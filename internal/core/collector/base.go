package collector

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type BaseCollector struct {
	id       string
	typ      string
	dataType string
	
	timers   map[string]*Timer
	mu       sync.RWMutex
	
	isRunning bool
	startTime time.Time
	metrics   CollectorMetrics
	
	eventBus EventBus
}

// EventBus 接口（临时，后续移到 event 包）
type EventBus interface {
	Publish(event interface{}) error
	PublishAsync(event interface{})
}

func NewBaseCollector(id, typ, dataType string) *BaseCollector {
	return &BaseCollector{
		id:       id,
		typ:      typ,
		dataType: dataType,
		timers:   make(map[string]*Timer),
		metrics: CollectorMetrics{
			TimerMetrics: make(map[string]TimerMetrics),
		},
	}
}

func (c *BaseCollector) ID() string {
	return c.id
}

func (c *BaseCollector) Type() string {
	return c.typ
}

func (c *BaseCollector) DataType() string {
	return c.dataType
}

func (c *BaseCollector) Initialize(ctx context.Context) error {
	return nil
}

func (c *BaseCollector) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.isRunning {
		return fmt.Errorf("采集器 %s 已经在运行", c.id)
	}
	
	c.startTime = time.Now()
	c.metrics.StartTime = c.startTime
	
	// 启动所有定时器
	for name, timer := range c.timers {
		if err := c.startTimer(ctx, name, timer); err != nil {
			return fmt.Errorf("启动定时器 %s 失败: %w", name, err)
		}
	}
	
	c.isRunning = true
	log.Printf("采集器 %s 已启动", c.id)
	
	return nil
}

func (c *BaseCollector) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if !c.isRunning {
		return fmt.Errorf("采集器 %s 未在运行", c.id)
	}
	
	// 停止所有定时器
	for name, timer := range c.timers {
		c.stopTimer(name, timer)
	}
	
	c.isRunning = false
	log.Printf("采集器 %s 已停止", c.id)
	
	return nil
}

func (c *BaseCollector) AddTimer(name string, interval time.Duration, handler TimerHandler) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if _, exists := c.timers[name]; exists {
		return fmt.Errorf("定时器 %s 已存在", name)
	}
	
	timer := &Timer{
		Name:     name,
		Interval: interval,
		Handler:  handler,
		NextRun:  time.Now().Add(interval),
	}
	
	c.timers[name] = timer
	
	// 如果采集器正在运行，立即启动定时器
	if c.isRunning {
		// 注意：这里没有context可用，所以不能启动定时器
		// 定时器只能在Start方法被调用时启动
		return fmt.Errorf("不能在运行时添加定时器")
	}
	
	return nil
}

func (c *BaseCollector) RemoveTimer(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	timer, exists := c.timers[name]
	if !exists {
		return fmt.Errorf("定时器 %s 不存在", name)
	}
	
	c.stopTimer(name, timer)
	delete(c.timers, name)
	
	return nil
}

func (c *BaseCollector) GetTimers() map[string]*Timer {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	timers := make(map[string]*Timer)
	for k, v := range c.timers {
		timers[k] = v
	}
	
	return timers
}

func (c *BaseCollector) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return c.isRunning
}

func (c *BaseCollector) GetStatus() CollectorStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	status := CollectorStatus{
		ID:         c.id,
		Type:       c.typ,
		DataType:   c.dataType,
		IsRunning:  c.isRunning,
		StartTime:  c.startTime,
		LastUpdate: time.Now(),
		Timers:     make(map[string]TimerStatus),
	}
	
	for name, timer := range c.timers {
		status.Timers[name] = TimerStatus{
			Name:       timer.Name,
			Interval:   timer.Interval,
			LastRun:    timer.LastRun,
			NextRun:    timer.NextRun,
			RunCount:   timer.RunCount,
			ErrorCount: timer.ErrorCount,
		}
	}
	
	return status
}

func (c *BaseCollector) GetMetrics() CollectorMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return c.metrics
}

func (c *BaseCollector) SetEventBus(eventBus EventBus) {
	c.eventBus = eventBus
}

func (c *BaseCollector) PublishEvent(event interface{}) error {
	if c.eventBus == nil {
		return fmt.Errorf("事件总线未设置")
	}
	
	c.eventBus.PublishAsync(event)
	c.metrics.EventsPublished++
	
	return nil
}

func (c *BaseCollector) startTimer(ctx context.Context, name string, timer *Timer) error {
	timerCtx, cancel := context.WithCancel(ctx)
	timer.cancel = cancel
	timer.ticker = time.NewTicker(timer.Interval)
	
	go func() {
		log.Printf("定时器 %s.%s 已启动，间隔: %v", c.id, name, timer.Interval)
		
		// 立即执行一次
		c.executeTimer(timerCtx, timer)
		
		for {
			select {
			case <-timerCtx.Done():
				log.Printf("定时器 %s.%s 已停止", c.id, name)
				return
			case <-timer.ticker.C:
				c.executeTimer(timerCtx, timer)
			}
		}
	}()
	
	return nil
}

func (c *BaseCollector) stopTimer(name string, timer *Timer) {
	if timer.ticker != nil {
		timer.ticker.Stop()
	}
	
	if timer.cancel != nil {
		timer.cancel()
	}
}

func (c *BaseCollector) executeTimer(ctx context.Context, timer *Timer) {
	timer.LastRun = time.Now()
	timer.NextRun = time.Now().Add(timer.Interval)
	
	start := time.Now()
	
	if err := timer.Handler(ctx); err != nil {
		timer.ErrorCount++
		c.metrics.ErrorsTotal++
		c.metrics.LastError = err
		c.metrics.LastErrorTime = time.Now()
		log.Printf("定时器 %s.%s 执行失败: %v", c.id, timer.Name, err)
	} else {
		timer.RunCount++
		log.Printf("定时器 %s.%s 执行成功", c.id, timer.Name)
	}
	
	// 更新指标
	latency := time.Since(start)
	if timerMetrics, exists := c.metrics.TimerMetrics[timer.Name]; exists {
		timerMetrics.RunCount = timer.RunCount
		timerMetrics.ErrorCount = timer.ErrorCount
		timerMetrics.LastRun = timer.LastRun
		timerMetrics.AvgLatency = (timerMetrics.AvgLatency + latency) / 2
		c.metrics.TimerMetrics[timer.Name] = timerMetrics
	} else {
		c.metrics.TimerMetrics[timer.Name] = TimerMetrics{
			RunCount:   timer.RunCount,
			ErrorCount: timer.ErrorCount,
			LastRun:    timer.LastRun,
			AvgLatency: latency,
		}
	}
}