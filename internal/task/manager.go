package task

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mooyang-code/data-collector/internal/collector"
	"github.com/mooyang-code/data-collector/internal/event"
	"github.com/mooyang-code/data-collector/internal/metrics"
	"github.com/mooyang-code/data-collector/pkg/logger"
	"github.com/mooyang-code/data-collector/pkg/model"
)

// manager 任务管理器实现
type manager struct {
	config           Config
	eventBus         event.Notifier
	metricsCollector metrics.Collector
	collectorManager collector.Manager

	// 任务存储和调度
	tasks        map[string]*model.Task
	runningTasks map[string]*taskExecution
	scheduler    TaskScheduler

	// 状态管理
	mu               sync.RWMutex
	started          bool
	concurrencyLimit chan struct{}
}

// taskExecution 任务执行上下文
type taskExecution struct {
	task       *model.Task
	cancel     context.CancelFunc
	startTime  time.Time
	lastRun    time.Time
	runCount   int64
	errorCount int64
}

// NewManager 创建新的任务管理器
func NewManager(cfg Config, log logger.Logger, bus event.Notifier, metricsCollector metrics.Collector, collectorManager collector.Manager) Manager {
	taskLogger := log.With("component", "task_manager")

	return &manager{
		config:           cfg,
		eventBus:         bus,
		metricsCollector: metricsCollector,
		collectorManager: collectorManager,
		tasks:            make(map[string]*model.Task),
		runningTasks:     make(map[string]*taskExecution),
		scheduler:        NewTaskScheduler(taskLogger),
		concurrencyLimit: make(chan struct{}, cfg.MaxConcurrent),
	}
}

func (m *manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return fmt.Errorf("task manager already started")
	}

	// 确保存储目录存在
	if err := m.ensureStorePath(); err != nil {
		return fmt.Errorf("failed to ensure store path: %w", err)
	}

	// 加载任务
	if err := m.loadTasks(); err != nil {

	}

	// 启动调度器
	if err := m.scheduler.Start(ctx); err != nil {
		return fmt.Errorf("failed to start scheduler: %w", err)
	}

	// 重新调度所有任务
	for _, task := range m.tasks {
		if task.Status == model.TaskStatusRunning {
			if err := m.scheduleTask(task); err != nil {
			}
		}
	}

	// 订阅配置变更事件
	m.subscribeEvents()

	m.started = true
	return nil
}

func (m *manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return fmt.Errorf("task manager not started")
	}

	// 停止所有运行中的任务
	for taskID := range m.runningTasks {
		m.stopTaskExecution(taskID)
	}

	// 停止调度器
	if m.scheduler != nil {
		if err := m.scheduler.Stop(ctx); err != nil {

		}
	}

	// 保存任务状态
	if err := m.saveTasks(); err != nil {

	}

	m.started = false
	return nil
}

func (m *manager) CreateTask(ctx context.Context, task *model.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}

	if task.ID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	// 检查任务是否已存在
	if _, exists := m.tasks[task.ID]; exists {
		return fmt.Errorf("task %s already exists", task.ID)
	}

	// 设置创建时间
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	task.Status = model.TaskStatusPending

	// 初始化统计信息
	if task.Statistics == nil {
		task.Statistics = &model.TaskStats{}
	}

	// 存储任务
	m.tasks[task.ID] = task

	// 保存到文件
	if err := m.saveTasks(); err != nil {

	}

	// 发送事件
	m.publishTaskEvent("task.created", task)

	return nil
}

func (m *manager) UpdateTask(ctx context.Context, task *model.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}

	// 检查任务是否存在
	existingTask, exists := m.tasks[task.ID]
	if !exists {
		return fmt.Errorf("task %s not found", task.ID)
	}

	// 如果任务正在运行，先停止它
	if existingTask.Status == model.TaskStatusRunning {
		m.stopTaskExecution(task.ID)
		m.unscheduleTask(task.ID)
	}

	// 保留创建时间和统计信息
	task.CreatedAt = existingTask.CreatedAt
	if task.Statistics == nil {
		task.Statistics = existingTask.Statistics
	}
	task.UpdatedAt = time.Now()

	// 更新任务
	m.tasks[task.ID] = task

	// 如果新任务状态为运行，重新调度
	if task.Status == model.TaskStatusRunning {
		if err := m.scheduleTask(task); err != nil {
			task.Status = model.TaskStatusError
		}
	}

	// 保存到文件
	if err := m.saveTasks(); err != nil {

	}

	// 发送事件
	m.publishTaskEvent("task.updated", task)

	return nil
}

func (m *manager) DeleteTask(ctx context.Context, taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	// 停止任务执行
	m.stopTaskExecution(taskID)
	m.unscheduleTask(taskID)

	// 从存储中删除
	delete(m.tasks, taskID)

	// 保存到文件
	if err := m.saveTasks(); err != nil {

	}

	// 发送事件
	m.publishTaskEvent("task.deleted", task)

	return nil
}

