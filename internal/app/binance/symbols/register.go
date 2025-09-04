// Package symbols 币安交易对采集器自注册（最好是中文注释！）
package symbols

import (
	"fmt"
	"time"

	"github.com/mooyang-code/data-collector/configs"
	"github.com/mooyang-code/data-collector/internal/app"
	"trpc.group/trpc-go/trpc-go/log"
)

// init 函数在包被导入时自动执行，注册采集器创建器
func init() {
	// 注册现货交易对采集器
	app.RegisterCollectorCreator("binance", "symbols", "spot", createBinanceSpotSymbolCollector)
	
	// 注册合约交易对采集器
	app.RegisterCollectorCreator("binance", "symbols", "futures", createBinanceFuturesSymbolCollector)
	
	log.Info("币安交易对采集器注册完成")
}

// createBinanceSpotSymbolCollector 创建币安现货交易对采集器
func createBinanceSpotSymbolCollector(appName, collectorName string, config *configs.Collector) (app.Collector, error) {
	// 转换为币安交易对采集器配置
	binanceConfig := &BinanceSymbolConfig{
		Exchange:           "binance",
		BaseURL:           "https://api.binance.com", // 现货API地址
		BufferSize:        1000,
		EnableAutoRefresh: config.Schedule.EnableAutoRefresh,
		RefreshInterval:   5 * time.Minute,
		EnableFiltering:   len(config.Config.Filters) > 0,
		AllowedQuotes:     config.Config.Filters,
	}

	// 解析触发间隔
	if config.Schedule.TriggerInterval != "" {
		if interval, err := time.ParseDuration(config.Schedule.TriggerInterval); err == nil {
			binanceConfig.RefreshInterval = interval
		}
	}

	// 创建具体的采集器实例
	collector, err := NewBinanceSymbolCollector(binanceConfig)
	if err != nil {
		return nil, fmt.Errorf("创建币安现货交易对采集器失败: %w", err)
	}

	// 包装为通用接口
	wrapper := &BinanceSymbolCollectorWrapper{
		collector:     collector,
		id:           fmt.Sprintf("%s.%s", appName, collectorName),
		collectorType: "binance.symbols.spot",
		dataType:     config.DataType,
	}

	return wrapper, nil
}

// createBinanceFuturesSymbolCollector 创建币安合约交易对采集器
func createBinanceFuturesSymbolCollector(appName, collectorName string, config *configs.Collector) (app.Collector, error) {
	// 转换为币安交易对采集器配置（合约版本）
	binanceConfig := &BinanceSymbolConfig{
		Exchange:           "binance",
		BaseURL:           "https://fapi.binance.com", // 合约API地址
		BufferSize:        1000,
		EnableAutoRefresh: config.Schedule.EnableAutoRefresh,
		RefreshInterval:   5 * time.Minute,
		EnableFiltering:   len(config.Config.Filters) > 0,
		AllowedQuotes:     config.Config.Filters,
	}

	// 解析触发间隔
	if config.Schedule.TriggerInterval != "" {
		if interval, err := time.ParseDuration(config.Schedule.TriggerInterval); err == nil {
			binanceConfig.RefreshInterval = interval
		}
	}

	// 创建具体的采集器实例
	collector, err := NewBinanceSymbolCollector(binanceConfig)
	if err != nil {
		return nil, fmt.Errorf("创建币安合约交易对采集器失败: %w", err)
	}

	// 包装为通用接口
	wrapper := &BinanceSymbolCollectorWrapper{
		collector:     collector,
		id:           fmt.Sprintf("%s.%s", appName, collectorName),
		collectorType: "binance.symbols.futures",
		dataType:     config.DataType,
	}

	return wrapper, nil
}

// BinanceSymbolCollectorWrapper 币安交易对采集器包装器
type BinanceSymbolCollectorWrapper struct {
	collector     *BinanceSymbolCollector
	id           string
	collectorType string
	dataType     string
	running      bool
}

// Initialize 初始化采集器
func (w *BinanceSymbolCollectorWrapper) Initialize(ctx context.Context) error {
	return w.collector.Initialize(ctx)
}

// StartCollection 启动采集
func (w *BinanceSymbolCollectorWrapper) StartCollection(ctx context.Context) error {
	err := w.collector.StartCollection(ctx)
	if err == nil {
		w.running = true
	}
	return err
}

// StopCollection 停止采集
func (w *BinanceSymbolCollectorWrapper) StopCollection(ctx context.Context) error {
	err := w.collector.StopCollection(ctx)
	if err == nil {
		w.running = false
	}
	return err
}

// IsRunning 检查是否运行中
func (w *BinanceSymbolCollectorWrapper) IsRunning() bool {
	return w.running
}

// GetID 获取采集器ID
func (w *BinanceSymbolCollectorWrapper) GetID() string {
	return w.id
}

// GetType 获取采集器类型
func (w *BinanceSymbolCollectorWrapper) GetType() string {
	return w.collectorType
}

// GetDataType 获取数据类型
func (w *BinanceSymbolCollectorWrapper) GetDataType() string {
	return w.dataType
}
