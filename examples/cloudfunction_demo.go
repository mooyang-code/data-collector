package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/tencentyun/scf-go-lib/functioncontext"

	"github.com/mooyang-code/data-collector/internal/bootstrap"
	"github.com/mooyang-code/data-collector/internal/handler"
	"github.com/mooyang-code/data-collector/pkg/model"
)

// 演示云函数处理器的使用
func main() {
	fmt.Println("========================================")
	fmt.Println("   云函数处理器演示")
	fmt.Println("========================================")

	// 创建启动器
	cfg := bootstrap.DefaultConfig()
	cfg.Logger.Level = "info"
	bs := bootstrap.New(cfg)

	// 创建云函数处理器
	h := handler.NewCloudFunctionHandler(bs)

	// 模拟云函数上下文
	funcCtx := &functioncontext.FunctionContext{
		FunctionName:       "data-collector-demo",
		FunctionVersion:    "$LATEST",
		Namespace:          "production",
		TencentcloudRegion: "ap-shanghai",
		RequestID:          "demo-request-001",
		MemoryLimitInMb:    256,
		TimeLimitInMs:      30000,
		Environment: map[string]string{
			"ENVIRONMENT": "production",
			"LOG_LEVEL":   "info",
		},
	}

	ctx := functioncontext.NewContext(context.Background(), funcCtx)

	// 演示不同类型的云函数事件

	// 1. 健康检查事件
	fmt.Println("\n1. 测试健康检查事件:")
	healthEvent := model.CloudFunctionEvent{
		Type:      model.EventTypeHealth,
		Action:    "health_check",
		Data:      make(map[string]interface{}),
		RequestID: "health-001",
	}
	
	response := handleEvent(h, ctx, healthEvent)
	fmt.Printf("健康检查响应: %s\n", response.Message)

	// 2. 配置同步事件
	fmt.Println("\n2. 测试配置同步事件:")
	configEvent := model.CloudFunctionEvent{
		Type:   model.EventTypeConfig,
		Action: "sync",
		Data: map[string]interface{}{
			"tasks": []map[string]interface{}{
				{
					"id":       "binance-btc-1m",
					"type":     "kline",
					"exchange": "binance",
					"symbol":   "BTCUSDT",
					"interval": "1m",
					"schedule": "*/1 * * * *",
				},
			},
			"server_config": map[string]interface{}{
				"server_url":         "https://api.moox.com",
				"heartbeat_interval": 30,
			},
		},
		RequestID: "config-001",
	}
	
	response = handleEvent(h, ctx, configEvent)
	fmt.Printf("配置同步响应: %s\n", response.Message)

	// 3. 心跳探测事件
	fmt.Println("\n3. 测试心跳探测事件:")
	heartbeatEvent := model.CloudFunctionEvent{
		Type:   model.EventTypeHeartbeat,
		Action: "probe",
		Data: map[string]interface{}{
			"server_url":  "https://api.moox.com",
			"server_port": 443,
			"auth_token":  "test-token-123",
		},
		RequestID: "heartbeat-001",
	}
	
	response = handleEvent(h, ctx, heartbeatEvent)
	fmt.Printf("心跳探测响应: %s\n", response.Message)

	// 4. 任务执行事件
	fmt.Println("\n4. 测试任务执行事件:")
	taskEvent := model.CloudFunctionEvent{
		Type:   model.EventTypeTask,
		Action: "timer_trigger",
		Data: map[string]interface{}{
			"trigger_name": "binance-kline-collector",
			"task_id":      "binance-btc-1m",
		},
		RequestID: "task-001",
	}
	
	response = handleEvent(h, ctx, taskEvent)
	fmt.Printf("任务执行响应: %s\n", response.Message)

	// 显示最终的节点信息
	fmt.Println("\n========================================")
	fmt.Println("最终节点信息:")
	nodeInfo := bs.GetNodeInfo()
	fmt.Printf("节点ID: %s\n", nodeInfo.NodeID)
	fmt.Printf("区域: %s\n", nodeInfo.Region)
	fmt.Printf("命名空间: %s\n", nodeInfo.Namespace)
	fmt.Printf("版本: %s\n", nodeInfo.Version)
	fmt.Printf("支持的采集器: %v\n", nodeInfo.Capabilities)
	fmt.Printf("元数据: %v\n", nodeInfo.Metadata)
	fmt.Println("========================================")

	// 优雅关闭
	if err := bs.Stop(ctx); err != nil {
		log.Printf("关闭系统时出错: %v", err)
	}

	fmt.Println("演示完成！")
}

// handleEvent 处理云函数事件的辅助函数
func handleEvent(h *handler.CloudFunctionHandler, ctx context.Context, event model.CloudFunctionEvent) *model.Response {
	eventData, err := json.Marshal(event)
	if err != nil {
		log.Printf("序列化事件失败: %v", err)
		return &model.Response{
			Success: false,
			Message: fmt.Sprintf("事件序列化失败: %v", err),
		}
	}

	result, err := h.HandleRequest(ctx, eventData)
	if err != nil {
		log.Printf("处理事件失败: %v", err)
		return &model.Response{
			Success: false,
			Message: fmt.Sprintf("事件处理失败: %v", err),
		}
	}

	response, ok := result.(*model.Response)
	if !ok {
		log.Printf("响应类型转换失败")
		return &model.Response{
			Success: false,
			Message: "响应类型错误",
		}
	}

	return response
}