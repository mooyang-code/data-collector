package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/mooyang-code/data-collector/internal/event"
	"github.com/mooyang-code/data-collector/pkg/logger"
	"github.com/mooyang-code/data-collector/pkg/model"
)

// manager 配置管理器实现
type manager struct {
	config   Config
	logger   logger.Logger
	eventBus event.Notifier

	// 配置存储
	serverConfig *ServerConfig
	nodeConfig   *NodeConfig
	taskConfigs  map[string]*model.Task

	// 状态管理
	mu      sync.RWMutex
	started bool
}

// NewManager 创建新的配置管理器
func NewManager(cfg Config, log logger.Logger, bus event.Notifier) Manager {
	return &manager{
		config:      cfg,
		logger:      log.With("component", "config_manager"),
		eventBus:    bus,
		taskConfigs: make(map[string]*model.Task),
	}
}

func (m *manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return fmt.Errorf("config manager already started")
	}

	// 确保存储目录存在
	if err := m.ensureStorePath(); err != nil {
		return fmt.Errorf("failed to ensure store path: %w", err)
	}

	// 加载本地配置
	if err := m.loadConfig(); err != nil {
		m.logger.Warn("failed to load local config, using defaults", "error", err)
		m.initDefaultConfig()
	}

	m.started = true
	m.logger.Info("config manager started")
	return nil
}

func (m *manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return fmt.Errorf("config manager not started")
	}

	// 保存配置
	if err := m.saveConfig(); err != nil {
		m.logger.Error("failed to save config on stop", "error", err)
	}

	m.started = false
	m.logger.Info("config manager stopped")
	return nil
}

func (m *manager) SyncConfig(ctx context.Context, config *ServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if config == nil {
		return fmt.Errorf("server config cannot be nil")
	}

	m.logger.Info("syncing server config",
		"server_ip", config.ServerIP,
		"server_port", config.ServerPort,
		"heartbeat_interval", config.HeartbeatInterval)

	// 更新配置
	m.serverConfig = config

	// 保存到本地
	if err := m.saveConfig(); err != nil {
		m.logger.Error("failed to save config after sync", "error", err)
	}

	return nil
}

func (m *manager) GetServerConfig(ctx context.Context) (*ServerConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.serverConfig == nil {
		// 返回默认配置
		return &ServerConfig{
			ServerIP:          "localhost",
			ServerPort:        20103,
			AuthToken:         "",
			HeartbeatInterval: 30,
		}, nil
	}

	// 创建副本以避免并发修改
	copy := *m.serverConfig
	return &copy, nil
}

func (m *manager) GetNodeConfig(ctx context.Context) (*NodeConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.nodeConfig == nil {
		return nil, fmt.Errorf("node config not initialized")
	}

	// 创建副本以避免并发修改
	result := *m.nodeConfig
	result.RunningTasks = make([]string, len(m.nodeConfig.RunningTasks))
	copy(result.RunningTasks, m.nodeConfig.RunningTasks)
	result.Capabilities = make([]model.CollectorType, len(m.nodeConfig.Capabilities))
	copy(result.Capabilities, m.nodeConfig.Capabilities)

	return &result, nil
}

func (m *manager) UpdateNodeConfig(ctx context.Context, config *NodeConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if config == nil {
		return fmt.Errorf("node config cannot be nil")
	}

	m.logger.Info("updating node config",
		"node_id", config.NodeID,
		"node_type", config.NodeType,
		"region", config.Region,
		"namespace", config.Namespace)

	// 更新配置
	m.nodeConfig = config

	// 保存到本地
	if err := m.saveConfig(); err != nil {
		m.logger.Error("failed to save config after update", "error", err)
	}

	return nil
}

// Task配置管理方法
func (m *manager) GetTaskConfigs(ctx context.Context) (map[string]*model.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 创建副本
	copy := make(map[string]*model.Task)
	for k, v := range m.taskConfigs {
		taskCopy := *v
		copy[k] = &taskCopy
	}

	return copy, nil
}

func (m *manager) UpdateTaskConfigs(ctx context.Context, tasks map[string]*model.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info("updating task configs", "count", len(tasks))

	// 更新任务配置
	m.taskConfigs = make(map[string]*model.Task)
	for k, v := range tasks {
		taskCopy := *v
		m.taskConfigs[k] = &taskCopy
	}

	// 保存到本地
	if err := m.saveConfig(); err != nil {
		m.logger.Error("failed to save config after task update", "error", err)
	}

	return nil
}

// 私有辅助方法
func (m *manager) ensureStorePath() error {
	dir := filepath.Dir(m.config.StorePath)
	return os.MkdirAll(dir, 0755)
}

func (m *manager) loadConfig() error {
	data, err := os.ReadFile(m.config.StorePath)
	if err != nil {
		if os.IsNotExist(err) {
			m.logger.Info("config file not found, will create new one", "path", m.config.StorePath)
			return nil
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config struct {
		Server *ServerConfig          `json:"server"`
		Node   *NodeConfig            `json:"node"`
		Tasks  map[string]*model.Task `json:"tasks"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	m.serverConfig = config.Server
	m.nodeConfig = config.Node
	if config.Tasks != nil {
		m.taskConfigs = config.Tasks
	}

	m.logger.Info("config loaded from file", "path", m.config.StorePath)
	return nil
}

func (m *manager) saveConfig() error {
	config := struct {
		Server *ServerConfig          `json:"server"`
		Node   *NodeConfig            `json:"node"`
		Tasks  map[string]*model.Task `json:"tasks"`
	}{
		Server: m.serverConfig,
		Node:   m.nodeConfig,
		Tasks:  m.taskConfigs,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(m.config.StorePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	m.logger.Debug("config saved to file", "path", m.config.StorePath)
	return nil
}

func (m *manager) initDefaultConfig() {
	// 初始化默认节点配置
	m.nodeConfig = &NodeConfig{
		NodeID:       "default-node",
		NodeType:     "standalone",
		Region:       "",
		Namespace:    "",
		RunningTasks: []string{},
		Capabilities: []model.CollectorType{
			model.CollectorTypeBinance,
			model.CollectorTypeOKX,
			model.CollectorTypeHuobi,
		},
	}

	// 初始化默认服务器配置
	m.serverConfig = &ServerConfig{
		ServerIP:          "localhost",
		ServerPort:        20103,
		AuthToken:         "",
		HeartbeatInterval: 30,
	}

	m.logger.Info("initialized default config")
}
