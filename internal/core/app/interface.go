package app

import (
	"context"
	"time"
)

type SourceType string

const (
	SourceTypeMarket     SourceType = "market"     // SourceTypeMarket 市场数据源类型
	SourceTypeSocial     SourceType = "social"     // SourceTypeSocial 社交数据源类型
	SourceTypeBlockchain SourceType = "blockchain" // SourceTypeBlockchain 区块链数据源类型
	SourceTypeNews       SourceType = "news"       // SourceTypeNews 新闻数据源类型
)

type App interface {
	// 基础信息
	ID() string
	Type() SourceType
	Name() string
	
	// 生命周期管理
	Initialize(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	
	// 采集器管理
	RegisterCollector(collector Collector) error
	GetCollector(id string) (Collector, error)
	ListCollectors() []Collector
	
	// 事件处理
	OnEvent(event Event) error
	
	// 健康检查
	HealthCheck() error
	GetMetrics() AppMetrics
}

// Collector 接口定义（临时，后续移到 collector 包）
type Collector interface {
	ID() string
	Type() string
	DataType() string
	Initialize(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	IsRunning() bool
	GetStatus() CollectorStatus
}

// CollectorStatus 采集器状态
type CollectorStatus struct {
	ID         string
	Type       string
	DataType   string
	IsRunning  bool
	StartTime  time.Time
	Timers     map[string]TimerStatus
}

// TimerStatus 定时器状态
type TimerStatus struct {
	Name       string
	Interval   time.Duration
	LastRun    time.Time
	NextRun    time.Time
	RunCount   int64
	ErrorCount int64
}

// Event 接口定义（临时，后续移到 event 包）
type Event interface {
	ID() string
	Type() string
	Source() string
	Timestamp() time.Time
	Data() interface{}
}

type AppMetrics struct {
	StartTime        time.Time
	CollectorsTotal  int
	CollectorsActive int
	EventsProcessed  int64
	ErrorsTotal      int64
	LastError        error
	LastErrorTime    time.Time
}

type AppStatus string

const (
	AppStatusInitialized AppStatus = "initialized" // AppStatusInitialized 应用已初始化状态
	AppStatusRunning     AppStatus = "running"     // AppStatusRunning 应用运行中状态
	AppStatusStopped     AppStatus = "stopped"     // AppStatusStopped 应用已停止状态
	AppStatusError       AppStatus = "error"       // AppStatusError 应用错误状态
)

type HealthStatus struct {
	Status    AppStatus
	Message   string
	Checks    map[string]CheckResult
	Timestamp time.Time
}

type CheckResult struct {
	Status  string
	Message string
	Error   string
}