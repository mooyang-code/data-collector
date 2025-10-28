package collector

import (
	"context"
	"encoding/json"

	"github.com/mooyang-code/data-collector/pkg/model"
)

// Collector 数据采集器接口
type Collector interface {
	// 采集器元信息
	Name() string
	Type() model.CollectorType
	
	// 生命周期
	Init(config json.RawMessage) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	
	// 数据采集
	Collect(ctx context.Context, taskType model.TaskType, params *model.CollectParams) (*model.CollectResult, error)
	
	// 健康检查
	HealthCheck(ctx context.Context) error
	
	// 获取状态信息
	Status() Status
}

// Status 采集器状态
type Status struct {
	Name        string            `json:"name"`
	Type        model.CollectorType `json:"type"`
	State       State             `json:"state"`
	LastError   string            `json:"last_error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// State 采集器状态枚举
type State string

const (
	StateUninitialized State = "uninitialized"
	StateInitialized   State = "initialized"
	StateRunning       State = "running"
	StateStopped       State = "stopped"
	StateError         State = "error"
)

// Factory 采集器工厂函数
type Factory func() Collector

// Registry 采集器注册中心接口
type Registry interface {
	// 注册采集器
	Register(name string, factory Factory) error
	
	// 创建采集器实例
	Create(name string) (Collector, error)
	
	// 获取所有已注册的采集器类型
	List() []string
	
	// 检查采集器是否已注册
	IsRegistered(name string) bool
}

// Manager 采集器管理器接口
type Manager interface {
	// 创建并初始化采集器
	Create(ctx context.Context, name string, config json.RawMessage) (Collector, error)
	
	// 获取采集器
	Get(name string) (Collector, error)
	
	// 获取采集器（按交易所名称）
	GetCollector(exchange string) (Collector, error)
	
	// 启动采集器
	Start(ctx context.Context, name string) error
	
	// 停止采集器
	Stop(ctx context.Context, name string) error
	
	// 移除采集器
	Remove(ctx context.Context, name string) error
	
	// 获取所有采集器状态
	ListStatus() []Status
	
	// 健康检查
	HealthCheck(ctx context.Context) map[string]error
}