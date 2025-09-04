package timer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"trpc.group/trpc-go/trpc-go/log"

	"github.com/robfig/cron/v3"
)

// CronTimer 基于cron的定时器实现
type CronTimer struct {
	config *Config
	cron   *cron.Cron

	// 任务存储
	jobs map[JobID]*Job
	mu   sync.RWMutex

	// 执行历史
	executions map[JobID][]*JobExecution
	execMu     sync.RWMutex

	// 运行状态
	running bool
	runMu   sync.RWMutex



	// 并发控制
	semaphore chan struct{}
}

// NewCronTimer 创建新的定时器
func NewCronTimer(config *Config) (*CronTimer, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// 创建cron调度器
	location, err := config.GetLocation()
	if err != nil {
		return nil, fmt.Errorf("invalid timezone: %w", err)
	}

	cronScheduler := cron.New(
		cron.WithLocation(location),
		cron.WithSeconds(), // 支持秒级调度
		cron.WithChain(cron.Recover(cron.DefaultLogger)),
	)

	timer := &CronTimer{
		config:     config,
		cron:       cronScheduler,
		jobs:       make(map[JobID]*Job),
		executions: make(map[JobID][]*JobExecution),
		semaphore:  make(chan struct{}, config.MaxConcurrentJobs),
	}
	return timer, nil
}

// Start 启动定时器
func (t *CronTimer) Start(ctx context.Context) error {
	t.runMu.Lock()
	defer t.runMu.Unlock()

	if t.running {
		return fmt.Errorf("timer is already running")
	}

	t.cron.Start()
	t.running = true

	log.Info("Timer started successfully")

	// 启动清理协程
	go t.cleanupRoutine(ctx)

	return nil
}

// Stop 停止定时器
func (t *CronTimer) Stop(ctx context.Context) error {
	t.runMu.Lock()
	defer t.runMu.Unlock()

	if !t.running {
		return nil
	}

	// 停止cron调度器
	cronCtx := t.cron.Stop()



	// 等待所有任务完成或超时
	select {
	case <-cronCtx.Done():
		log.Info("All jobs completed gracefully")
	case <-time.After(30 * time.Second):
		log.Warn("Timeout waiting for jobs to complete")
	}

	t.running = false
	log.Info("Timer stopped successfully")

	return nil
}

// AddJob 添加定时任务
func (t *CronTimer) AddJob(job *Job) error {
	if job == nil {
		return fmt.Errorf("job cannot be nil")
	}

	if job.ID == "" {
		return fmt.Errorf("job ID cannot be empty")
	}

	if job.CronExpr == "" {
		return fmt.Errorf("cron expression cannot be empty")
	}

	if job.Func == nil {
		return fmt.Errorf("job function cannot be nil")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// 检查任务是否已存在
	if _, exists := t.jobs[job.ID]; exists {
		return fmt.Errorf("job with ID %s already exists", job.ID)
	}

	// 添加到cron调度器
	entryID, err := t.cron.AddFunc(job.CronExpr, func() {
		t.executeJob(context.Background(), job.ID)
	})
	if err != nil {
		return fmt.Errorf("failed to add job to cron: %w", err)
	}

	// 计算下次执行时间
	if t.running {
		entry := t.cron.Entry(entryID)
		job.NextRunAt = &entry.Next
	}

	// 存储任务
	t.jobs[job.ID] = job

	log.Infof("Job added successfully - id: %s, name: %s, cron: %s", job.ID, job.Name, job.CronExpr)

	return nil
}

// RemoveJob 移除定时任务
func (t *CronTimer) RemoveJob(jobID JobID) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	job, exists := t.jobs[jobID]
	if !exists {
		return fmt.Errorf("job with ID %s not found", jobID)
	}

	// 从cron调度器中移除
	// 注意：cron/v3没有直接的RemoveJob方法，需要重新创建调度器或使用EntryID
	// 这里我们标记为禁用
	job.Enabled = false

	delete(t.jobs, jobID)

	log.Infof("Job removed successfully - id: %s", jobID)

	return nil
}

// GetJob 获取任务信息
func (t *CronTimer) GetJob(jobID JobID) (*Job, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	job, exists := t.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job with ID %s not found", jobID)
	}

	// 返回副本以避免并发修改
	jobCopy := *job
	return &jobCopy, nil
}

// ListJobs 列出所有任务
func (t *CronTimer) ListJobs() ([]*Job, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	jobs := make([]*Job, 0, len(t.jobs))
	for _, job := range t.jobs {
		jobCopy := *job
		jobs = append(jobs, &jobCopy)
	}

	return jobs, nil
}

// EnableJob 启用任务
func (t *CronTimer) EnableJob(jobID JobID) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	job, exists := t.jobs[jobID]
	if !exists {
		return fmt.Errorf("job with ID %s not found", jobID)
	}

	job.Enabled = true
	log.Infof("Job enabled - id: %s", jobID)

	return nil
}

