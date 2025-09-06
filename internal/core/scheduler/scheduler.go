package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Scheduler 调度器接口
type Scheduler interface {
	// 添加任务
	AddTask(name string, interval time.Duration, handler func(context.Context) error) error
	// 移除任务
	RemoveTask(name string) error
	// 启动调度器
	Start(ctx context.Context) error
	// 停止调度器
	Stop() error
	// 获取任务状态
	GetTaskStatus(name string) (*TaskStatus, error)
	// 列出所有任务
	ListTasks() map[string]*TaskStatus
}

// TaskStatus 任务状态
type TaskStatus struct {
	Name       string
	Interval   time.Duration
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
	Interval time.Duration
	Handler  func(context.Context) error
	
	// 运行时状态
	ticker     *time.Ticker
	cancel     context.CancelFunc
	lastRun    time.Time
	nextRun    time.Time
	runCount   int64
	errorCount int64
	isRunning  bool
	lastError  error
	mu         sync.RWMutex
}

// TimeScheduler 基于时间的调度器
type TimeScheduler struct {
	tasks map[string]*Task
	mu    sync.RWMutex
	
	wg     sync.WaitGroup
	
	started bool
	stopChan chan struct{}
}

// NewTimeScheduler 创建时间调度器
func NewTimeScheduler() *TimeScheduler {
	return &TimeScheduler{
		tasks: make(map[string]*Task),
	}
}

// AddTask 添加定时任务
func (s *TimeScheduler) AddTask(name string, interval time.Duration, handler func(context.Context) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.tasks[name]; exists {
		return fmt.Errorf("任务 %s 已存在", name)
	}
	
	task := &Task{
		Name:     name,
		Interval: interval,
		Handler:  handler,
		nextRun:  time.Now().Add(interval),
	}
	
	s.tasks[name] = task
	
	// 如果调度器已启动，立即启动这个任务
	if s.started {
		// 注意: 这里没有context可用，所以不能启动任务
		// 任务只能在Start方法被调用时启动
		return fmt.Errorf("不能在运行时添加任务")
	}
	
	return nil
}

// RemoveTask 移除任务
func (s *TimeScheduler) RemoveTask(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	task, exists := s.tasks[name]
	if !exists {
		return fmt.Errorf("任务 %s 不存在", name)
	}
	
	// 停止任务
	s.stopTask(task)
	
	delete(s.tasks, name)
	return nil
}

// Start 启动调度器
func (s *TimeScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.started {
		return fmt.Errorf("调度器已经启动")
	}
	
	s.stopChan = make(chan struct{})
	
	// 启动所有任务
	for _, task := range s.tasks {
		s.startTask(ctx, task)
	}
	
	s.started = true
	return nil
}

// Stop 停止调度器
func (s *TimeScheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.started {
		return fmt.Errorf("调度器未启动")
	}
	
	// 停止所有任务
	for _, task := range s.tasks {
		s.stopTask(task)
	}
	
	if s.stopChan != nil {
		close(s.stopChan)
	}
	
	// 等待所有任务完成
	s.wg.Wait()
	
	s.started = false
	return nil
}

// GetTaskStatus 获取任务状态
func (s *TimeScheduler) GetTaskStatus(name string) (*TaskStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	task, exists := s.tasks[name]
	if !exists {
		return nil, fmt.Errorf("任务 %s 不存在", name)
	}
	
	return task.GetStatus(), nil
}

// ListTasks 列出所有任务
func (s *TimeScheduler) ListTasks() map[string]*TaskStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	statuses := make(map[string]*TaskStatus)
	for name, task := range s.tasks {
		statuses[name] = task.GetStatus()
	}
	
	return statuses
}

// startTask 启动单个任务
func (s *TimeScheduler) startTask(ctx context.Context, task *Task) {
	taskCtx, cancel := context.WithCancel(ctx)
	task.cancel = cancel
	task.ticker = time.NewTicker(task.Interval)
	
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer task.ticker.Stop()
		
		// 立即执行一次
		s.executeTask(taskCtx, task)
		
		// 定时执行
		for {
			select {
			case <-taskCtx.Done():
				return
			case <-s.stopChan:
				return
			case <-task.ticker.C:
				s.executeTask(taskCtx, task)
			}
		}
	}()
}

// stopTask 停止单个任务
func (s *TimeScheduler) stopTask(task *Task) {
	if task.ticker != nil {
		task.ticker.Stop()
	}
	
	if task.cancel != nil {
		task.cancel()
	}
}

// executeTask 执行任务
func (s *TimeScheduler) executeTask(ctx context.Context, task *Task) {
	task.mu.Lock()
	if task.isRunning {
		task.mu.Unlock()
		return // 避免重复执行
	}
	task.isRunning = true
	task.lastRun = time.Now()
	task.nextRun = time.Now().Add(task.Interval)
	task.mu.Unlock()
	
	// 执行任务
	err := task.Handler(ctx)
	
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

// GetStatus 获取任务状态
func (t *Task) GetStatus() *TaskStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	return &TaskStatus{
		Name:       t.Name,
		Interval:   t.Interval,
		LastRun:    t.lastRun,
		NextRun:    t.nextRun,
		RunCount:   t.runCount,
		ErrorCount: t.errorCount,
		IsRunning:  t.isRunning,
		LastError:  t.lastError,
	}
}

// CronScheduler 基于Cron表达式的调度器（预留接口）
type CronScheduler struct {
	*TimeScheduler
}

// NewCronScheduler 创建Cron调度器
func NewCronScheduler() *CronScheduler {
	return &CronScheduler{
		TimeScheduler: NewTimeScheduler(),
	}
}

// AddCronTask 添加Cron任务
func (s *CronScheduler) AddCronTask(name string, cronExpr string, handler func(context.Context) error) error {
	// TODO: 解析cron表达式并转换为定时任务
	return fmt.Errorf("Cron调度器暂未实现")
}