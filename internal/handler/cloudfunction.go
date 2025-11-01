package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tencentyun/scf-go-lib/cloudfunction"
	"github.com/tencentyun/scf-go-lib/functioncontext"

	"github.com/mooyang-code/data-collector/internal/bootstrap"
	"github.com/mooyang-code/data-collector/internal/heartbeat"
	"github.com/mooyang-code/data-collector/pkg/logger"
	"github.com/mooyang-code/data-collector/pkg/model"
)

// CloudFunctionHandler 云函数处理器
type CloudFunctionHandler struct {
	bootstrap        *bootstrap.Bootstrap // 用于在接收到请求时，确保系统已初始化，若未初始化，则执行初始化工作
	logger           logger.Logger
	heartbeatManager heartbeat.Manager // 心跳管理器，用于处理心跳探测功能
	heartbeatConfig  heartbeat.Config  // 心跳配置，用于获取默认端口
	started          bool
}

// NewCloudFunctionHandler 创建云函数处理器
func NewCloudFunctionHandler(bs *bootstrap.Bootstrap, heartbeatManager heartbeat.Manager, heartbeatConfig heartbeat.Config) *CloudFunctionHandler {
	var handlerLogger logger.Logger
	if bs.GetLogger() != nil {
		handlerLogger = bs.GetLogger().With("component", "cloud_function_handler")
	} else {
		handlerLogger = logger.NewDefault().With("component", "cloud_function_handler")
	}

	return &CloudFunctionHandler{
		bootstrap:        bs,
		logger:           handlerLogger,
		heartbeatManager: heartbeatManager,
		heartbeatConfig:  heartbeatConfig,
	}
}

// ensureStarted 确保系统已启动
func (h *CloudFunctionHandler) ensureStarted(ctx context.Context) error {
	if h.started {
		return nil
	}

	// 检查bootstrap状态
	if h.bootstrap.GetState() != "running" {
		// 初始化系统
		if err := h.bootstrap.Init(ctx); err != nil {
			return fmt.Errorf("failed to init bootstrap: %w", err)
		}

		// 从云函数上下文更新配置（在Init之后，这样节点信息已经创建）
		h.updateConfigFromContext(ctx)

		// 启动系统
		if err := h.bootstrap.Start(ctx); err != nil {
			return fmt.Errorf("failed to start bootstrap: %w", err)
		}
	}

	h.started = true
	logger.InfoContextf(ctx, "云函数系统启动成功")
	return nil
}

// HandleRequest 处理云函数请求 - 通用处理器
func (h *CloudFunctionHandler) HandleRequest(ctx context.Context, event json.RawMessage) (interface{}, error) {
	// 从上下文获取云函数信息
	funcCtx, _ := functioncontext.FromContext(ctx)

	// 创建带上下文的logger，自动携带request_id和function_name
	requestLogger := h.logger.With(
		"function_name", funcCtx.FunctionName,
		"request_id", funcCtx.RequestID,
	)

	// 将 logger 存储到 context 中，后续可以通过 logger.FromContext(ctx) 获取
	ctx = logger.WithContext(ctx, requestLogger)
	logger.InfoContextf(ctx, "云函数入口 HandleRequest", "event_size", len(event))

	// 确保系统已初始化
	if err := h.ensureStarted(ctx); err != nil {
		return h.errorResponse("system_not_ready", fmt.Sprintf("failed to start system: %v", err)), nil
	}

	// 解析事件
	var cfEvent model.CloudFunctionEvent
	if err := json.Unmarshal(event, &cfEvent); err != nil {
		// 解析失败，直接报错
		logger.ErrorContextf(ctx, "解析云函数事件失败", "error", err, "event", string(event))
		return h.errorResponse("invalid_event", fmt.Sprintf("failed to parse event: %v", err)), nil
	}

	// 设置默认值
	if cfEvent.Timestamp == "" {
		cfEvent.Timestamp = time.Now().Format(time.RFC3339)
	}
	if cfEvent.RequestID == "" {
		cfEvent.RequestID = funcCtx.RequestID
	}

	return h.handleCloudFunctionEvent(ctx, cfEvent)
}

