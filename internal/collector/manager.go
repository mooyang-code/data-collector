// Package collector 采集器管理器实现（最好是中文注释！）
package collector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mooyang-code/data-collector/internal/app"
	"github.com/mooyang-code/data-collector/model"
	"trpc.group/trpc-go/trpc-go/log"
)

// CollectorManagerImpl 采集器管理器实现 - 管理多种类型的数据采集器
type CollectorManagerImpl struct {
	// 基础信息
	id   string
	name string

	// 采集器管理
	collectors       map[string]app.Collector
	collectorsByType map[model.DataType][]app.Collector

	// 状态管理
	running   bool
	startTime time.Time

	// 同步控制
	mu     sync.RWMutex

	// 监控数据
	lastError  error
	errorCount int

	// 配置
	maxConcurrentCollectors int
	healthCheckInterval     time.Duration
}

// NewCollectorManager 创建采集器管理器
func NewCollectorManager() types.CollectorManager {
	return &CollectorManagerImpl{
		id:                      fmt.Sprintf("collector-manager-%d", time.Now().Unix()),
		name:                    "默认采集器管理器",
		collectors:              make(map[string]types.Collector),
		collectorsByType:        make(map[types.DataType][]types.Collector),
		maxConcurrentCollectors: 50,
		healthCheckInterval:     30 * time.Second,
	}
}

// NewCollectorManager 创建新的采集器管理器
func NewCollectorManager(id string) *CollectorManagerImpl {
	return &CollectorManagerImpl{
		id:                      id,
		name:                    fmt.Sprintf("collector-manager-%s", id),
		collectors:              make(map[string]types.Collector),
		collectorsByType:        make(map[types.DataType][]types.Collector),
		maxConcurrentCollectors: 10,
		healthCheckInterval:     30 * time.Second,
	}
}

// ================================
// 采集器管理接口实现
// ================================

// AddCollector 添加采集器
func (cm *CollectorManagerImpl) AddCollector(collector types.Collector) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	collectorID := collector.GetID()
	
	// 检查是否已存在
	if _, exists := cm.collectors[collectorID]; exists {
		return fmt.Errorf("采集器 %s 已存在", collectorID)
	}
	
	// 添加到映射
	cm.collectors[collectorID] = collector
	
	// 按类型分组
	dataType := collector.GetDataType()
	cm.collectorsByType[dataType] = append(cm.collectorsByType[dataType], collector)
	
	log.Infof("成功添加采集器: %s (类型: %s, 数据类型: %s)", 
		collectorID, collector.GetType(), dataType)
	
	// 如果管理器正在运行，启动新添加的采集器
	if cm.running {
		if err := collector.Start(context.Background()); err != nil {
			log.Errorf("启动新添加的采集器 %s 失败: %v", collectorID, err)
			return fmt.Errorf("启动采集器失败: %w", err)
		}
	}
	
	return nil
}

// RemoveCollector 移除采集器
func (cm *CollectorManagerImpl) RemoveCollector(collectorID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	collector, exists := cm.collectors[collectorID]
	if !exists {
		return fmt.Errorf("采集器 %s 不存在", collectorID)
	}
	
	// 停止采集器
	if collector.IsRunning() {
		if err := collector.Stop(cm.ctx); err != nil {
			log.Errorf("停止采集器 %s 失败: %v", collectorID, err)
		}
	}
	
	// 从映射中删除
	delete(cm.collectors, collectorID)
	
	// 从类型分组中删除
	dataType := collector.GetDataType()
	collectors := cm.collectorsByType[dataType]
	for i, c := range collectors {
		if c.GetID() == collectorID {
			cm.collectorsByType[dataType] = append(collectors[:i], collectors[i+1:]...)
			break
		}
	}
	
	// 如果该类型没有采集器了，删除类型映射
	if len(cm.collectorsByType[dataType]) == 0 {
		delete(cm.collectorsByType, dataType)
	}
	
	log.Infof("成功移除采集器: %s", collectorID)
	return nil
}

// GetCollector 获取采集器
func (cm *CollectorManagerImpl) GetCollector(collectorID string) (types.Collector, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	collector, exists := cm.collectors[collectorID]
	return collector, exists
}

// GetCollectors 获取所有采集器
func (cm *CollectorManagerImpl) GetCollectors() []types.Collector {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	collectors := make([]types.Collector, 0, len(cm.collectors))
	for _, collector := range cm.collectors {
		collectors = append(collectors, collector)
	}
	return collectors
}

// GetCollectorsByType 根据数据类型获取采集器
func (cm *CollectorManagerImpl) GetCollectorsByType(dataType types.DataType) []types.Collector {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	collectors := cm.collectorsByType[dataType]
	result := make([]types.Collector, len(collectors))
	copy(result, collectors)
	return result
}

// GetCollectorsByStatus 根据状态获取采集器
func (cm *CollectorManagerImpl) GetCollectorsByStatus(status types.CollectorState) []types.Collector {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	var result []types.Collector
	for _, collector := range cm.collectors {
		if collector.GetStatus().State == status {
			result = append(result, collector)
		}
	}
	return result
}

// ================================
// 生命周期管理接口实现
// ================================

