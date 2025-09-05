// Package app 应用实现
package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mooyang-code/data-collector/configs"
	"trpc.group/trpc-go/trpc-go/log"
)

// BaseApp 基础应用实现
type BaseApp struct {
	id               string
	config           *AppConfig
	dataset          Dataset
	collectorManager CollectorManager
	storage          Storage
	status           AppStatus
	metrics          AppMetrics
	mutex            sync.RWMutex
	running          bool
}

// NewDefaultApp 创建默认应用
func NewDefaultApp(config *AppConfig) (App, error) {
	if config == nil {
		return nil, fmt.Errorf("应用配置不能为空")
	}

	app := &BaseApp{
		id:     config.ID,
		config: config,
		status: AppStatus{
			State: "created",
		},
	}

	return app, nil
}

// NewDataCollectorApp 创建数据采集器应用
func NewDataCollectorApp(config *AppConfig) (App, error) {
	if config == nil {
		return nil, fmt.Errorf("应用配置不能为空")
	}

	// 创建采集器管理器
	collectorManager := NewDataCollectorManager()

	app := &BaseApp{
		id:     config.ID,
		config: config,
		status: AppStatus{
			State: "created",
		},
		dataset:          &DefaultDataset{name: "data-collector"},
		collectorManager: collectorManager,
		storage:          &DefaultStorage{},
	}

	// 根据配置初始化采集器
	if err := app.initializeCollectors(config); err != nil {
		return nil, fmt.Errorf("初始化采集器失败: %w", err)
	}

	return app, nil
}

// GetID 获取应用ID
func (a *BaseApp) GetID() string {
	return a.id
}

// GetDataset 获取数据集
func (a *BaseApp) GetDataset() Dataset {
	return a.dataset
}

// GetCollectorManager 获取采集器管理器
func (a *BaseApp) GetCollectorManager() CollectorManager {
	return a.collectorManager
}

// GetStorage 获取存储
func (a *BaseApp) GetStorage() Storage {
	return a.storage
}

// Initialize 初始化应用
func (a *BaseApp) Initialize(config *AppConfig) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.config = config
	a.status.State = "initialized"

	log.Infof("应用初始化完成: %s", a.id)
	return nil
}

// Start 启动应用
func (a *BaseApp) Start(ctx context.Context) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if a.running {
		return fmt.Errorf("应用已在运行: %s", a.id)
	}

	a.status.State = "running"
	a.status.StartTime = time.Now()
	a.running = true

	// 启动采集器管理器
	if a.collectorManager != nil {
		if err := a.collectorManager.Start(ctx); err != nil {
			a.status.State = "error"
			a.status.LastError = err.Error()
			a.running = false
			return fmt.Errorf("启动采集器管理器失败: %w", err)
		}
	}

	log.Infof("应用启动成功: %s", a.id)
	return nil
}

// Stop 停止应用
func (a *BaseApp) Stop(ctx context.Context) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if !a.running {
		return nil
	}

	// 停止采集器管理器
	if a.collectorManager != nil {
		if err := a.collectorManager.Stop(ctx); err != nil {
			log.Errorf("停止采集器管理器失败: %v", err)
		}
	}

	a.status.State = "stopped"
	a.running = false

	log.Infof("应用停止成功: %s", a.id)
	return nil
}

// IsRunning 检查应用是否运行中
func (a *BaseApp) IsRunning() bool {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.running
}

// GetStatus 获取应用状态
func (a *BaseApp) GetStatus() AppStatus {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.status
}

// GetMetrics 获取应用指标
func (a *BaseApp) GetMetrics() AppMetrics {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	if a.running && !a.status.StartTime.IsZero() {
		a.metrics.Uptime = time.Since(a.status.StartTime)
	}

	return a.metrics
}

