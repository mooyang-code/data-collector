// Package klines 币安K线采集器实现
package klines

import (
	"context"
	"fmt"
	"time"

	"github.com/mooyang-code/data-collector/internal/datatype/klines"
	"github.com/mooyang-code/data-collector/internal/infra/timer"
	"trpc.group/trpc-go/trpc-go/log"
)

// BinanceKlineCollector 币安K线采集器
type BinanceKlineCollector struct {
	*klines.BaseKlineCollector
	adapter *BinanceKlineAdapter
	config  *BinanceKlineConfig
}

// NewBinanceKlineCollector 创建币安K线采集器
func NewBinanceKlineCollector(config *BinanceKlineConfig) (*BinanceKlineCollector, error) {
	if config == nil {
		config = DefaultBinanceKlineConfig()
	}
	
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}
	
	// 创建适配器
	adapter := NewBinanceKlineAdapter(config.BaseURL)
	
	// 创建基础采集器配置
	baseConfig := &klines.CollectorConfig{
		EnableTimer:       true,
		Timezone:          "UTC",
		MaxConcurrentJobs: 10,
		JobTimeout:        5 * time.Minute,
		MaxRetries:        config.RetryCount,
	}
	
	// 创建基础采集器
	baseCollector, err := klines.NewBaseKlineCollector(config.Exchange, config.BufferSize, baseConfig)
	if err != nil {
		return nil, fmt.Errorf("创建基础采集器失败: %w", err)
	}
	
	collector := &BinanceKlineCollector{
		BaseKlineCollector: baseCollector,
		adapter:            adapter,
		config:             config,
	}
	
	// 注册适配器
	klines.RegisterKlineAdapter(config.Exchange, adapter)
	
	return collector, nil
}

// Initialize 初始化采集器
func (c *BinanceKlineCollector) Initialize(ctx context.Context) error {
	log.Infof("初始化币安K线采集器...")
	
	// 为每个交易对和周期组合创建定时任务
	for _, symbol := range c.config.Symbols {
		for _, interval := range c.config.Intervals {
			timerConfig, exists := c.config.GetTimerConfig(interval)
			if !exists || !timerConfig.Enabled {
				log.Warnf("跳过未配置或未启用的周期: %s", interval)
				continue
			}
			
			// 创建获取K线数据的函数
			fetchFunc := c.createFetchFunc(symbol, interval)
			
			// 添加定时任务
			if err := c.AddTimerJob(symbol, string(interval), timerConfig.CronExpr, fetchFunc); err != nil {
				log.Errorf("添加定时任务失败 - symbol: %s, interval: %s, error: %v", symbol, interval, err)
				continue
			}
			
			log.Infof("添加定时任务成功 - symbol: %s, interval: %s, cron: %s", symbol, interval, timerConfig.CronExpr)
		}
	}
	
	log.Infof("币安K线采集器初始化完成")
	return nil
}

// createFetchFunc 创建获取K线数据的函数
func (c *BinanceKlineCollector) createFetchFunc(symbol string, interval klines.Interval) timer.JobFunc {
	return func(ctx context.Context) error {
		log.Debugf("开始获取K线数据 - symbol: %s, interval: %s", symbol, interval)
		
		// 计算时间范围（获取最近的数据）
		endTime := time.Now()
		startTime := endTime.Add(-interval.ToDuration() * time.Duration(c.config.HistoryLimit))
		
		// 使用适配器获取历史K线数据
		klineData, err := c.adapter.FetchHistoryKlines(ctx, symbol, interval, startTime, endTime, c.config.HistoryLimit)
		if err != nil {
			log.Errorf("获取K线数据失败 - symbol: %s, interval: %s, error: %v", symbol, interval, err)
			return err
		}
		
		// 处理获取到的数据
		for _, kline := range klineData {
			// 转换为KlineRecord格式
			record := kline.ToKlineRecord()
			
			// 发送事件
			c.Emit(record, "binance_timer")
			
			log.Debugf("发送K线数据 - symbol: %s, interval: %s, openTime: %s", 
				symbol, interval, kline.OpenTime.Format("2006-01-02 15:04:05"))
		}
		
		log.Debugf("完成获取K线数据 - symbol: %s, interval: %s, count: %d", symbol, interval, len(klineData))
		return nil
	}
}

// StartCollection 启动采集
func (c *BinanceKlineCollector) StartCollection(ctx context.Context) error {
	log.Infof("启动币安K线采集...")
	
	// 启动基础采集器
	if err := c.Start(ctx); err != nil {
		return fmt.Errorf("启动基础采集器失败: %w", err)
	}
	
	// 如果启用回补，执行历史数据回补
	if c.config.EnableBackfill {
		go c.backfillHistoryData(ctx)
	}
	
	log.Infof("币安K线采集启动完成")
	return nil
}

// backfillHistoryData 回补历史数据
func (c *BinanceKlineCollector) backfillHistoryData(ctx context.Context) {
	log.Infof("开始回补历史数据...")
	
	for _, symbol := range c.config.Symbols {
		for _, interval := range c.config.Intervals {
			select {
			case <-ctx.Done():
				log.Infof("历史数据回补被取消")
				return
			default:
			}
			
			// 计算回补时间范围
			endTime := time.Now()
			startTime := endTime.AddDate(0, 0, -c.config.BackfillDays)
			
			log.Infof("回补历史数据 - symbol: %s, interval: %s, 时间范围: %s ~ %s", 
				symbol, interval, startTime.Format("2006-01-02"), endTime.Format("2006-01-02"))
			
			// 分批获取历史数据
			batchSize := c.config.HistoryLimit
			currentTime := startTime
			
			for currentTime.Before(endTime) {
				batchEndTime := currentTime.Add(time.Duration(batchSize) * interval.ToDuration())
				if batchEndTime.After(endTime) {
					batchEndTime = endTime
				}
				
				// 获取这一批数据
				klineData, err := c.adapter.FetchHistoryKlines(ctx, symbol, interval, currentTime, batchEndTime, batchSize)
				if err != nil {
					log.Errorf("回补历史数据失败 - symbol: %s, interval: %s, error: %v", symbol, interval, err)
					break
				}
				
				// 处理数据
				for _, kline := range klineData {
					record := kline.ToKlineRecord()
					c.Emit(record, "binance_backfill")
				}
				
				log.Debugf("回补历史数据批次完成 - symbol: %s, interval: %s, count: %d", symbol, interval, len(klineData))
				
				// 更新时间
				currentTime = batchEndTime
				
				// 避免请求过于频繁
				time.Sleep(c.config.RetryInterval)
			}
		}
	}
	
	log.Infof("历史数据回补完成")
}

// StopCollection 停止采集
func (c *BinanceKlineCollector) StopCollection(ctx context.Context) error {
	log.Infof("停止币安K线采集...")
	
	// 停止基础采集器
	if err := c.Close(); err != nil {
		return fmt.Errorf("停止基础采集器失败: %w", err)
	}
	
	log.Infof("币安K线采集停止完成")
	return nil
}

// GetConfig 获取配置
func (c *BinanceKlineCollector) GetConfig() *BinanceKlineConfig {
	return c.config
}

// GetAdapter 获取适配器
func (c *BinanceKlineCollector) GetAdapter() *BinanceKlineAdapter {
	return c.adapter
}

// UpdateConfig 更新配置
func (c *BinanceKlineCollector) UpdateConfig(config *BinanceKlineConfig) error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}
	
	c.config = config
	
	// 更新适配器配置
	c.adapter = NewBinanceKlineAdapter(config.BaseURL)
	
	log.Infof("币安K线采集器配置更新完成")
	return nil
}
