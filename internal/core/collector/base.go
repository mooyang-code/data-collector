// Package collector 基础采集器实现
package collector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"trpc.group/trpc-go/trpc-go/log"
)

// BaseCollector 基础采集器实现
type BaseCollector struct {
	id       string
	typ      string
	exchange string
	
	status   Status
	metrics  Metrics
	
	timers   map[string]*timerInstance
	timerMu  sync.RWMutex
	
	ctx      context.Context
	cancel   context.CancelFunc
	running  bool
	mu       sync.RWMutex
}

// timerInstance 定时器实例
type timerInstance struct {
	*Timer
	ticker *time.Ticker
	done   chan struct{}
}

// NewBaseCollector 创建基础采集器
func NewBaseCollector(id, typ, exchange string) *BaseCollector {
	return &BaseCollector{
		id:       id,
		typ:      typ,
		exchange: exchange,
		timers:   make(map[string]*timerInstance),
		status: Status{
			State: "idle",
		},
		metrics: Metrics{
			Custom: make(map[string]interface{}),
		},
	}
}

// ID 获取采集器ID
func (c *BaseCollector) ID() string {
	return c.id
}

// Type 获取采集器类型
func (c *BaseCollector) Type() string {
	return c.typ
}

// Exchange 获取交易所名称
func (c *BaseCollector) Exchange() string {
	return c.exchange
}

// Initialize 初始化采集器
func (c *BaseCollector) Initialize(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.updateStatus("initializing", "正在初始化")
	
	// 创建采集器专用的context
	c.ctx, c.cancel = context.WithCancel(ctx)
	
	c.updateStatus("initialized", "初始化完成")
	return nil
}

// Start 启动采集器
func (c *BaseCollector) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.running {
		return fmt.Errorf("采集器已在运行中")
	}
	
	c.updateStatus("running", "正在运行")
	c.status.StartTime = time.Now()
	c.running = true
	
	// 启动所有定时器
	c.startAllTimers()
	
	log.Infof("采集器启动成功: %s", c.id)
	return nil
}

// Stop 停止采集器
func (c *BaseCollector) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if !c.running {
		return nil
	}
	
	c.updateStatus("stopping", "正在停止")
	
	// 停止所有定时器
	c.stopAllTimers()
	
	// 取消context
	if c.cancel != nil {
		c.cancel()
	}
	
	c.running = false
	c.updateStatus("stopped", "已停止")
	
	log.Infof("采集器停止成功: %s", c.id)
	return nil
}

// IsRunning 检查是否运行中
func (c *BaseCollector) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

// GetStatus 获取状态
func (c *BaseCollector) GetStatus() Status {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.status
}

// GetMetrics 获取指标
func (c *BaseCollector) GetMetrics() Metrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.metrics
}

// AddTimer 添加定时器
func (c *BaseCollector) AddTimer(name string, interval time.Duration, handler TimerHandler) error {
	c.timerMu.Lock()
	defer c.timerMu.Unlock()
	
	if _, exists := c.timers[name]; exists {
		return fmt.Errorf("定时器已存在: %s", name)
	}
	
	timer := &Timer{
		Name:     name,
		Interval: interval,
		Handler:  handler,
		Running:  false,
	}
	
	instance := &timerInstance{
		Timer: timer,
		done:  make(chan struct{}),
	}
	
	c.timers[name] = instance
	
	// 如果采集器正在运行，立即启动定时器
	if c.running {
		c.startTimer(instance)
	}
	
	log.Infof("添加定时器成功: %s.%s (间隔: %v)", c.id, name, interval)
	return nil
}

// RemoveTimer 移除定时器
func (c *BaseCollector) RemoveTimer(name string) error {
	c.timerMu.Lock()
	defer c.timerMu.Unlock()
	
	instance, exists := c.timers[name]
	if !exists {
		return fmt.Errorf("定时器不存在: %s", name)
	}
	
	// 停止定时器
	c.stopTimer(instance)
	
	// 从映射中删除
	delete(c.timers, name)
	
	log.Infof("移除定时器成功: %s.%s", c.id, name)
	return nil
}

// GetTimers 获取所有定时器
func (c *BaseCollector) GetTimers() map[string]*Timer {
	c.timerMu.RLock()
	defer c.timerMu.RUnlock()
	
	result := make(map[string]*Timer)
	for name, instance := range c.timers {
		result[name] = instance.Timer
	}
	return result
}

// 内部方法

// updateStatus 更新状态
func (c *BaseCollector) updateStatus(state, message string) {
	c.status.State = state
	c.status.Message = message
	c.status.LastUpdate = time.Now()
}

// updateMetrics 更新指标
func (c *BaseCollector) UpdateMetrics(updates func(*Metrics)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	updates(&c.metrics)
}

// startAllTimers 启动所有定时器
func (c *BaseCollector) startAllTimers() {
	c.timerMu.Lock()
	defer c.timerMu.Unlock()
	
	for _, instance := range c.timers {
		c.startTimer(instance)
	}
}

// stopAllTimers 停止所有定时器
func (c *BaseCollector) stopAllTimers() {
	c.timerMu.Lock()
	defer c.timerMu.Unlock()
	
	for _, instance := range c.timers {
		c.stopTimer(instance)
	}
}

// startTimer 启动单个定时器
func (c *BaseCollector) startTimer(instance *timerInstance) {
	if instance.Running {
		return
	}
	
	instance.ticker = time.NewTicker(instance.Interval)
	instance.Running = true
	instance.NextRun = time.Now().Add(instance.Interval)
	
	// 在新的goroutine中运行定时器
	go func() {
		log.Infof("定时器启动: %s.%s", c.id, instance.Name)
		
		// 首次立即执行
		c.executeTimer(instance)
		
		for {
			select {
			case <-instance.ticker.C:
				c.executeTimer(instance)
			case <-instance.done:
				return
			case <-c.ctx.Done():
				return
			}
		}
	}()
}

// stopTimer 停止单个定时器
func (c *BaseCollector) stopTimer(instance *timerInstance) {
	if !instance.Running {
		return
	}
	
	instance.Running = false
	
	if instance.ticker != nil {
		instance.ticker.Stop()
	}
	
	close(instance.done)
	
	log.Infof("定时器停止: %s.%s", c.id, instance.Name)
}

// executeTimer 执行定时器
func (c *BaseCollector) executeTimer(instance *timerInstance) {
	instance.LastRun = time.Now()
	instance.NextRun = time.Now().Add(instance.Interval)
	instance.RunCount++
	
	// 执行处理函数
	if err := instance.Handler(c.ctx); err != nil {
		instance.Errors++
		log.Errorf("定时器执行失败: %s.%s, error: %v", c.id, instance.Name, err)
		
		// 更新错误状态
		c.mu.Lock()
		c.status.LastError = err
		c.mu.Unlock()
	}
}