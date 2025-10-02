package manager

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mooyang-code/data-collector/internal/core/app"
	"github.com/mooyang-code/data-collector/internal/serverless/types"
)

// RunningTask 运行中的任务
type RunningTask struct {
	Config         types.TaskConfig
	Collector      app.Collector
	Timer          *time.Timer
	IsRunning      bool
	LastExecutedAt time.Time
	NextExecuteAt  time.Time
	ExecutionCount int64
	ErrorCount     int64
	cancelFunc     context.CancelFunc
	mu             sync.Mutex
}

// TaskManager 任务管理器
type TaskManager struct {
	tasks      map[string]*RunningTask
	appManager *app.Manager
	mu         sync.RWMutex
}

// NewTaskManager 创建任务管理器
func NewTaskManager(appManager *app.Manager) *TaskManager {
	return &TaskManager{
		tasks:      make(map[string]*RunningTask),
		appManager: appManager,
	}
}

// StartTask 启动任务
func (m *TaskManager) StartTask(config types.TaskConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查任务是否已存在
	if _, exists := m.tasks[config.TaskID]; exists {
		log.Printf("[TaskManager] Task %s already exists, updating config", config.TaskID)
		m.stopTaskLocked(config.TaskID)
	}

	// 创建采集器
	col, err := m.createCollector(config)
	if err != nil {
		return fmt.Errorf("failed to create collector: %w", err)
	}

	// 解析执行间隔
	interval, err := time.ParseDuration(config.Interval)
	if err != nil {
		return fmt.Errorf("invalid interval %s: %w", config.Interval, err)
	}

	// 创建任务上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 创建运行任务
	task := &RunningTask{
		Config:     config,
		Collector:  col,
		cancelFunc: cancel,
		IsRunning:  false,
	}

	// 启动定时器
	task.startTimer(ctx, interval)

	m.tasks[config.TaskID] = task
	log.Printf("[TaskManager] Started task %s with interval %s", config.TaskID, interval)

	return nil
}

// StopTask 停止任务
func (m *TaskManager) StopTask(taskID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stopTaskLocked(taskID)
}

// stopTaskLocked 停止任务（需要已持有锁）
func (m *TaskManager) stopTaskLocked(taskID string) {
	task, exists := m.tasks[taskID]
	if !exists {
		return
	}

	log.Printf("[TaskManager] Stopping task %s", taskID)

	// 取消任务上下文
	if task.cancelFunc != nil {
		task.cancelFunc()
	}

	// 停止定时器
	if task.Timer != nil {
		task.Timer.Stop()
	}

	// 停止采集器
	if task.Collector != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := task.Collector.Stop(ctx); err != nil {
			log.Printf("[TaskManager] Failed to stop collector for task %s: %v", taskID, err)
		}
	}

	delete(m.tasks, taskID)
}

// GetRunningTasks 获取所有运行中的任务
func (m *TaskManager) GetRunningTasks() map[string]types.TaskConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks := make(map[string]types.TaskConfig)
	for id, task := range m.tasks {
		tasks[id] = task.Config
	}
	return tasks
}

// GetTasksStatus 获取任务状态
func (m *TaskManager) GetTasksStatus() []types.RunningTaskInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var status []types.RunningTaskInfo
	for _, task := range m.tasks {
		task.mu.Lock()
		info := types.RunningTaskInfo{
			TaskID:        task.Config.TaskID,
			CollectorType: task.Config.CollectorType,
			Source:        task.Config.Source,
			StartTime:     task.LastExecutedAt,
			LastExecTime:  task.LastExecutedAt,
			ExecCount:     task.ExecutionCount,
			ErrorCount:    task.ErrorCount,
		}
		task.mu.Unlock()
		status = append(status, info)
	}
	return status
}

// StopAll 停止所有任务
func (m *TaskManager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Printf("[TaskManager] Stopping all tasks")
	for taskID := range m.tasks {
		m.stopTaskLocked(taskID)
	}
}

// createCollector 创建采集器
func (m *TaskManager) createCollector(config types.TaskConfig) (app.Collector, error) {
	// 获取或创建App
	appID := config.Source
	app, err := m.appManager.GetApp(appID)
	if err != nil {
		// 尝试创建新的App
		// 这里需要根据实际的App创建逻辑来实现
		return nil, fmt.Errorf("app %s not found: %w", appID, err)
	}

	// 从App获取采集器
	collectors := app.ListCollectors()
	for _, col := range collectors {
		if col.DataType() == config.CollectorType {
			return col, nil
		}
	}

	return nil, fmt.Errorf("collector type %s not found in app %s", config.CollectorType, appID)
}

// startTimer 启动定时器
func (task *RunningTask) startTimer(ctx context.Context, interval time.Duration) {
	// 立即执行一次
	go task.execute(ctx)

	// 创建定时器
	task.Timer = time.NewTimer(interval)

	go func() {
		for {
			select {
			case <-task.Timer.C:
				task.execute(ctx)
				task.Timer.Reset(interval)
			case <-ctx.Done():
				return
			}
		}
	}()
}

// execute 执行采集任务
func (task *RunningTask) execute(ctx context.Context) {
	task.mu.Lock()
	task.IsRunning = true
	task.LastExecutedAt = time.Now()
	task.NextExecuteAt = time.Now().Add(time.Hour) // 这里应该根据实际间隔计算
	task.mu.Unlock()

	defer func() {
		task.mu.Lock()
		task.IsRunning = false
		task.ExecutionCount++
		task.mu.Unlock()
	}()

	log.Printf("[TaskManager] Executing task %s", task.Config.TaskID)

	// 执行采集
	if err := task.Collector.Start(ctx); err != nil {
		log.Printf("[TaskManager] Failed to execute task %s: %v", task.Config.TaskID, err)
		task.mu.Lock()
		task.ErrorCount++
		task.mu.Unlock()
	}
}