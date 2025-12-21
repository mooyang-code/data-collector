package cloudfunction

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mooyang-code/data-collector/internal/heartbeat"
	"github.com/mooyang-code/data-collector/pkg/config"
	"github.com/mooyang-code/data-collector/pkg/model"
	"github.com/tencentyun/scf-go-lib/cloudfunction"
	"github.com/tencentyun/scf-go-lib/functioncontext"
	"trpc.group/trpc-go/trpc-go/log"
)

// CloudFunctionHandler 云函数处理器
type CloudFunctionHandler struct{}

// NewCloudFunctionHandler 创建云函数处理器
func NewCloudFunctionHandler() *CloudFunctionHandler {
	return &CloudFunctionHandler{}
}

// RegisterCloudFunction 注册云函数处理器（在内部启动协程）
func RegisterCloudFunction() {
	handler := NewCloudFunctionHandler()
	go func() {
		cloudfunction.Start(handler.HandleRequest)
	}()
}

// HandleRequest 处理云函数请求 - 通用处理器【入口方法】
func (h *CloudFunctionHandler) HandleRequest(ctx context.Context, event json.RawMessage) (interface{}, error) {
	// 从上下文获取云函数信息
	funcCtx, _ := functioncontext.FromContext(ctx)

	// 解析事件
	var cfEvent model.CloudFunctionEvent
	if err := json.Unmarshal(event, &cfEvent); err != nil {
		// 解析失败，直接报错
		fmt.Printf("解析云函数事件失败, error: %v, event: %s", err, string(event))
		return h.errorResponse("invalid_event", fmt.Sprintf("failed to parse event: %v", err)), nil
	}

	// 设置默认值
	if cfEvent.Timestamp == "" {
		cfEvent.Timestamp = time.Now().Format(time.RFC3339)
	}
	if cfEvent.RequestID == "" {
		cfEvent.RequestID = funcCtx.RequestID
	}
	return h.processCloudFunctionEvent(ctx, cfEvent)
}

// processCloudFunctionEvent 处理云函数事件
func (h *CloudFunctionHandler) processCloudFunctionEvent(ctx context.Context, event model.CloudFunctionEvent) (*model.Response, error) {
	fmt.Printf("处理云函数事件, action: %s, data: %v", event.Action, event.Data)

	// 根据事件类型处理
	switch event.Action {
	case model.EventActionConfig:
		return h.handleConfig(ctx, event)

	case model.EventActionTask:
		return h.handleTask(ctx, event)

	case model.EventActionHealth:
		return h.handleHealth(ctx, event)

	default:
		return h.errorResponse("unknown_event_type", "unknown event Action: "+string(event.Action)), nil
	}
}

// handleConfig 处理配置事件
func (h *CloudFunctionHandler) handleConfig(ctx context.Context, event model.CloudFunctionEvent) (*model.Response, error) {
	// TODO: 实现配置同步逻辑
	log.Infof("处理配置事件, action: %s", event.Action)

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
func (h *CloudFunctionHandler) handleTask(ctx context.Context, event model.CloudFunctionEvent) (*model.Response, error) {
	// TODO: 实现任务处理逻辑
	log.Infof("处理任务事件, action: %s", event.Action)

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
	log.InfoContextf(ctx, "[handleHealth] 执行健康检查, source=%s, ServerIP=%s, ServerPort=%d",
		event.Source, event.ServerIP, event.ServerPort)

	// 处理心跳探测请求（服务端主动发送的探测）
	if event.Source != "heartbeat_probe" {
		// 不是moox后台来的心跳探测请求，直接进行普通健康检查
		log.InfoContextf(ctx, "[handleHealth] 非探测请求，直接返回健康检查响应")
		return h.buildHealthResponse(ctx, event)
	}

	// 调用函数式心跳模块处理探测请求
	log.InfoContextf(ctx, "[handleHealth] 检测到探测请求，调用 ProcessProbe")
	_, err := heartbeat.ProcessProbe(ctx, event)
	if err != nil {
		log.ErrorContextf(ctx, "[handleHealth] 处理心跳探测请求失败: %v", err)
		// 探测处理失败不影响健康检查响应
	} else {
		log.InfoContextf(ctx, "[handleHealth] ProcessProbe 执行成功")
	}

	// 构建健康检查响应
	return h.buildHealthResponse(ctx, event)
}

// buildHealthResponse 构建健康检查响应
func (h *CloudFunctionHandler) buildHealthResponse(ctx context.Context, event model.CloudFunctionEvent) (*model.Response, error) {
	// 从云函数上下文获取信息
	funcCtx, _ := functioncontext.FromContext(ctx)

	// 获取节点ID：优先使用全局配置，降级使用函数名
	nodeID, _ := config.GetNodeInfo()
	if nodeID == "" && funcCtx.FunctionName != "" {
		nodeID = funcCtx.FunctionName
	}
	if nodeID == "" {
		nodeID = "cloud-function" // 最后降级
	}

	// 构建节点信息
	nodeInfo := &model.NodeInfo{
		NodeID:       nodeID,
		NodeType:     "scf",
		Region:       funcCtx.TencentcloudRegion,
		Namespace:    funcCtx.Namespace,
		Version:      funcCtx.FunctionVersion,
		RunningTasks: make([]string, 0),
		Capabilities: []model.CollectorType{
			model.CollectorTypeBinance,
			model.CollectorTypeOKX,
			model.CollectorTypeHuobi,
		},
		Metadata: map[string]string{
			"function_name": funcCtx.FunctionName,
			"request_id":    event.RequestID,
		},
	}

	// 如果是心跳探测请求，使用探测源中的节点ID（优先级最高）
	if event.Source == "heartbeat_probe" && event.Data != nil {
		if probeNodeID, ok := event.Data["node_id"].(string); ok && probeNodeID != "" {
			nodeInfo.NodeID = probeNodeID
		}
	}

	return &model.Response{
		Success: true,
		Message: "cloud function is healthy",
		Data: map[string]interface{}{
			"node_info": nodeInfo,
			"timestamp": time.Now(),
			"status":    "healthy",
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
