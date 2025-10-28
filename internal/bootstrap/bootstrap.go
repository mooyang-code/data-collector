package bootstrap

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/mooyang-code/data-collector/internal/collector"
	_ "github.com/mooyang-code/data-collector/internal/collector/exchanges" // 注册采集器
	"github.com/mooyang-code/data-collector/internal/config"
	"github.com/mooyang-code/data-collector/internal/event"
	"github.com/mooyang-code/data-collector/internal/heartbeat"
	"github.com/mooyang-code/data-collector/internal/metrics"
	"github.com/mooyang-code/data-collector/internal/task"
	"github.com/mooyang-code/data-collector/pkg/logger"
	"github.com/mooyang-code/data-collector/pkg/model"
)

// Bootstrap 系统启动器
type Bootstrap struct {
	// 配置
	config *Config

	// 核心组件
	logger           logger.Logger
	eventBus         event.Notifier
	metricsCollector metrics.Collector

	// 管理器
	configManager    config.Manager
	taskManager      task.Manager
	heartbeatManager heartbeat.Manager
	collectorManager collector.Manager

	// 运行时状态
	nodeInfo  *model.NodeInfo
	startTime time.Time
	mu        sync.RWMutex
	state     State
}

// State 启动器状态
type State string

const (
	StateUninitialized State = "uninitialized"
	StateInitializing  State = "initializing"
	StateRunning       State = "running"
	StateStopping      State = "stopping"
	StateStopped       State = "stopped"
	StateError         State = "error"
)

// Config 启动器配置
type Config struct {
	NodeID      string `json:"node_id" yaml:"node_id"`
	NodeType    string `json:"node_type" yaml:"node_type"`
	Region      string `json:"region" yaml:"region"`
	Namespace   string `json:"namespace" yaml:"namespace"`
	Version     string `json:"version" yaml:"version"`
	Environment string `json:"environment" yaml:"environment"`

	// 组件配置
	Logger   logger.Config  `json:"logger" yaml:"logger"`
	EventBus event.Config   `json:"event_bus" yaml:"event_bus"`
	Metrics  metrics.Config `json:"metrics" yaml:"metrics"`

	// 管理器配置
	Task      task.Config      `json:"task" yaml:"task"`
	Heartbeat heartbeat.Config `json:"heartbeat" yaml:"heartbeat"`
	Config    config.Config    `json:"config" yaml:"config"`
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		NodeID:      "", // NodeID 将在 Init 时动态获取
		NodeType:    "", // NodeType 将在 Init 时根据运行环境动态判断
		Region:      "",
		Namespace:   "",
		Version:     "1.0.0",
		Environment: "",

		Logger:   logger.Config{Level: "info", Format: "json"},
		EventBus: event.DefaultConfig,
		Metrics:  metrics.DefaultConfig,

		Task:      task.DefaultConfig,
		Heartbeat: heartbeat.DefaultConfig,
		Config:    config.DefaultConfig,
	}
}

// New 创建新的启动器
func New(cfg *Config) *Bootstrap {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// 设置节点ID到日志配置
	cfg.Logger.NodeID = cfg.NodeID
	return &Bootstrap{
		config:    cfg,
		startTime: time.Now(),
		state:     StateUninitialized,
	}
}

// Init 初始化启动器
func (b *Bootstrap) Init(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.state != StateUninitialized {
		return fmt.Errorf("bootstrap already initialized")
	}

	b.state = StateInitializing

	// 动态获取NodeID（如果为空）
	if b.config.NodeID == "" {
		hostname, err := os.Hostname()
		if err != nil || hostname == "" {
			hostname = "unknown-host"
		}
		b.config.NodeID = hostname
	}

	// 动态判断NodeType（如果为空）
	if b.config.NodeType == "" {
		// 默认为独立运行模式，使用操作系统类型
		b.config.NodeType = runtime.GOOS // linux, darwin, windows 等
	}

	// 初始化日志
	b.config.Logger.NodeID = b.config.NodeID
	b.logger = logger.New(b.config.Logger)
	logger.SetGlobal(b.logger)

	b.logger.Info("initializing bootstrap",
		"node_id", b.config.NodeID,
		"node_type", b.config.NodeType,
		"version", b.config.Version)

	// 初始化节点信息
	b.nodeInfo = &model.NodeInfo{
		NodeID:       b.config.NodeID,
		NodeType:     b.config.NodeType,
		Region:       b.config.Region,
		Namespace:    b.config.Namespace,
		Version:      b.config.Version,
		RunningTasks: make([]string, 0),
		Capabilities: []model.CollectorType{
			model.CollectorTypeBinance,
			model.CollectorTypeOKX,
			model.CollectorTypeHuobi,
		},
		Metadata: map[string]string{
			"go_version": runtime.Version(),
			"os":         runtime.GOOS,
			"arch":       runtime.GOARCH,
			"hostname":   b.config.NodeID,
		},
	}

	// 初始化基础组件
	if err := b.initInfrastructure(); err != nil {
		b.state = StateError
		return fmt.Errorf("failed to initialize infrastructure: %w", err)
	}

	// 初始化管理器
	if err := b.initManagers(); err != nil {
		b.state = StateError
		return fmt.Errorf("failed to initialize managers: %w", err)
	}

	b.logger.Info("bootstrap initialized successfully")
	return nil
}

