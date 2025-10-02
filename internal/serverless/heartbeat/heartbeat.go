package heartbeat

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/mooyang-code/data-collector/internal/serverless/manager"
	pb "github.com/mooyang-code/data-collector/proto/gen"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
)

// HeartbeatService 心跳服务
type HeartbeatService struct {
	nodeID      string
	mooxURL     string
	taskManager *manager.TaskManager
	interval    time.Duration
	rpcClient   pb.CloudNodeAPIClientProxy
	ticker      *time.Ticker
	mu          sync.RWMutex
	appInfo     *pb.AppInfo
}

// NewHeartbeatService 创建心跳服务
func NewHeartbeatService(nodeID, mooxURL string, taskManager *manager.TaskManager) *HeartbeatService {
	return &HeartbeatService{
		nodeID:      nodeID,
		mooxURL:     mooxURL,
		taskManager: taskManager,
		interval:    5 * time.Second,
		appInfo: &pb.AppInfo{
			AppId:  "data-collector",
			AppKey: "collector-key", // TODO: 从配置读取
		},
	}
}

// Start 启动心跳服务
func (h *HeartbeatService) Start(ctx context.Context) {
	// 初始化随机种子
	rand.Seed(time.Now().UnixNano())

	// 初始化RPC客户端
	if err := h.initRPCClient(); err != nil {
		log.Printf("[Heartbeat] Failed to init RPC client: %v", err)
		return
	}

	log.Printf("[Heartbeat] Starting RPC heartbeat service for node %s", h.nodeID)

	// 立即发送第一次心跳
	go h.sendHeartbeat(ctx)

	// 启动定时心跳
	h.ticker = time.NewTicker(h.interval)
	go h.heartbeatLoop(ctx)
}

// Stop 停止心跳服务
func (h *HeartbeatService) Stop() {
	if h.ticker != nil {
		h.ticker.Stop()
	}
}

// UpdateMooxURL 更新Moox服务地址
func (h *HeartbeatService) UpdateMooxURL(url string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.mooxURL = url
	// 重新初始化RPC客户端
	if err := h.initRPCClient(); err != nil {
		log.Printf("[Heartbeat] Failed to reinit RPC client: %v", err)
	} else {
		log.Printf("[Heartbeat] Updated Moox URL to: %s", url)
	}
}

// initRPCClient 初始化RPC客户端
func (h *HeartbeatService) initRPCClient() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.mooxURL == "" {
		return fmt.Errorf("moox URL not configured")
	}

	// 解析Moox URL提取主机和端口
	// TODO: 从配置或服务发现获取实际的RPC端口
	target := fmt.Sprintf("trpc.moox.server.CloudNodeAPI")

	// 创建客户端配置
	opts := []client.Option{
		client.WithTarget(target),
		client.WithNamespace("Production"),
		client.WithTimeout(10 * time.Second),
	}

	// 创建RPC客户端
	h.rpcClient = pb.NewCloudNodeAPIClientProxy(opts...)
	return nil
}

// heartbeatLoop 心跳循环
func (h *HeartbeatService) heartbeatLoop(ctx context.Context) {
	for {
		select {
		case <-h.ticker.C:
			// 添加0-2秒的随机延迟
			randomDelay := time.Duration(rand.Intn(2000)) * time.Millisecond
			time.Sleep(randomDelay)

			if err := h.sendHeartbeat(ctx); err != nil {
				log.Printf("[Heartbeat] Failed to send heartbeat: %v", err)
			}
		case <-ctx.Done():
			log.Printf("[Heartbeat] Stopping heartbeat loop")
			h.ticker.Stop()
			return
		}
	}
}

// sendHeartbeat 发送心跳
func (h *HeartbeatService) sendHeartbeat(ctx context.Context) error {
	h.mu.RLock()
	rpcClient := h.rpcClient
	h.mu.RUnlock()

	if rpcClient == nil {
		return fmt.Errorf("rpc client not initialized")
	}

	// 获取任务状态
	tasksStatus := h.taskManager.GetTasksStatus()

	// 判断节点状态
	status := "idle"
	if len(tasksStatus) > 0 {
		status = "running"
	}

	// 转换任务信息到proto格式
	runningTasks := make([]*pb.RunningTaskInfo, len(tasksStatus))
	for i, task := range tasksStatus {
		runningTasks[i] = &pb.RunningTaskInfo{
			TaskId:        task.TaskID,
			CollectorType: task.CollectorType,
			Source:        task.Source,
			StartTime:     task.StartTime.Unix(),
			LastExecTime:  task.LastExecTime.Unix(),
			ExecCount:     task.ExecCount,
			ErrorCount:    task.ErrorCount,
		}
	}

	// 构建心跳请求
	req := &pb.HeartbeatReq{
		AppInfo:      h.appInfo,
		NodeId:       h.nodeID,
		Timestamp:    time.Now().Unix(),
		Status:       status,
		RunningTasks: runningTasks,
	}

	// 发送RPC请求
	ctx = trpc.BackgroundContext()
	rsp, err := rpcClient.Heartbeat(ctx, req)
	if err != nil {
		return fmt.Errorf("rpc heartbeat failed: %w", err)
	}

	// 检查响应
	if rsp.RetInfo.Code != pb.EnumMooxErrorCode_SUCCESS {
		return fmt.Errorf("heartbeat failed: %s", rsp.RetInfo.Msg)
	}

	if !rsp.Success {
		return fmt.Errorf("heartbeat not successful")
	}
	return nil
}
