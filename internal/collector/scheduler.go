package collector

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"trpc.group/trpc-go/trpc-go"
)

// Scheduler 调度器接口
type Scheduler interface {
	// AddTask 添加定时任务（支持时间间隔），ctx 用于克隆链路追踪信息
	AddTask(ctx context.Context, name string, interval time.Duration, handler func(context.Context) error) error
	// AddCronTask 添加Cron任务（支持Cron表达式，实现整点运行），ctx 用于克隆链路追踪信息
	AddCronTask(ctx context.Context, name string, cronExpr string, handler func(context.Context) error) error
	// RemoveTask 移除任务
	RemoveTask(name string) error
	// Start 启动调度器
	Start(ctx context.Context) error
	// Stop 停止调度器
	Stop() error
	// GetTaskStatus 获取任务状态
	GetTaskStatus(name string) (*TaskStatus, error)
	// ListTasks 列出所有任务
	ListTasks() map[string]*TaskStatus
}

// TaskStatus 任务状态
type TaskStatus struct {
	Name       string
	Type       string        // "interval" 或 "cron"
	Interval   time.Duration // 用于interval类型
	CronExpr   string        // 用于cron类型
	LastRun    time.Time
	NextRun    time.Time
	RunCount   int64
	ErrorCount int64
	IsRunning  bool
	LastError  error
}

// Task 任务定义
type Task struct {
	Name     string
	Type     string        // "interval" 或 "cron"
	Interval time.Duration // 用于interval类型
	CronExpr string        // 用于cron类型
	Handler  func(context.Context) error
	EntryID  cron.EntryID    // cron任务ID
	BaseCtx  context.Context // 克隆的上下文，保留链路追踪信息

	// 运行时状态
	lastRun    time.Time
	runCount   int64
	errorCount int64
	isRunning  bool
	lastError  error
	mu         sync.RWMutex
}

// CronScheduler 基于Cron的调度器
type CronScheduler struct {
	cron    *cron.Cron
	tasks   map[string]*Task
	mu      sync.RWMutex
	started bool
	logger  *log.Logger
}

// NewCronScheduler 创建基于Cron的调度器
func NewCronScheduler() *CronScheduler {
	// 创建支持秒级精度的cron调度器
	c := cron.New(cron.WithSeconds())

	return &CronScheduler{
		cron:   c,
		tasks:  make(map[string]*Task),
		logger: log.New(log.Writer(), "[调度器] ", log.LstdFlags|log.Lshortfile),
	}
}

// AddTask 添加定时任务（支持时间间隔）
func (s *CronScheduler) AddTask(ctx context.Context, name string, interval time.Duration, handler func(context.Context) error) error {
	// 将时间间隔转换为cron表达式
	cronExpr := s.intervalToCron(interval)
	return s.AddCronTask(ctx, name, cronExpr, handler)
}

// AddCronTask 添加Cron任务（支持Cron表达式，实现整点运行）
func (s *CronScheduler) AddCronTask(ctx context.Context, name string, cronExpr string, handler func(context.Context) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[name]; exists {
		return fmt.Errorf("任务 %s 已存在", name)
	}

	// 使用 trpc.CloneContext 克隆上下文，保留链路追踪信息，分离超时控制
	baseCtx := trpc.CloneContext(ctx)

	task := &Task{
		Name:     name,
		Type:     "cron",
		CronExpr: cronExpr,
		Handler:  handler,
		BaseCtx:  baseCtx,
	}

	// 创建任务包装函数，传入task引用
	wrappedHandler := s.wrapHandler(task)

	// 添加到cron调度器
	entryID, err := s.cron.AddFunc(cronExpr, wrappedHandler)
	if err != nil {
		return fmt.Errorf("添加任务失败 [%s]: %v", name, err)
	}

	task.EntryID = entryID
	s.tasks[name] = task
	s.logger.Printf("成功添加任务: %s - Cron: %s", name, cronExpr)
	return nil
}

// RemoveTask 移除任务
func (s *CronScheduler) RemoveTask(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[name]
	if !exists {
		return fmt.Errorf("任务 %s 不存在", name)
	}

	// 从cron调度器中移除任务
	s.cron.Remove(task.EntryID)

	delete(s.tasks, name)
	s.logger.Printf("成功移除任务: %s", name)
	return nil
}

// Start 启动调度器
func (s *CronScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("调度器已经启动")
	}

	// 启动cron调度器
	s.cron.Start()
	s.started = true

	s.logger.Printf("调度器已启动，共有 %d 个任务", len(s.tasks))
	return nil
}

// Stop 停止调度器
func (s *CronScheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return fmt.Errorf("调度器未启动")
	}

	s.logger.Println("正在停止调度器...")

	// 停止cron调度器，等待正在运行的任务完成
	ctx := s.cron.Stop()

	// 等待所有任务完成或超时
	select {
	case <-ctx.Done():
		s.logger.Println("调度器已优雅停止")
	case <-time.After(30 * time.Second):
		s.logger.Println("调度器停止超时，强制退出")
	}

	s.started = false
	return nil
}

// GetTaskStatus 获取任务状态
func (s *CronScheduler) GetTaskStatus(name string) (*TaskStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, exists := s.tasks[name]
	if !exists {
		return nil, fmt.Errorf("任务 %s 不存在", name)
	}

	return task.GetStatus(s.cron), nil
}

// ListTasks 列出所有任务
func (s *CronScheduler) ListTasks() map[string]*TaskStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	statuses := make(map[string]*TaskStatus)
	for name, task := range s.tasks {
		statuses[name] = task.GetStatus(s.cron)
	}
	return statuses
}

// intervalToCron 将时间间隔转换为cron表达式
func (s *CronScheduler) intervalToCron(interval time.Duration) string {
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
func (s *CronScheduler) wrapHandler(task *Task) func() {
	return func() {
		// 使用克隆的上下文执行任务，保留链路追踪信息
		ctx := task.BaseCtx
		if ctx == nil {
			ctx = context.Background()
		}
		s.executeTask(ctx, task)
	}
}

// executeTask 执行任务
func (s *CronScheduler) executeTask(ctx context.Context, task *Task) {
	task.mu.Lock()
	if task.isRunning {
		task.mu.Unlock()
		return // 避免重复执行
	}
	task.isRunning = true
	task.lastRun = time.Now()
	task.mu.Unlock()

	// 执行任务
	err := task.Handler(ctx)

	task.mu.Lock()
	defer task.mu.Unlock()

	task.isRunning = false
	if err != nil {
		task.errorCount++
		task.lastError = err
		s.logger.Printf("任务 %s 执行失败: %v", task.Name, err)
	} else {
		task.runCount++
		task.lastError = nil
	}
}

// GetStatus 获取任务状态
func (t *Task) GetStatus(c *cron.Cron) *TaskStatus {
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

	return &TaskStatus{
		Name:       t.Name,
		Type:       t.Type,
		Interval:   t.Interval,
		CronExpr:   t.CronExpr,
		LastRun:    t.lastRun,
		NextRun:    nextRun,
		RunCount:   t.runCount,
		ErrorCount: t.errorCount,
		IsRunning:  t.isRunning,
		LastError:  t.lastError,
	}
}
