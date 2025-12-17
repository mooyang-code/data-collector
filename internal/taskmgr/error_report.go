package taskmgr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mooyang-code/data-collector/pkg/config"
	"trpc.group/trpc-go/trpc-go/log"
)

// TaskErrorPayload 任务错误上报载荷
type TaskErrorPayload struct {
	TaskID    string    `json:"task_id"`
	NodeID    string    `json:"node_id"`
	ErrorType string    `json:"error_type"`
	ErrorMsg  string    `json:"error_msg"`
	Timestamp time.Time `json:"timestamp"`
}

// ReportError 上报任务错误到服务端
// 异步执行，不阻塞主流程
func ReportError(ctx context.Context, taskID, errorType string, err error) {
	go func() {
		if reportErr := doReportError(ctx, taskID, errorType, err); reportErr != nil {
			log.ErrorContextf(ctx, "上报任务错误失败 [%s]: %v", taskID, reportErr)
		}
	}()
}

// doReportError 执行错误上报
func doReportError(ctx context.Context, taskID, errorType string, err error) error {
	serverIP, serverPort := config.GetServerInfo()
	if serverIP == "" || serverPort <= 0 {
		return fmt.Errorf("服务端地址未配置")
	}

	nodeID, _ := config.GetNodeInfo()

	payload := &TaskErrorPayload{
		TaskID:    taskID,
		NodeID:    nodeID,
		ErrorType: errorType,
		ErrorMsg:  err.Error(),
		Timestamp: time.Now(),
	}

	data, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return fmt.Errorf("序列化错误载荷失败: %w", marshalErr)
	}

	url := fmt.Sprintf("http://%s:%d/gateway/collectmgr/ReportTaskError", serverIP, serverPort)

	client := &http.Client{Timeout: 5 * time.Second}
	req, reqErr := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if reqErr != nil {
		return fmt.Errorf("创建请求失败: %w", reqErr)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, respErr := client.Do(req)
	if respErr != nil {
		return fmt.Errorf("发送请求失败: %w", respErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务端返回错误状态码: %d", resp.StatusCode)
	}

	log.InfoContextf(ctx, "任务错误已上报 [%s, %s]", taskID, errorType)
	return nil
}
