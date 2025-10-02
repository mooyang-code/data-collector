package serverless

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/mooyang-code/data-collector/internal/config"
	"github.com/mooyang-code/data-collector/internal/core/app"
	"github.com/mooyang-code/data-collector/internal/serverless/cache"
	"github.com/mooyang-code/data-collector/internal/serverless/heartbeat"
	"github.com/mooyang-code/data-collector/internal/serverless/manager"
	"github.com/tencentyun/scf-go-lib/cloudfunction"
	"github.com/tencentyun/scf-go-lib/functioncontext"
)

// Handler 云函数处理器
type Handler struct {
	config           *config.Config
	appManager       *app.Manager
	taskManager      *manager.TaskManager
	taskSync         *manager.TaskSynchronizer
	heartbeatService *heartbeat.HeartbeatService
	initialized      bool
	nodeID           string
	mooxURL          string
	mu               sync.RWMutex
}

// NewHandler 创建云函数处理器
func NewHandler(cfg *config.Config, appManager *app.Manager) (*Handler, error) {
	return &Handler{
		config:      cfg,
		appManager:  appManager,
		initialized: false,
	}, nil
}

// Start 启动云函数处理器
func (h *Handler) Start() {
	log.Printf("[Handler] Starting cloud function handler, waiting for initialization...")
	// 启动云函数处理
	cloudfunction.Start(h.handleRequest)
}

// initialize 初始化处理器
func (h *Handler) initialize(nodeID, mooxURL string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.initialized {
		return nil
	}

	log.Printf("[Handler] Initializing with nodeID=%s, mooxURL=%s", nodeID, mooxURL)

	// 保存节点ID和Moox URL
	h.nodeID = nodeID
	h.mooxURL = mooxURL

	// 获取缓存服务地址
	cacheEndpoint := h.config.Runtime.Serverless.CacheEndpoint
	if cacheEndpoint == "" {
		return fmt.Errorf("cache endpoint not configured")
	}

	// 1. 创建缓存客户端
	cacheClient := cache.NewAPICacheClient(cacheEndpoint)

	// 2. 创建任务管理器
	h.taskManager = manager.NewTaskManager(h.appManager)

	// 3. 创建任务同步器
	h.taskSync = manager.NewTaskSynchronizer(nodeID, cacheClient, h.taskManager)

	// 4. 创建心跳服务
	h.heartbeatService = heartbeat.NewHeartbeatService(nodeID, mooxURL, h.taskManager)

	// 启动后台服务
	ctx := context.Background()

	// 启动任务同步
	go h.taskSync.Start(ctx)
	log.Printf("[Handler] Task synchronizer started")

	// 启动心跳服务
	go h.heartbeatService.Start(ctx)
	log.Printf("[Handler] Heartbeat service started")

	h.initialized = true
	log.Printf("[Handler] Initialization completed")
	return nil
}

// handleRequest 处理云函数请求
func (h *Handler) handleRequest(ctx context.Context, event interface{}) (interface{}, error) {
	// 打印函数上下文信息
	funcCtx, _ := functioncontext.FromContext(ctx)
	log.Printf("云函数被调用: FunctionName=%s, RequestID=%s",
		funcCtx.FunctionName, funcCtx.RequestID)

	// 解析事件数据
	eventData, err := h.parseEvent(event)
	if err != nil {
		return h.errorResponse("解析事件失败", err), nil
	}

	// 根据事件类型处理
	switch eventData["_action"] {
	case "init":
		return h.handleInit(ctx, eventData)
	case "status":
		return h.handleStatus(ctx)
	default:
		return h.errorResponse("未知的操作类型", fmt.Errorf("action: %v", eventData["action"])), nil
	}
}

// parseEvent 解析云函数事件
func (h *Handler) parseEvent(event interface{}) (map[string]interface{}, error) {
	// 尝试直接转换为map
	if eventMap, ok := event.(map[string]interface{}); ok {
		return eventMap, nil
	}

	// 尝试JSON解析
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("序列化事件失败: %w", err)
	}

	var eventMap map[string]interface{}
	if err := json.Unmarshal(eventBytes, &eventMap); err != nil {
		return nil, fmt.Errorf("反序列化事件失败: %w", err)
	}

	return eventMap, nil
}

// InitRequest 初始化请求
type InitRequest struct {
	NodeID  string `json:"node_id"`
	MooxURL string `json:"moox_url"`
}

// handleInit 处理初始化请求
func (h *Handler) handleInit(ctx context.Context, eventData map[string]interface{}) (interface{}, error) {
	log.Printf("[Handler] Processing init request")

	// 解析初始化请求
	var request InitRequest
	if data, ok := eventData["data"]; ok {
		dataBytes, _ := json.Marshal(data)
		json.Unmarshal(dataBytes, &request)
	}

	// 验证必要参数
	if request.NodeID == "" {
		return h.errorResponse("节点ID不能为空", nil), nil
	}
	if request.MooxURL == "" {
		return h.errorResponse("Moox URL不能为空", nil), nil
	}

	// 执行初始化
	if err := h.initialize(request.NodeID, request.MooxURL); err != nil {
		return h.errorResponse("初始化失败", err), nil
	}

	return h.successResponse(map[string]interface{}{
		"message": "Initialization completed",
		"node_id": request.NodeID,
	}), nil
}

// handleStatus 获取状态
func (h *Handler) handleStatus(ctx context.Context) (interface{}, error) {
	h.mu.RLock()
	initialized := h.initialized
	nodeID := h.nodeID
	h.mu.RUnlock()

	if !initialized {
		return h.successResponse(map[string]interface{}{
			"node_id":       "uninitialized",
			"initialized":   false,
			"running_tasks": 0,
			"tasks":         []interface{}{},
		}), nil
	}

	tasksStatus := h.taskManager.GetTasksStatus()
	return h.successResponse(map[string]interface{}{
		"node_id":       nodeID,
		"initialized":   true,
		"running_tasks": len(tasksStatus),
		"tasks":         tasksStatus,
	}), nil
}

// successResponse 构建成功响应
func (h *Handler) successResponse(data interface{}) interface{} {
	return map[string]interface{}{
		"success": true,
		"data":    data,
	}
}

// errorResponse 构建错误响应
func (h *Handler) errorResponse(message string, err error) interface{} {
	response := map[string]interface{}{
		"success": false,
		"message": message,
	}

	if err != nil {
		response["error"] = err.Error()
	}

	return response
}

// Stop 停止处理器
func (h *Handler) Stop() {
	log.Printf("[Handler] Stopping handler")

	h.mu.RLock()
	initialized := h.initialized
	h.mu.RUnlock()

	if !initialized {
		return
	}

	// 停止所有任务
	if h.taskManager != nil {
		h.taskManager.StopAll()
	}

	// 停止心跳
	if h.heartbeatService != nil {
		h.heartbeatService.Stop()
	}
}