// Start 启动系统
func (b *Bootstrap) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.state != StateInitializing {
		return fmt.Errorf("bootstrap not initialized or already started")
	}
	b.logger.Info("starting bootstrap")

	// 启动事件总线
	if err := b.eventBus.Start(ctx); err != nil {
		b.state = StateError
		return fmt.Errorf("failed to start event bus: %w", err)
	}

	// 启动配置管理器
	if err := b.configManager.Start(ctx); err != nil {
		b.state = StateError
		return fmt.Errorf("failed to start config manager: %w", err)
	}

	// 启动任务管理器
	if err := b.taskManager.Start(ctx); err != nil {
		b.state = StateError
		return fmt.Errorf("failed to start task manager: %w", err)
	}

	// 启动心跳管理器
	if err := b.heartbeatManager.Start(ctx); err != nil {
		b.state = StateError
		return fmt.Errorf("failed to start heartbeat manager: %w", err)
	}

	// 启动指标上报
	b.startMetricsReporting()

	b.state = StateRunning
	b.logger.Info("bootstrap started successfully",
		"startup_duration_ms", time.Since(b.startTime).Milliseconds())
	return nil
}

// Stop 停止系统
func (b *Bootstrap) Stop(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.state != StateRunning {
		return fmt.Errorf("bootstrap not running")
	}

	b.state = StateStopping
	b.logger.Info("stopping bootstrap")

	// 停止心跳管理器
	if err := b.heartbeatManager.Stop(ctx); err != nil {
		b.logger.Error("failed to stop heartbeat manager", "error", err)
	}

	// 停止任务管理器
	if err := b.taskManager.Stop(ctx); err != nil {
		b.logger.Error("failed to stop task manager", "error", err)
	}

	// 停止配置管理器
	if err := b.configManager.Stop(ctx); err != nil {
		b.logger.Error("failed to stop config manager", "error", err)
	}

	// 停止事件总线
	if err := b.eventBus.Stop(ctx); err != nil {
		b.logger.Error("failed to stop event bus", "error", err)
	}


	b.state = StateStopped
	b.logger.Info("bootstrap stopped successfully")

	return nil
}

// GetState 获取启动器状态
func (b *Bootstrap) GetState() State {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.state
}

// GetNodeInfo 获取节点信息
func (b *Bootstrap) GetNodeInfo() *model.NodeInfo {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.nodeInfo
}

// GetManagers 获取管理器实例（用于handler调用）
func (b *Bootstrap) GetManagers() (task.Manager, config.Manager, heartbeat.Manager) {
	return b.taskManager, b.configManager, b.heartbeatManager
}

// GetLogger 获取日志实例
func (b *Bootstrap) GetLogger() logger.Logger {
	return b.logger
}

// initInfrastructure 初始化基础设施
func (b *Bootstrap) initInfrastructure() error {
	// 初始化事件总线
	b.eventBus = event.NewNotifier(b.config.EventBus, b.logger)

	// 初始化指标收集器
	b.metricsCollector = metrics.New(b.config.Metrics)

	// 初始化采集器管理器
	b.collectorManager = collector.NewManager(b.logger, b.metricsCollector)
	return nil
}

// initManagers 初始化管理器
func (b *Bootstrap) initManagers() error {
	// 初始化配置管理器
	b.configManager = config.NewManager(b.config.Config, b.logger, b.eventBus)

	// 初始化任务管理器
	b.taskManager = task.NewManager(b.config.Task, b.logger, b.eventBus, b.metricsCollector, b.collectorManager)

	// 初始化心跳管理器
	b.heartbeatManager = heartbeat.NewManager(b.config.Heartbeat, b.logger, b.nodeInfo, b.taskManager, b.metricsCollector)
	return nil
}

// startMetricsReporting 启动指标上报
func (b *Bootstrap) startMetricsReporting() {
	if !b.config.Metrics.Enabled {
		return
	}

	go func() {
		ticker := time.NewTicker(b.config.Metrics.ReportInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				b.reportSystemMetrics()
			}
		}
	}()
}

// reportSystemMetrics 上报系统指标
func (b *Bootstrap) reportSystemMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 上报内存使用情况
	b.metricsCollector.SetGauge(metrics.MetricMemoryUsage, float64(m.Alloc))

	// 上报协程数量
	b.metricsCollector.SetGauge(metrics.MetricGoroutines, float64(runtime.NumGoroutine()))

	// CPU使用率需要外部监控系统计算，这里设置为0
	b.metricsCollector.SetGauge(metrics.MetricCPUUsage, 0)
}
