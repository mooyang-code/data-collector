package manager

import (
	"context"
	"log"
	"time"

	"github.com/mooyang-code/data-collector/internal/serverless/cache"
	"github.com/mooyang-code/data-collector/internal/serverless/types"
)

// TaskSynchronizer 任务同步器
type TaskSynchronizer struct {
	nodeID       string
	cacheClient  cache.CacheClient
	taskManager  *TaskManager
	syncInterval time.Duration
}

// NewTaskSynchronizer 创建任务同步器
func NewTaskSynchronizer(nodeID string, cacheClient cache.CacheClient, taskManager *TaskManager) *TaskSynchronizer {
	return &TaskSynchronizer{
		nodeID:       nodeID,
		cacheClient:  cacheClient,
		taskManager:  taskManager,
		syncInterval: 2 * time.Second,
	}
}

// Start 启动任务同步
func (s *TaskSynchronizer) Start(ctx context.Context) {
	log.Printf("[TaskSync] Starting task synchronizer for node %s", s.nodeID)

	// 立即执行一次同步
	s.syncTasks(ctx)

	// 启动定时同步
	ticker := time.NewTicker(s.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.syncTasks(ctx)
		case <-ctx.Done():
			log.Printf("[TaskSync] Stopping task synchronizer")
			return
		}
	}
}

// syncTasks 同步任务
func (s *TaskSynchronizer) syncTasks(ctx context.Context) {
	// 1. 从缓存获取本节点的任务配置
	expectedTasks, err := s.cacheClient.GetNodeTasks(s.nodeID)
	if err != nil {
		log.Printf("[TaskSync] Failed to get tasks from cache: %v", err)
		return
	}

	log.Printf("[TaskSync] Got %d tasks from cache", len(expectedTasks))

	// 2. 获取当前运行的任务
	runningTasks := s.taskManager.GetRunningTasks()

	// 3. 对比差异
	toAdd, toStop := s.diffTasks(expectedTasks, runningTasks)

	// 4. 停止不需要的任务
	for _, taskID := range toStop {
		log.Printf("[TaskSync] Stopping task: %s", taskID)
		s.taskManager.StopTask(taskID)
	}

	// 5. 启动新任务
	for _, task := range toAdd {
		log.Printf("[TaskSync] Starting task: %s", task.TaskID)
		if err := s.taskManager.StartTask(task); err != nil {
			log.Printf("[TaskSync] Failed to start task %s: %v", task.TaskID, err)
		}
	}
}

// diffTasks 对比任务差异
func (s *TaskSynchronizer) diffTasks(expected []types.TaskConfig, running map[string]types.TaskConfig) (toAdd []types.TaskConfig, toStop []string) {
	// 创建期望任务的映射
	expectedMap := make(map[string]types.TaskConfig)
	for _, task := range expected {
		expectedMap[task.TaskID] = task
	}

	// 找出需要停止的任务
	for taskID := range running {
		if _, exists := expectedMap[taskID]; !exists {
			toStop = append(toStop, taskID)
		}
	}

	// 找出需要新增或更新的任务
	for _, task := range expected {
		runningTask, exists := running[task.TaskID]
		if !exists {
			// 新任务
			toAdd = append(toAdd, task)
		} else if s.taskConfigChanged(task, runningTask) {
			// 配置变更，需要重启
			toStop = append(toStop, task.TaskID)
			toAdd = append(toAdd, task)
		}
	}

	return toAdd, toStop
}

// taskConfigChanged 检查任务配置是否变更
func (s *TaskSynchronizer) taskConfigChanged(new, old types.TaskConfig) bool {
	// 比较主要配置项
	if new.CollectorType != old.CollectorType {
		return true
	}
	if new.Source != old.Source {
		return true
	}
	if new.Interval != old.Interval {
		return true
	}
	
	// TODO: 深度比较Config字段
	// 这里简化处理，认为Config总是可能变化
	return true
}