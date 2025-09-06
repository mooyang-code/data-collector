package collector

import (
	"context"
	"time"
)

type Collector interface {
	// 基础信息
	ID() string
	Type() string
	DataType() string
	
	// 生命周期
	Initialize(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	
	// 定时器管理
	AddTimer(name string, interval time.Duration, handler TimerHandler) error
	RemoveTimer(name string) error
	GetTimers() map[string]*Timer
	
	// 状态监控
	IsRunning() bool
	GetStatus() CollectorStatus
	GetMetrics() CollectorMetrics
}

type TimerHandler func(ctx context.Context) error

type Timer struct {
	Name       string
	Interval   time.Duration
	Handler    TimerHandler
	LastRun    time.Time
	NextRun    time.Time
	RunCount   int64
	ErrorCount int64
	ticker     *time.Ticker
	cancel     context.CancelFunc
}

type CollectorStatus struct {
	ID         string
	Type       string
	DataType   string
	IsRunning  bool
	StartTime  time.Time
	LastUpdate time.Time
	Timers     map[string]TimerStatus
}

type TimerStatus struct {
	Name       string
	Interval   time.Duration
	LastRun    time.Time
	NextRun    time.Time
	RunCount   int64
	ErrorCount int64
}

type CollectorMetrics struct {
	StartTime       time.Time
	DataCollected   int64
	EventsPublished int64
	ErrorsTotal     int64
	LastError       error
	LastErrorTime   time.Time
	TimerMetrics    map[string]TimerMetrics
}

type TimerMetrics struct {
	RunCount   int64
	ErrorCount int64
	LastRun    time.Time
	AvgLatency time.Duration
}