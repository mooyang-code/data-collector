// Package app 应用程序管理模块（最好是中文注释！）
package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mooyang-code/data-collector/configs"
	"trpc.group/trpc-go/trpc-go/log"
)

// AppManager 应用管理器接口
type AppManager interface {
	LoadConfig(configPath string) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	AddApp(appConfig *AppConfig) error
	RemoveApp(appID string) error
	GetApp(appID string) (App, bool)
	GetAllApps() []App
	GetStats() AppManagerStats
	GetConfig() *configs.Config
	GetAppConfig(appName string) (*configs.AppConfig, error)
	GetEnabledApps() map[string]*configs.AppConfig
}

// AppManagerImpl 应用管理器实现
type AppManagerImpl struct {
	factory AppFactory
	apps    map[string]App
	mutex   sync.RWMutex
	config  *configs.Config
}

// NewAppManager 创建新的应用管理器
func NewAppManager(factory AppFactory) AppManager {
	// 初始化采集器注册
	InitCollectors()

	return &AppManagerImpl{
		factory: factory,
		apps:    make(map[string]App),
	}
}

// LoadConfig 加载配置
func (m *AppManagerImpl) LoadConfig(configPath string) error {
	log.Infof("加载配置文件: %s", configPath)

	// 使用 configs.Load 函数加载配置文件
	config, err := configs.Load(configPath)
	if err != nil {
		log.Errorf("加载配置文件失败: %v", err)
		return fmt.Errorf("加载配置文件失败: %w", err)
	}

	m.config = config
	log.Info("配置加载完成")

	// 根据配置初始化应用
	if err := m.initializeAppsFromConfig(); err != nil {
		log.Errorf("初始化应用失败: %v", err)
		return fmt.Errorf("初始化应用失败: %w", err)
	}
	return nil
}

// Start 启动应用管理器
func (m *AppManagerImpl) Start(ctx context.Context) error {
	log.Info("启动应用管理器...")

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 启动所有应用
	for id, app := range m.apps {
		if err := app.Start(ctx); err != nil {
			log.Errorf("启动应用失败: %s, error: %v", id, err)
			return fmt.Errorf("启动应用 %s 失败: %w", id, err)
		}
		log.Infof("应用启动成功: %s", id)
	}

	log.Info("应用管理器启动完成")
	return nil
}

// Stop 停止应用管理器
func (m *AppManagerImpl) Stop(ctx context.Context) error {
	log.Info("停止应用管理器...")

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 停止所有应用
	for id, app := range m.apps {
		if err := app.Stop(ctx); err != nil {
			log.Errorf("停止应用失败: %s, error: %v", id, err)
		} else {
			log.Infof("应用停止成功: %s", id)
		}
	}

	log.Info("应用管理器停止完成")
	return nil
}

// AddApp 添加应用
func (m *AppManagerImpl) AddApp(appConfig *AppConfig) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	app, err := m.factory.CreateApp(appConfig)
	if err != nil {
		return fmt.Errorf("创建应用失败: %w", err)
	}

	m.apps[appConfig.ID] = app
	log.Infof("添加应用成功: %s", appConfig.ID)
	return nil
}

// RemoveApp 移除应用
func (m *AppManagerImpl) RemoveApp(appID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if app, exists := m.apps[appID]; exists {
		// 停止应用
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := app.Stop(ctx); err != nil {
			log.Errorf("停止应用失败: %s, error: %v", appID, err)
		}

		delete(m.apps, appID)
		log.Infof("移除应用成功: %s", appID)
		return nil
	}
	return fmt.Errorf("应用不存在: %s", appID)
}

// GetApp 获取应用
func (m *AppManagerImpl) GetApp(appID string) (App, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	app, exists := m.apps[appID]
	return app, exists
}

// GetAllApps 获取所有应用
func (m *AppManagerImpl) GetAllApps() []App {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	apps := make([]App, 0, len(m.apps))
	for _, app := range m.apps {
		apps = append(apps, app)
	}

	return apps
}

// GetStats 获取统计信息
func (m *AppManagerImpl) GetStats() AppManagerStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stats := AppManagerStats{
		TotalApps: len(m.apps),
	}

	for _, app := range m.apps {
		if app.IsRunning() {
			stats.RunningApps++
		}

		status := app.GetStatus()
		if status.LastError != "" {
			stats.ErrorApps++
		}
	}
	return stats
}

// GetConfig 获取配置
func (m *AppManagerImpl) GetConfig() *configs.Config {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.config
}

// GetAppConfig 获取指定应用配置
func (m *AppManagerImpl) GetAppConfig(appName string) (*configs.AppConfig, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.config == nil {
		return nil, fmt.Errorf("配置未加载")
	}
	return m.config.GetAppConfig(appName)
}

// GetEnabledApps 获取所有启用的应用配置
func (m *AppManagerImpl) GetEnabledApps() map[string]*configs.AppConfig {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.config == nil {
		return make(map[string]*configs.AppConfig)
	}
	return m.config.GetEnabledApps()
}

// initializeAppsFromConfig 根据配置初始化应用
func (m *AppManagerImpl) initializeAppsFromConfig() error {
	if m.config == nil {
		return fmt.Errorf("配置未加载")
	}

	// 获取所有启用的应用配置
	enabledApps := m.config.GetEnabledApps()

	for appName, appConfig := range enabledApps {
		log.Infof("初始化应用: %s", appName)

		// 转换配置格式
		legacyConfig := &AppConfig{
			ID:          appName,
			Type:        "data-collector", // 默认类型
			Name:        appConfig.Name,
			Description: fmt.Sprintf("%s 数据采集应用", appName),
			Enabled:     appConfig.Enabled,
			Config:      make(map[string]interface{}),
		}

		// 将 configs.AppConfig 转换为 map[string]interface{}
		legacyConfig.Config["baseConfig"] = appConfig.BaseConfig
		legacyConfig.Config["collectors"] = appConfig.Collectors
		legacyConfig.Config["appName"] = appName

		// 创建并添加应用
		if err := m.AddApp(legacyConfig); err != nil {
			log.Errorf("添加应用失败: %s, error: %v", appName, err)
			return fmt.Errorf("添加应用 %s 失败: %w", appName, err)
		}

		log.Infof("应用初始化完成: %s", appName)
	}

	log.Infof("所有应用初始化完成，共 %d 个应用", len(enabledApps))
	return nil
}
