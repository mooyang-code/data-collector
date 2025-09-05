// Package app 应用工厂模块
package app

import (
	"fmt"

	"trpc.group/trpc-go/trpc-go/log"
)

// AppFactory 应用工厂接口
type AppFactory interface {
	CreateApp(config *AppConfig) (App, error)
	GetSupportedTypes() []string
}

// AppFactoryImpl 应用工厂实现
type AppFactoryImpl struct {
	creators map[string]AppCreator
}

// AppCreator 应用创建器函数类型
type AppCreator func(config *AppConfig) (App, error)

// NewAppFactory 创建新的应用工厂
func NewAppFactory() AppFactory {
	factory := &AppFactoryImpl{
		creators: make(map[string]AppCreator),
	}

	// 注册默认的应用创建器
	factory.registerDefaultCreators()
	return factory
}

// registerDefaultCreators 注册默认的应用创建器
func (f *AppFactoryImpl) registerDefaultCreators() {
	// 使用注册表中的应用
	registry := GetAppRegistry()
	apps := registry.GetAllApps()

	// 为每个注册的应用创建工厂入口
	for name := range apps {
		appName := name // 避免闭包问题
		f.creators[appName] = func(config *AppConfig) (App, error) {
			return registry.CreateApp(appName, config)
		}
	}

	// 保留默认的数据采集器应用类型
	f.creators["data-collector"] = func(config *AppConfig) (App, error) {
		// 如果配置中有 appName，使用注册表创建
		if appNameInterface, ok := config.Config["appName"]; ok {
			if appName, ok := appNameInterface.(string); ok {
				if HasApp(appName) {
					return registry.CreateApp(appName, config)
				}
			}
		}
		// 否则使用默认实现
		return NewDataCollectorApp(config)
	}

	// 注册默认应用
	f.creators["default"] = func(config *AppConfig) (App, error) {
		return NewDefaultApp(config)
	}

	if len(apps) > 0 {
		log.Infof("从注册表加载了 %d 个应用类型", len(apps))
	}
	log.Info("默认应用创建器注册完成")
}

// CreateApp 创建应用
func (f *AppFactoryImpl) CreateApp(config *AppConfig) (App, error) {
	if config == nil {
		return nil, fmt.Errorf("应用配置不能为空")
	}

	appType := config.Type
	if appType == "" {
		appType = "default"
	}

	// 优先从注册表查找
	registry := GetAppRegistry()
	if descriptor, exists := registry.GetApp(appType); exists {
		app, err := descriptor.Creator(config)
		if err != nil {
			return nil, fmt.Errorf("创建应用失败: %w", err)
		}
		log.Infof("创建应用成功: %s (类型: %s，来自注册表)", config.ID, appType)
		return app, nil
	}

	// 从工厂创建器查找
	creator, exists := f.creators[appType]
	if !exists {
		return nil, fmt.Errorf("不支持的应用类型: %s", appType)
	}

	app, err := creator(config)
	if err != nil {
		return nil, fmt.Errorf("创建应用失败: %w", err)
	}

	log.Infof("创建应用成功: %s (类型: %s)", config.ID, appType)
	return app, nil
}

// GetSupportedTypes 获取支持的应用类型
func (f *AppFactoryImpl) GetSupportedTypes() []string {
	types := make([]string, 0, len(f.creators))
	for appType := range f.creators {
		types = append(types, appType)
	}
	return types
}

// RegisterCreator 注册应用创建器
func (f *AppFactoryImpl) RegisterCreator(appType string, creator AppCreator) {
	f.creators[appType] = creator
	log.Infof("注册应用创建器: %s", appType)
}
