package app

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type BaseApp struct {
	id         string
	name       string
	sourceType SourceType
	
	collectors map[string]Collector
	mu         sync.RWMutex
	
	status     AppStatus
	metrics    AppMetrics
	
	eventBus   EventBus
}

// EventBus 接口（临时，后续移到 event 包）
type EventBus interface {
	Publish(event Event) error
	Subscribe(pattern string, handler func(Event) error) error
}

func NewBaseApp(id, name string, sourceType SourceType) *BaseApp {
	return &BaseApp{
		id:         id,
		name:       name,
		sourceType: sourceType,
		collectors: make(map[string]Collector),
		status:     AppStatusInitialized,
		metrics: AppMetrics{
			StartTime: time.Now(),
		},
	}
}

func (a *BaseApp) ID() string {
	return a.id
}

func (a *BaseApp) Type() SourceType {
	return a.sourceType
}

func (a *BaseApp) Name() string {
	return a.name
}

func (a *BaseApp) Initialize(ctx context.Context) error {
	// 初始化所有采集器
	for _, collector := range a.collectors {
		if err := collector.Initialize(ctx); err != nil {
			return fmt.Errorf("初始化采集器 %s 失败: %w", collector.ID(), err)
		}
	}
	
	return nil
}

func (a *BaseApp) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	if a.status == AppStatusRunning {
		return fmt.Errorf("App %s 已经在运行", a.id)
	}
	
	// 启动所有采集器
	for _, collector := range a.collectors {
		if err := collector.Start(ctx); err != nil {
			return fmt.Errorf("启动采集器 %s 失败: %w", collector.ID(), err)
		}
		a.metrics.CollectorsActive++
	}
	
	a.status = AppStatusRunning
	return nil
}

func (a *BaseApp) Stop(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	if a.status != AppStatusRunning {
		return fmt.Errorf("App %s 未在运行", a.id)
	}
	
	// 停止所有采集器
	var errs []error
	for _, collector := range a.collectors {
		if err := collector.Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("停止采集器 %s 失败: %w", collector.ID(), err))
		}
		a.metrics.CollectorsActive--
	}
	
	a.status = AppStatusStopped
	
	if len(errs) > 0 {
		return fmt.Errorf("停止App时发生错误: %v", errs)
	}
	
	return nil
}

func (a *BaseApp) RegisterCollector(collector Collector) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	if _, exists := a.collectors[collector.ID()]; exists {
		return fmt.Errorf("采集器 %s 已存在", collector.ID())
	}
	
	a.collectors[collector.ID()] = collector
	a.metrics.CollectorsTotal++
	
	return nil
}

func (a *BaseApp) GetCollector(id string) (Collector, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	collector, exists := a.collectors[id]
	if !exists {
		return nil, fmt.Errorf("采集器 %s 不存在", id)
	}
	
	return collector, nil
}

func (a *BaseApp) ListCollectors() []Collector {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	collectors := make([]Collector, 0, len(a.collectors))
	for _, c := range a.collectors {
		collectors = append(collectors, c)
	}
	
	return collectors
}

func (a *BaseApp) OnEvent(event Event) error {
	a.metrics.EventsProcessed++
	// 基础实现，具体App可以覆盖此方法
	return nil
}

func (a *BaseApp) HealthCheck() error {
	if a.status != AppStatusRunning {
		return fmt.Errorf("App状态异常: %s", a.status)
	}
	
	// 检查所有采集器
	for id, c := range a.collectors {
		// 这里可以添加采集器的健康检查逻辑
		if c == nil {
			return fmt.Errorf("采集器 %s 为空", id)
		}
	}
	
	return nil
}

func (a *BaseApp) GetMetrics() AppMetrics {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	return a.metrics
}

func (a *BaseApp) SetEventBus(eventBus EventBus) {
	a.eventBus = eventBus
}

func (a *BaseApp) GetEventBus() EventBus {
	return a.eventBus
}