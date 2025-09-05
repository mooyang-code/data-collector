// Package klines 币安K线采集器自注册
package klines

import (
	"context"
	"fmt"
	"time"

	"github.com/mooyang-code/data-collector/configs"
	"github.com/mooyang-code/data-collector/internal/app"
	"github.com/mooyang-code/data-collector/internal/datatype/klines"
	"trpc.group/trpc-go/trpc-go/log"
)

// init 函数在包被导入时自动执行，注册采集器创建器
func init() {
	// 注册现货K线采集器
	err := app.NewCollectorCreatorBuilder().
		WithExchange("binance", "币安").
		WithDataType("klines", "K线").
		WithMarketType("spot", "现货").
		WithDescription("采集币安现货市场的K线数据").
		WithCreator(createBinanceSpotKlineCollector).
		Register()
	if err != nil {
		log.Errorf("注册币安现货K线采集器失败: %v", err)
	}

	// 注册合约K线采集器
	err = app.NewCollectorCreatorBuilder().
		WithExchange("binance", "币安").
		WithDataType("klines", "K线").
		WithMarketType("futures", "合约").
		WithDescription("采集币安合约市场的K线数据").
		WithCreator(createBinanceFuturesKlineCollector).
		Register()
	if err != nil {
		log.Errorf("注册币安合约K线采集器失败: %v", err)
	}
	
	log.Info("币安K线采集器注册完成")
}

// createBinanceSpotKlineCollector 创建币安现货K线采集器
func createBinanceSpotKlineCollector(appName, collectorName string, config *configs.Collector) (app.Collector, error) {
	// 转换为币安K线采集器配置
	binanceConfig := &BinanceKlineConfig{
		Exchange:       "binance",
		BaseURL:        "https://api.binance.com", // 现货API地址
		BufferSize:     config.Config.BufferSize,
		Symbols:        config.Config.Filters, // 使用过滤器作为交易对列表
		Intervals:      convertToKlineIntervals(config.Config.Intervals),
		HistoryLimit:   config.Config.MaxKlinesPerSymbol,
		EnableBackfill: config.Config.EnableBackfill,
		BackfillDays:   7, // 默认回补7天
		RetryCount:     config.Schedule.MaxRetries,
		RetryInterval:  30 * time.Second,
	}

	// 解析重试间隔
	if config.Schedule.RetryInterval != "" {
		if interval, err := time.ParseDuration(config.Schedule.RetryInterval); err == nil {
			binanceConfig.RetryInterval = interval
		}
	}

	// 解析回补时间
	if config.Config.BackfillLookback != "" {
		if duration, err := time.ParseDuration(config.Config.BackfillLookback); err == nil {
			binanceConfig.BackfillDays = int(duration.Hours() / 24)
		}
	}

	// 创建具体的采集器实例
	collector, err := NewBinanceKlineCollector(binanceConfig)
	if err != nil {
		return nil, fmt.Errorf("创建币安现货K线采集器失败: %w", err)
	}

	// 包装为通用接口
	wrapper := &BinanceKlineCollectorWrapper{
		collector:     collector,
		id:            fmt.Sprintf("%s.%s", appName, collectorName),
		collectorType: "binance.klines.spot",
		dataType:      config.DataType,
	}
	return wrapper, nil
}

// createBinanceFuturesKlineCollector 创建币安合约K线采集器
func createBinanceFuturesKlineCollector(appName, collectorName string, config *configs.Collector) (app.Collector, error) {
	// 转换为币安K线采集器配置（合约版本）
	binanceConfig := &BinanceKlineConfig{
		Exchange:       "binance",
		BaseURL:        "https://fapi.binance.com", // 合约API地址
		BufferSize:     config.Config.BufferSize,
		Symbols:        config.Config.Filters, // 使用过滤器作为交易对列表
		Intervals:      convertToKlineIntervals(config.Config.Intervals),
		HistoryLimit:   config.Config.MaxKlinesPerSymbol,
		EnableBackfill: config.Config.EnableBackfill,
		BackfillDays:   7, // 默认回补7天
		RetryCount:     config.Schedule.MaxRetries,
		RetryInterval:  30 * time.Second,
	}

	// 解析重试间隔
	if config.Schedule.RetryInterval != "" {
		if interval, err := time.ParseDuration(config.Schedule.RetryInterval); err == nil {
			binanceConfig.RetryInterval = interval
		}
	}

	// 解析回补时间
	if config.Config.BackfillLookback != "" {
		if duration, err := time.ParseDuration(config.Config.BackfillLookback); err == nil {
			binanceConfig.BackfillDays = int(duration.Hours() / 24)
		}
	}

	// 创建具体的采集器实例
	collector, err := NewBinanceKlineCollector(binanceConfig)
	if err != nil {
		return nil, fmt.Errorf("创建币安合约K线采集器失败: %w", err)
	}

	// 包装为通用接口
	wrapper := &BinanceKlineCollectorWrapper{
		collector:     collector,
		id:            fmt.Sprintf("%s.%s", appName, collectorName),
		collectorType: "binance.klines.futures",
		dataType:      config.DataType,
	}
	return wrapper, nil
}

// convertToKlineIntervals 转换K线间隔
func convertToKlineIntervals(intervals []string) []klines.Interval {
	result := make([]klines.Interval, 0, len(intervals))
	for _, interval := range intervals {
		switch interval {
		case "1m":
			result = append(result, klines.Interval1m)
		case "5m":
			result = append(result, klines.Interval5m)
		case "15m":
			result = append(result, klines.Interval15m)
		case "30m":
			result = append(result, klines.Interval30m)
		case "1h":
			result = append(result, klines.Interval1h)
		case "4h":
			result = append(result, klines.Interval4h)
		case "1d":
			result = append(result, klines.Interval1d)
		default:
			log.Warnf("不支持的K线间隔: %s", interval)
		}
	}
	return result
}

// BinanceKlineCollectorWrapper 币安K线采集器包装器
type BinanceKlineCollectorWrapper struct {
	collector     *BinanceKlineCollector
	id            string
	collectorType string
	dataType      string
	running       bool
}

// Initialize 初始化采集器
func (w *BinanceKlineCollectorWrapper) Initialize(ctx context.Context) error {
	return w.collector.Initialize(ctx)
}

// StartCollection 启动采集
func (w *BinanceKlineCollectorWrapper) StartCollection(ctx context.Context) error {
	err := w.collector.StartCollection(ctx)
	if err == nil {
		w.running = true
	}
	return err
}

// StopCollection 停止采集
func (w *BinanceKlineCollectorWrapper) StopCollection(ctx context.Context) error {
	err := w.collector.StopCollection(ctx)
	if err == nil {
		w.running = false
	}
	return err
}

// IsRunning 检查是否运行中
func (w *BinanceKlineCollectorWrapper) IsRunning() bool {
	return w.running
}

// GetID 获取采集器ID
func (w *BinanceKlineCollectorWrapper) GetID() string {
	return w.id
}

// GetType 获取采集器类型
func (w *BinanceKlineCollectorWrapper) GetType() string {
	return w.collectorType
}

// GetDataType 获取数据类型
func (w *BinanceKlineCollectorWrapper) GetDataType() string {
	return w.dataType
}