// Start 启动采集器管理器
func (cm *CollectorManagerImpl) Start(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	if cm.running {
		return fmt.Errorf("采集器管理器已经在运行")
	}
	
	log.Infof("启动采集器管理器: %s", cm.id)

	cm.startTime = time.Now()

	// 启动所有采集器
	var errors []error
	for collectorID, collector := range cm.collectors {
		if err := collector.Start(ctx); err != nil {
			log.Errorf("启动采集器 %s 失败: %v", collectorID, err)
			errors = append(errors, fmt.Errorf("启动采集器 %s 失败: %w", collectorID, err))
		} else {
			log.Infof("成功启动采集器: %s", collectorID)
		}
	}
	
	cm.running = true
	
	// 启动监控协程
	go cm.monitorLoop()
	go cm.healthCheckLoop()
	
	log.Infof("采集器管理器启动成功，管理 %d 个采集器", len(cm.collectors))
	
	if len(errors) > 0 {
		return fmt.Errorf("启动部分采集器失败: %v", errors)
	}
	
	return nil
}

// Stop 停止采集器管理器
func (cm *CollectorManagerImpl) Stop(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	if !cm.running {
		return nil
	}
	
	log.Infof("停止采集器管理器: %s", cm.id)
	
	// 停止所有采集器
	var errors []error
	for collectorID, collector := range cm.collectors {
		if collector.IsRunning() {
			if err := collector.Stop(ctx); err != nil {
				log.Errorf("停止采集器 %s 失败: %v", collectorID, err)
				errors = append(errors, fmt.Errorf("停止采集器 %s 失败: %w", collectorID, err))
			} else {
				log.Infof("成功停止采集器: %s", collectorID)
			}
		}
	}
	

	
	cm.running = false
	
	log.Infof("采集器管理器已停止")
	
	if len(errors) > 0 {
		return fmt.Errorf("停止部分采集器失败: %v", errors)
	}
	
	return nil
}

// IsRunning 检查是否正在运行
func (cm *CollectorManagerImpl) IsRunning() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.running
}



// GetHealth 获取健康状态
func (cm *CollectorManagerImpl) GetHealth() *types.HealthStatus {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	health := &types.HealthStatus{
		Timestamp: time.Now(),
		Checks:    make(map[string]string),
	}
	
	// 检查运行状态
	if cm.running {
		health.Checks["running"] = "healthy"
	} else {
		health.Checks["running"] = "stopped"
	}
	
	// 检查采集器状态
	totalCollectors := len(cm.collectors)
	runningCollectors := 0
	errorCollectors := 0
	
	for _, collector := range cm.collectors {
		if collector.IsRunning() {
			runningCollectors++
		}
		
		collectorHealth := collector.GetHealth()
		if collectorHealth.Status == "unhealthy" {
			errorCollectors++
		}
	}
	
	if totalCollectors == 0 {
		health.Status = "healthy"
		health.Checks["collectors"] = "no collectors configured"
	} else if errorCollectors > 0 {
		health.Status = "unhealthy"
		health.Checks["collectors"] = fmt.Sprintf("%d collectors in error state", errorCollectors)
		health.ErrorCount = errorCollectors
	} else if runningCollectors == totalCollectors {
		health.Status = "healthy"
		health.Checks["collectors"] = fmt.Sprintf("all %d collectors running", totalCollectors)
	} else if runningCollectors > 0 {
		health.Status = "degraded"
		health.Checks["collectors"] = fmt.Sprintf("%d/%d collectors running", runningCollectors, totalCollectors)
	} else {
		health.Status = "unhealthy"
		health.Checks["collectors"] = "no collectors running"
	}
	
	if cm.lastError != nil {
		health.LastError = cm.lastError.Error()
	}
	
	return health
}

// ================================
// 私有方法
// ================================

// monitorLoop 监控循环
func (cm *CollectorManagerImpl) monitorLoop() {
	ticker := time.NewTicker(60 * time.Second) // 每分钟更新一次统计
	defer ticker.Stop()

	for {
		select {
		case <-cm.ctx.Done():
			return
		case <-ticker.C:
			cm.mu.Lock()
			cm.updateMetrics()
			cm.mu.Unlock()

			log.Debugf("采集器管理器统计: 总数=%d, 运行中=%d",
				cm.metrics.TotalCollectors, cm.metrics.RunningCollectors)
		}
	}
}

// healthCheckLoop 健康检查循环
func (cm *CollectorManagerImpl) healthCheckLoop() {
	ticker := time.NewTicker(cm.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cm.ctx.Done():
			return
		case <-ticker.C:
			cm.performHealthCheck()
		}
	}
}

// updateMetrics 更新指标
func (cm *CollectorManagerImpl) updateMetrics() {
	cm.metrics.TotalCollectors = len(cm.collectors)
	cm.metrics.RunningCollectors = 0

	for _, collector := range cm.collectors {
		if collector.IsRunning() {
			cm.metrics.RunningCollectors++
		}
	}
}

// performHealthCheck 执行健康检查
func (cm *CollectorManagerImpl) performHealthCheck() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 检查每个采集器的健康状态
	for collectorID, collector := range cm.collectors {
		health := collector.GetHealth()
		if health.Status != "healthy" {
			log.Warnf("采集器 %s 健康状态异常: %s", collectorID, health.Status)

			// 如果采集器处于错误状态，可以考虑重启
			if health.Status == "unhealthy" && collector.IsRunning() {
				log.Warnf("考虑重启异常采集器: %s", collectorID)
				// 这里可以添加自动重启逻辑
			}
		}
	}
}