// DisableJob 禁用任务
func (t *CronTimer) DisableJob(jobID JobID) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	job, exists := t.jobs[jobID]
	if !exists {
		return fmt.Errorf("job with ID %s not found", jobID)
	}

	job.Enabled = false
	log.Infof("Job disabled - id: %s", jobID)

	return nil
}

// TriggerJob 手动触发任务
func (t *CronTimer) TriggerJob(jobID JobID) error {
	t.mu.RLock()
	job, exists := t.jobs[jobID]
	t.mu.RUnlock()

	if !exists {
		return fmt.Errorf("job with ID %s not found", jobID)
	}

	if !job.Enabled {
		return fmt.Errorf("job %s is disabled", jobID)
	}

	go t.executeJob(context.Background(), jobID)

	log.Infof("Job triggered manually - id: %s", jobID)

	return nil
}

// IsRunning 检查定时器是否运行中
func (t *CronTimer) IsRunning() bool {
	t.runMu.RLock()
	defer t.runMu.RUnlock()
	return t.running
}

// GetJobExecutions 获取任务执行历史
func (t *CronTimer) GetJobExecutions(jobID JobID, limit int) ([]*JobExecution, error) {
	t.execMu.RLock()
	defer t.execMu.RUnlock()

	executions, exists := t.executions[jobID]
	if !exists {
		return []*JobExecution{}, nil
	}

	// 返回最近的执行记录
	start := 0
	if limit > 0 && len(executions) > limit {
		start = len(executions) - limit
	}

	result := make([]*JobExecution, len(executions)-start)
	copy(result, executions[start:])

	return result, nil
}

// executeJob 执行任务
func (t *CronTimer) executeJob(ctx context.Context, jobID JobID) {
	// 获取信号量，控制并发数
	select {
	case t.semaphore <- struct{}{}:
		defer func() { <-t.semaphore }()
	case <-ctx.Done():
		return
	}

	t.mu.RLock()
	job, exists := t.jobs[jobID]
	t.mu.RUnlock()

	if !exists || !job.Enabled {
		return
	}

	// 创建执行记录
	execution := &JobExecution{
		ID:        fmt.Sprintf("%s_%d", jobID, time.Now().UnixNano()),
		JobID:     jobID,
		StartTime: time.Now(),
		Status:    JobStatusRunning,
	}

	// 更新任务状态
	t.mu.Lock()
	job.Status = JobStatusRunning
	job.RunCount++
	now := time.Now()
	job.LastRunAt = &now
	t.mu.Unlock()

	log.Infof("Job execution started - id: %s, name: %s", jobID, job.Name)

	// 执行任务
	jobCtx, cancel := context.WithTimeout(ctx, job.Timeout)
	defer cancel()

	err := job.Func(jobCtx)

	// 更新执行结果
	execution.EndTime = &[]time.Time{time.Now()}[0]
	execution.Duration = execution.EndTime.Sub(execution.StartTime)

	t.mu.Lock()
	if err != nil {
		execution.Status = JobStatusFailed
		execution.Error = err.Error()
		job.Status = JobStatusFailed
		job.FailCount++
		job.LastError = err.Error()
		job.RetryCount++

		log.Errorf("Job execution failed - id: %s, error: %v", jobID, err)

		// 检查是否需要重试
		if job.RetryCount < job.MaxRetries {
			log.Infof("Job will retry - id: %s, retry: %d/%d", jobID, job.RetryCount, job.MaxRetries)
			go func() {
				time.Sleep(time.Minute) // 等待1分钟后重试
				t.executeJob(ctx, jobID)
			}()
		}
	} else {
		execution.Status = JobStatusCompleted
		job.Status = JobStatusCompleted
		job.RetryCount = 0 // 重置重试计数
		job.LastError = ""

		log.Infof("Job execution completed - id: %s, duration: %v", jobID, execution.Duration)
	}
	t.mu.Unlock()

	// 保存执行记录
	t.saveExecution(execution)
}

// saveExecution 保存执行记录
func (t *CronTimer) saveExecution(execution *JobExecution) {
	t.execMu.Lock()
	defer t.execMu.Unlock()

	executions := t.executions[execution.JobID]
	executions = append(executions, execution)

	// 限制历史记录数量
	maxHistory := 100
	if len(executions) > maxHistory {
		executions = executions[len(executions)-maxHistory:]
	}

	t.executions[execution.JobID] = executions
}

// cleanupRoutine 清理协程
func (t *CronTimer) cleanupRoutine(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.cleanup()
		case <-ctx.Done():
			return
		}
	}
}

// cleanup 清理过期的执行记录
func (t *CronTimer) cleanup() {
	t.execMu.Lock()
	defer t.execMu.Unlock()

	cutoff := time.Now().Add(-t.config.HistoryRetention)

	for jobID, executions := range t.executions {
		filtered := make([]*JobExecution, 0, len(executions))
		for _, exec := range executions {
			if exec.StartTime.After(cutoff) {
				filtered = append(filtered, exec)
			}
		}

		if len(filtered) != len(executions) {
			t.executions[jobID] = filtered
			log.Debugf("Cleaned up execution history for job %s: %d -> %d",
				jobID, len(executions), len(filtered))
		}
	}
}
