// Package app 应用工厂模块（最好是中文注释！）
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
	// 注册数据采集器应用
	f.creators["data-collector"] = func(config *AppConfig) (App, error) {
		return NewDataCollectorApp(config)
	}

	// 注册默认应用
	f.creators["default"] = func(config *AppConfig) (App, error) {
		return NewDefaultApp(config)
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
