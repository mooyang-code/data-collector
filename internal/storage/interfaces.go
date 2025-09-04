// Package storage 提供数据存储抽象接口
package storage

import (
	"context"
	"time"

	"github.com/mooyang-code/data-collector/internal/datatype/klines"
	"github.com/mooyang-code/data-collector/internal/datatype/symbols"
	"github.com/mooyang-code/data-collector/model"
)

// 类型别名，方便在Storage接口中使用
type (
	DataRecord    = model.DataRecord
	QueryRequest  = model.QueryRequest
	QueryResult   = model.QueryResult
	StorageStats  = model.StorageStats
)

// SymbolWriter 交易对写入器接口
type SymbolWriter interface {
	// WriteSymbol 写入单个交易对
	WriteSymbol(ctx context.Context, symbol *symbols.SymbolMeta) error
	
	// WriteSymbols 批量写入交易对
	WriteSymbols(ctx context.Context, symbols []*symbols.SymbolMeta) error
	
	// UpsertSymbol 更新或插入交易对
	UpsertSymbol(ctx context.Context, symbol *symbols.SymbolMeta) error
	
	// UpsertSymbols 批量更新或插入交易对
	UpsertSymbols(ctx context.Context, symbols []*symbols.SymbolMeta) error
	
	// DeleteSymbol 删除交易对
	DeleteSymbol(ctx context.Context, exchange, symbol string) error
	
	// Close 关闭写入器
	Close() error
}

// SymbolReader 交易对读取器接口
type SymbolReader interface {
	// GetSymbol 获取单个交易对
	GetSymbol(ctx context.Context, exchange, symbol string) (*symbols.SymbolMeta, error)
	
	// GetSymbols 获取交易对列表
	GetSymbols(ctx context.Context, filter *symbols.SymbolFilter) ([]*symbols.SymbolMeta, error)
	
	// GetSymbolsByExchange 按交易所获取交易对
	GetSymbolsByExchange(ctx context.Context, exchange string) ([]*symbols.SymbolMeta, error)
	
	// CountSymbols 统计交易对数量
	CountSymbols(ctx context.Context, filter *symbols.SymbolFilter) (int64, error)
	
	// Close 关闭读取器
	Close() error
}

// KlineWriter K线写入器接口
type KlineWriter interface {
	// WriteKline 写入单个K线
	WriteKline(ctx context.Context, kline *klines.Kline) error
	
	// WriteKlines 批量写入K线
	WriteKlines(ctx context.Context, klines []*klines.Kline) error
	
	// Close 关闭写入器
	Close() error
}

// KlineReader K线读取器接口
type KlineReader interface {
	// GetKlines 获取K线数据
	GetKlines(ctx context.Context, filter *klines.KlineFilter) ([]*klines.Kline, error)
	
	// GetLatestKline 获取最新K线
	GetLatestKline(ctx context.Context, exchange, symbol string, interval klines.Interval) (*klines.Kline, error)
	
	// GetKlineRange 获取K线时间范围
	GetKlineRange(ctx context.Context, exchange, symbol string, interval klines.Interval) (*klines.KlineRange, error)
	
	// CountKlines 统计K线数量
	CountKlines(ctx context.Context, filter *klines.KlineFilter) (int64, error)
	
	// Close 关闭读取器
	Close() error
}

// BatchWriter 批量写入器接口
type BatchWriter interface {
	// AddSymbol 添加交易对到批次
	AddSymbol(symbol *symbols.SymbolMeta) error
	
	// AddKline 添加K线到批次
	AddKline(kline *klines.Kline) error
	
	// Flush 刷新批次
	Flush(ctx context.Context) error
	
	// Size 获取批次大小
	Size() int
	
	// Clear 清空批次
	Clear()
	
	// Close 关闭批量写入器
	Close() error
}

// SnapshotStore 快照存储接口
type SnapshotStore interface {
	// SaveSymbolSnapshot 保存交易对快照
	SaveSymbolSnapshot(ctx context.Context, snapshot *symbols.SymbolSnapshot) error
	
	// LoadSymbolSnapshot 加载交易对快照
	LoadSymbolSnapshot(ctx context.Context, exchange string) (*symbols.SymbolSnapshot, error)
	
	// DeleteSymbolSnapshot 删除交易对快照
	DeleteSymbolSnapshot(ctx context.Context, exchange string) error
	
	// ListSymbolSnapshots 列出交易对快照
	ListSymbolSnapshots(ctx context.Context) ([]*symbols.SymbolSnapshot, error)
	
	// Close 关闭快照存储
	Close() error
}

// MetricsStore 指标存储接口
type MetricsStore interface {
	// WriteMetric 写入指标
	WriteMetric(ctx context.Context, metric *Metric) error
	
	// WriteMetrics 批量写入指标
	WriteMetrics(ctx context.Context, metrics []*Metric) error
	
	// GetMetrics 获取指标
	GetMetrics(ctx context.Context, filter *MetricFilter) ([]*Metric, error)
	
	// Close 关闭指标存储
	Close() error
}

// Metric 指标数据
type Metric struct {
	Name      string                 `json:"name"`      // 指标名称
	Value     float64                `json:"value"`     // 指标值
	Labels    map[string]string      `json:"labels"`    // 标签
	Timestamp time.Time              `json:"timestamp"` // 时间戳
	Extra     map[string]interface{} `json:"extra"`     // 扩展字段
}

