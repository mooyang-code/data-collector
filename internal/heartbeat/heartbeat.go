package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"

	"github.com/avast/retry-go"
	"github.com/mooyang-code/data-collector/pkg/config"
	"github.com/mooyang-code/data-collector/pkg/model"
	"github.com/tencentyun/scf-go-lib/functioncontext"
	"trpc.group/trpc-go/trpc-go/log"
)

// ServerResponse 服务端响应结构
type ServerResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    []any  `json:"data"` // 统一响应格式中data为数组
	Total   *int64 `json:"total,omitempty"`
}

// ScheduledHeartbeat 框架定时器入口函数 - 定时心跳
func ScheduledHeartbeat(ctx context.Context, _ string) error {
	nodeID, version := config.GetNodeInfo()
	log.WithContextFields(ctx, "func", "ScheduledHeartbeat", "version", version, "nodeID", nodeID)

	log.DebugContextf(ctx, "ScheduledHeartbeat Enter")
	if err := ReportHeartbeat(ctx); err != nil {
		log.ErrorContextf(ctx, "scheduled heartbeat failed: %v", err)
		return err
	}
	log.DebugContextf(ctx, "ScheduledHeartbeat Success")
	return nil
}

// ReportHeartbeat 发送心跳上报服务端
func ReportHeartbeat(ctx context.Context) error {
	serverIP, serverPort := config.GetServerInfo()
	nodeID, localVersion := config.GetNodeInfo()
	log.InfoContextf(ctx, "ReportHeartbeat 开始: serverIP=%s:%d, nodeID=%s, version=%s", serverIP, serverPort, nodeID, localVersion)

	// 检查NodeID是否配置
	if nodeID == "" {
		log.WarnContextf(ctx, "NodeID 为空，跳过心跳上报。请确保服务端探测请求已触发 ProcessProbe")
		return nil
	}
	if serverIP == "" {
		log.WarnContextf(ctx, "服务端 IP 未配置，跳过心跳上报")
		return nil
	}

	// 构建本节点负载信息
	payload, err := buildPayloadInfo()
	if err != nil {
		log.ErrorContextf(ctx, "failed to build heartbeat payload: %v", err)
		return fmt.Errorf("failed to build heartbeat payload: %w", err)
	}

	// 发送心跳并获取包版本信息
	packageVersion, err := sendToServer(ctx, payload, serverIP, serverPort)
	if err != nil {
		log.ErrorContextf(ctx, "failed to send heartbeat: %v", err)
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}

	// 检查版本一致性，如果不一致则终止服务（避免云平台同时保留多个版本节点的运行）
	if packageVersion != "" && packageVersion != localVersion {
		log.FatalContextf(ctx, "版本不一致，终止服务 - 本地版本: %s, 服务端版本: %s", localVersion, packageVersion)
	}
	return nil
}

// ProcessProbe 处理心跳探测请求【服务端来的探测请求】
func ProcessProbe(ctx context.Context, event model.CloudFunctionEvent) (*model.Response, error) {
	log.InfoContextf(ctx, "[ProcessProbe] 开始处理探测请求")

	// 从上下文获取云函数信息，更新NodeID
	funcCtx, ok := functioncontext.FromContext(ctx)
	if ok && funcCtx.FunctionName != "" {
		currentNodeID, currentVersion := config.GetNodeInfo()
		log.InfoContextf(ctx, "[ProcessProbe] 当前 NodeID=%s, 云函数名=%s", currentNodeID, funcCtx.FunctionName)

		// 无条件更新 NodeID 为云函数名称
		config.UpdateNodeInfo(funcCtx.FunctionName, currentVersion)
		log.InfoContextf(ctx, "[ProcessProbe] NodeID 已更新为 %s", funcCtx.FunctionName)
	} else {
		log.WarnContextf(ctx, "[ProcessProbe] 无法从上下文获取云函数信息, ok=%v", ok)
	}

	// 更新服务端连接信息的配置（用于本节点 主动上报心跳和拉取配置）
	log.InfoContextf(ctx, "[ProcessProbe] event.ServerIP=%s, event.ServerPort=%d", event.ServerIP, event.ServerPort)
	if event.ServerIP != "" && event.ServerPort > 0 {
		log.InfoContextf(ctx, "[ProcessProbe] 更新服务端地址 %s:%d", event.ServerIP, event.ServerPort)
		config.UpdateServerInfo(event.ServerIP, event.ServerPort)
		config.UpdateCompassMap(map[string]string{
			config.MooxServerServiceName: fmt.Sprintf("%s:%d", event.ServerIP, event.ServerPort),
		})

		// 验证更新是否成功
		verifyIP, verifyPort := config.GetServerInfo()
		log.InfoContextf(ctx, "[ProcessProbe] 验证更新后的服务端地址: %s:%d", verifyIP, verifyPort)
	} else {
		log.WarnContextf(ctx, "[ProcessProbe] 服务端地址信息缺失 ServerIP=%s, ServerPort=%d", event.ServerIP, event.ServerPort)
	}

	// 构建响应数据
	probeResponse, err := buildProbeResponse()
	if err != nil {
		return &model.Response{
			Success: false,
			Message: fmt.Sprintf("failed to build response: %v", err),
		}, nil
	}

	return &model.Response{
		Success:   true,
		Message:   "probe handled successfully",
		Data:      probeResponse,
		Timestamp: time.Now(),
	}, nil
}