// updateConfigFromContext 从云函数上下文更新配置
func (h *CloudFunctionHandler) updateConfigFromContext(ctx context.Context) {
	funcCtx, ok := functioncontext.FromContext(ctx)
	if !ok {
		logger.WarnContextf(ctx, "无法获取函数上下文，使用默认配置")
		return
	}

	logger.InfoContextf(ctx, "从函数上下文更新配置",
		"function_name", funcCtx.FunctionName,
		"region", funcCtx.TencentcloudRegion,
		"namespace", funcCtx.Namespace)

	// 更新节点信息
	nodeInfo := h.bootstrap.GetNodeInfo()
	if nodeInfo != nil {
		// 使用云函数名称作为节点ID
		if funcCtx.FunctionName != "" {
			nodeInfo.NodeID = funcCtx.FunctionName
		}

		// 在云函数环境中，NodeType 设置为 "scf"
		nodeInfo.NodeType = "scf"

		// 从云函数上下文获取区域信息
		if funcCtx.TencentcloudRegion != "" {
			nodeInfo.Region = funcCtx.TencentcloudRegion
		}

		// 从云函数上下文获取命名空间信息
		if funcCtx.Namespace != "" {
			nodeInfo.Namespace = funcCtx.Namespace
		}

		// 添加云函数特定的元数据
		if nodeInfo.Metadata == nil {
			nodeInfo.Metadata = make(map[string]string)
		}
		nodeInfo.Metadata["function_name"] = funcCtx.FunctionName
		nodeInfo.Metadata["function_version"] = funcCtx.FunctionVersion
		nodeInfo.Metadata["region"] = funcCtx.TencentcloudRegion
		nodeInfo.Metadata["namespace"] = funcCtx.Namespace

		logger.InfoContextf(ctx, "已从函数上下文更新节点配置",
			"function_name", funcCtx.FunctionName,
			"region", funcCtx.TencentcloudRegion,
			"namespace", funcCtx.Namespace,
			"version", funcCtx.FunctionVersion)
	}
}

// handleCloudFunctionEvent 处理云函数事件
func (h *CloudFunctionHandler) handleCloudFunctionEvent(ctx context.Context, event model.CloudFunctionEvent) (*model.Response, error) {
	logger.InfoContextf(ctx, "处理云函数事件", "action", event.Action, "data", event.Data)

	// 获取管理器
	taskManager, configManager, _ := h.bootstrap.GetManagers()

	// 根据事件类型处理
	switch event.Action {
	case model.EventActionConfig:
		return h.handleConfig(ctx, event, configManager)

	case model.EventActionTask:
		return h.handleTask(ctx, event, taskManager)

	case model.EventActionHealth:
		return h.handleHealth(ctx, event)

	default:
		return h.errorResponse("unknown_event_type", "unknown event Action: "+string(event.Action)), nil
	}
}

// handleConfig 处理配置事件
func (h *CloudFunctionHandler) handleConfig(ctx context.Context, event model.CloudFunctionEvent, configManager any) (*model.Response, error) {
	// TODO: 实现配置同步逻辑
	logger.InfoContextf(ctx, "处理配置事件", "action", event.Action)

	return &model.Response{
		Success: true,
		Message: "config synchronized successfully",
		Data: map[string]interface{}{
			"action": event.Action,
		},
		RequestID: event.RequestID,
		Timestamp: time.Now(),
	}, nil
}

// handleTask 处理任务事件
func (h *CloudFunctionHandler) handleTask(ctx context.Context, event model.CloudFunctionEvent, taskManager any) (*model.Response, error) {
	// TODO: 实现任务处理逻辑
	logger.InfoContextf(ctx, "处理任务事件", "action", event.Action)

	return &model.Response{
		Success: true,
		Message: "task processed successfully",
		Data: map[string]interface{}{
			"action": event.Action,
		},
		RequestID: event.RequestID,
		Timestamp: time.Now(),
	}, nil
}

