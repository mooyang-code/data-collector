// Package app 应用注册表
package app

import (
	"fmt"
	"sync"

	"trpc.group/trpc-go/trpc-go/log"
)

// AppCreatorFunc 应用创建函数类型
type AppCreatorFunc func(config *AppConfig) (App, error)

// AppDescriptor 应用描述信息
type AppDescriptor struct {
	Name        string         // 应用名称（如 binance、okx）
	DisplayName string         // 显示名称（如 币安、欧易）
	Description string         // 应用描述
	Creator     AppCreatorFunc // 应用创建函数
}

// appRegistry 应用注册表实例
var appRegistry = &AppRegistry{
	apps: make(map[string]*AppDescriptor),
}

// AppRegistry 应用注册表
type AppRegistry struct {
	apps  map[string]*AppDescriptor
	mutex sync.RWMutex
}

// GetAppRegistry 获取全局应用注册表
func GetAppRegistry() *AppRegistry {
	return appRegistry
}

// RegisterApp 注册应用
func (r *AppRegistry) RegisterApp(descriptor *AppDescriptor) error {
	if descriptor == nil {
		return fmt.Errorf("应用描述不能为空")
	}

	if descriptor.Name == "" {
		return fmt.Errorf("应用名称不能为空")
	}

	if descriptor.Creator == nil {
		return fmt.Errorf("应用创建函数不能为空")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.apps[descriptor.Name]; exists {
		return fmt.Errorf("应用已注册: %s", descriptor.Name)
	}

	r.apps[descriptor.Name] = descriptor
	log.Infof("注册应用成功: %s (%s)", descriptor.Name, descriptor.DisplayName)
	return nil
}

// GetApp 获取应用描述
func (r *AppRegistry) GetApp(name string) (*AppDescriptor, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	app, exists := r.apps[name]
	return app, exists
}

// GetAllApps 获取所有注册的应用
func (r *AppRegistry) GetAllApps() map[string]*AppDescriptor {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// 创建副本返回
	apps := make(map[string]*AppDescriptor)
	for k, v := range r.apps {
		apps[k] = v
	}
	return apps
}

// CreateApp 创建应用实例
func (r *AppRegistry) CreateApp(name string, config *AppConfig) (App, error) {
	descriptor, exists := r.GetApp(name)
	if !exists {
		return nil, fmt.Errorf("未找到应用: %s", name)
	}

	return descriptor.Creator(config)
}

// RegisterCreator 全局注册应用（便捷方法）
func RegisterCreator(name, displayName, description string, creator AppCreatorFunc) error {
	return appRegistry.RegisterApp(&AppDescriptor{
		Name:        name,
		DisplayName: displayName,
		Description: description,
		Creator:     creator,
	})
}

// HasApp 检查应用是否已注册
func HasApp(name string) bool {
	_, exists := appRegistry.GetApp(name)
	return exists
}

// HasApp 检查应用是否已注册（方法形式）
func (r *AppRegistry) HasApp(name string) bool {
	_, exists := r.GetApp(name)
	return exists
}

// ListRegisteredApps 列出所有已注册的应用
func ListRegisteredApps() []string {
	apps := appRegistry.GetAllApps()
	names := make([]string, 0, len(apps))
	for name := range apps {
		names = append(names, name)
	}
	return names
}