func buildPayloadInfo() (*model.HeartbeatPayload, error) {
	// 从全局配置获取节点信息
	nodeID, version := config.GetNodeInfo()

	// 获取节点指标
	nodeMetrics := collectNodeMetrics()

	// 构建心跳负载
	payload := &model.HeartbeatPayload{
		NodeID:       nodeID,
		NodeType:     "scf",
		Timestamp:    time.Now(),
		RunningTasks: []*model.TaskSummary{},
		Metrics:      nodeMetrics,
		Metadata: map[string]interface{}{
			"version":    version,
			"go_version": runtime.Version(),
			"os":         runtime.GOOS,
			"arch":       runtime.GOARCH,
		},
	}
	return payload, nil
}

func collectNodeMetrics() *model.NodeMetrics {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return &model.NodeMetrics{
		CPUUsage:    0,
		MemoryUsage: float64(memStats.Alloc) / 1024 / 1024, // MB
		TaskCount:   0,
		SuccessRate: 100,
		ErrorCount:  0,
		Timestamp:   time.Now(),
	}
}

func sendToServer(ctx context.Context, payload *model.HeartbeatPayload, serverIP string, serverPort int) (string, error) {
	log.InfoContextf(ctx, "sending heartbeat, node_id: %s", payload.NodeID)
	// 检查必要参数
	if serverIP == "" || serverPort <= 0 {
		return "", fmt.Errorf("invalid server address: %s:%d", serverIP, serverPort)
	}

	packageVersion, err := executeReport(ctx, payload, serverIP, serverPort)
	if err != nil {
		return "", fmt.Errorf("failed to send heartbeat: %w", err)
	}
	return packageVersion, nil
}

// executeReport 准备并发送心跳请求
func executeReport(ctx context.Context, payload *model.HeartbeatPayload, serverIP string, serverPort int) (string, error) {
	url := fmt.Sprintf("http://%s:%d/gateway/cloudnode/ReportHeartbeatInner", serverIP, serverPort)

	// 构建请求体
	apiPayload := map[string]interface{}{
		"node_id":   payload.NodeID,
		"node_type": payload.NodeType,
		"metadata":  payload.Metadata,
	}

	// 序列化请求数据
	data, err := json.Marshal(apiPayload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal heartbeat payload: %w", err)
	}

	// 创建HTTP客户端
	httpClient := &http.Client{Timeout: 5 * time.Second}
	var packageVersion string

	// 使用重试机制发送请求
	err = retry.Do(
		func() error {
			return sendSingleHeartbeat(ctx, url, data, httpClient, &packageVersion)
		},
		retry.Attempts(5),
		retry.Delay(1*time.Second),
		retry.DelayType(retry.BackOffDelay),
		retry.LastErrorOnly(true),
		retry.OnRetry(func(n uint, err error) {
			log.WarnContextf(ctx, "retrying heartbeat request, attempt: %d, error: %v", n+1, err)
		}),
		retry.Context(ctx),
	)
	return packageVersion, err
}

// sendSingleHeartbeat 发送单次心跳请求
func sendSingleHeartbeat(ctx context.Context, url string, data []byte, httpClient *http.Client, packageVersion *string) error {
	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create heartbeat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 发送请求并检查错误
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respData, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("heartbeat request failed with status: %d, response: %s", resp.StatusCode, string(respData))
	}
	log.DebugContextf(ctx, "heartbeat sent successfully, status: %d", resp.StatusCode)

	// 读取和解析响应
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// 解析服务端响应
	version, parseErr := parseServerResponse(respData)
	if parseErr != nil {
		log.WarnContextf(ctx, "failed to parse server response: %v", parseErr)
		return nil // 不影响心跳上报，只记录警告
	}
	*packageVersion = version
	return nil
}

// parseServerResponse 解析服务端响应，提取包版本信息
func parseServerResponse(respData []byte) (string, error) {
	var serverResp ServerResponse
	if err := json.Unmarshal(respData, &serverResp); err != nil {
		return "", fmt.Errorf("failed to parse server response: %w", err)
	}
	// 检查响应状态码（200表示成功）
	if serverResp.Code != 200 {
		return "", fmt.Errorf("server returned error code: %d, message: %s", serverResp.Code, serverResp.Message)
	}

	// 检查数据数组
	if serverResp.Data == nil || len(serverResp.Data) == 0 {
		return "", nil // 返回空版本而不是错误
	}

	// 获取第一个数据元素并提取包版本
	firstData := serverResp.Data[0]
	if dataMap, ok := firstData.(map[string]interface{}); ok {
		if packageVersion, exists := dataMap["package_version"]; exists {
			if versionStr, ok := packageVersion.(string); ok {
				return versionStr, nil
			}
		}
	}
	return "", nil // 如果没有找到package_version字段，返回空字符串而不是错误
}