func (m *manager) GetTask(ctx context.Context, taskID string) (*model.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	// 创建副本以避免并发修改
	taskCopy := *task
	if task.Statistics != nil {
		statsCopy := *task.Statistics
		taskCopy.Statistics = &statsCopy
	}

	return &taskCopy, nil
}

func (m *manager) ListTasks(ctx context.Context) ([]*model.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks := make([]*model.Task, 0, len(m.tasks))
	for _, task := range m.tasks {
		// 创建副本
		taskCopy := *task
		if task.Statistics != nil {
			statsCopy := *task.Statistics
			taskCopy.Statistics = &statsCopy
		}
		tasks = append(tasks, &taskCopy)
	}

	return tasks, nil
}

func (m *manager) StartTask(ctx context.Context, taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	if task.Status == model.TaskStatusRunning {
		return fmt.Errorf("task %s is already running", taskID)
	}

	// 更新任务状态
	task.Status = model.TaskStatusRunning
	task.UpdatedAt = time.Now()

	// 调度任务
	if err := m.scheduleTask(task); err != nil {
		task.Status = model.TaskStatusError
		return fmt.Errorf("failed to schedule task: %w", err)
	}

	// 保存状态
	if err := m.saveTasks(); err != nil {

	}

	// 发送事件
	m.publishTaskEvent("task.started", task)

	return nil
}

func (m *manager) StopTask(ctx context.Context, taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	if task.Status != model.TaskStatusRunning {
		return fmt.Errorf("task %s is not running", taskID)
	}

	// 停止任务执行
	m.stopTaskExecution(taskID)
	m.unscheduleTask(taskID)

	// 更新任务状态
	task.Status = model.TaskStatusStopped
	task.UpdatedAt = time.Now()

	// 保存状态
	if err := m.saveTasks(); err != nil {

	}

	// 发送事件
	m.publishTaskEvent("task.stopped", task)

	return nil
}

func (m *manager) GetTaskStatus(ctx context.Context, taskID string) (*model.TaskStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	status := task.Status
	return &status, nil
}

func (m *manager) GetRunningTasks(ctx context.Context) ([]*model.TaskSummary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var summaries []*model.TaskSummary
	for _, task := range m.tasks {
		if task.Status == model.TaskStatusRunning {
			summary := &model.TaskSummary{
				ID:      task.ID,
				Type:    task.Type,
				Status:  task.Status,
				LastRun: task.LastRun,
				NextRun: task.NextRun,
			}
			summaries = append(summaries, summary)
		}
	}

	return summaries, nil
}

// 私有辅助方法
func (m *manager) ensureStorePath() error {
	dir := filepath.Dir(m.config.StorePath)
	return os.MkdirAll(dir, 0755)
}

func (m *manager) loadTasks() error {
	data, err := os.ReadFile(m.config.StorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read task file: %w", err)
	}

	var tasks []*model.Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return fmt.Errorf("failed to unmarshal tasks: %w", err)
	}

	// 加载任务到内存
	for _, task := range tasks {
		m.tasks[task.ID] = task
	}

	return nil
}

func (m *manager) saveTasks() error {
	tasks := make([]*model.Task, 0, len(m.tasks))
	for _, task := range m.tasks {
		tasks = append(tasks, task)
	}

	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tasks: %w", err)
	}

	if err := os.WriteFile(m.config.StorePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write task file: %w", err)
	}

	return nil
}

func (m *manager) scheduleTask(task *model.Task) error {
	// 移除现有的调度
	m.unscheduleTask(task.ID)

	// 创建任务处理器
	handler := func(ctx context.Context, t *model.Task) error {
		m.executeTask(t.ID)
		return nil
	}

	var err error

	// 根据调度类型选择合适的调度方法
	if task.Schedule != "" {
		// 使用Cron表达式调度
		err = m.scheduler.AddCronTask(task, task.Schedule, handler)
	} else if task.Interval != "" {
		// 解析时间间隔
		interval, parseErr := time.ParseDuration(task.Interval)
		if parseErr != nil {
			return fmt.Errorf("invalid interval format for task %s: %w", task.ID, parseErr)
		}
		// 使用时间间隔调度
		err = m.scheduler.AddTask(task, interval, handler)
	} else {
		return fmt.Errorf("task %s has no schedule or interval", task.ID)
	}

	if err != nil {
		return fmt.Errorf("failed to schedule task %s: %w", task.ID, err)
	}

	// 获取调度状态并更新下次运行时间
	if status, err := m.scheduler.GetTaskStatus(task.ID); err == nil {
		task.NextRun = &status.NextRun
	}

	return nil
}

func (m *manager) unscheduleTask(taskID string) {
	if err := m.scheduler.RemoveTask(taskID); err != nil {
		// 任务不存在时不记录错误，这是正常情况
	}
}

