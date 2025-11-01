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
	nodeInfo         *model.NodeInfo
	taskManager      task.Manager
	metricsCollector metrics.Collector

	// 服务端信息
	serverIP   string
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
func NewManager(cfg Config, nodeInfo *model.NodeInfo, taskManager task.Manager, metricsCollector metrics.Collector) Manager {
	return &manager{
		config:           cfg,
		nodeInfo:         nodeInfo,
		taskManager:      taskManager,
		metricsCollector: metricsCollector,
		serverIP:         cfg.ServerIP,
		serverPort:       cfg.ServerPort,
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
	logger.Info("心跳管理器启动成功", "component", "heartbeat_manager", "interval", m.config.Interval)
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
	logger.Info("heartbeat manager stopped",
		"component", "heartbeat_manager",
		"report_count", m.reportCount,
		"error_count", m.errorCount)
	return nil
}

func (m *manager) Report(ctx context.Context) error {
	if m.serverIP == "" {
		logger.Debug("no server IP configured, skipping heartbeat report", "component", "heartbeat_manager")
		return nil
	}

	// 构建心跳负载
	payload, err := m.buildHeartbeatPayload(ctx)
	if err != nil {
		logger.Error("failed to build heartbeat payload", "component", "heartbeat_manager", "error", err)
		return fmt.Errorf("failed to build heartbeat payload: %w", err)
	}

	// 发送心跳
	if err := m.sendHeartbeat(ctx, payload); err != nil {
		m.mu.Lock()
		m.errorCount++
		m.mu.Unlock()
		logger.Error("failed to send heartbeat", "component", "heartbeat_manager", "error", err, "report_count", m.reportCount)
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}

	m.mu.Lock()
	m.lastReport = time.Now()
	m.reportCount++
	m.mu.Unlock()

	logger.Debug("heartbeat report sent successfully", "component", "heartbeat_manager")
	return nil
}

func (m *manager) HandleProbe(ctx context.Context, probeData map[string]interface{}) (*model.Response, error) {
	logger.Info("handling heartbeat probe", "component", "heartbeat_manager", "probe_data_keys", getMapKeys(probeData))

	// 更新服务端信息
	if serverIP, ok := probeData["server_ip"].(string); ok {
		m.serverIP = serverIP
		logger.Info("server IP updated from probe", "component", "heartbeat_manager", "server_ip", serverIP)
	}

	if serverPort, ok := probeData["server_port"].(float64); ok {
		m.serverPort = int(serverPort)
		logger.Info("server port updated from probe", "component", "heartbeat_manager", "server_port", m.serverPort)
	}

	if authToken, ok := probeData["auth_token"].(string); ok {
		m.authToken = authToken
		logger.Info("auth token updated from probe", "component", "heartbeat_manager")
	}

	// 构建响应数据
	probeResponse, err := m.buildProbeResponse(ctx)
	if err != nil {
		logger.Error("failed to build probe response", "component", "heartbeat_manager", "error", err)
		return &model.Response{
			Success: false,
			Message: fmt.Sprintf("failed to build response: %v", err),
		}, nil
	}

	logger.Info("probe handled successfully", "component", "heartbeat_manager",
		"node_id", probeResponse.NodeID, "state", probeResponse.State)
	return &model.Response{
		Success:   true,
		Message:   "probe handled successfully",
		Data:      probeResponse,
		Timestamp: time.Now(),
	}, nil
}

func (m *manager) UpdateServerInfo(serverIP string, serverPort int, authToken string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.serverIP = serverIP
	m.serverPort = serverPort
	m.authToken = authToken

	logger.Info("server info updated",
		"component", "heartbeat_manager",
		"server_ip", serverIP,
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
				logger.Error("heartbeat report failed", "component", "heartbeat_manager", "error", err,
					"report_count", m.reportCount, "error_count", m.errorCount)
			} else {
				logger.Debug("heartbeat loop report completed",
					"component", "heartbeat_manager", "report_count", m.reportCount)
			}
		}
	}
}

