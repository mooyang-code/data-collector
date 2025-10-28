package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/avast/retry-go"
	"github.com/mooyang-code/data-collector/internal/metrics"
	"github.com/mooyang-code/data-collector/internal/task"
	"github.com/mooyang-code/data-collector/pkg/logger"
	"github.com/mooyang-code/data-collector/pkg/model"
)

// manager 心跳管理器实现
type manager struct {
	config           Config
	logger           logger.Logger
	nodeInfo         *model.NodeInfo
	taskManager      task.Manager
	metricsCollector metrics.Collector

	// 服务端信息
	serverURL  string
	serverPort int
	authToken  string
	httpClient *http.Client

	// 状态管理
	mu          sync.RWMutex
	started     bool
	ticker      *time.Ticker
	lastReport  time.Time
	reportCount int64
	errorCount  int64
}

// NewManager 创建新的心跳管理器
func NewManager(cfg Config, log logger.Logger, nodeInfo *model.NodeInfo, taskManager task.Manager, metricsCollector metrics.Collector) Manager {
	return &manager{
		config:           cfg,
		logger:           log.With("component", "heartbeat_manager"),
		nodeInfo:         nodeInfo,
		taskManager:      taskManager,
		metricsCollector: metricsCollector,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (m *manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return fmt.Errorf("heartbeat manager already started")
	}

	m.ticker = time.NewTicker(m.config.Interval)

	// 启动心跳上报循环
	go m.heartbeatLoop()

	m.started = true
	m.logger.Info("heartbeat manager started", "interval", m.config.Interval)
	return nil
}

func (m *manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return fmt.Errorf("heartbeat manager not started")
	}

	// 停止定时器
	if m.ticker != nil {
		m.ticker.Stop()
	}

	m.started = false
	m.logger.Info("heartbeat manager stopped",
		"report_count", m.reportCount,
		"error_count", m.errorCount)
	return nil
}

func (m *manager) Report(ctx context.Context) error {
	if m.serverURL == "" {
		m.logger.Debug("no server URL configured, skipping heartbeat report")
		return nil
	}

	// 构建心跳负载
	payload, err := m.buildHeartbeatPayload(ctx)
	if err != nil {
		return fmt.Errorf("failed to build heartbeat payload: %w", err)
	}

	// 发送心跳
	if err := m.sendHeartbeat(ctx, payload); err != nil {
		m.mu.Lock()
		m.errorCount++
		m.mu.Unlock()
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}

	m.mu.Lock()
	m.lastReport = time.Now()
	m.reportCount++
	m.mu.Unlock()

	m.logger.Debug("heartbeat report sent successfully")
	return nil
}

func (m *manager) HandleProbe(ctx context.Context, probeData map[string]interface{}) (*model.Response, error) {
	m.logger.Info("handling heartbeat probe", "probe_data_keys", getMapKeys(probeData))

	// 更新服务端信息
	if serverURL, ok := probeData["server_url"].(string); ok {
		m.serverURL = serverURL
		m.logger.Info("server URL updated from probe", "server_url", serverURL)
	}

	if serverPort, ok := probeData["server_port"].(float64); ok {
		m.serverPort = int(serverPort)
		m.logger.Info("server port updated from probe", "server_port", m.serverPort)
	}

	if authToken, ok := probeData["auth_token"].(string); ok {
		m.authToken = authToken
		m.logger.Info("auth token updated from probe")
	}

	// 构建响应数据
	responseData, err := m.buildProbeResponse(ctx)
	if err != nil {
		m.logger.Error("failed to build probe response", "error", err)
		return &model.Response{
			Success: false,
			Message: fmt.Sprintf("failed to build response: %v", err),
		}, nil
	}

	return &model.Response{
		Success:   true,
		Message:   "probe handled successfully",
		Data:      responseData,
		Timestamp: time.Now(),
	}, nil
}

func (m *manager) UpdateServerInfo(serverURL string, serverPort int, authToken string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.serverURL = serverURL
	m.serverPort = serverPort
	m.authToken = authToken

	m.logger.Info("server info updated",
		"server_url", serverURL,
		"server_port", serverPort,
		"has_auth_token", authToken != "")
	return nil
}

// 私有辅助方法
func (m *manager) heartbeatLoop() {
	defer m.ticker.Stop()

	for {
		select {
		case <-m.ticker.C:
			if err := m.Report(context.Background()); err != nil {
				m.logger.Error("heartbeat report failed", "error", err)
			}
		}
	}
}