// initializeCollectors 初始化采集器
func (a *BaseApp) initializeCollectors(config *AppConfig) error {
	// 从配置中获取采集器信息
	collectorsConfig, ok := config.Config["collectors"]
	if !ok {
		log.Infof("应用 %s 没有配置采集器", a.id)
		return nil
	}

	appName, _ := config.Config["appName"].(string)
	if appName == "" {
		appName = a.id
	}

	// 类型断言获取采集器配置
	collectors, ok := collectorsConfig.(map[string]configs.Collector)
	if !ok {
		log.Warnf("应用 %s 的采集器配置格式不正确", a.id)
		return nil
	}

	// 获取数据采集器管理器
	dataCollectorManager, ok := a.collectorManager.(*DataCollectorManager)
	if !ok {
		return fmt.Errorf("采集器管理器类型不正确")
	}

	// 注册每个启用的采集器
	for collectorName, collector := range collectors {
		if !collector.Enabled {
			log.Infof("跳过未启用的采集器: %s.%s", appName, collectorName)
			continue
		}

		log.Infof("注册采集器: %s.%s (类型: %s)", appName, collectorName, collector.DataType)

		// 根据采集器类型注册
		if err := dataCollectorManager.RegisterCollector(appName, collectorName, &collector); err != nil {
			log.Errorf("注册采集器失败: %s.%s, error: %v", appName, collectorName, err)
			return fmt.Errorf("注册采集器 %s.%s 失败: %w", appName, collectorName, err)
		}
	}

	log.Infof("应用 %s 采集器初始化完成", a.id)
	return nil
}

// DefaultDataset 默认数据集实现
type DefaultDataset struct {
	name        string
	description string
}

func (d *DefaultDataset) GetName() string {
	return d.name
}

func (d *DefaultDataset) GetDescription() string {
	return d.description
}

// DataCollectorManager 数据采集器管理器实现
type DataCollectorManager struct {
	running          bool
	collectors       map[string]*CollectorInfo
	collectorsByType map[string][]*CollectorInfo
	collectorFactory CollectorFactory
	mutex            sync.RWMutex
}

// CollectorInfo 采集器信息
type CollectorInfo struct {
	AppName       string
	CollectorName string
	Config        *configs.Collector
	Instance      Collector // 具体的采集器实例
	Running       bool
}

// NewDataCollectorManager 创建数据采集器管理器
func NewDataCollectorManager() *DataCollectorManager {
	return &DataCollectorManager{
		collectors:       make(map[string]*CollectorInfo),
		collectorsByType: make(map[string][]*CollectorInfo),
		collectorFactory: NewCollectorFactory(),
	}
}

// RegisterCollector 注册采集器
func (dcm *DataCollectorManager) RegisterCollector(appName, collectorName string, config *configs.Collector) error {
	dcm.mutex.Lock()
	defer dcm.mutex.Unlock()

	collectorID := fmt.Sprintf("%s.%s", appName, collectorName)

	// 检查是否已存在
	if _, exists := dcm.collectors[collectorID]; exists {
		return fmt.Errorf("采集器 %s 已存在", collectorID)
	}

	// 使用工厂创建采集器实例
	collectorInstance, err := dcm.collectorFactory.CreateCollector(appName, collectorName, config)
	if err != nil {
		log.Warnf("创建采集器实例失败: %s, error: %v, 将使用占位符", collectorID, err)
		collectorInstance = nil // 允许为空，稍后可以重试创建
	}

	// 创建采集器信息
	collectorInfo := &CollectorInfo{
		AppName:       appName,
		CollectorName: collectorName,
		Config:        config,
		Instance:      collectorInstance,
		Running:       false,
	}

	// 添加到映射
	dcm.collectors[collectorID] = collectorInfo

	// 按类型分组
	dataType := config.DataType
	dcm.collectorsByType[dataType] = append(dcm.collectorsByType[dataType], collectorInfo)

	log.Infof("成功注册采集器: %s (数据类型: %s, 市场类型: %s)",
		collectorID, config.DataType, config.MarketType)
	return nil
}

// Start 启动采集器管理器
func (dcm *DataCollectorManager) Start(ctx context.Context) error {
	dcm.mutex.Lock()
	defer dcm.mutex.Unlock()

	dcm.running = true
	log.Info("数据采集器管理器启动")

	// 启动所有注册的采集器
	for collectorID, collectorInfo := range dcm.collectors {
		if err := dcm.startCollector(ctx, collectorInfo); err != nil {
			log.Errorf("启动采集器失败: %s, error: %v", collectorID, err)
		} else {
			collectorInfo.Running = true
			log.Infof("采集器启动成功: %s", collectorID)
		}
	}
	return nil
}

