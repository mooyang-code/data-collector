package task

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mooyang-code/data-collector/pkg/logger"
	"github.com/mooyang-code/data-collector/pkg/model"
	"github.com/robfig/cron/v3"
)

// TaskScheduler 任务调度器接口
type TaskScheduler interface {
	// AddTask 添加定时任务（支持时间间隔）
	AddTask(task *model.Task, interval time.Duration, handler TaskHandler) error
	// AddCronTask 添加Cron任务（支持Cron表达式）
	AddCronTask(task *model.Task, cronExpr string, handler TaskHandler) error
	// RemoveTask 移除任务
	RemoveTask(taskID string) error
	// Start 启动调度器
	Start(ctx context.Context) error
	// Stop 停止调度器
	Stop(ctx context.Context) error
	// GetTaskStatus 获取任务状态
	GetTaskStatus(taskID string) (*TaskScheduleStatus, error)
	// ListTasks 列出所有任务
	ListTasks() map[string]*TaskScheduleStatus
	// IsRunning 检查调度器是否运行中
	IsRunning() bool
}

// TaskHandler 任务处理器函数类型
type TaskHandler func(ctx context.Context, task *model.Task) error

// TaskScheduleStatus 任务调度状态
type TaskScheduleStatus struct {
	TaskID     string        `json:"task_id"`
	TaskName   string        `json:"task_name"`
	Type       string        `json:"type"`      // "interval" 或 "cron"
	Interval   time.Duration `json:"interval"`  // 用于interval类型
	CronExpr   string        `json:"cron_expr"` // 用于cron类型
	LastRun    time.Time     `json:"last_run"`
	NextRun    time.Time     `json:"next_run"`
	RunCount   int64         `json:"run_count"`
	ErrorCount int64         `json:"error_count"`
	IsRunning  bool          `json:"is_running"`
	LastError  error         `json:"last_error,omitempty"`
	CreatedAt  time.Time     `json:"created_at"`
}

// scheduledTask 调度任务内部结构
type scheduledTask struct {
	Task     *model.Task
	Handler  TaskHandler
	Type     string        // "interval" 或 "cron"
	Interval time.Duration // 用于interval类型
	CronExpr string        // 用于cron类型
	EntryID  cron.EntryID  // cron任务ID

	// 运行时状态
	lastRun    time.Time
	runCount   int64
	errorCount int64
	isRunning  bool
	lastError  error
	createdAt  time.Time
	mu         sync.RWMutex
}

// cronTaskScheduler 基于Cron的任务调度器实现
type cronTaskScheduler struct {
	cron    *cron.Cron
	tasks   map[string]*scheduledTask
	mu      sync.RWMutex
	started bool
}

// NewTaskScheduler 创建新的任务调度器
func NewTaskScheduler(log logger.Logger) TaskScheduler {
	// 创建支持秒级精度的cron调度器
	c := cron.New(cron.WithSeconds())

	return &cronTaskScheduler{
		cron:  c,
		tasks: make(map[string]*scheduledTask),
	}
}

// AddTask 添加定时任务（支持时间间隔）
func (s *cronTaskScheduler) AddTask(task *model.Task, interval time.Duration, handler TaskHandler) error {
	if interval <= 0 {
		return fmt.Errorf("任务 %s 的间隔时间必须大于0", task.ID)
	}

	// 将时间间隔转换为cron表达式
	cronExpr := s.intervalToCron(interval)
	return s.AddCronTask(task, cronExpr, handler)
}

// AddCronTask 添加Cron任务（支持Cron表达式）
func (s *cronTaskScheduler) AddCronTask(task *model.Task, cronExpr string, handler TaskHandler) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[task.ID]; exists {
		return fmt.Errorf("任务 %s 已存在", task.ID)
	}

	scheduledTask := &scheduledTask{
		Task:      task,
		Handler:   handler,
		Type:      "cron",
		CronExpr:  cronExpr,
		createdAt: time.Now(),
	}

	// 创建任务包装函数
	wrappedHandler := s.wrapHandler(scheduledTask)

	// 添加到cron调度器
	entryID, err := s.cron.AddFunc(cronExpr, wrappedHandler)
	if err != nil {
		return fmt.Errorf("添加任务失败 [%s]: %w", task.ID, err)
	}

	scheduledTask.EntryID = entryID
	s.tasks[task.ID] = scheduledTask

	return nil
}