func (m *manager) buildHeartbeatPayload(ctx context.Context) (*model.HeartbeatPayload, error) {
	// 获取运行中的任务
	runningTasks, err := m.taskManager.GetRunningTasks(ctx)
	if err != nil {
		m.logger.Warn("failed to get running tasks for heartbeat", "error", err)
		runningTasks = []*model.TaskSummary{}
	}

	// 获取节点指标
	nodeMetrics := m.collectNodeMetrics()

	// 构建心跳负载
	payload := &model.HeartbeatPayload{
		NodeID:       m.nodeInfo.NodeID,
		NodeType:     m.nodeInfo.NodeType,
		Timestamp:    time.Now(),
		RunningTasks: runningTasks,
		Metrics:      nodeMetrics,
		Metadata: map[string]interface{}{
			"version":      m.nodeInfo.Version,
			"region":       m.nodeInfo.Region,
			"namespace":    m.nodeInfo.Namespace,
			"capabilities": m.nodeInfo.Capabilities,
			"last_report":  m.lastReport,
			"report_count": m.reportCount,
			"error_count":  m.errorCount,
		},
	}

	return payload, nil
}

func (m *manager) collectNodeMetrics() *model.NodeMetrics {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// 计算成功率
	var successRate float64
	if m.reportCount > 0 {
		successRate = float64(m.reportCount-m.errorCount) / float64(m.reportCount) * 100
	}

	return &model.NodeMetrics{
		CPUUsage:    0,                                     // CPU使用率需要外部监控
		MemoryUsage: float64(memStats.Alloc) / 1024 / 1024, // MB
		TaskCount:   len(m.nodeInfo.RunningTasks),
		SuccessRate: successRate,
		ErrorCount:  m.errorCount,
		Timestamp:   time.Now(),
	}
}

func (m *manager) sendHeartbeat(ctx context.Context, payload *model.HeartbeatPayload) error {
	url := fmt.Sprintf("%s:%d/api/v1/heartbeat", m.serverURL, m.serverPort)

	// 序列化负载
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat payload: %w", err)
	}

	// 使用retry库进行重试
	return retry.Do(
		func() error {
			// 创建请求 (每次重试都要重新创建，因为body会被消费)
			req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
			if err != nil {
				return fmt.Errorf("failed to create heartbeat request: %w", err)
			}

			// 设置请求头
			req.Header.Set("Content-Type", "application/json")
			if m.authToken != "" {
				req.Header.Set("Authorization", "Bearer "+m.authToken)
			}

			// 发送请求
			resp, err := m.httpClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			// 检查响应状态
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				m.logger.Debug("heartbeat sent successfully", "status", resp.StatusCode)
				return nil
			}
			return fmt.Errorf("heartbeat request failed with status: %d", resp.StatusCode)
		},
		retry.Attempts(5),
		retry.Delay(1*time.Second),
		retry.DelayType(retry.BackOffDelay),
		retry.LastErrorOnly(true),
		retry.OnRetry(func(n uint, err error) {
			m.logger.Debug("retrying heartbeat request", "attempt", n+1, "error", err)
		}),
		retry.Context(ctx),
	)
}

func (m *manager) buildProbeResponse(ctx context.Context) (map[string]interface{}, error) {
	// 获取运行中的任务
	runningTasks, err := m.taskManager.GetRunningTasks(ctx)
	if err != nil {
		m.logger.Warn("failed to get running tasks for probe response", "error", err)
		runningTasks = []*model.TaskSummary{}
	}

	// 获取所有任务
	allTasks, err := m.taskManager.ListTasks(ctx)
	if err != nil {
		m.logger.Warn("failed to get all tasks for probe response", "error", err)
		allTasks = []*model.Task{}
	}

	// 构建任务统计
	taskStats := map[string]interface{}{
		"total":   len(allTasks),
		"running": len(runningTasks),
		"pending": 0,
		"stopped": 0,
		"error":   0,
	}

	for _, task := range allTasks {
		switch task.Status {
		case model.TaskStatusPending:
			taskStats["pending"] = taskStats["pending"].(int) + 1
		case model.TaskStatusStopped:
			taskStats["stopped"] = taskStats["stopped"].(int) + 1
		case model.TaskStatusError:
			taskStats["error"] = taskStats["error"].(int) + 1
		}
	}

	return map[string]interface{}{
		"node_info":     m.nodeInfo,
		"running_tasks": runningTasks,
		"task_stats":    taskStats,
		"metrics":       m.collectNodeMetrics(),
		"system_info": map[string]interface{}{
			"go_version":    runtime.Version(),
			"os":            runtime.GOOS,
			"arch":          runtime.GOARCH,
			"num_cpu":       runtime.NumCPU(),
			"num_goroutine": runtime.NumGoroutine(),
		},
		"heartbeat_stats": map[string]interface{}{
			"last_report":  m.lastReport,
			"report_count": m.reportCount,
			"error_count":  m.errorCount,
			"interval":     m.config.Interval.String(),
		},
	}, nil
}

func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
