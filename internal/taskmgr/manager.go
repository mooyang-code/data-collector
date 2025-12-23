package taskmgr

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"sync"
	"time"

	"github.com/mooyang-code/data-collector/internal/collector"
	"github.com/mooyang-code/data-collector/pkg/config"
	"trpc.group/trpc-go/trpc-go/log"
)

// TaskManager 任务管理器
// 负责定时同步任务配置，管理 Collector 生命周期
type TaskManager struct {
	registry *collector.CollectorRegistry

	// 运行中的任务：TaskID → RunningTask
	runningTasks map[string]*RunningTask
	mu           sync.RWMutex
}

// RunningTask 运行中的任务
type RunningTask struct {
	TaskInstance *config.CollectorTaskInstanceCache
	Collector    collector.Collector
	StartedAt    time.Time
	ParamsHash   string // 用于检测配置变更
}

// 全局单例
var (
	defaultManager *TaskManager
	managerOnce    sync.Once
)

// GetManager 获取任务管理器单例
func GetManager() *TaskManager {
	return defaultManager
}

// InitManager 初始化任务管理器
func InitManager() {
	managerOnce.Do(func() {
		defaultManager = &TaskManager{
			registry:     collector.GetRegistry(),
			runningTasks: make(map[string]*RunningTask),
		}
	})
}

// Sync 同步任务配置
// 对比远程配置与本地运行状态，执行增/删/改操作
func (m *TaskManager) Sync(ctx context.Context) error {
	log.DebugContext(ctx, "开始同步任务配置...")

	// 动态获取 NodeID
	nodeID, _ := config.GetNodeInfo()
	if nodeID == "" {
		log.DebugContext(ctx, "NodeID 为空，跳过本次任务同步")
		return nil
	}

	// 获取本节点最新任务列表
	newTasks := config.GetTaskInstancesByNode(nodeID)
	log.DebugContextf(ctx, "NodeID: %s; 从远程获取到 %d 个任务配置", nodeID, len(newTasks))

	// 构建新任务 Map
	newTaskMap := make(map[string]*config.CollectorTaskInstanceCache)
	for _, task := range newTasks {
		newTaskMap[task.TaskID] = task
		log.DebugContextf(ctx, "NodeID: %s; 远程任务: TaskID=%s, DataSource=%s, DataType=%s, InstType=%s,"+
			" Symbol=%s, Intervals=%v, ParamsHash=%s",
			nodeID, task.TaskID, task.DataSource, task.DataType, task.InstType,
			task.Symbol, task.Intervals, hashString(task.TaskParams))
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	log.DebugContextf(ctx, "NodeID: %s; 当前运行中任务数: %d", nodeID, len(m.runningTasks))
	for taskID, running := range m.runningTasks {
		log.DebugContextf(ctx, "运行中任务: TaskID=%s, ParamsHash=%s, StartedAt=%v",
			taskID, running.ParamsHash, running.StartedAt)
	}

	// 检测需要删除的任务（本地有，远程无）
	for taskID := range m.runningTasks {
		if _, exists := newTaskMap[taskID]; !exists {
			log.InfoContextf(ctx, "NodeID: %s; 任务 %s 已从远程删除，停止采集器", nodeID, taskID)
			m.stopTaskLocked(ctx, taskID)
		}
	}

	// 检测需要新增/更新的任务
	for taskID, newTask := range newTaskMap {
		if running, exists := m.runningTasks[taskID]; exists {
			// 检查是否需要更新（TaskParams 变化）
			newHash := hashString(newTask.TaskParams)
			if running.ParamsHash != newHash {
				log.DebugContextf(ctx, "任务 %s 配置变更详情: 旧Params=%s, 新Params=%s",
					taskID, running.TaskInstance.TaskParams, newTask.TaskParams)
				m.stopTaskLocked(ctx, taskID)
				m.startTaskLocked(ctx, newTask)
			} else {
				log.DebugContextf(ctx, "任务 %s 配置无变化，跳过", taskID)
			}
		} else {
			// 新任务
			log.InfoContextf(ctx, "发现新任务 %s，启动采集器", taskID)
			m.startTaskLocked(ctx, newTask)
		}
	}

	log.DebugContextf(ctx, "NodeID: %s; 任务同步完成，当前运行中任务数: %d", nodeID, len(m.runningTasks))
	return nil
}

// startTaskLocked 启动任务（调用前需持有锁）
func (m *TaskManager) startTaskLocked(ctx context.Context, task *config.CollectorTaskInstanceCache) {
	// 将 taskID 注入日志上下文
	log.WithContextFields(ctx, "taskID", task.TaskID)

	// 构建 Collector 配置
	collectorConfig := map[string]interface{}{
		"inst_type": task.InstType,
		"symbol":    task.Symbol,
		"intervals": task.Intervals,
	}

	// 从 Registry 创建 Collector
	c, err := m.registry.CreateCollector(task.DataSource, task.DataType, collectorConfig)
	if err != nil {
		log.ErrorContextf(ctx, "创建采集器失败 [%s]: %v", task.TaskID, err)
		ReportError(ctx, task.TaskID, "create_failed", err)
		return
	}

	// 初始化 Collector
	if err := c.Initialize(ctx); err != nil {
		log.ErrorContextf(ctx, "初始化采集器失败 [%s]: %v", task.TaskID, err)
		ReportError(ctx, task.TaskID, "init_failed", err)
		return
	}

	// 启动 Collector
	if err := c.Start(ctx); err != nil {
		log.ErrorContextf(ctx, "启动采集器失败 [%s]: %v", task.TaskID, err)
		ReportError(ctx, task.TaskID, "start_failed", err)
		return
	}

	// 记录运行状态
	m.runningTasks[task.TaskID] = &RunningTask{
		TaskInstance: task,
		Collector:    c,
		StartedAt:    time.Now(),
		ParamsHash:   hashString(task.TaskParams),
	}

	log.InfoContextf(ctx, "任务 %s 启动成功 [%s/%s, inst_type=%s, symbol=%s, intervals=%v]",
		task.TaskID, task.DataSource, task.DataType, task.InstType, task.Symbol, task.Intervals)
}

// stopTaskLocked 停止任务（调用前需持有锁）
func (m *TaskManager) stopTaskLocked(ctx context.Context, taskID string) {
	running, exists := m.runningTasks[taskID]
	if !exists {
		return
	}

	// 停止 Collector
	if err := running.Collector.Stop(ctx); err != nil {
		log.WarnContextf(ctx, "停止采集器失败 [%s]: %v", taskID, err)
	}

	delete(m.runningTasks, taskID)
	log.InfoContextf(ctx, "任务 %s 已停止", taskID)
}

// GetRunningTasks 获取所有运行中的任务（用于状态查询）
func (m *TaskManager) GetRunningTasks() map[string]*RunningTask {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*RunningTask)
	for k, v := range m.runningTasks {
		result[k] = v
	}
	return result
}

// GetRunningTaskCount 获取运行中任务数量
func (m *TaskManager) GetRunningTaskCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.runningTasks)
}

// hashString 计算字符串的 MD5 哈希
func hashString(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