// RemoveTask 移除任务
func (s *cronTaskScheduler) RemoveTask(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("任务 %s 不存在", taskID)
	}

	// 从cron调度器中移除任务
	s.cron.Remove(task.EntryID)

	delete(s.tasks, taskID)
	return nil
}

// Start 启动调度器
func (s *cronTaskScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("调度器已经启动")
	}

	// 启动cron调度器
	s.cron.Start()
	s.started = true

	return nil
}

// Stop 停止调度器
func (s *cronTaskScheduler) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return fmt.Errorf("调度器未启动")
	}

	// 停止cron调度器，等待正在运行的任务完成
	cronCtx := s.cron.Stop()

	// 等待所有任务完成或外部context超时
	select {
	case <-cronCtx.Done():
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(30 * time.Second):
	}

	s.started = false
	return nil
}

// GetTaskStatus 获取任务状态
func (s *cronTaskScheduler) GetTaskStatus(taskID string) (*TaskScheduleStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("任务 %s 不存在", taskID)
	}

	return task.getStatus(s.cron), nil
}

// ListTasks 列出所有任务
func (s *cronTaskScheduler) ListTasks() map[string]*TaskScheduleStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	statuses := make(map[string]*TaskScheduleStatus)
	for taskID, task := range s.tasks {
		statuses[taskID] = task.getStatus(s.cron)
	}
	return statuses
}

// IsRunning 检查调度器是否运行中
func (s *cronTaskScheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.started
}

// intervalToCron 将时间间隔转换为cron表达式
func (s *cronTaskScheduler) intervalToCron(interval time.Duration) string {
	seconds := int(interval.Seconds())

	// 支持常见的时间间隔
	switch {
	case seconds < 60:
		// 每N秒执行一次
		return fmt.Sprintf("*/%d * * * * *", seconds)
	case seconds%60 == 0 && seconds < 3600:
		// 每N分钟执行一次
		minutes := seconds / 60
		return fmt.Sprintf("0 */%d * * * *", minutes)
	case seconds%3600 == 0 && seconds < 86400:
		// 每N小时执行一次（整点运行）
		hours := seconds / 3600
		return fmt.Sprintf("0 0 */%d * * *", hours)
	case seconds%86400 == 0:
		// 每N天执行一次（每天凌晨执行）
		days := seconds / 86400
		if days == 1 {
			return "0 0 0 * * *" // 每天凌晨0点
		}
		return fmt.Sprintf("0 0 0 */%d * *", days)
	default:
		// 其他情况，默认每N秒执行一次
		return fmt.Sprintf("*/%d * * * * *", seconds)
	}
}

// wrapHandler 包装任务处理函数
func (s *cronTaskScheduler) wrapHandler(task *scheduledTask) func() {
	return func() {
		// 使用背景context执行任务
		ctx := context.Background()
		s.executeTask(ctx, task)
	}
}

// executeTask 执行任务
func (s *cronTaskScheduler) executeTask(ctx context.Context, task *scheduledTask) {
	task.mu.Lock()
	if task.isRunning {
		task.mu.Unlock()
		return // 避免重复执行
	}
	task.isRunning = true
	task.lastRun = time.Now()
	task.mu.Unlock()

	// 执行任务
	err := task.Handler(ctx, task.Task)

	task.mu.Lock()
	defer task.mu.Unlock()

	task.isRunning = false
	if err != nil {
		task.errorCount++
		task.lastError = err
	} else {
		task.runCount++
		task.lastError = nil
	}
}

// getStatus 获取任务状态
func (t *scheduledTask) getStatus(c *cron.Cron) *TaskScheduleStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// 从cron调度器获取下次运行时间
	var nextRun time.Time
	for _, entry := range c.Entries() {
		if entry.ID == t.EntryID {
			nextRun = entry.Next
			break
		}
	}

	return &TaskScheduleStatus{
		TaskID:     t.Task.ID,
		TaskName:   t.Task.ID, // 使用ID作为名称
		Type:       t.Type,
		Interval:   t.Interval,
		CronExpr:   t.CronExpr,
		LastRun:    t.lastRun,
		NextRun:    nextRun,
		RunCount:   t.runCount,
		ErrorCount: t.errorCount,
		IsRunning:  t.isRunning,
		LastError:  t.lastError,
		CreatedAt:  t.createdAt,
	}
}
