// Package trigger 触发器实现
package trigger

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mooyang-code/data-collector/model"
	"trpc.group/trpc-go/trpc-go/log"
)

// BaseTrigger 基础触发器实现
type BaseTrigger struct {
	id           string
	triggerType  model.TriggerType
	name         string
	status       model.TriggerStatus
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	startTime    time.Time
	triggerCount int64
	errorCount   int64
	lastTrigger  time.Time
	callback     model.TriggerCallback
}

// NewBaseTrigger 创建基础触发器
func NewBaseTrigger(id string, triggerType types.TriggerType, name string) *BaseTrigger {
	return &BaseTrigger{
		id:          id,
		triggerType: triggerType,
		name:        name,
		status: types.TriggerStatus{
			ID:    id,
			Type:  triggerType,
			Name:  name,
			State: types.TriggerStateUnknown,
		},
		metrics: types.TriggerMetrics{
			ID:   id,
			Type: triggerType,
		},
	}
}

// GetID 获取触发器ID
func (t *BaseTrigger) GetID() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.id
}

// GetType 获取触发器类型
func (t *BaseTrigger) GetType() types.TriggerType {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.triggerType
}

// GetName 获取触发器名称
func (t *BaseTrigger) GetName() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.name
}

// Initialize 初始化触发器
func (t *BaseTrigger) Initialize(config interface{}) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.status.State != types.TriggerStateUnknown {
		return fmt.Errorf("触发器 %s 已经初始化", t.id)
	}

	t.status.State = types.TriggerStateInitialized
	t.status.LastUpdate = time.Now()

	log.Infof("触发器 %s (%s) 初始化完成", t.name, t.triggerType)
	return nil
}

// Start 启动触发器
func (t *BaseTrigger) Start(ctx context.Context, callback types.TriggerCallback) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.status.State == types.TriggerStateRunning {
		return fmt.Errorf("触发器 %s 已经在运行", t.id)
	}

	if callback == nil {
		return fmt.Errorf("触发器回调函数不能为空")
	}

	t.ctx, t.cancel = context.WithCancel(ctx)
	t.callback = callback
	t.status.State = types.TriggerStateStarting
	t.status.LastUpdate = time.Now()
	t.startTime = time.Now()

	// 重置计数器
	atomic.StoreInt64(&t.triggerCount, 0)
	atomic.StoreInt64(&t.errorCount, 0)

	t.status.State = types.TriggerStateRunning
	t.status.StartTime = t.startTime
	t.status.LastUpdate = time.Now()

	log.Infof("触发器 %s (%s) 启动完成", t.name, t.triggerType)
	return nil
}

// Stop 停止触发器
func (t *BaseTrigger) Stop(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.status.State != types.TriggerStateRunning {
		return fmt.Errorf("触发器 %s 未在运行", t.id)
	}

	t.status.State = types.TriggerStateStopping
	t.status.LastUpdate = time.Now()

	// 取消上下文
	if t.cancel != nil {
		t.cancel()
	}

	t.status.State = types.TriggerStateStopped
	t.status.LastUpdate = time.Now()

	log.Infof("触发器 %s (%s) 停止完成", t.name, t.triggerType)
	return nil
}

// IsRunning 检查触发器是否运行中
func (t *BaseTrigger) IsRunning() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.status.State == types.TriggerStateRunning
}

// GetStatus 获取触发器状态
func (t *BaseTrigger) GetStatus() types.TriggerStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()

	status := t.status
	status.TriggerCount = atomic.LoadInt64(&t.triggerCount)
	status.ErrorCount = int(atomic.LoadInt64(&t.errorCount))
	status.LastTrigger = t.lastTrigger
	return status
}

// executeTrigger 执行触发器回调
func (t *BaseTrigger) executeTrigger() {
	if t.callback == nil {
		return
	}

	// 增加触发计数
	atomic.AddInt64(&t.triggerCount, 1)

	// 更新最后触发时间
	t.mu.Lock()
	t.lastTrigger = time.Now()
	t.status.LastTrigger = t.lastTrigger
	t.mu.Unlock()

	// 执行回调
	if err := t.callback(t.ctx); err != nil {
		atomic.AddInt64(&t.errorCount, 1)
		t.mu.Lock()
		t.status.ErrorCount++
		t.status.LastError = err.Error()
		t.mu.Unlock()
		log.Errorf("触发器 %s 执行回调失败: %v", t.id, err)
	}
}

// GetContext 获取上下文
func (t *BaseTrigger) GetContext() context.Context {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.ctx
}

// TimerTrigger 定时触发器
type TimerTrigger struct {
	*BaseTrigger
	interval time.Duration
	ticker   *time.Ticker
}

// TimerTriggerConfig 定时触发器配置
type TimerTriggerConfig struct {
	Interval string `yaml:"interval" json:"interval"` // 触发间隔，如 "1m", "5s", "1h"
}

// NewTimerTrigger 创建定时触发器
func NewTimerTrigger(id, name string, interval time.Duration) *TimerTrigger {
	return &TimerTrigger{
		BaseTrigger: NewBaseTrigger(id, types.TriggerTypeTimer, name),
		interval:    interval,
	}
}

// Initialize 初始化定时触发器
func (t *TimerTrigger) Initialize(config interface{}) error {
	if err := t.BaseTrigger.Initialize(config); err != nil {
		return err
	}

	// 解析配置
	if cfg, ok := config.(*TimerTriggerConfig); ok && cfg != nil {
		if cfg.Interval != "" {
			if interval, err := time.ParseDuration(cfg.Interval); err == nil {
				t.interval = interval
			} else {
				return fmt.Errorf("无效的触发间隔: %s", cfg.Interval)
			}
		}
	}

	if t.interval <= 0 {
		t.interval = 1 * time.Minute // 默认1分钟
	}

	log.Infof("定时触发器 %s 初始化完成，间隔: %v", t.GetName(), t.interval)
	return nil
}

// Start 启动定时触发器
func (t *TimerTrigger) Start(ctx context.Context, callback types.TriggerCallback) error {
	if err := t.BaseTrigger.Start(ctx, callback); err != nil {
		return err
	}

	// 启动定时器
	t.ticker = time.NewTicker(t.interval)
	go t.timerLoop()

	log.Infof("定时触发器 %s 启动，间隔: %v", t.GetName(), t.interval)
	return nil
}

// Stop 停止定时触发器
func (t *TimerTrigger) Stop(ctx context.Context) error {
	if t.ticker != nil {
		t.ticker.Stop()
		t.ticker = nil
	}

	return t.BaseTrigger.Stop(ctx)
}

// timerLoop 定时器循环
func (t *TimerTrigger) timerLoop() {
	for {
		select {
		case <-t.GetContext().Done():
			log.Infof("定时触发器 %s 循环退出", t.GetName())
			return
		case <-t.ticker.C:
			t.executeTrigger()
		}
	}
}

// GetInterval 获取触发间隔
func (t *TimerTrigger) GetInterval() time.Duration {
	return t.interval
}
