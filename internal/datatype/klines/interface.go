package klines

import (
	"context"
	"errors"
	"time"
)

var (
	ErrClosed         = errors.New("collector_closed")
	ErrAlreadyStarted = errors.New("collector_already_started")
	ErrInvalidParam   = errors.New("invalid_param")
	ErrNotSupported   = errors.New("not_supported")
)

// KlineCollector K线采集器接口
type KlineCollector interface {
	// Start 启动采集器
	Start(ctx context.Context) error
	
	// Close 关闭采集器
	Close() error

	// Subscribe 订阅单个 (symbol, interval)
	Subscribe(symbol, interval string) error
	
	// Unsubscribe 取消订阅单个 (symbol, interval)
	Unsubscribe(symbol, interval string) error

	// Subscriptions 当前订阅快照
	Subscriptions() []SymbolInterval

	// GetKlines 获取历史K线
	GetKlines(ctx context.Context, query KlineQuery) ([]*KlineRecord, error)

	// Events 实时事件通道
	Events() <-chan KlineEvent

	// GetRateLimit 速率限制信息（若无可返回 nil）
	GetRateLimit() *RateLimit
}

// 速率限制结构
type RateLimit struct {
	RequestsPerSecond int
	RequestsPerMinute int
	RequestsPerHour   int
	LastRequest       time.Time
	RequestCount      int
}

// (symbol, interval) 组合
type SymbolInterval struct {
	Symbol   string
	Interval string
}

// 历史查询条件
type KlineQuery struct {
	Symbol   string
	Interval string
	StartMs  int64 // 0 表示未指定
	EndMs    int64 // 0 表示未指定
	Limit    int   // 0 表示不限制
}

// 单条 K 线
type KlineRecord struct {
	Exchange    string
	Symbol      string
	Interval    string
	OpenTimeMs  int64
	CloseTimeMs int64
	Open, High, Low, Close string
	Volume string
	Closed bool
}

// 实时事件
type KlineEvent struct {
	Record *KlineRecord
	Source string    // "ws" / "rest" / ...
	Ts     time.Time // 本地接收时间
}

// KlineAdapter K线适配器接口
type KlineAdapter interface {
	// GetExchange 获取交易所名称
	GetExchange() string
	
	// SubscribeKlines 订阅实时K线
	SubscribeKlines(ctx context.Context, subscriptions []*KlineSubscription, eventCh chan<- *KlineEvent) error
	
	// UnsubscribeKlines 取消订阅
	UnsubscribeKlines(ctx context.Context, subscriptions []*KlineSubscription) error
	
	// FetchHistoryKlines 获取历史K线
	FetchHistoryKlines(ctx context.Context, symbol string, interval Interval, startTime, endTime time.Time, limit int) ([]*Kline, error)
	
	// GetSupportedIntervals 获取支持的周期
	GetSupportedIntervals() []Interval
	
	// IsSymbolSupported 检查是否支持该交易对
	IsSymbolSupported(symbol string) bool
}



// EventProcessor K线事件处理器接口
type EventProcessor interface {
	// ProcessEvent 处理事件
	ProcessEvent(ctx context.Context, event *KlineEvent) error
	
	// GetName 获取处理器名称
	GetName() string
}

// Registry K线适配器注册表
type Registry struct {
	adapters map[string]KlineAdapter
}

// NewRegistry 创建新的注册表
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[string]KlineAdapter),
	}
}

// Register 注册适配器
func (r *Registry) Register(exchange string, adapter KlineAdapter) {
	r.adapters[exchange] = adapter
}

// Get 获取适配器
func (r *Registry) Get(exchange string) (KlineAdapter, bool) {
	adapter, exists := r.adapters[exchange]
	return adapter, exists
}

// List 列出所有注册的交易所
func (r *Registry) List() []string {
	exchanges := make([]string, 0, len(r.adapters))
	for exchange := range r.adapters {
		exchanges = append(exchanges, exchange)
	}
	return exchanges
}

// 全局注册表
var globalKlineRegistry = NewRegistry()

// RegisterKlineAdapter 注册K线适配器
func RegisterKlineAdapter(exchange string, adapter KlineAdapter) {
	globalKlineRegistry.Register(exchange, adapter)
}

// GetKlineAdapter 获取K线适配器
func GetKlineAdapter(exchange string) (KlineAdapter, bool) {
	return globalKlineRegistry.Get(exchange)
}

// ListRegisteredKlineExchanges 列出已注册的K线交易所
func ListRegisteredKlineExchanges() []string {
	return globalKlineRegistry.List()
}