func (m *manager) buildHeartbeatPayload(ctx context.Context) (*model.HeartbeatPayload, error) {
	// 获取运行中的任务
	runningTasks, err := m.taskManager.GetRunningTasks(ctx)
	if err != nil {
		logger.Warn("failed to get running tasks for heartbeat", "component", "heartbeat_manager", "error", err)
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
	logger.Info("主动上报心跳", "component", "heartbeat_manager", "node_id", payload.NodeID)
	if m.serverIP == "" {
		if m.config.ServerIP != "" {
			m.serverIP = m.config.ServerIP
		}
	}
	if m.serverPort == 0 {
		if m.config.ServerPort != 0 {
			m.serverPort = m.config.ServerPort
		}
	}

	url := fmt.Sprintf("http://%s:%d/gateway/cloudnode/ReportHeartbeat", m.serverIP, m.serverPort)

	// 构建符合后台API规范的请求体
	apiPayload := map[string]interface{}{
		"node_id":   payload.NodeID,
		"node_type": payload.NodeType,
		"metadata":  payload.Metadata,
	}

	// 序列化负载
	data, err := json.Marshal(apiPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat payload: %w", err)
	}
	logger.Info("sendHeartbeat req data:" + string(data))

	// 使用retry库进行重试
	if err := retry.Do(
		func() error {
			// 创建请求 (每次重试都要重新创建，因为body会被消费)
			req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
			if err != nil {
				logger.Error("failed to create heartbeat request", "component", "heartbeat_manager", "error", err, "url", url)
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
				logger.Error("heartbeat HTTP request failed", "component", "heartbeat_manager", "error", err, "url", url)
				return err
			}
			defer resp.Body.Close()

			// 检查响应状态
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				logger.Debug("heartbeat sent successfully", "component", "heartbeat_manager", "status", resp.StatusCode)
				return nil
			}
			logger.Error("heartbeat request failed with non-200 status", "component", "heartbeat_manager",
				"status", resp.StatusCode, "status_text", http.StatusText(resp.StatusCode))
			return fmt.Errorf("heartbeat request failed with status: %d", resp.StatusCode)
		},
		retry.Attempts(5),
		retry.Delay(1*time.Second),
		retry.DelayType(retry.BackOffDelay),
		retry.LastErrorOnly(true),
		retry.OnRetry(func(n uint, err error) {
			logger.Warn("retrying heartbeat request", "component", "heartbeat_manager", "attempt", n+1, "error", err, "url", url)
		}),
		retry.Context(ctx),
	); err != nil {
		logger.Error("heartbeat send failed after all retries", "component", "heartbeat_manager", "error", err,
			"attempts", 5, "url", url)
		return err
	}

	logger.Info("心跳上报成功")
	return nil
}

func (m *manager) buildProbeResponse(ctx context.Context) (*model.ProbeResponse, error) {
	// 获取运行中的任务
	runningTasks, err := m.taskManager.GetRunningTasks(ctx)
	if err != nil {
		logger.Warn("failed to get running tasks for probe response", "component", "heartbeat_manager", "error", err)
		runningTasks = []*model.TaskSummary{}
	}

	// 获取所有任务
	allTasks, err := m.taskManager.ListTasks(ctx)
	if err != nil {
		logger.Warn("failed to get all tasks for probe response", "component", "heartbeat_manager", "error", err)
		allTasks = []*model.Task{}
	}

	// 构建任务统计
	taskStats := model.TaskStatsInfo{
		Total:   len(allTasks),
		Running: len(runningTasks),
		Pending: 0,
		Stopped: 0,
		Error:   0,
	}

	for _, task := range allTasks {
		switch task.Status {
		case model.TaskStatusPending:
			taskStats.Pending++
		case model.TaskStatusStopped:
			taskStats.Stopped++
		case model.TaskStatusError:
			taskStats.Error++
		}
	}

	// 构建系统信息
	systemInfo := model.SystemInfo{
		GoVersion:    runtime.Version(),
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: runtime.NumGoroutine(),
	}

	// 构建心跳统计信息
	heartbeatInfo := model.HeartbeatInfo{
		LastReport:  m.lastReport,
		ReportCount: m.reportCount,
		ErrorCount:  m.errorCount,
		Interval:    m.config.Interval.String(),
		ServerIP:    m.serverIP,
		ServerPort:  m.serverPort,
	}

	// 获取指标
	nodeMetrics := m.collectNodeMetrics()

	return &model.ProbeResponse{
		NodeID: m.nodeInfo.NodeID,
		State:  "running",
		Details: model.ProbeDetails{
			NodeInfo:      m.nodeInfo,
			RunningTasks:  runningTasks,
			TaskStats:     taskStats,
			Metrics:       nodeMetrics,
			SystemInfo:    systemInfo,
			HeartbeatInfo: heartbeatInfo,
		},
		Timestamp: time.Now(),
	}, nil
}

func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
