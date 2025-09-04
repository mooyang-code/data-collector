// Package collector 采集器相关接口定义（最好是中文注释！）
package collector

import (
	"context"

	"github.com/mooyang-code/data-collector/configs"
	"github.com/mooyang-code/data-collector/model"
)

// Collector 数据采集器接口 - 执行具体的数据采集逻辑
type Collector interface {
	GetID() string
	GetType() string
	GetDataType() model.DataType
	GetTrigger() Trigger
	SetTrigger(trigger Trigger) error
	Initialize(config *configs.CollectorConfig) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	IsRunning() bool
	GetStatus() model.CollectorStatus
	Collect(ctx context.Context) error
}

// CollectorManager 采集器管理器接口 - 管理多种类型的数据采集器
type CollectorManager interface {
	AddCollector(collector Collector) error
	RemoveCollector(collectorID string) error
	GetCollector(collectorID string) (Collector, bool)
	GetCollectors() []Collector
	GetCollectorsByType(dataType types.DataType) []Collector
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	IsRunning() bool
}

// Trigger 触发器接口 - 控制采集器的执行时机
type Trigger interface {
	GetID() string
	GetType() types.TriggerType
	Initialize(config interface{}) error
	Start(ctx context.Context, callback types.TriggerCallback) error
	Stop(ctx context.Context) error
	IsRunning() bool
	GetStatus() types.TriggerStatus
}

// CollectorFactory 采集器工厂接口
type CollectorFactory interface {
	CreateCollector(collectorType string, config *types.CollectorConfig) (Collector, error)
	GetSupportedTypes() []string
	RegisterCollectorType(collectorType string, creator CollectorCreator) error
}

// CollectorCreator 采集器创建函数
type CollectorCreator func(config *types.CollectorConfig) (Collector, error)
