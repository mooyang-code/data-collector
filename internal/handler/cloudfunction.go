package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tencentyun/scf-go-lib/cloudfunction"
	"github.com/tencentyun/scf-go-lib/functioncontext"

	"github.com/mooyang-code/data-collector/internal/bootstrap"
	"github.com/mooyang-code/data-collector/pkg/logger"
	"github.com/mooyang-code/data-collector/pkg/model"
)

// CloudFunctionHandler 云函数处理器
type CloudFunctionHandler struct {
	bootstrap *bootstrap.Bootstrap // 用于在接收到请求时，确保系统已初始化，若未初始化，则执行初始化工作
	logger    logger.Logger
	started   bool
}

// NewCloudFunctionHandler 创建云函数处理器
func NewCloudFunctionHandler(bs *bootstrap.Bootstrap) *CloudFunctionHandler {
	var handlerLogger logger.Logger
	if bs.GetLogger() != nil {
		handlerLogger = bs.GetLogger().With("component", "cloud_function_handler")
	} else {
		handlerLogger = logger.NewDefault().With("component", "cloud_function_handler")
	}

	return &CloudFunctionHandler{
		bootstrap: bs,
		logger:    handlerLogger,
	}
}

// HandleRequest 处理云函数请求 - 通用处理器
func (h *CloudFunctionHandler) HandleRequest(ctx context.Context, event json.RawMessage) (interface{}, error) {
	// 从上下文获取云函数信息
	funcCtx, _ := functioncontext.FromContext(ctx)
	h.logger.Info("received cloud function request",
		"function_name", funcCtx.FunctionName,
		"request_id", funcCtx.RequestID,
		"event_size", len(event))

	// 确保系统已初始化
	if err := h.ensureStarted(ctx); err != nil {
		return h.errorResponse("system_not_ready", fmt.Sprintf("failed to start system: %v", err)), nil
	}

	// 解析事件
	var cfEvent model.CloudFunctionEvent
	if err := json.Unmarshal(event, &cfEvent); err != nil {
		// 解析失败，直接报错
		h.logger.Error("failed to parse cloud function event", "error", err, "event", string(event))
		return h.errorResponse("invalid_event", fmt.Sprintf("failed to parse event: %v", err)), nil
	}

	// 设置默认值
	if cfEvent.Timestamp.IsZero() {
		cfEvent.Timestamp = time.Now()
	}
	if cfEvent.RequestID == "" {
		cfEvent.RequestID = fmt.Sprintf("req-%d", time.Now().UnixNano())
	}

	return h.handleCloudFunctionEvent(ctx, cfEvent)
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
	h.logger.Info("cloud function system started successfully")
	return nil
}

// updateConfigFromContext 从云函数上下文更新配置
func (h *CloudFunctionHandler) updateConfigFromContext(ctx context.Context) {
	funcCtx, ok := functioncontext.FromContext(ctx)
	if !ok {
		h.logger.Warn("failed to get function context, using default config")
		return
	}

	h.logger.Info("updating config from function context",
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

		h.logger.Info("updated node config from function context",
			"function_name", funcCtx.FunctionName,
			"region", funcCtx.TencentcloudRegion,
			"namespace", funcCtx.Namespace,
			"version", funcCtx.FunctionVersion)
	}
}

// handleCloudFunctionEvent 处理云函数事件
func (h *CloudFunctionHandler) handleCloudFunctionEvent(ctx context.Context, event model.CloudFunctionEvent) (*model.Response, error) {
	h.logger.Info("handling cloud function event",
		"type", event.Type,
		"action", event.Action,
		"request_id", event.RequestID)

	// 获取管理器
	taskManager, configManager, _ := h.bootstrap.GetManagers()

	// 根据事件类型处理
	switch event.Type {
	case model.EventTypeConfig:
		return h.handleConfig(ctx, event, configManager)

	case model.EventTypeTask:
		return h.handleTask(ctx, event, taskManager)

	case model.EventTypeHealth:
		return h.handleHealth(ctx, event)

	default:
		return h.errorResponse("unknown_event_type", "unknown event type: "+string(event.Type)), nil
	}
}

// handleConfig 处理配置事件
func (h *CloudFunctionHandler) handleConfig(ctx context.Context, event model.CloudFunctionEvent, configManager any) (*model.Response, error) {
	// TODO: 实现配置同步逻辑

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

// handleHealth 处理健康检查事件
func (h *CloudFunctionHandler) handleHealth(ctx context.Context, event model.CloudFunctionEvent) (*model.Response, error) {
	// 简单返回节点健康即可
	return &model.Response{
		Success: true,
		Message: "cloud function is healthy",
		Data: map[string]interface{}{
			"state":     h.bootstrap.GetState(),
			"node_info": h.bootstrap.GetNodeInfo(),
			"timestamp": time.Now(),
		},
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
func RegisterCloudFunction(bs *bootstrap.Bootstrap) {
	handler := NewCloudFunctionHandler(bs)

	// 注册通用处理器
	cloudfunction.Start(handler.HandleRequest)
}