// Stop 停止采集器管理器
func (dcm *DataCollectorManager) Stop(ctx context.Context) error {
	dcm.mutex.Lock()
	defer dcm.mutex.Unlock()

	// 停止所有采集器
	for collectorID, collectorInfo := range dcm.collectors {
		if collectorInfo.Running {
			if err := dcm.stopCollector(ctx, collectorInfo); err != nil {
				log.Errorf("停止采集器失败: %s, error: %v", collectorID, err)
			} else {
				collectorInfo.Running = false
				log.Infof("采集器停止成功: %s", collectorID)
			}
		}
	}

	dcm.running = false
	log.Info("数据采集器管理器停止")
	return nil
}

// IsRunning 检查管理器是否运行中
func (dcm *DataCollectorManager) IsRunning() bool {
	dcm.mutex.RLock()
	defer dcm.mutex.RUnlock()
	return dcm.running
}

// startCollector 启动单个采集器
func (dcm *DataCollectorManager) startCollector(ctx context.Context, info *CollectorInfo) error {
	log.Infof("启动采集器: %s.%s (类型: %s)", info.AppName, info.CollectorName, info.Config.DataType)

	// 如果没有采集器实例，尝试创建
	if info.Instance == nil {
		collectorInstance, err := dcm.collectorFactory.CreateCollector(info.AppName, info.CollectorName, info.Config)
		if err != nil {
			return fmt.Errorf("创建采集器实例失败: %w", err)
		}
		info.Instance = collectorInstance
	}

	// 初始化采集器
	if err := info.Instance.Initialize(ctx); err != nil {
		return fmt.Errorf("初始化采集器失败: %w", err)
	}

	// 启动采集器
	if err := info.Instance.StartCollection(ctx); err != nil {
		return fmt.Errorf("启动采集器失败: %w", err)
	}

	log.Infof("采集器启动成功: %s.%s", info.AppName, info.CollectorName)
	return nil
}

// stopCollector 停止单个采集器
func (dcm *DataCollectorManager) stopCollector(ctx context.Context, info *CollectorInfo) error {
	log.Infof("停止采集器: %s.%s", info.AppName, info.CollectorName)

	if info.Instance == nil {
		log.Warnf("采集器实例为空，无需停止: %s.%s", info.AppName, info.CollectorName)
		return nil
	}

	// 停止采集器
	if err := info.Instance.StopCollection(ctx); err != nil {
		return fmt.Errorf("停止采集器失败: %w", err)
	}

	log.Infof("采集器停止成功: %s.%s", info.AppName, info.CollectorName)
	return nil
}

// GetCollectors 获取所有采集器信息
func (dcm *DataCollectorManager) GetCollectors() map[string]*CollectorInfo {
	dcm.mutex.RLock()
	defer dcm.mutex.RUnlock()

	result := make(map[string]*CollectorInfo)
	for id, info := range dcm.collectors {
		result[id] = info
	}
	return result
}

// GetCollectorsByType 根据数据类型获取采集器
func (dcm *DataCollectorManager) GetCollectorsByType(dataType string) []*CollectorInfo {
	dcm.mutex.RLock()
	defer dcm.mutex.RUnlock()

	return dcm.collectorsByType[dataType]
}

// DefaultCollectorManager 默认采集器管理器实现
type DefaultCollectorManager struct {
	running bool
}

func (c *DefaultCollectorManager) Start(ctx context.Context) error {
	c.running = true
	log.Info("采集器管理器启动")
	return nil
}

func (c *DefaultCollectorManager) Stop(ctx context.Context) error {
	c.running = false
	log.Info("采集器管理器停止")
	return nil
}

func (c *DefaultCollectorManager) IsRunning() bool {
	return c.running
}

// DefaultStorage 默认存储实现
type DefaultStorage struct{}

func (s *DefaultStorage) Save(data interface{}) error {
	log.Debugf("保存数据: %+v", data)
	return nil
}

func (s *DefaultStorage) Load(key string) (interface{}, error) {
	log.Debugf("加载数据: %s", key)
	return nil, nil
}
