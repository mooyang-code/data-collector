// Package symbols OKX交易对采集器自注册示例（最好是中文注释！）
package symbols

import (
	"context"
	"fmt"
	"time"

	"github.com/mooyang-code/data-collector/configs"
	"github.com/mooyang-code/data-collector/internal/app"
	"trpc.group/trpc-go/trpc-go/log"
)

// init 函数在包被导入时自动执行，注册采集器创建器
func init() {
	// 注册OKX现货交易对采集器
	app.RegisterCollectorCreator("okx", "symbols", "spot", createOKXSpotSymbolCollector)
	
	// 注册OKX合约交易对采集器
	app.RegisterCollectorCreator("okx", "symbols", "futures", createOKXFuturesSymbolCollector)
	
	log.Info("OKX交易对采集器注册完成")
}

// createOKXSpotSymbolCollector 创建OKX现货交易对采集器
func createOKXSpotSymbolCollector(appName, collectorName string, config *configs.Collector) (app.Collector, error) {
	// 这里是示例实现，实际需要根据OKX的API实现
	log.Infof("创建OKX现货交易对采集器: %s.%s", appName, collectorName)
	
	// 创建OKX采集器实例（这里用模拟实现）
	collector := &OKXSymbolCollector{
		exchange:   "okx",
		marketType: "spot",
		baseURL:    "https://www.okx.com/api/v5",
		config:     config,
	}
	
	// 包装为通用接口
	wrapper := &OKXSymbolCollectorWrapper{
		collector:     collector,
		id:           fmt.Sprintf("%s.%s", appName, collectorName),
		collectorType: "okx.symbols.spot",
		dataType:     config.DataType,
	}
	
	return wrapper, nil
}

// createOKXFuturesSymbolCollector 创建OKX合约交易对采集器
func createOKXFuturesSymbolCollector(appName, collectorName string, config *configs.Collector) (app.Collector, error) {
	// 这里是示例实现，实际需要根据OKX的API实现
	log.Infof("创建OKX合约交易对采集器: %s.%s", appName, collectorName)
	
	// 创建OKX采集器实例（这里用模拟实现）
	collector := &OKXSymbolCollector{
		exchange:   "okx",
		marketType: "futures",
		baseURL:    "https://www.okx.com/api/v5",
		config:     config,
	}
	
	// 包装为通用接口
	wrapper := &OKXSymbolCollectorWrapper{
		collector:     collector,
		id:           fmt.Sprintf("%s.%s", appName, collectorName),
		collectorType: "okx.symbols.futures",
		dataType:     config.DataType,
	}
	
	return wrapper, nil
}

// OKXSymbolCollector OKX交易对采集器（示例实现）
type OKXSymbolCollector struct {
	exchange   string
	marketType string
	baseURL    string
	config     *configs.Collector
	running    bool
}

// Initialize 初始化采集器
func (c *OKXSymbolCollector) Initialize(ctx context.Context) error {
	log.Infof("初始化OKX交易对采集器: %s.%s", c.exchange, c.marketType)
	// 这里可以添加实际的初始化逻辑
	return nil
}

// StartCollection 启动采集
func (c *OKXSymbolCollector) StartCollection(ctx context.Context) error {
	log.Infof("启动OKX交易对采集: %s.%s", c.exchange, c.marketType)
	c.running = true
	// 这里可以添加实际的采集逻辑
	return nil
}

// StopCollection 停止采集
func (c *OKXSymbolCollector) StopCollection(ctx context.Context) error {
	log.Infof("停止OKX交易对采集: %s.%s", c.exchange, c.marketType)
	c.running = false
	// 这里可以添加实际的停止逻辑
	return nil
}

// IsRunning 检查是否运行中
func (c *OKXSymbolCollector) IsRunning() bool {
	return c.running
}

// OKXSymbolCollectorWrapper OKX交易对采集器包装器
type OKXSymbolCollectorWrapper struct {
	collector     *OKXSymbolCollector
	id           string
	collectorType string
	dataType     string
	running      bool
}

// Initialize 初始化采集器
func (w *OKXSymbolCollectorWrapper) Initialize(ctx context.Context) error {
	return w.collector.Initialize(ctx)
}

// StartCollection 启动采集
func (w *OKXSymbolCollectorWrapper) StartCollection(ctx context.Context) error {
	err := w.collector.StartCollection(ctx)
	if err == nil {
		w.running = true
	}
	return err
}

// StopCollection 停止采集
func (w *OKXSymbolCollectorWrapper) StopCollection(ctx context.Context) error {
	err := w.collector.StopCollection(ctx)
	if err == nil {
		w.running = false
	}
	return err
}

// IsRunning 检查是否运行中
func (w *OKXSymbolCollectorWrapper) IsRunning() bool {
	return w.running
}

// GetID 获取采集器ID
func (w *OKXSymbolCollectorWrapper) GetID() string {
	return w.id
}

// GetType 获取采集器类型
func (w *OKXSymbolCollectorWrapper) GetType() string {
	return w.collectorType
}

// GetDataType 获取数据类型
func (w *OKXSymbolCollectorWrapper) GetDataType() string {
	return w.dataType
}
