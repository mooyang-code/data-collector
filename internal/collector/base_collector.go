// Package collector 基础采集器实现
package collector

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mooyang-code/data-collector/configs"
	"github.com/mooyang-code/data-collector/internal/app"
	"github.com/mooyang-code/data-collector/model"
	"trpc.group/trpc-go/trpc-go/log"
)

// BaseCollector 基础采集器实现
type BaseCollector struct {
	id            string
	collectorType string
	name          string
	dataType      model.DataType
	config        *configs.CollectorConfig
	status        model.CollectorStatus
	mu            sync.RWMutex
	startTime     time.Time
	dataCount     int64
	errorCount    int64

	// 触发器
	trigger app.Trigger

	// 存储接口
	storage app.Storage
}

// NewBaseCollector 创建基础采集器实例
func NewBaseCollector(id, collectorType, name string, dataType model.DataType) *BaseCollector {
	return &BaseCollector{
		id:            id,
		collectorType: collectorType,
		name:          name,
		dataType:      dataType,
		status: model.CollectorStatus{
			ID:    id,
			Type:  collectorType,
			Name:  name,
			State: model.CollectorStateUnknown,
		},
	}
}

// GetID 获取采集器ID
func (c *BaseCollector) GetID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.id
}

// GetType 获取采集器类型
func (c *BaseCollector) GetType() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.collectorType
}

// GetName 获取采集器名称
func (c *BaseCollector) GetName() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.name
}

// GetDataType 获取数据类型
func (c *BaseCollector) GetDataType() model.DataType {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.dataType
}

// GetTrigger 获取触发器
func (c *BaseCollector) GetTrigger() app.Trigger {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.trigger
}

// SetTrigger 设置触发器
func (c *BaseCollector) SetTrigger(trigger app.Trigger) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if trigger == nil {
		return fmt.Errorf("触发器不能为空")
	}

	c.trigger = trigger
	log.Infof("采集器 %s 设置触发器 %s (%s)", c.id, trigger.GetID(), trigger.GetType())
	return nil
}

// Initialize 初始化采集器
func (c *BaseCollector) Initialize(config *configs.CollectorConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.status.State != model.CollectorStateUnknown {
		return fmt.Errorf("采集器 %s 已经初始化", c.id)
	}

	c.config = config
	c.status.State = model.CollectorStateInitialized
	c.status.LastUpdate = time.Now()

	log.Infof("采集器 %s (%s) 初始化完成", c.name, c.collectorType)
	return nil
}

// Start 启动采集器
func (c *BaseCollector) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.status.State == model.CollectorStateRunning {
		return fmt.Errorf("采集器 %s 已经在运行", c.id)
	}

	if c.status.State != model.CollectorStateInitialized {
		return fmt.Errorf("采集器 %s 未初始化", c.id)
	}

	c.status.State = model.CollectorStateStarting
	c.status.LastUpdate = time.Now()
	c.startTime = time.Now()

	// 重置计数器
	atomic.StoreInt64(&c.dataCount, 0)
	atomic.StoreInt64(&c.errorCount, 0)

	// 启动触发器
	if c.trigger != nil {
		if err := c.trigger.Start(ctx, c.triggerCallback); err != nil {
			c.status.State = model.CollectorStateError
			c.status.LastError = fmt.Sprintf("启动触发器失败: %v", err)
			return fmt.Errorf("启动触发器失败: %w", err)
		}
	}

	c.status.State = model.CollectorStateRunning
	c.status.StartTime = c.startTime
	c.status.LastUpdate = time.Now()

	log.Infof("采集器 %s (%s) 启动完成", c.name, c.collectorType)
	return nil
}

// Stop 停止采集器
func (c *BaseCollector) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.status.State != model.CollectorStateRunning {
		return fmt.Errorf("采集器 %s 未在运行", c.id)
	}

	c.status.State = model.CollectorStateStopping
	c.status.LastUpdate = time.Now()

	// 停止触发器
	if c.trigger != nil {
		if err := c.trigger.Stop(ctx); err != nil {
			log.Errorf("停止触发器失败: %v", err)
		}
	}



	c.status.State = model.CollectorStateStopped
	c.status.LastUpdate = time.Now()

	log.Infof("采集器 %s (%s) 停止完成", c.name, c.collectorType)
	return nil
}

// IsRunning 检查采集器是否运行中
func (c *BaseCollector) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.status.State == model.CollectorStateRunning
}

// GetStatus 获取采集器状态
func (c *BaseCollector) GetStatus() model.CollectorStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()

	status := c.status
	status.ErrorCount = int(atomic.LoadInt64(&c.errorCount))
	return status
}



// Collect 采集数据（基础实现，子类应该重写）
func (c *BaseCollector) Collect(ctx context.Context) error {
	// 基础实现只是记录日志，具体的采集逻辑由子类实现
	log.Infof("采集器 %s 执行数据采集", c.id)
	
	// 增加数据计数
	c.IncrementDataCount()
	
	return nil
}

// triggerCallback 触发器回调函数
func (c *BaseCollector) triggerCallback(ctx context.Context) error {
	return c.Collect(ctx)
}

// IncrementDataCount 增加数据计数
func (c *BaseCollector) IncrementDataCount() {
	atomic.AddInt64(&c.dataCount, 1)
}

// IncrementErrorCount 增加错误计数
func (c *BaseCollector) IncrementErrorCount() {
	atomic.AddInt64(&c.errorCount, 1)
	c.mu.Lock()
	c.status.ErrorCount++
	c.mu.Unlock()
}

// SetStorage 设置存储接口
func (c *BaseCollector) SetStorage(storage app.Storage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.storage = storage
}

// GetStorage 获取存储接口
func (c *BaseCollector) GetStorage() app.Storage {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.storage
}

// StoreData 存储数据
func (c *BaseCollector) StoreData(ctx context.Context, data *model.DataRecord) error {
	c.mu.RLock()
	storage := c.storage
	c.mu.RUnlock()

	if storage == nil {
		// 如果没有配置存储，直接返回（不报错）
		return nil
	}

	// 增加数据计数
	c.IncrementDataCount()

	return storage.Store(ctx, data)
}

// StoreBatchData 批量存储数据
func (c *BaseCollector) StoreBatchData(ctx context.Context, data []*model.DataRecord) error {
	c.mu.RLock()
	storage := c.storage
	c.mu.RUnlock()

	if storage == nil {
		// 如果没有配置存储，直接返回（不报错）
		return nil
	}

	// 增加数据计数
	for range data {
		c.IncrementDataCount()
	}

	return storage.StoreBatch(ctx, data)
}
