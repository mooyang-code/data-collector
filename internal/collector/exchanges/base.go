package exchanges

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/mooyang-code/data-collector/internal/collector"
	"github.com/mooyang-code/data-collector/pkg/model"
)

// BaseCollector 基础采集器，提供通用功能
type BaseCollector struct {
	name      string
	colType   model.CollectorType
	state     collector.State
	lastError string
	config    map[string]interface{}
	mu        sync.RWMutex
}

// NewBaseCollector 创建基础采集器
func NewBaseCollector(name string, colType model.CollectorType) *BaseCollector {
	return &BaseCollector{
		name:    name,
		colType: colType,
		state:   collector.StateUninitialized,
		config:  make(map[string]interface{}),
	}
}

func (b *BaseCollector) Name() string {
	return b.name
}

func (b *BaseCollector) Type() model.CollectorType {
	return b.colType
}

func (b *BaseCollector) Init(config json.RawMessage) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if config != nil {
		if err := json.Unmarshal(config, &b.config); err != nil {
			b.state = collector.StateError
			b.lastError = fmt.Sprintf("failed to parse config: %v", err)
			return err
		}
	}
	
	b.state = collector.StateInitialized
	b.lastError = ""
	return nil
}

func (b *BaseCollector) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if b.state != collector.StateInitialized && b.state != collector.StateStopped {
		return fmt.Errorf("collector not initialized")
	}
	
	b.state = collector.StateRunning
	b.lastError = ""
	return nil
}

func (b *BaseCollector) Stop(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.state = collector.StateStopped
	return nil
}

func (b *BaseCollector) HealthCheck(ctx context.Context) error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	if b.state == collector.StateError {
		return fmt.Errorf("collector in error state: %s", b.lastError)
	}
	
	return nil
}

func (b *BaseCollector) Status() collector.Status {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	return collector.Status{
		Name:      b.name,
		Type:      b.colType,
		State:     b.state,
		LastError: b.lastError,
		Metadata: map[string]interface{}{
			"initialized_at": time.Now(),
		},
	}
}

// 受保护的方法供子类使用
func (b *BaseCollector) SetError(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.state = collector.StateError
	b.lastError = err.Error()
}

func (b *BaseCollector) GetConfig(key string) (interface{}, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	value, exists := b.config[key]
	return value, exists
}

func (b *BaseCollector) GetState() collector.State {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	return b.state
}