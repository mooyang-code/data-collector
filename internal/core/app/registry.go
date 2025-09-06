package app

import (
	"fmt"
	"sync"
)

type AppCreatorFunc func(config *AppConfig) (App, error)

type AppDescriptor struct {
	Name        string         // 唯一标识，如 "binance"
	Type        SourceType     // 数据源类型
	DisplayName string         // 显示名称，如 "币安"
	Description string         // 描述
	Creator     AppCreatorFunc // 创建函数
}

// AppConfig 现在定义在 manager.go 中

type AppRegistry struct {
	apps map[string]*AppDescriptor
	mu   sync.RWMutex
}

var globalRegistry = &AppRegistry{
	apps: make(map[string]*AppDescriptor),
}

// Register 注册App
func Register(name string, descriptor *AppDescriptor) error {
	return globalRegistry.Register(name, descriptor)
}

// RegisterCreator 注册App创建函数（简化版）
func RegisterCreator(name, displayName, description string, sourceType SourceType, creator AppCreatorFunc) error {
	return globalRegistry.Register(name, &AppDescriptor{
		Name:        name,
		Type:        sourceType,
		DisplayName: displayName,
		Description: description,
		Creator:     creator,
	})
}

// GetRegistry 获取全局注册表
func GetRegistry() *AppRegistry {
	return globalRegistry
}

func (r *AppRegistry) Register(name string, descriptor *AppDescriptor) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.apps[name]; exists {
		return fmt.Errorf("App %s 已经注册", name)
	}
	
	descriptor.Name = name
	r.apps[name] = descriptor
	return nil
}

func (r *AppRegistry) Get(name string) (*AppDescriptor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	descriptor, exists := r.apps[name]
	if !exists {
		return nil, fmt.Errorf("App %s 未注册", name)
	}
	
	return descriptor, nil
}

func (r *AppRegistry) List() []*AppDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	descriptors := make([]*AppDescriptor, 0, len(r.apps))
	for _, desc := range r.apps {
		descriptors = append(descriptors, desc)
	}
	
	return descriptors
}

func (r *AppRegistry) CreateApp(name string, config *AppConfig) (App, error) {
	descriptor, err := r.Get(name)
	if err != nil {
		return nil, err
	}
	
	if config == nil {
		config = &AppConfig{
			ID:       name,
			Name:     descriptor.DisplayName,
			Type:     string(descriptor.Type),
			Settings: make(map[string]interface{}),
		}
	}
	
	return descriptor.Creator(config)
}