// handleHealth 处理健康检查事件（包括心跳探测功能）
func (h *CloudFunctionHandler) handleHealth(ctx context.Context, event model.CloudFunctionEvent) (*model.Response, error) {
	logger.InfoContextf(ctx, "执行健康检查", "source", event.Source)

	// 处理心跳探测请求（服务端主动发送的探测）
	if h.heartbeatManager == nil || event.Data == nil {
		logger.InfoContextf(ctx, " 提前返回 heartbeatManager is nil")
		return h.buildHealthResponse(ctx, event)
	}
	if event.Source != "heartbeat_probe" {
		// 不是moox后台来的心跳探测请求，直接进行普通健康检查
		return h.buildHealthResponse(ctx, event)
	}
	logger.InfoContextf(ctx, "处理心跳探测请求", "source", event.Source)

	// 提取服务端IP信息
	var serverIP string
	if ip, ok := event.Data["internal_ip"].(string); ok && ip != "" {
		serverIP = ip
		logger.InfoContextf(ctx, "从心跳探测获取到内部IP", "internal_ip", serverIP)
	} else if ip, ok := event.Data["public_ip"].(string); ok && ip != "" {
		serverIP = ip
		logger.InfoContextf(ctx, "从心跳探测获取到公网IP", "public_ip", serverIP)
	}

	// 如果没有获取到服务端IP，从配置中读取作为兜底
	if serverIP == "" {
		if h.heartbeatConfig.ServerIP != "" {
			serverIP = h.heartbeatConfig.ServerIP
			logger.InfoContextf(ctx, "未获取到服务端IP，使用配置默认IP",
				"server_ip", serverIP, "config_source", "heartbeat.server_ip")
		}
	}

	// 提取服务端端口信息
	serverPort := h.heartbeatConfig.ServerPort // 从配置中获取默认端口
	if portValue, ok := event.Data["server_port"].(float64); ok && portValue > 0 {
		serverPort = int(portValue)
		logger.DebugContextf(ctx, "从心跳探测获取到服务端端口", "server_port", serverPort)
	} else {
		logger.DebugContextf(ctx, "未获取到服务端端口，使用配置默认端口",
			"default_port", serverPort, "config_source", "heartbeat.server_port")
	}

	// 更新服务端信息（只有当获取到有效的服务端IP时才更新）
	if serverIP != "" {
		if err := h.heartbeatManager.UpdateServerInfo(serverIP, serverPort, ""); err != nil {
			logger.WarnContextf(ctx, "更新服务端信息失败", "server_ip", serverIP, "server_port", serverPort, "error", err)
		}
	}

	// 构建健康检查响应
	return h.buildHealthResponse(ctx, event)
}

// buildHealthResponse 构建健康检查响应
func (h *CloudFunctionHandler) buildHealthResponse(ctx context.Context, event model.CloudFunctionEvent) (*model.Response, error) {
	nodeInfo := h.bootstrap.GetNodeInfo()
	healthData := map[string]interface{}{
		"state":     h.bootstrap.GetState(),
		"node_info": nodeInfo,
		"timestamp": time.Now(),
	}

	return &model.Response{
		Success:   true,
		Message:   "cloud function is healthy",
		Data:      healthData,
		RequestID: event.RequestID,
		Timestamp: time.Now(),
	}, nil
}

// errorResponse 创建错误响应
func (h *CloudFunctionHandler) errorResponse(code, message string) *model.Response {
	return &model.Response{
		Success:   false,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// RegisterCloudFunction 注册云函数处理器
func RegisterCloudFunction(bs *bootstrap.Bootstrap, heartbeatManager heartbeat.Manager, heartbeatConfig heartbeat.Config) {
	handler := NewCloudFunctionHandler(bs, heartbeatManager, heartbeatConfig)

	// 注册通用处理器
	cloudfunction.Start(handler.HandleRequest)
}
