package symbols

import (
	"context"
	"errors"
	"time"
)

var (
	ErrClosed         = errors.New("collector_closed")
	ErrAlreadyStarted = errors.New("collector_already_started")
	ErrNotSupported   = errors.New("not_supported")
	ErrNotFound       = errors.New("symbol_not_found")
)

// RateLimit 速率限制结构
type RateLimit struct {
	RequestsPerSecond int
	RequestsPerMinute int
	RequestsPerHour   int
	LastRequest       time.Time
	RequestCount      int
}

// SymbolEventType 事件类型
type SymbolEventType int

const (
	SymbolAdded SymbolEventType = iota + 1
	SymbolUpdated
	SymbolRemoved
	SnapshotEnd
)

// String 返回事件类型字符串
func (t SymbolEventType) String() string {
	switch t {
	case SymbolAdded:
		return "added"
	case SymbolUpdated:
		return "updated"
	case SymbolRemoved:
		return "removed"
	case SnapshotEnd:
		return "snapshot_end"
	default:
		return "unknown"
	}
}

// SymbolEvent 交易对事件
type SymbolEvent struct {
	Type   SymbolEventType `json:"type"`   // 事件类型
	Symbol *SymbolMeta     `json:"symbol"` // 交易对信息
	Ts     time.Time       `json:"ts"`     // 事件时间
	Source string          `json:"source"` // 数据源 (rest/ws/internal)
}

// SymbolsCollector 交易对采集器接口（扩展版本）
type SymbolsCollector interface {
	// 生命周期管理
	Start(ctx context.Context) error
	Close() error

	// 数据操作
	Refresh(ctx context.Context) error
	Symbols() []*SymbolMeta
	Symbol(symbol string) (*SymbolMeta, error)

	// 事件管理
	Events() <-chan SymbolEvent

	// 速率限制
	GetRateLimit() *RateLimit

	// 健康检查
	HealthCheck(ctx context.Context) error
}

// SymbolAdapter 交易对适配器接口（保持向后兼容）
type SymbolAdapter interface {
	// GetExchange 获取交易所名称
	GetExchange() string

	// FetchAll 获取所有交易对
	FetchAll(ctx context.Context) ([]*SymbolMeta, error)

	// FetchSymbol 获取单个交易对
	FetchSymbol(ctx context.Context, symbol string) (*SymbolMeta, error)

	// IsSupported 检查是否支持该交易对
	IsSupported(symbol string) bool
}

// CollectorConfig 采集器配置
type CollectorConfig struct {
	Exchange          string        `yaml:"exchange"`          // 交易所名称
	RefreshInterval   time.Duration `yaml:"refreshInterval"`   // 刷新间隔
	EnableAutoRefresh bool          `yaml:"enableAutoRefresh"` // 启用自动刷新
	MaxRetries        int           `yaml:"maxRetries"`        // 最大重试次数
	RetryInterval     time.Duration `yaml:"retryInterval"`     // 重试间隔
	EnablePersistence bool          `yaml:"enablePersistence"` // 启用持久化
	EnableMetrics     bool          `yaml:"enableMetrics"`     // 启用指标
	BufferSize        int           `yaml:"bufferSize"`        // 事件缓冲区大小
}

// DefaultCollectorConfig 默认配置
func DefaultCollectorConfig() *CollectorConfig {
	return &CollectorConfig{
		RefreshInterval:   5 * time.Minute,
		EnableAutoRefresh: true,
		MaxRetries:        3,
		RetryInterval:     30 * time.Second,
		EnablePersistence: true,
		EnableMetrics:     true,
		BufferSize:        256,
	}
}

// Registry 交易对适配器注册表
type Registry struct {
	adapters map[string]SymbolAdapter
}

// NewRegistry 创建新的注册表
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[string]SymbolAdapter),
	}
}

// Register 注册适配器
func (r *Registry) Register(exchange string, adapter SymbolAdapter) {
	r.adapters[exchange] = adapter
}

// Get 获取适配器
func (r *Registry) Get(exchange string) (SymbolAdapter, bool) {
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
var globalSymbolRegistry = NewRegistry()

// RegisterSymbolAdapter 注册交易对适配器
func RegisterSymbolAdapter(exchange string, adapter SymbolAdapter) {
	globalSymbolRegistry.Register(exchange, adapter)
}

// GetSymbolAdapter 获取交易对适配器
func GetSymbolAdapter(exchange string) (SymbolAdapter, bool) {
	return globalSymbolRegistry.Get(exchange)
}

// ListRegisteredSymbolExchanges 列出已注册的交易对交易所
func ListRegisteredSymbolExchanges() []string {
	return globalSymbolRegistry.List()
}
