// Package trigger 触发器相关接口定义（最好是中文注释！）
package trigger

import (
	"context"

	"github.com/mooyang-code/data-collector/model"
)

// Trigger 触发器接口 - 控制采集器的执行时机
type Trigger interface {
	GetID() string
	GetType() model.TriggerType
	Initialize(config interface{}) error
	Start(ctx context.Context, callback model.TriggerCallback) error
	Stop(ctx context.Context) error
	IsRunning() bool
	GetStatus() model.TriggerStatus
}

// TriggerFactory 触发器工厂接口
type TriggerFactory interface {
	CreateTrigger(triggerType model.TriggerType, config *model.TriggerConfig) (Trigger, error)
	GetSupportedTypes() []model.TriggerType
	RegisterTriggerType(triggerType model.TriggerType, creator TriggerCreator) error
}

// TriggerCreator 触发器创建函数
type TriggerCreator func(config *model.TriggerConfig) (Trigger, error)

// TriggerManager 触发器管理器接口
type TriggerManager interface {
	AddTrigger(trigger Trigger) error
	RemoveTrigger(triggerID string) error
	GetTrigger(triggerID string) (Trigger, bool)
	GetTriggers() []Trigger
	GetTriggersByType(triggerType model.TriggerType) []Trigger
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	IsRunning() bool
}
