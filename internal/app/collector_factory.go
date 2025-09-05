// Package app 采集器工厂实现
package app

import (
	"fmt"

	"github.com/mooyang-code/data-collector/configs"
)

// CollectorFactoryImpl 采集器工厂实现
type CollectorFactoryImpl struct {
	registry *CollectorRegistry
}

// NewCollectorFactory 创建采集器工厂
func NewCollectorFactory() CollectorFactory {
	return &CollectorFactoryImpl{
		registry: GetGlobalRegistry(),
	}
}

// CreateCollector 创建采集器
func (f *CollectorFactoryImpl) CreateCollector(appName, collectorName string, config interface{}) (Collector, error) {
	// 从配置中获取数据类型和市场类型
	collectorConfig, ok := config.(*configs.Collector)
	if !ok {
		return nil, fmt.Errorf("配置类型不正确")
	}

	// 使用注册中心创建采集器
	return f.registry.CreateCollector(appName, collectorName, collectorConfig)
}

// GetSupportedTypes 获取支持的采集器类型
func (f *CollectorFactoryImpl) GetSupportedTypes() []string {
	return f.registry.GetSupportedTypes()
}
