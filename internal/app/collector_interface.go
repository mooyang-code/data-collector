// Package app 采集器接口定义
package app

import (
	"context"
)

// Collector 通用采集器接口
type Collector interface {
	// Initialize 初始化采集器
	Initialize(ctx context.Context) error

	// StartCollection 启动采集
	StartCollection(ctx context.Context) error

	// StopCollection 停止采集
	StopCollection(ctx context.Context) error

	// IsRunning 检查是否运行中
	IsRunning() bool

	// GetID 获取采集器ID
	GetID() string

	// GetType 获取采集器类型
	GetType() string

	// GetDataType 获取数据类型
	GetDataType() string
}

// CollectorFactory 采集器工厂接口
type CollectorFactory interface {
	// CreateCollector 创建采集器
	CreateCollector(appName, collectorName string, config interface{}) (Collector, error)

	// GetSupportedTypes 获取支持的采集器类型
	GetSupportedTypes() []string
}
