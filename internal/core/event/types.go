package event

import (
	"time"
)

// Event 事件接口
type Event interface {
	ID() string
	Type() string           // 事件类型，如 "data.kline.collected"
	Source() string         // 事件源，如 "binance.kline.collector"
	Timestamp() time.Time
	Data() interface{}
}

// BaseEvent 基础事件实现
type BaseEvent struct {
	id        string
	eventType string
	source    string
	timestamp time.Time
	data      interface{}
}

func NewEvent(eventType, source string, data interface{}) Event {
	return &BaseEvent{
		id:        generateEventID(),
		eventType: eventType,
		source:    source,
		timestamp: time.Now(),
		data:      data,
	}
}

func (e *BaseEvent) ID() string {
	return e.id
}

func (e *BaseEvent) Type() string {
	return e.eventType
}

func (e *BaseEvent) Source() string {
	return e.source
}

func (e *BaseEvent) Timestamp() time.Time {
	return e.timestamp
}

func (e *BaseEvent) Data() interface{} {
	return e.data
}

// 事件类型常量
const (
	EventDataCollected = "data.*.collected"       // EventDataCollected 数据采集完成事件
	EventDataProcessed = "data.*.processed"       // EventDataProcessed 数据处理完成事件
	EventDataStored    = "data.*.stored"          // EventDataStored 数据存储完成事件

	EventKlineCollected     = "data.kline.collected"     // EventKlineCollected K线数据采集事件
	EventTickerCollected    = "data.ticker.collected"    // EventTickerCollected 行情数据采集事件
	EventOrderBookCollected = "data.orderbook.collected" // EventOrderBookCollected 订单簿数据采集事件
	EventTradeCollected     = "data.trade.collected"     // EventTradeCollected 交易数据采集事件

	EventAppStarted      = "app.*.started"      // EventAppStarted 应用启动事件
	EventAppStopped      = "app.*.stopped"      // EventAppStopped 应用停止事件
	EventCollectorError  = "collector.*.error" // EventCollectorError 采集器错误事件

	EventAnomalyDetected = "analysis.anomaly.*" // EventAnomalyDetected 异常检测事件
	EventSignalGenerated = "analysis.signal.*" // EventSignalGenerated 信号生成事件
)

// DataEvent 数据事件
type DataEvent struct {
	BaseEvent
	Exchange   string      `json:"exchange"`
	Symbol     string      `json:"symbol"`
	DataType   string      `json:"data_type"`
	Count      int         `json:"count"`
	RawData    interface{} `json:"raw_data"`
}

// ErrorEvent 错误事件
type ErrorEvent struct {
	BaseEvent
	Error   error  `json:"error"`
	Message string `json:"message"`
	Level   string `json:"level"` // info, warning, error, critical
}

// SystemEvent 系统事件
type SystemEvent struct {
	BaseEvent
	Action  string                 `json:"action"`
	Details map[string]interface{} `json:"details"`
}

// 辅助函数
func generateEventID() string {
	// 简化实现，实际应使用 UUID 或其他唯一ID生成器
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, n)
	for i := range result {
		result[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(result)
}