package timer

import (
	"context"
	"time"
)

// JobID 任务ID类型
type JobID string

// JobFunc 任务执行函数类型
type JobFunc func(ctx context.Context) error

// JobStatus 任务状态
type JobStatus int

const (
	// JobStatusPending 等待执行
	JobStatusPending JobStatus = iota
	// JobStatusRunning 正在执行
	JobStatusRunning
	// JobStatusCompleted 执行完成
	JobStatusCompleted
	// JobStatusFailed 执行失败
	JobStatusFailed
	// JobStatusCancelled 已取消
	JobStatusCancelled
)

// String 返回状态字符串
func (s JobStatus) String() string {
	switch s {
	case JobStatusPending:
		return "pending"
	case JobStatusRunning:
		return "running"
	case JobStatusCompleted:
		return "completed"
	case JobStatusFailed:
		return "failed"
	case JobStatusCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// Job 定时任务定义
type Job struct {
	// 任务ID
	ID JobID `json:"id"`
	
	// 任务名称
	Name string `json:"name"`
	
	// 任务描述
	Description string `json:"description"`
	
	// Cron表达式
	CronExpr string `json:"cronExpr"`
	
	// 任务执行函数
	Func JobFunc `json:"-"`
	
	// 任务状态
	Status JobStatus `json:"status"`
	
	// 创建时间
	CreatedAt time.Time `json:"createdAt"`
	
	// 上次执行时间
	LastRunAt *time.Time `json:"lastRunAt,omitempty"`
	
	// 下次执行时间
	NextRunAt *time.Time `json:"nextRunAt,omitempty"`
	
	// 执行次数
	RunCount int64 `json:"runCount"`
	
	// 失败次数
	FailCount int64 `json:"failCount"`
	
	// 最后一次错误
	LastError string `json:"lastError,omitempty"`
	
	// 任务超时时间
	Timeout time.Duration `json:"timeout"`
	
	// 是否启用
	Enabled bool `json:"enabled"`
	
	// 最大重试次数
	MaxRetries int `json:"maxRetries"`
	
	// 当前重试次数
	RetryCount int `json:"retryCount"`
}

// JobExecution 任务执行记录
type JobExecution struct {
	// 执行ID
	ID string `json:"id"`
	
	// 任务ID
	JobID JobID `json:"jobId"`
	
	// 开始时间
	StartTime time.Time `json:"startTime"`
	
	// 结束时间
	EndTime *time.Time `json:"endTime,omitempty"`
	
	// 执行状态
	Status JobStatus `json:"status"`
	
	// 错误信息
	Error string `json:"error,omitempty"`
	
	// 执行耗时
	Duration time.Duration `json:"duration"`
}

// Timer 定时器接口
type Timer interface {
	// Start 启动定时器
	Start(ctx context.Context) error
	
	// Stop 停止定时器
	Stop(ctx context.Context) error
	
	// AddJob 添加定时任务
	AddJob(job *Job) error
	
	// RemoveJob 移除定时任务
	RemoveJob(jobID JobID) error
	
	// GetJob 获取任务信息
	GetJob(jobID JobID) (*Job, error)
	
	// ListJobs 列出所有任务
	ListJobs() ([]*Job, error)
	
	// EnableJob 启用任务
	EnableJob(jobID JobID) error
	
	// DisableJob 禁用任务
	DisableJob(jobID JobID) error
	
	// TriggerJob 手动触发任务
	TriggerJob(jobID JobID) error
	
	// GetJobExecutions 获取任务执行历史
	GetJobExecutions(jobID JobID, limit int) ([]*JobExecution, error)
	
	// IsRunning 检查定时器是否运行中
	IsRunning() bool
}

// JobBuilder 任务构建器
type JobBuilder struct {
	job *Job
}

// NewJobBuilder 创建任务构建器
func NewJobBuilder() *JobBuilder {
	return &JobBuilder{
		job: &Job{
			Status:     JobStatusPending,
			CreatedAt:  time.Now(),
			Enabled:    true,
			MaxRetries: 3,
			Timeout:    30 * time.Minute,
		},
	}
}

// WithID 设置任务ID
func (b *JobBuilder) WithID(id JobID) *JobBuilder {
	b.job.ID = id
	return b
}

// WithName 设置任务名称
func (b *JobBuilder) WithName(name string) *JobBuilder {
	b.job.Name = name
	return b
}

// WithDescription 设置任务描述
func (b *JobBuilder) WithDescription(desc string) *JobBuilder {
	b.job.Description = desc
	return b
}

// WithCron 设置Cron表达式
func (b *JobBuilder) WithCron(cronExpr string) *JobBuilder {
	b.job.CronExpr = cronExpr
	return b
}

// WithFunc 设置任务执行函数
func (b *JobBuilder) WithFunc(fn JobFunc) *JobBuilder {
	b.job.Func = fn
	return b
}

// WithTimeout 设置任务超时时间
func (b *JobBuilder) WithTimeout(timeout time.Duration) *JobBuilder {
	b.job.Timeout = timeout
	return b
}

// WithMaxRetries 设置最大重试次数
func (b *JobBuilder) WithMaxRetries(maxRetries int) *JobBuilder {
	b.job.MaxRetries = maxRetries
	return b
}

// WithEnabled 设置是否启用
func (b *JobBuilder) WithEnabled(enabled bool) *JobBuilder {
	b.job.Enabled = enabled
	return b
}

// Build 构建任务
func (b *JobBuilder) Build() *Job {
	return b.job
}
