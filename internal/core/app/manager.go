package app

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/mooyang-code/data-collector/internal/core/event"
)

// ManagerConfig 管理器配置
type ManagerConfig struct {
	MaxConcurrent int
	EventBus      event.EventBus
}

// AppConfig 应用配置
type AppConfig struct {
	ID       string
	Name     string
	Type     string
	Enabled  bool
	Settings map[string]interface{}
	EventBus event.EventBus
}

// Manager App管理器
type Manager struct {
	apps          map[string]App
	mu            sync.RWMutex
	eventBus      event.EventBus
	maxConcurrent int
}

// NewManager 创建新的管理器
func NewManager(config *ManagerConfig) *Manager {
	return &Manager{
		apps:          make(map[string]App),
		eventBus:      config.EventBus,
		maxConcurrent: config.MaxConcurrent,
	}
}

// CreateApp 创建并注册App
func (m *Manager) CreateApp(config *AppConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 从注册中心创建App
	registry := GetRegistry()

	// 配置已经是正确的格式

	app, err := registry.CreateApp(config.ID, config)
	if err != nil {
		return fmt.Errorf("创建App失败: %w", err)
	}

	// 设置事件总线
	if setter, ok := app.(interface{ SetEventBus(EventBus) }); ok {
		// 使用适配器
		appEventBus := NewEventBusAdapter(m.eventBus)
		setter.SetEventBus(appEventBus)
	}

	m.apps[config.ID] = app
	log.Printf("App %s 创建成功", config.ID)
	return nil
}

// StartApp 启动指定的App
func (m *Manager) StartApp(ctx context.Context, id string) error {
	m.mu.RLock()
	app, exists := m.apps[id]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("app %s 不存在", id)
	}

	// 初始化
	if err := app.Initialize(ctx); err != nil {
		return fmt.Errorf("初始化App %s 失败: %w", id, err)
	}

	// 启动
	if err := app.Start(ctx); err != nil {
		return fmt.Errorf("启动App %s 失败: %w", id, err)
	}
	return nil
}

// StopApp 停止指定的App
func (m *Manager) StopApp(ctx context.Context, id string) error {
	m.mu.RLock()
	app, exists := m.apps[id]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("app %s 不存在", id)
	}
	return app.Stop(ctx)
}

// GetApp 获取App
func (m *Manager) GetApp(id string) (App, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	app, exists := m.apps[id]
	if !exists {
		return nil, fmt.Errorf("app %s 不存在", id)
	}
	return app, nil
}

// ListApps 列出所有App
func (m *Manager) ListApps() []AppInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	infos := make([]AppInfo, 0, len(m.apps))
	for _, app := range m.apps {
		info := AppInfo{
			ID:     app.ID(),
			Name:   app.Name(),
			Type:   string(app.Type()),
			Status: AppStatusRunning, // TODO: 从App获取实际状态
		}
		infos = append(infos, info)
	}
	return infos
}

// AppInfo App信息
type AppInfo struct {
	ID     string
	Name   string
	Type   string
	Status AppStatus
}

// Shutdown 关闭管理器
func (m *Manager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 停止所有App
	for id, app := range m.apps {
		if err := app.Stop(ctx); err != nil {
			log.Printf("停止App %s 失败: %v", id, err)
		}
	}
	return nil
}

// Initialize 初始化管理器
func (m *Manager) Initialize(ctx context.Context) error {
	return nil
}
