package collector

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mooyang-code/data-collector/internal/core/scheduler"
)

type BaseCollector struct {
	id       string
	typ      string
	dataType string

	timers    map[string]*Timer
	scheduler scheduler.Scheduler // 使用 Cron 调度器
	mu        sync.RWMutex

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
		id:        id,
		typ:       typ,
		dataType:  dataType,
		timers:    make(map[string]*Timer),
		scheduler: scheduler.NewCronScheduler(),
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

	// 启动调度器
	if err := c.scheduler.Start(ctx); err != nil {
		return fmt.Errorf("启动调度器失败: %w", err)
	}

	c.isRunning = true
	log.Printf("采集器 %s 已启动（使用Cron调度器，整点执行）", c.id)

	return nil
}

func (c *BaseCollector) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isRunning {
		return fmt.Errorf("采集器 %s 未在运行", c.id)
	}

	// 停止调度器
	if err := c.scheduler.Stop(); err != nil {
		log.Printf("停止调度器失败: %v", err)
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

	// 包装 handler 以更新 metrics
	wrappedHandler := c.wrapHandler(name, handler)

	// 添加到调度器
	taskName := fmt.Sprintf("%s.%s", c.id, name)
	if err := c.scheduler.AddTask(taskName, interval, wrappedHandler); err != nil {
		return fmt.Errorf("添加任务到调度器失败: %w", err)
	}

	// 保存定时器信息
	timer := &Timer{
		Name:     name,
		Interval: interval,
		Handler:  handler,
	}
	c.timers[name] = timer

	// 获取任务状态以显示下次执行时间
	if taskStatus, err := c.scheduler.GetTaskStatus(taskName); err == nil {
		log.Printf("采集器 %s: 添加定时器 %s，间隔 %v，下次执行时间: %s（整点执行）",
			c.id, name, interval, taskStatus.NextRun.Format("15:04:05"))
	} else {
		log.Printf("采集器 %s: 添加定时器 %s，间隔 %v", c.id, name, interval)
	}
	return nil
}

func (c *BaseCollector) RemoveTimer(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.timers[name]; !exists {
		return fmt.Errorf("定时器 %s 不存在", name)
	}

	taskName := fmt.Sprintf("%s.%s", c.id, name)
	if err := c.scheduler.RemoveTask(taskName); err != nil {
		return fmt.Errorf("从调度器移除任务失败: %w", err)
	}

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

	// 从调度器获取任务状态
	taskStatuses := c.scheduler.ListTasks()
	for name, timer := range c.timers {
		taskName := fmt.Sprintf("%s.%s", c.id, name)
		if taskStatus, exists := taskStatuses[taskName]; exists {
			status.Timers[name] = TimerStatus{
				Name:       timer.Name,
				Interval:   timer.Interval,
				LastRun:    taskStatus.LastRun,
				NextRun:    taskStatus.NextRun,
				RunCount:   taskStatus.RunCount,
				ErrorCount: taskStatus.ErrorCount,
			}
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

// wrapHandler 包装处理函数以更新 metrics
func (c *BaseCollector) wrapHandler(name string, handler TimerHandler) func(context.Context) error {
	return func(ctx context.Context) error {
		start := time.Now()

		// 执行原始 handler
		err := handler(ctx)

		// 更新 metrics
		latency := time.Since(start)
		c.mu.Lock()
		defer c.mu.Unlock()

		if err != nil {
			c.metrics.ErrorsTotal++
			c.metrics.LastError = err
			c.metrics.LastErrorTime = time.Now()
			log.Printf("定时器 %s.%s 执行失败: %v", c.id, name, err)
		} else {
			log.Printf("定时器 %s.%s 执行成功，执行时间: %s，耗时: %v",
				c.id, name, time.Now().Format("15:04:05"), latency)
		}

		// 更新定时器指标
		if timerMetrics, exists := c.metrics.TimerMetrics[name]; exists {
			if err != nil {
				timerMetrics.ErrorCount++
			} else {
				timerMetrics.RunCount++
			}
			timerMetrics.LastRun = time.Now()
			timerMetrics.AvgLatency = (timerMetrics.AvgLatency + latency) / 2
			c.metrics.TimerMetrics[name] = timerMetrics
		} else {
			c.metrics.TimerMetrics[name] = TimerMetrics{
				RunCount:   1,
				ErrorCount: 0,
				LastRun:    time.Now(),
				AvgLatency: latency,
			}
			if err != nil {
				c.metrics.TimerMetrics[name] = TimerMetrics{
					RunCount:   0,
					ErrorCount: 1,
					LastRun:    time.Now(),
					AvgLatency: latency,
				}
			}
		}

		return err
	}
}

// AddCronTimer 添加 Cron 定时器（支持更灵活的调度）
func (c *BaseCollector) AddCronTimer(name string, cronExpr string, handler TimerHandler) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.timers[name]; exists {
		return fmt.Errorf("定时器 %s 已存在", name)
	}

	// 包装 handler
	wrappedHandler := c.wrapHandler(name, handler)

	// 添加到调度器
	taskName := fmt.Sprintf("%s.%s", c.id, name)
	if err := c.scheduler.AddCronTask(taskName, cronExpr, wrappedHandler); err != nil {
		return fmt.Errorf("添加 Cron 任务到调度器失败: %w", err)
	}

	// 保存定时器信息（使用特殊标记表示 cron 类型）
	timer := &Timer{
		Name:     name,
		Interval: -1, // 使用 -1 表示这是 cron 类型
		Handler:  handler,
	}
	c.timers[name] = timer

	// 获取任务状态以显示下次执行时间
	if taskStatus, err := c.scheduler.GetTaskStatus(taskName); err == nil {
		log.Printf("采集器 %s: 添加 Cron 定时器 %s，表达式 %s，下次执行时间: %s",
			c.id, name, cronExpr, taskStatus.NextRun.Format("2006-01-02 15:04:05"))
	} else {
		log.Printf("采集器 %s: 添加 Cron 定时器 %s，表达式 %s", c.id, name, cronExpr)
	}

	return nil
}
