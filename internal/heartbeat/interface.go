package heartbeat

import (
	"context"
	"time"

	"github.com/mooyang-code/data-collector/pkg/model"
)

// Manager 心跳管理器接口
type Manager interface {
	// Start 生命周期
	Start(ctx context.Context) error
	Stop(ctx context.Context) error

	// Report 心跳上报
	Report(ctx context.Context) error

	// HandleProbe 处理心跳探测请求
	HandleProbe(ctx context.Context, probeData map[string]interface{}) (*model.Response, error)

	// UpdateServerInfo 更新服务端信息
	UpdateServerInfo(serverIP string, serverPort int, authToken string) error
}

// Client 心跳客户端接口
type Client interface {
	Report(ctx context.Context, payload *model.HeartbeatPayload) error
	RegisterNode(ctx context.Context, node *model.NodeInfo) error
}

// Config 心跳管理器配置
type Config struct {
	Interval      time.Duration `json:"interval" yaml:"interval"`
	Timeout       time.Duration `json:"timeout" yaml:"timeout"`
	RetryCount    int           `json:"retry_count" yaml:"retry_count"`
	RetryInterval time.Duration `json:"retry_interval" yaml:"retry_interval"`
	ServerIP      string        `json:"server_ip" yaml:"server_ip"`
	ServerPort    int           `json:"server_port" yaml:"server_port"`
}

// HeartbeatDefaultConfig 心跳默认配置
var HeartbeatDefaultConfig = Config{
	Interval:      10 * time.Second,
	Timeout:       5 * time.Second,
	RetryCount:    3,
	RetryInterval: 5 * time.Second,
	ServerIP:      "43.132.204.177",
	ServerPort:    20103,
}
