package event

import (
	"context"
	"time"
)

// Event 事件接口
type Event interface {
	ID() string
	Type() string           // 事件类型，如 "data.kline.collected"
	Source() string         // 事件源，如 "binance.kline.collector"
	Timestamp() time.Time
	Data() interface{}
	Context() context.Context
}

// BaseEvent 基础事件实现
type BaseEvent struct {
	id        string
	eventType string
	source    string
	timestamp time.Time
	data      interface{}
	ctx       context.Context
}

func NewEvent(eventType, source string, data interface{}) Event {
	return &BaseEvent{
		id:        generateEventID(),
		eventType: eventType,
		source:    source,
		timestamp: time.Now(),
		data:      data,
		ctx:       context.Background(),
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

func (e *BaseEvent) Context() context.Context {
	return e.ctx
}

// 事件类型常量
const (
	// 数据事件
	EventDataCollected = "data.*.collected"
	EventDataProcessed = "data.*.processed"
	EventDataStored    = "data.*.stored"
	
	// 具体数据类型事件
	EventKlineCollected     = "data.kline.collected"
	EventTickerCollected    = "data.ticker.collected"
	EventOrderBookCollected = "data.orderbook.collected"
	EventTradeCollected     = "data.trade.collected"
	
	// 系统事件
	EventAppStarted      = "app.*.started"
	EventAppStopped      = "app.*.stopped"
	EventCollectorError  = "collector.*.error"
	
	// 分析事件
	EventAnomalyDetected = "analysis.anomaly.*"
	EventSignalGenerated = "analysis.signal.*"
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