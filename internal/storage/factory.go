// Package storage 存储工厂
package storage

import (
	"fmt"
	"strings"
)

// Factory 存储工厂
type Factory struct {
	backends map[string]func() StorageBackend
}

// NewFactory 创建存储工厂
func NewFactory() *Factory {
	factory := &Factory{
		backends: make(map[string]func() StorageBackend),
	}
	
	// 注册默认后端
	factory.RegisterBackend("memory", func() StorageBackend {
		return NewMemoryBackend()
	})
	
	return factory
}

// RegisterBackend 注册存储后端
func (f *Factory) RegisterBackend(name string, createFunc func() StorageBackend) {
	f.backends[strings.ToLower(name)] = createFunc
}

// CreateBackend 创建存储后端
func (f *Factory) CreateBackend(config *Config) (StorageBackend, error) {
	if config == nil {
		return nil, fmt.Errorf("存储配置不能为空")
	}
	
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("存储配置验证失败: %w", err)
	}
	
	if !config.Enabled {
		return nil, fmt.Errorf("存储未启用")
	}
	
	backendName := strings.ToLower(config.Backend)
	createFunc, exists := f.backends[backendName]
	if !exists {
		return nil, fmt.Errorf("不支持的存储后端: %s", config.Backend)
	}
	
	backend := createFunc()
	if err := backend.Initialize(config); err != nil {
		return nil, fmt.Errorf("初始化存储后端失败: %w", err)
	}
	
	return backend, nil
}

// GetSupportedBackends 获取支持的后端类型
func (f *Factory) GetSupportedBackends() []string {
	backends := make([]string, 0, len(f.backends))
	for name := range f.backends {
		backends = append(backends, name)
	}
	return backends
}

// ValidateConfig 验证配置
func (f *Factory) ValidateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("存储配置不能为空")
	}
	
	if !config.Enabled {
		return nil // 未启用时不需要验证
	}
	
	backendName := strings.ToLower(config.Backend)
	if _, exists := f.backends[backendName]; !exists {
		return fmt.Errorf("不支持的存储后端: %s", config.Backend)
	}
	
	return config.Validate()
}

// DefaultFactory 默认存储工厂实例
var DefaultFactory = NewFactory()
