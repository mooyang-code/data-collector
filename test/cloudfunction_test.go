package test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/tencentyun/scf-go-lib/functioncontext"

	"github.com/mooyang-code/data-collector/internal/bootstrap"
	"github.com/mooyang-code/data-collector/internal/handler"
	"github.com/mooyang-code/data-collector/pkg/model"
)

func TestCloudFunctionHandler(t *testing.T) {
	// 创建启动器
	cfg := bootstrap.DefaultConfig()
	cfg.Logger.Level = "debug"
	bs := bootstrap.New(cfg)

	// 创建云函数处理器
	h := handler.NewCloudFunctionHandler(bs)

	// 创建带有云函数上下文的context
	funcCtx := &functioncontext.FunctionContext{
		FunctionName:       "test-data-collector",
		FunctionVersion:    "$LATEST",
		Namespace:          "default",
		TencentcloudRegion: "ap-guangzhou",
		RequestID:          "test-request-123",
		MemoryLimitInMb:    128,
		TimeLimitInMs:      3000,
	}
	
	ctx := functioncontext.NewContext(context.Background(), funcCtx)

	// 测试健康检查事件
	healthEvent := model.CloudFunctionEvent{
		Type:      model.EventTypeHealth,
		Action:    "health_check",
		Data:      make(map[string]interface{}),
		RequestID: "test-health-001",
	}

	eventData, _ := json.Marshal(healthEvent)
	response, err := h.HandleRequest(ctx, eventData)
	if err != nil {
		t.Fatalf("HandleRequest failed: %v", err)
	}

	resp, ok := response.(*model.Response)
	if !ok {
		t.Fatalf("Response type assertion failed")
	}

	if !resp.Success {
		t.Fatalf("Health check failed: %s", resp.Message)
	}

	t.Logf("Health check response: %+v", resp)

	// 验证节点信息是否从云函数上下文更新
	nodeInfo := bs.GetNodeInfo()
	if nodeInfo.NodeID != "test-data-collector" {
		t.Errorf("Expected NodeID to be 'test-data-collector', got '%s'", nodeInfo.NodeID)
	}
	if nodeInfo.Region != "ap-guangzhou" {
		t.Errorf("Expected Region to be 'ap-guangzhou', got '%s'", nodeInfo.Region)
	}
	if nodeInfo.Namespace != "default" {
		t.Errorf("Expected Namespace to be 'default', got '%s'", nodeInfo.Namespace)
	}

	t.Logf("Node info: %+v", nodeInfo)
}

func TestAPIGatewayHandler(t *testing.T) {
	// 测试API网关处理器的创建
	cfg := bootstrap.DefaultConfig()
	cfg.Logger.Level = "debug"
	bs := bootstrap.New(cfg)
	
	// 创建云函数处理器
	h := handler.NewCloudFunctionHandler(bs)
	
	if h == nil {
		t.Fatal("Failed to create CloudFunctionHandler")
	}
	
	t.Log("CloudFunctionHandler created successfully")
	t.Log("API Gateway handler would use events.APIGatewayRequest in actual deployment")
}