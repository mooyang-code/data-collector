package config

import (
	"context"

	"github.com/mooyang-code/data-collector/pkg/model"
)

// Manager 配置管理器接口
type Manager interface {
	// 生命周期
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	
	// 配置同步
	SyncConfig(ctx context.Context, config *ServerConfig) error
	GetServerConfig(ctx context.Context) (*ServerConfig, error)
	
	// 节点配置
	GetNodeConfig(ctx context.Context) (*NodeConfig, error)
	UpdateNodeConfig(ctx context.Context, config *NodeConfig) error
	
	// 任务配置
	GetTaskConfigs(ctx context.Context) (map[string]*model.Task, error)
	UpdateTaskConfigs(ctx context.Context, tasks map[string]*model.Task) error
}

// Store 配置存储接口
type Store interface {
	Get(key string) (interface{}, error)
	Set(key string, value interface{}) error
	Subscribe(key string, callback func(interface{})) error
}

// ServerConfig 服务端配置
type ServerConfig struct {
	ServerURL         string `json:"server_url" yaml:"server_url"`
	ServerPort        int    `json:"server_port" yaml:"server_port"`
	AuthToken         string `json:"auth_token" yaml:"auth_token"`
	HeartbeatInterval int    `json:"heartbeat_interval" yaml:"heartbeat_interval"`
}

// NodeConfig 节点配置
type NodeConfig struct {
	NodeID       string                `json:"node_id" yaml:"node_id"`
	NodeType     string                `json:"node_type" yaml:"node_type"`
	Region       string                `json:"region" yaml:"region"`
	Namespace    string                `json:"namespace" yaml:"namespace"`
	RunningTasks []string              `json:"running_tasks" yaml:"running_tasks"`
	Capabilities []model.CollectorType `json:"capabilities" yaml:"capabilities"`
}

// Config 配置管理器配置
type Config struct {
	StorePath string `json:"store_path" yaml:"store_path"`
}

// DefaultConfig 默认配置
var DefaultConfig = Config{
	StorePath: "/tmp/data-collector-config.json",
}