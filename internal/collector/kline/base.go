// Package kline K线数据采集器基类
package kline

import (
	"context"
	"fmt"
	"time"

	"github.com/mooyang-code/data-collector/internal/core/collector"
	"github.com/mooyang-code/data-collector/internal/datatype/klines"
	"trpc.group/trpc-go/trpc-go/log"
)

// Config K线采集器配置
type Config struct {
	Symbols        []string           // 交易对列表
	Intervals      []klines.Interval  // K线周期
	EnableBackfill bool              // 是否启用历史数据回补
	BackfillDays   int               // 回补天数
	BatchSize      int               // 批量大小
	RateLimit      time.Duration     // 速率限制
}

// BaseKlineCollector K线采集器基类
type BaseKlineCollector struct {
	*collector.BaseCollector
	
	config    Config
	dataStore DataStore
	api       ExchangeAPI
}

// DataStore 数据存储接口
type DataStore interface {
	SaveKlines(symbol string, interval klines.Interval, data []*klines.Kline) error
	GetLastKline(symbol string, interval klines.Interval) (*klines.Kline, error)
}

// ExchangeAPI 交易所API接口
type ExchangeAPI interface {
	GetKlines(ctx context.Context, symbol string, interval klines.Interval, start, end time.Time) ([]*klines.Kline, error)
	GetExchangeTime(ctx context.Context) (time.Time, error)
}

// NewBaseKlineCollector 创建K线采集器基类
func NewBaseKlineCollector(id, exchange string, config Config, api ExchangeAPI, store DataStore) *BaseKlineCollector {
	return &BaseKlineCollector{
		BaseCollector: collector.NewBaseCollector(id, "kline", exchange),
		config:        config,
		api:           api,
		dataStore:     store,
	}
}

// Initialize 初始化
func (c *BaseKlineCollector) Initialize(ctx context.Context) error {
	// 调用基类初始化
	if err := c.BaseCollector.Initialize(ctx); err != nil {
		return err
	}
	
	// 注册定时器
	for _, interval := range c.config.Intervals {
		timerName := fmt.Sprintf("collect_%s", interval)
		timerInterval := c.getTimerInterval(interval)
		
		err := c.AddTimer(timerName, timerInterval, c.createCollectHandler(interval))
		if err != nil {
			return fmt.Errorf("添加定时器失败 %s: %w", timerName, err)
		}
	}
	
	// 如果启用回补，添加回补定时器
	if c.config.EnableBackfill {
		err := c.AddTimer("backfill", 1*time.Hour, c.backfillHandler)
		if err != nil {
			return fmt.Errorf("添加回补定时器失败: %w", err)
		}
	}
	
	log.Infof("K线采集器初始化完成: %s", c.ID())
	return nil
}

// getTimerInterval 根据K线周期获取定时器间隔
func (c *BaseKlineCollector) getTimerInterval(interval klines.Interval) time.Duration {
	switch interval {
	case klines.Interval1m:
		return 1 * time.Minute
	case klines.Interval5m:
		return 5 * time.Minute
	case klines.Interval15m:
		return 15 * time.Minute
	case klines.Interval30m:
		return 30 * time.Minute
	case klines.Interval1h:
		return 1 * time.Hour
	case klines.Interval4h:
		return 4 * time.Hour
	case klines.Interval1d:
		return 24 * time.Hour
	default:
		return 1 * time.Hour
	}
}

// createCollectHandler 创建采集处理函数
func (c *BaseKlineCollector) createCollectHandler(interval klines.Interval) collector.TimerHandler {
	return func(ctx context.Context) error {
		log.Debugf("开始采集K线数据: %s, interval: %s", c.ID(), interval)
		
		successCount := 0
		errorCount := 0
		
		// 批量处理交易对
		for i := 0; i < len(c.config.Symbols); i += c.config.BatchSize {
			end := i + c.config.BatchSize
			if end > len(c.config.Symbols) {
				end = len(c.config.Symbols)
			}
			
			batch := c.config.Symbols[i:end]
			
			for _, symbol := range batch {
				if err := c.collectSymbolKlines(ctx, symbol, interval); err != nil {
					log.Errorf("采集K线失败: %s %s, error: %v", symbol, interval, err)
					errorCount++
				} else {
					successCount++
				}
				
				// 速率限制
				if c.config.RateLimit > 0 {
					time.Sleep(c.config.RateLimit)
				}
			}
		}
		
		// 更新指标
		c.UpdateMetrics(func(m *collector.Metrics) {
			m.DataPoints += int64(successCount)
			m.ErrorCount += int64(errorCount)
			m.LastDataTime = time.Now()
			m.Custom["success_rate"] = float64(successCount) / float64(successCount+errorCount)
		})
		
		log.Infof("K线采集完成: %s, 成功: %d, 失败: %d", c.ID(), successCount, errorCount)
		return nil
	}
}