// MetricFilter 指标过滤器
type MetricFilter struct {
	Name      string            `json:"name,omitempty"`      // 指标名称
	Labels    map[string]string `json:"labels,omitempty"`    // 标签过滤
	StartTime time.Time         `json:"startTime,omitempty"` // 开始时间
	EndTime   time.Time         `json:"endTime,omitempty"`   // 结束时间
	Limit     int               `json:"limit,omitempty"`     // 数量限制
}

// Storage 存储器接口 - 提供统一的数据存储接口
type Storage interface {
	GetID() string
	GetType() string
	Initialize(config interface{}) error
	Store(ctx context.Context, data *DataRecord) error
	StoreBatch(ctx context.Context, data []*DataRecord) error
	Query(ctx context.Context, query *QueryRequest) (*QueryResult, error)
	Close() error
	GetStats() *StorageStats
}

// StorageBackend 存储后端接口
type StorageBackend interface {
	// GetSymbolWriter 获取交易对写入器
	GetSymbolWriter() (SymbolWriter, error)

	// GetSymbolReader 获取交易对读取器
	GetSymbolReader() (SymbolReader, error)

	// GetKlineWriter 获取K线写入器
	GetKlineWriter() (KlineWriter, error)

	// GetKlineReader 获取K线读取器
	GetKlineReader() (KlineReader, error)

	// GetBatchWriter 获取批量写入器
	GetBatchWriter() (BatchWriter, error)

	// GetSnapshotStore 获取快照存储
	GetSnapshotStore() (SnapshotStore, error)

	// Ping 检查连接
	Ping(ctx context.Context) error

	// Close 关闭后端
	Close() error

	// GetName 获取后端名称
	GetName() string
}

// Config 存储配置
type Config struct {
	Backend string                 `yaml:"backend"` // 后端类型
	Options map[string]interface{} `yaml:"options"` // 后端选项
}

// Factory 存储工厂接口
type Factory interface {
	// CreateBackend 创建存储后端
	CreateBackend(config *Config) (StorageBackend, error)
	
	// GetSupportedBackends 获取支持的后端类型
	GetSupportedBackends() []string
}

// Stats 存储统计信息
type Stats struct {
	Backend       string    `json:"backend"`       // 后端类型
	Connected     bool      `json:"connected"`     // 是否连接
	LastPing      time.Time `json:"lastPing"`      // 最后ping时间
	WriteCount    int64     `json:"writeCount"`    // 写入次数
	ReadCount     int64     `json:"readCount"`     // 读取次数
	ErrorCount    int64     `json:"errorCount"`    // 错误次数
	LastError     string    `json:"lastError"`     // 最后错误
	SymbolCount   int64     `json:"symbolCount"`   // 交易对数量
	KlineCount    int64     `json:"klineCount"`    // K线数量
	SnapshotCount int64     `json:"snapshotCount"` // 快照数量
}

// HealthCheck 健康检查接口
type HealthCheck interface {
	// Check 执行健康检查
	Check(ctx context.Context) error
	
	// GetStats 获取统计信息
	GetStats() *Stats
}

// Transaction 事务接口
type Transaction interface {
	// Commit 提交事务
	Commit(ctx context.Context) error
	
	// Rollback 回滚事务
	Rollback(ctx context.Context) error
	
	// GetSymbolWriter 获取事务内的交易对写入器
	GetSymbolWriter() (SymbolWriter, error)
	
	// GetKlineWriter 获取事务内的K线写入器
	GetKlineWriter() (KlineWriter, error)
}

// TransactionalBackend 支持事务的存储后端
type TransactionalBackend interface {
	StorageBackend
	
	// BeginTransaction 开始事务
	BeginTransaction(ctx context.Context) (Transaction, error)
}

// CacheBackend 缓存后端接口
type CacheBackend interface {
	// Get 获取缓存
	Get(ctx context.Context, key string) (interface{}, error)
	
	// Set 设置缓存
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	
	// Delete 删除缓存
	Delete(ctx context.Context, key string) error
	
	// Clear 清空缓存
	Clear(ctx context.Context) error
	
	// Close 关闭缓存
	Close() error
}

// CompressionType 压缩类型
type CompressionType string

const (
	CompressionNone CompressionType = "none"
	CompressionGzip CompressionType = "gzip"
	CompressionLZ4  CompressionType = "lz4"
	CompressionZstd CompressionType = "zstd"
)

// ArchiveConfig 归档配置
type ArchiveConfig struct {
	Enabled     bool            `yaml:"enabled"`     // 是否启用归档
	Interval    time.Duration   `yaml:"interval"`    // 归档间隔
	Retention   time.Duration   `yaml:"retention"`   // 保留时间
	Compression CompressionType `yaml:"compression"` // 压缩类型
	Path        string          `yaml:"path"`        // 归档路径
}

// ArchiveBackend 归档后端接口
type ArchiveBackend interface {
	// Archive 归档数据
	Archive(ctx context.Context, config *ArchiveConfig) error
	
	// Restore 恢复数据
	Restore(ctx context.Context, archivePath string) error
	
	// ListArchives 列出归档
	ListArchives(ctx context.Context) ([]string, error)
	
	// DeleteArchive 删除归档
	DeleteArchive(ctx context.Context, archivePath string) error
	
	// Close 关闭归档后端
	Close() error
}
