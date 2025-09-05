// Package collector 采集器核心接口定义
package collector

import (
	"context"
	"time"
)

// Collector 采集器基础接口
type Collector interface {
	// 基础信息
	ID() string
	Type() string
	Exchange() string
	
	// 生命周期管理
	Initialize(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	
	// 状态管理
	IsRunning() bool
	GetStatus() Status
	GetMetrics() Metrics
}

// TimedCollector 定时采集器接口
type TimedCollector interface {
	Collector
	
	// 定时器管理
	AddTimer(name string, interval time.Duration, handler TimerHandler) error
	RemoveTimer(name string) error
	GetTimers() map[string]*Timer
}

// StreamCollector 流式采集器接口
type StreamCollector interface {
	Collector
	
	// WebSocket连接管理
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	Subscribe(topics []string) error
	Unsubscribe(topics []string) error
}

// Timer 定时器
type Timer struct {
	Name     string
	Interval time.Duration
	Handler  TimerHandler
	Running  bool
	LastRun  time.Time
	NextRun  time.Time
	RunCount int64
	Errors   int64
}

// TimerHandler 定时器处理函数
type TimerHandler func(ctx context.Context) error

// Status 采集器状态
type Status struct {
	State      string    // idle, initializing, running, stopping, stopped, error
	StartTime  time.Time
	LastUpdate time.Time
	LastError  error
	Message    string
}

// Metrics 采集器指标
type Metrics struct {
	DataPoints   int64         // 数据点数量
	LastDataTime time.Time     // 最后数据时间
	ErrorCount   int64         // 错误次数
	Latency      time.Duration // 延迟
	Custom       map[string]interface{}
}