// collectSymbolKlines 采集单个交易对的K线数据
func (c *BaseKlineCollector) collectSymbolKlines(ctx context.Context, symbol string, interval klines.Interval) error {
	// 获取最后一条K线
	lastKline, err := c.dataStore.GetLastKline(symbol, interval)
	if err != nil && err.Error() != "not found" {
		return fmt.Errorf("获取最后K线失败: %w", err)
	}
	
	// 确定时间范围
	endTime := time.Now()
	startTime := endTime.Add(-c.getTimerInterval(interval) * 2) // 获取两个周期的数据
	
	if lastKline != nil {
		startTime = lastKline.OpenTime.Add(c.getTimerInterval(interval))
	}
	
	// 调用API获取数据
	klineData, err := c.api.GetKlines(ctx, symbol, interval, startTime, endTime)
	if err != nil {
		return fmt.Errorf("API获取K线失败: %w", err)
	}
	
	if len(klineData) == 0 {
		return nil
	}
	
	// 保存数据
	if err := c.dataStore.SaveKlines(symbol, interval, klineData); err != nil {
		return fmt.Errorf("保存K线失败: %w", err)
	}
	
	log.Debugf("成功采集K线: %s %s, 数量: %d", symbol, interval, len(klineData))
	return nil
}

// backfillHandler 历史数据回补处理函数
func (c *BaseKlineCollector) backfillHandler(ctx context.Context) error {
	log.Infof("开始执行历史数据回补: %s", c.ID())
	
	for _, symbol := range c.config.Symbols {
		for _, interval := range c.config.Intervals {
			if err := c.backfillSymbol(ctx, symbol, interval); err != nil {
				log.Errorf("回补失败: %s %s, error: %v", symbol, interval, err)
			}
			
			// 速率限制
			if c.config.RateLimit > 0 {
				time.Sleep(c.config.RateLimit)
			}
		}
	}
	
	log.Infof("历史数据回补完成: %s", c.ID())
	return nil
}

// backfillSymbol 回补单个交易对的历史数据
func (c *BaseKlineCollector) backfillSymbol(ctx context.Context, symbol string, interval klines.Interval) error {
	// 获取最早的K线时间
	lastKline, err := c.dataStore.GetLastKline(symbol, interval)
	if err != nil && err.Error() != "not found" {
		return fmt.Errorf("获取最后K线失败: %w", err)
	}
	
	// 确定回补的时间范围
	endTime := time.Now()
	if lastKline != nil {
		endTime = lastKline.OpenTime
	}
	
	startTime := endTime.AddDate(0, 0, -c.config.BackfillDays)
	
	// 分批获取数据
	batchDuration := c.getTimerInterval(interval) * 1000 // 每批1000条
	
	for currentEnd := endTime; currentEnd.After(startTime); {
		currentStart := currentEnd.Add(-batchDuration)
		if currentStart.Before(startTime) {
			currentStart = startTime
		}
		
		// 获取K线数据
		klineData, err := c.api.GetKlines(ctx, symbol, interval, currentStart, currentEnd)
		if err != nil {
			return fmt.Errorf("回补获取K线失败: %w", err)
		}
		
		if len(klineData) > 0 {
			// 保存数据
			if err := c.dataStore.SaveKlines(symbol, interval, klineData); err != nil {
				return fmt.Errorf("回补保存K线失败: %w", err)
			}
			
			log.Debugf("回补K线成功: %s %s, 时间: %s - %s, 数量: %d", 
				symbol, interval, currentStart.Format(time.RFC3339), 
				currentEnd.Format(time.RFC3339), len(klineData))
		}
		
		// 更新时间窗口
		currentEnd = currentStart
		
		// 速率限制
		if c.config.RateLimit > 0 {
			time.Sleep(c.config.RateLimit)
		}
	}
	
	return nil
}