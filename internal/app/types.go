// Package app 应用类型定义（最好是中文注释！）
package app

import (
	"context"
	"time"
)

// App 应用接口
type App interface {
	GetID() string
	GetDataset() Dataset
	GetCollectorManager() CollectorManager
	GetStorage() Storage
	Initialize(config *AppConfig) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	IsRunning() bool
	GetStatus() AppStatus
	GetMetrics() AppMetrics
}

// Dataset 数据集接口
type Dataset interface {
	GetName() string
	GetDescription() string
}

// CollectorManager 采集器管理器接口
type CollectorManager interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	IsRunning() bool
}

// Storage 存储接口
type Storage interface {
	Save(data interface{}) error
	Load(key string) (interface{}, error)
}

// AppConfig 应用配置
type AppConfig struct {
	ID          string                 `yaml:"id"`
	Type        string                 `yaml:"type"`
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Enabled     bool                   `yaml:"enabled"`
	Config      map[string]interface{} `yaml:"config"`
}

// Config 全局配置
type Config struct {
	ConfigPath string
	Apps       []*AppConfig `yaml:"apps"`
}

// AppStatus 应用状态
type AppStatus struct {
	State     string    `json:"state"`
	StartTime time.Time `json:"start_time"`
	LastError string    `json:"last_error,omitempty"`
}

// AppMetrics 应用指标
type AppMetrics struct {
	Uptime        time.Duration `json:"uptime"`
	RequestCount  int64         `json:"request_count"`
	ErrorCount    int64         `json:"error_count"`
	LastRequestAt time.Time     `json:"last_request_at"`
}

// AppManagerStats 应用管理器统计信息
type AppManagerStats struct {
	TotalApps   int           `json:"total_apps"`
	RunningApps int           `json:"running_apps"`
	ErrorApps   int           `json:"error_apps"`
	TotalUptime time.Duration `json:"total_uptime"`
}