func (m *manager) executeTask(taskID string) {
	// 获取并发限制
	select {
	case m.concurrencyLimit <- struct{}{}:
		defer func() { <-m.concurrencyLimit }()
	default:
		return
	}

	m.mu.Lock()
	task, exists := m.tasks[taskID]
	if !exists {
		m.mu.Unlock()
		return
	}

	// 检查是否已经在运行
	if _, running := m.runningTasks[taskID]; running {
		m.mu.Unlock()
		return
	}

	// 创建执行上下文
	_, cancel := context.WithCancel(context.Background())
	execution := &taskExecution{
		task:      task,
		cancel:    cancel,
		startTime: time.Now(),
		lastRun:   time.Now(),
	}

	m.runningTasks[taskID] = execution
	m.mu.Unlock()

	// 执行任务
	m.doExecuteTask(execution)
}

func (m *manager) doExecuteTask(execution *taskExecution) {
	taskID := execution.task.ID
	startTime := time.Now()

	defer func() {
		// 清理执行上下文
		m.mu.Lock()
		delete(m.runningTasks, taskID)
		execution.cancel()
		m.mu.Unlock()

		// 更新任务统计
		duration := time.Since(startTime)
		m.updateTaskStats(taskID, duration, nil)
	}()

	// 获取采集器
	collector, err := m.collectorManager.GetCollector(execution.task.Exchange)
	if err != nil {
		m.updateTaskStats(taskID, time.Since(startTime), err)
		return
	}

	// 构建采集参数
	params := &model.CollectParams{
		Symbol:   execution.task.Symbol,
		Interval: execution.task.Interval,
		Options:  make(map[string]interface{}),
	}

	// 解析任务特定配置
	if execution.task.Config != nil {
		var taskConfig map[string]interface{}
		if err := json.Unmarshal(execution.task.Config, &taskConfig); err == nil {
			params.Options = taskConfig
		}
	}

	// 执行采集
	result, err := collector.Collect(context.Background(), execution.task.Type, params)
	if err != nil {
		m.updateTaskStats(taskID, time.Since(startTime), err)
		return
	}

	// 记录指标
	if m.metricsCollector != nil {
		m.metricsCollector.IncrementCounter("task_executions_total", map[string]string{
			"task_id":  taskID,
			"type":     string(execution.task.Type),
			"exchange": execution.task.Exchange,
			"status":   "success",
		})

		m.metricsCollector.SetGauge("task_execution_duration_seconds", float64(time.Since(startTime).Seconds()), "task_id", taskID)
	}

	// 发送执行结果事件
	m.publishTaskEvent("task.executed", map[string]interface{}{
		"task_id":  taskID,
		"result":   result,
		"duration": time.Since(startTime).Milliseconds(),
	})
}

func (m *manager) stopTaskExecution(taskID string) {
	if execution, exists := m.runningTasks[taskID]; exists {
		execution.cancel()
		delete(m.runningTasks, taskID)
	}
}

func (m *manager) updateTaskStats(taskID string, duration time.Duration, execErr error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return
	}

	if task.Statistics == nil {
		task.Statistics = &model.TaskStats{}
	}

	task.Statistics.TotalRuns++
	now := time.Now()
	task.LastRun = &now
	task.UpdatedAt = now

	if execErr != nil {
		task.Statistics.FailedRuns++
		task.Statistics.LastError = &now
		task.Statistics.LastErrorMsg = execErr.Error()
	} else {
		task.Statistics.SuccessRuns++
		task.Statistics.LastSuccess = &now
	}

	// 更新平均执行时间
	if task.Statistics.TotalRuns > 0 {
		oldAvg := task.Statistics.AvgDuration
		newCount := float64(task.Statistics.TotalRuns)
		task.Statistics.AvgDuration = (oldAvg*(newCount-1) + duration.Seconds()) / newCount
	}

	// 异步保存（避免阻塞）
	go func() {
		if err := m.saveTasks(); err != nil {
			
		}
	}()
}

func (m *manager) publishTaskEvent(eventType string, data interface{}) {
	if m.eventBus != nil {
		metadata := map[string]interface{}{
			"timestamp": time.Now(),
			"source":    "task_manager",
		}

		if err := m.eventBus.PublishWithMetadata(eventType, data, metadata); err != nil {
		}
	}
}

func (m *manager) subscribeEvents() {
	if m.eventBus != nil {
		// 订阅配置变更事件
		handler := func(notification event.Notification) {
			m.handleConfigChange(notification.Data)
		}

		if err := m.eventBus.Subscribe("config.changed", handler); err != nil {
		}
	}
}

func (m *manager) handleConfigChange(data interface{}) {
	eventData, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	configType, ok := eventData["type"].(string)
	if !ok {
		return
	}

	if configType == "task_configs" {
		// 任务配置变更，重新加载任务
		// TODO: 实现配置变更处理逻辑
	}
}
