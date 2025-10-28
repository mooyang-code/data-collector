package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/mooyang-code/data-collector/internal/metrics"
	"github.com/mooyang-code/data-collector/pkg/errors"
)

// manager 采集器管理器实现
type manager struct {
	registry   Registry
	collectors map[string]Collector
	metrics    metrics.Collector
	mu         sync.RWMutex
}

// NewManager 创建新的采集器管理器
func NewManager(metricsCollector metrics.Collector) Manager {
	return &manager{
		registry:   GetRegistry(),
		collectors: make(map[string]Collector),
		metrics:    metricsCollector,
	}
}

func (m *manager) Create(ctx context.Context, name string, config json.RawMessage) (Collector, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否已存在
	if _, exists := m.collectors[name]; exists {
		return nil, errors.NewAppError(errors.ErrCodeCollectorError,
			fmt.Sprintf("collector %s already exists", name), nil)
	}

	// 从注册中心创建采集器
	collector, err := m.registry.Create(name)
	if err != nil {
		return nil, err
	}

	// 初始化采集器
	if err := collector.Init(config); err != nil {
		return nil, errors.NewAppError(errors.ErrCodeCollectorError,
			fmt.Sprintf("failed to initialize collector %s", name), err)
	}

	// 保存到管理器
	m.collectors[name] = collector
	return collector, nil
}

func (m *manager) Get(name string) (Collector, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	collector, exists := m.collectors[name]
	if !exists {
		return nil, errors.NewAppError(errors.ErrCodeCollectorError,
			fmt.Sprintf("collector %s not found", name), errors.ErrCollectorNotFound)
	}

	return collector, nil
}

func (m *manager) GetCollector(exchange string) (Collector, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 直接通过交易所名称查找
	if collector, exists := m.collectors[exchange]; exists {
		return collector, nil
	}

	// 如果没有找到，尝试创建一个默认配置的采集器
	m.mu.RUnlock()
	m.mu.Lock()
	defer func() {
		m.mu.Unlock()
		m.mu.RLock()
	}()

	// 双重检查
	if collector, exists := m.collectors[exchange]; exists {
		return collector, nil
	}

	// 创建采集器（使用空配置）
	collector, err := m.registry.Create(exchange)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrCodeCollectorError,
			fmt.Sprintf("failed to create collector for exchange %s", exchange), err)
	}

	// 使用默认配置初始化
	if err := collector.Init(nil); err != nil {
		return nil, errors.NewAppError(errors.ErrCodeCollectorError,
			fmt.Sprintf("failed to initialize collector for exchange %s", exchange), err)
	}

	// 保存到管理器
	m.collectors[exchange] = collector

	return collector, nil
}

func (m *manager) Start(ctx context.Context, name string) error {
	collector, err := m.Get(name)
	if err != nil {
		return err
	}

	if err := collector.Start(ctx); err != nil {
		return errors.NewAppError(errors.ErrCodeCollectorError,
			fmt.Sprintf("failed to start collector %s", name), err)
	}

	return nil
}

func (m *manager) Stop(ctx context.Context, name string) error {
	collector, err := m.Get(name)
	if err != nil {
		return err
	}

	if err := collector.Stop(ctx); err != nil {
		return errors.NewAppError(errors.ErrCodeCollectorError,
			fmt.Sprintf("failed to stop collector %s", name), err)
	}

	return nil
}

func (m *manager) Remove(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	collector, exists := m.collectors[name]
	if !exists {
		return errors.NewAppError(errors.ErrCodeCollectorError,
			fmt.Sprintf("collector %s not found", name), errors.ErrCollectorNotFound)
	}

	// 先停止采集器
	if err := collector.Stop(ctx); err != nil {
	}

	// 从管理器中移除
	delete(m.collectors, name)

	return nil
}

func (m *manager) ListStatus() []Status {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statuses := make([]Status, 0, len(m.collectors))
	for _, collector := range m.collectors {
		statuses = append(statuses, collector.Status())
	}

	return statuses
}

func (m *manager) HealthCheck(ctx context.Context) map[string]error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make(map[string]error)
	for name, collector := range m.collectors {
		results[name] = collector.HealthCheck(ctx)
	}

	return results
}
