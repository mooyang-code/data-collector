package task

import (
	"context"

	"github.com/mooyang-code/data-collector/pkg/model"
)

// Manager 任务管理器接口
type Manager interface {
	// 生命周期
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	
	// 任务管理
	CreateTask(ctx context.Context, task *model.Task) error
	UpdateTask(ctx context.Context, task *model.Task) error
	DeleteTask(ctx context.Context, taskID string) error
	GetTask(ctx context.Context, taskID string) (*model.Task, error)
	ListTasks(ctx context.Context) ([]*model.Task, error)
	
	// 任务控制
	StartTask(ctx context.Context, taskID string) error
	StopTask(ctx context.Context, taskID string) error
	
	// 状态查询
	GetTaskStatus(ctx context.Context, taskID string) (*model.TaskStatus, error)
	GetRunningTasks(ctx context.Context) ([]*model.TaskSummary, error)
}

// Store 任务存储接口
type Store interface {
	Save(ctx context.Context, tasks []*model.Task) error
	Load(ctx context.Context) ([]*model.Task, error)
	Watch(ctx context.Context, onChange func([]*model.Task)) error
}

// Scheduler 任务调度器接口
type Scheduler interface {
	Schedule(task *model.Task) error
	Cancel(taskID string) error
	List() []*model.Task
}

// Config 任务管理器配置
type Config struct {
	StorePath    string `json:"store_path" yaml:"store_path"`
	MaxConcurrent int   `json:"max_concurrent" yaml:"max_concurrent"`
}

// DefaultConfig 默认配置
var DefaultConfig = Config{
	StorePath:    "/tmp/data-collector-tasks.json",
	MaxConcurrent: 10,
}