// BuildProbeResponseOptions 构建探测响应的选项
type BuildProbeResponseOptions struct {
	Config       *ProbeResponseConfig
	IncludeTasks bool
	CustomState  string
}

// BuildProbeResponseOption 构建选项函数类型
type BuildProbeResponseOption func(*BuildProbeResponseOptions)

// buildProbeResponse 构建心跳探测响应
func buildProbeResponse(options ...BuildProbeResponseOption) (*model.ProbeResponse, error) {
	// 1. 解析配置选项
	opts := &BuildProbeResponseOptions{
		Config: DefaultProbeResponseConfig(),
	}
	for _, option := range options {
		option(opts)
	}

	// 2. 获取节点信息
	nodeID, version, err := getNodeInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get node info: %w", err)
	}

	// 3. 获取系统信息
	systemInfo, err := getSystemInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get system info: %w", err)
	}

	// 4. 获取节点指标
	nodeMetrics, err := getNodeMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to get node metrics: %w", err)
	}

	// 5. 确定节点状态
	nodeState := determineNodeState(opts.CustomState, opts.Config.State)

	// 6. 构建运行任务信息
	var runningTasks []*model.TaskSummary
	if opts.IncludeTasks {
		runningTasks = getRunningTasks()
	}

	// 7. 获取心跳统计信息
	heartbeatInfo := getHeartbeatInfo(opts.Config)

	// 8. 构建完整的探测响应
	probeResponse := &model.ProbeResponse{
		NodeID:    nodeID,
		State:     nodeState,
		Timestamp: time.Now(),
		Details: model.ProbeDetails{
			NodeInfo:      createNodeInfo(nodeID, version),
			RunningTasks:  runningTasks,
			TaskStats:     getTaskStatistics(),
			Metrics:       nodeMetrics,
			SystemInfo:    systemInfo,
			HeartbeatInfo: heartbeatInfo,
		},
	}
	return probeResponse, nil
}

// ProbeResponseConfig 探测响应配置
type ProbeResponseConfig struct {
	State       string
	Interval    string
	ReportCount int64
	ErrorCount  int64
}

// DefaultProbeResponseConfig 默认探测响应配置
func DefaultProbeResponseConfig() *ProbeResponseConfig {
	return &ProbeResponseConfig{
		State:       "running",
		Interval:    "30s",
		ReportCount: 0,
		ErrorCount:  0,
	}
}

// getNodeInfo 获取节点信息
func getNodeInfo() (nodeID, version string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic while getting node info: %v", r)
		}
	}()

	nodeID, version = config.GetNodeInfo()
	if nodeID == "" {
		return "", "", fmt.Errorf("node ID is empty")
	}
	return nodeID, version, nil
}

// getSystemInfo 获取系统信息
func getSystemInfo() (model.SystemInfo, error) {
	return model.SystemInfo{
		GoVersion:    runtime.Version(),
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: runtime.NumGoroutine(),
	}, nil
}

// getNodeMetrics 获取节点指标
func getNodeMetrics() (*model.NodeMetrics, error) {
	return collectNodeMetrics(), nil
}

// determineNodeState 确定节点状态
func determineNodeState(customState, defaultState string) string {
	if customState != "" {
		return customState
	}
	return defaultState
}

// getRunningTasks 获取运行任务（目前返回空，未来可扩展）
func getRunningTasks() []*model.TaskSummary {
	// TODO: 从任务管理器获取实际运行的任务
	// 目前返回空切片，保持向后兼容
	return []*model.TaskSummary{}
}

// getTaskStatistics 获取任务统计信息
func getTaskStatistics() model.TaskStatsInfo {
	// TODO: 从任务管理器获取实际的统计数据
	// 目前返回默认值，保持向后兼容
	return model.TaskStatsInfo{
		Total:   0,
		Running: 0,
		Pending: 0,
		Stopped: 0,
		Error:   0,
	}
}

// getHeartbeatInfo 获取心跳统计信息
func getHeartbeatInfo(probeConfig *ProbeResponseConfig) model.HeartbeatInfo {
	// 从全局配置获取服务器信息
	serverIP, serverPort := config.GetServerInfo()

	return model.HeartbeatInfo{
		LastReport:  time.Now(),
		ReportCount: probeConfig.ReportCount,
		ErrorCount:  probeConfig.ErrorCount,
		Interval:    probeConfig.Interval,
		ServerIP:    serverIP,
		ServerPort:  serverPort,
	}
}

// createNodeInfo 创建节点信息
func createNodeInfo(nodeID, version string) *model.NodeInfo {
	return &model.NodeInfo{
		NodeID:       nodeID,
		NodeType:     "scf",
		Version:      version,
		RunningTasks: make([]string, 0),
		Capabilities: []model.CollectorType{
			model.CollectorTypeBinance,
			model.CollectorTypeOKX,
			model.CollectorTypeHuobi,
		},
		Metadata: map[string]string{
			"go_version": runtime.Version(),
			"os":         runtime.GOOS,
			"arch":       runtime.GOARCH,
		},
	}
}
