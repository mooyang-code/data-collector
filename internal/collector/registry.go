package collector

import (
	"fmt"
	"sync"

	"github.com/mooyang-code/data-collector/pkg/errors"
)

// registry 采集器注册中心实现
type registry struct {
	factories map[string]Factory
	mu        sync.RWMutex
}

// NewRegistry 创建新的采集器注册中心
func NewRegistry() Registry {
	return &registry{
		factories: make(map[string]Factory),
	}
}

func (r *registry) Register(name string, factory Factory) error {
	if name == "" {
		return errors.NewAppError(errors.ErrCodeInvalidRequest, "collector name cannot be empty", nil)
	}
	
	if factory == nil {
		return errors.NewAppError(errors.ErrCodeInvalidRequest, "collector factory cannot be nil", nil)
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.factories[name]; exists {
		return errors.NewAppError(errors.ErrCodeCollectorError, 
			fmt.Sprintf("collector %s already registered", name), nil)
	}
	
	r.factories[name] = factory
	return nil
}

func (r *registry) Create(name string) (Collector, error) {
	r.mu.RLock()
	factory, exists := r.factories[name]
	r.mu.RUnlock()
	
	if !exists {
		return nil, errors.NewAppError(errors.ErrCodeCollectorError,
			fmt.Sprintf("collector %s not registered", name), errors.ErrCollectorNotFound)
	}
	
	return factory(), nil
}

func (r *registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	
	return names
}

func (r *registry) IsRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	_, exists := r.factories[name]
	return exists
}

// 全局注册中心实例
var globalRegistry = NewRegistry()

// Register 注册采集器到全局注册中心
func Register(name string, factory Factory) error {
	return globalRegistry.Register(name, factory)
}

// Create 从全局注册中心创建采集器
func Create(name string) (Collector, error) {
	return globalRegistry.Create(name)
}

// List 列出全局注册中心中的所有采集器
func List() []string {
	return globalRegistry.List()
}

// IsRegistered 检查采集器是否在全局注册中心中注册
func IsRegistered(name string) bool {
	return globalRegistry.IsRegistered(name)
}

// GetRegistry 获取全局注册中心
func GetRegistry() Registry {
	return globalRegistry
}