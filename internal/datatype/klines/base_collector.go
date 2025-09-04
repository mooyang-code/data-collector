package klines

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mooyang-code/data-collector/internal/infra/timer"
	"trpc.group/trpc-go/trpc-go/log"
)

// CollectorConfig K线采集器配置
type CollectorConfig struct {
	// 是否启用定时器
	EnableTimer bool `yaml:"enableTimer" json:"enableTimer"`

	// 定时器时区
	Timezone string `yaml:"timezone" json:"timezone"`

	// 最大并发任务数
	MaxConcurrentJobs int `yaml:"maxConcurrentJobs" json:"maxConcurrentJobs"`

	// 任务超时时间
	JobTimeout time.Duration `yaml:"jobTimeout" json:"jobTimeout"`

	// 最大重试次数
	MaxRetries int `yaml:"maxRetries" json:"maxRetries"`
}

// DefaultCollectorConfig 返回默认配置
func DefaultCollectorConfig() *CollectorConfig {
	return &CollectorConfig{
		EnableTimer:       true,
		Timezone:          "UTC",
		MaxConcurrentJobs: 10,
		JobTimeout:        5 * time.Minute,
		MaxRetries:        3,
	}
}

// BaseKlineCollector 基础K线采集器实现（集成定时器功能）
type BaseKlineCollector struct {
	exchange string // 交易所名
	outCh    chan KlineEvent

	started atomic.Bool
	closed  atomic.Bool

	muSub sync.RWMutex
	subs  map[string]struct{} // key = symbol|interval

	// 速率限制（可选），实现方可直接修改
	muRL      sync.RWMutex
	rateLimit *RateLimit

	// 定时器相关
	timer  timer.Timer
	config *CollectorConfig

	// 任务管理
	jobs   map[string]timer.JobID // key = symbol|interval, value = jobID
	jobsMu sync.RWMutex
}

// NewBaseKlineCollector 创建基础K线采集器（包含定时器功能）
func NewBaseKlineCollector(exchange string, buf int, config *CollectorConfig) (*BaseKlineCollector, error) {
	if config == nil {
		config = DefaultCollectorConfig()
	}

	collector := &BaseKlineCollector{
		exchange: exchange,
		outCh:    make(chan KlineEvent, buf),
		subs:     make(map[string]struct{}),
		jobs:     make(map[string]timer.JobID),
		config:   config,
	}

	// 如果启用定时器，创建定时器实例
	if config.EnableTimer {
		timerConfig := &timer.Config{
			Enabled:           true,
			Timezone:          config.Timezone,
			MaxConcurrentJobs: config.MaxConcurrentJobs,
			JobTimeout:        config.JobTimeout,
			EnableRecovery:    false,
			HistoryRetention:  24 * time.Hour,
		}

		timerInstance, err := timer.NewCronTimer(timerConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create timer: %w", err)
		}
		collector.timer = timerInstance
	}

	return collector, nil
}

// Start 启动采集器
func (b *BaseKlineCollector) Start(ctx context.Context) error {
	if b.closed.Load() {
		return ErrClosed
	}
	if !b.started.CompareAndSwap(false, true) {
		return ErrAlreadyStarted
	}

	// 启动定时器（如果配置了）
	if b.timer != nil && b.config.EnableTimer {
		if err := b.timer.Start(ctx); err != nil {
			return fmt.Errorf("failed to start timer: %w", err)
		}
		log.Infof("K线采集器定时器已启动 - exchange: %s", b.exchange)
	}

	return nil
}

// Close 关闭采集器
func (b *BaseKlineCollector) Close() error {
	if b.closed.CompareAndSwap(false, true) {
		// 停止定时器
		if b.timer != nil && b.timer.IsRunning() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := b.timer.Stop(ctx); err != nil {
				log.Errorf("停止定时器失败: %v", err)
			}
		}

		// 清理任务映射
		b.jobsMu.Lock()
		b.jobs = make(map[string]timer.JobID)
		b.jobsMu.Unlock()

		close(b.outCh)
		log.Infof("K线采集器已关闭 - exchange: %s", b.exchange)
	}
	return nil
}

// Stop 停止采集器（兼容接口）
func (b *BaseKlineCollector) Stop(ctx context.Context) error {
	return b.Close()
}

// === 定时器相关方法 ===

// AddTimerJob 添加定时任务（用于定时获取K线数据）
// cronExpr: cron表达式，如 "0 * * * * *" (每分钟), "*/30 * * * * *" (每30秒)
func (b *BaseKlineCollector) AddTimerJob(symbol, interval, cronExpr string, fetchFunc timer.JobFunc) error {
	if b.timer == nil || !b.config.EnableTimer {
		return fmt.Errorf("timer not enabled")
	}

	if !b.timer.IsRunning() {
		return fmt.Errorf("timer not running")
	}

	return b.addTimerJob(symbol, interval, cronExpr, fetchFunc)
}

// RemoveTimerJob 移除定时任务
func (b *BaseKlineCollector) RemoveTimerJob(symbol, interval string) error {
	if b.timer == nil || !b.timer.IsRunning() {
		return nil
	}

	return b.removeTimerJob(symbol, interval)
}

// TriggerManualFetch 手动触发数据获取
func (b *BaseKlineCollector) TriggerManualFetch(symbol, interval string) error {
	if b.timer == nil || !b.timer.IsRunning() {
		return fmt.Errorf("timer not running")
	}

	key := makeSubscriptionKey(symbol, interval)
	b.jobsMu.RLock()
	jobID, exists := b.jobs[key]
	b.jobsMu.RUnlock()

	if !exists {
		return fmt.Errorf("no timer job found for %s %s", symbol, interval)
	}

	return b.timer.TriggerJob(jobID)
}

// GetTimerJobs 获取所有定时任务信息
func (b *BaseKlineCollector) GetTimerJobs() ([]*timer.Job, error) {
	if b.timer == nil || !b.timer.IsRunning() {
		return nil, fmt.Errorf("timer not running")
	}

	return b.timer.ListJobs()
}

// IsTimerEnabled 检查定时器是否启用
func (b *BaseKlineCollector) IsTimerEnabled() bool {
	return b.timer != nil && b.config.EnableTimer
}

// IsTimerRunning 检查定时器是否运行中
func (b *BaseKlineCollector) IsTimerRunning() bool {
	return b.timer != nil && b.timer.IsRunning()
}

// Subscribe 订阅单个 (symbol, interval) - WebSocket订阅
func (b *BaseKlineCollector) Subscribe(symbol, interval string) error {
	if symbol == "" || interval == "" {
		return ErrInvalidParam
	}
	key := symbol + "|" + interval
	b.muSub.Lock()
	b.subs[key] = struct{}{}
	b.muSub.Unlock()
	return nil
}

// Unsubscribe 取消订阅单个 (symbol, interval) - WebSocket取消订阅
func (b *BaseKlineCollector) Unsubscribe(symbol, interval string) error {
	if symbol == "" || interval == "" {
		return ErrInvalidParam
	}
	key := symbol + "|" + interval
	b.muSub.Lock()
	delete(b.subs, key)
	b.muSub.Unlock()
	return nil
}

// Subscriptions 当前订阅快照
func (b *BaseKlineCollector) Subscriptions() []SymbolInterval {
	b.muSub.RLock()
	res := make([]SymbolInterval, 0, len(b.subs))
	for k := range b.subs {
		// k = symbol|interval
		for i := 0; i < len(k); i++ {
			if k[i] == '|' {
				res = append(res, SymbolInterval{
					Symbol:   k[:i],
					Interval: k[i+1:],
				})
				break
			}
		}
	}
	b.muSub.RUnlock()
	return res
}

// Events 实时事件通道
func (b *BaseKlineCollector) Events() <-chan KlineEvent {
	return b.outCh
}

// Emit 供子类推送实时数据
func (b *BaseKlineCollector) Emit(rec *KlineRecord, source string) {
	if b.closed.Load() {
		return
	}
	evt := KlineEvent{
		Record: rec,
		Source: source,
		Ts:     time.Now(),
	}
	select {
	case b.outCh <- evt:
	default:
		// 默认丢弃；需要阻塞或扩展策略可在子类重写 Emit 包装
	}
}

// SetRateLimit 设置或更新速率限制（子类可调用）
func (b *BaseKlineCollector) SetRateLimit(rl *RateLimit) {
	b.muRL.Lock()
	b.rateLimit = rl
	b.muRL.Unlock()
}

// GetRateLimit 获取速率限制信息
func (b *BaseKlineCollector) GetRateLimit() *RateLimit {
	b.muRL.RLock()
	defer b.muRL.RUnlock()
	return b.rateLimit
}

// GetKlines 默认历史查询：不支持
func (b *BaseKlineCollector) GetKlines(ctx context.Context, q KlineQuery) ([]*KlineRecord, error) {
	return nil, ErrNotSupported
}

// IsSubscribed 检查是否已订阅
func (b *BaseKlineCollector) IsSubscribed(symbol, interval string) bool {
	key := symbol + "|" + interval
	b.muSub.RLock()
	_, exists := b.subs[key]
	b.muSub.RUnlock()
	return exists
}

// GetSubscriptionCount 获取订阅数量
func (b *BaseKlineCollector) GetSubscriptionCount() int {
	b.muSub.RLock()
	count := len(b.subs)
	b.muSub.RUnlock()
	return count
}

// GetExchange 获取交易所名称
func (b *BaseKlineCollector) GetExchange() string {
	return b.exchange
}

// IsStarted 检查是否已启动
func (b *BaseKlineCollector) IsStarted() bool {
	return b.started.Load()
}

// IsClosed 检查是否已关闭
func (b *BaseKlineCollector) IsClosed() bool {
	return b.closed.Load()
}

// ClearSubscriptions 清空所有订阅
func (b *BaseKlineCollector) ClearSubscriptions() {
	b.muSub.Lock()
	b.subs = make(map[string]struct{})
	b.muSub.Unlock()
}

// UpdateRateLimit 更新速率限制计数
func (b *BaseKlineCollector) UpdateRateLimit() {
	b.muRL.Lock()
	if b.rateLimit != nil {
		b.rateLimit.LastRequest = time.Now()
		b.rateLimit.RequestCount++
	}
	b.muRL.Unlock()
}





// === 定时器私有方法 ===

// addTimerJob 添加定时任务
func (b *BaseKlineCollector) addTimerJob(symbol, interval, cronExpr string, fetchFunc timer.JobFunc) error {
	key := makeSubscriptionKey(symbol, interval)

	b.jobsMu.Lock()
	defer b.jobsMu.Unlock()

	// 检查是否已存在
	if _, exists := b.jobs[key]; exists {
		return nil // 已存在，不重复添加
	}

	// 创建定时任务
	jobID := timer.JobID(fmt.Sprintf("kline_%s_%s_%s", b.exchange, symbol, interval))
	job := timer.NewJobBuilder().
		WithID(jobID).
		WithName(fmt.Sprintf("K线获取-%s-%s-%s", b.exchange, symbol, interval)).
		WithDescription(fmt.Sprintf("定时获取%s交易所%s交易对%s周期的K线数据", b.exchange, symbol, interval)).
		WithCron(cronExpr).
		WithFunc(fetchFunc).
		WithTimeout(b.config.JobTimeout).
		WithMaxRetries(b.config.MaxRetries).
		Build()

	// 添加到定时器
	if err := b.timer.AddJob(job); err != nil {
		return fmt.Errorf("failed to add timer job: %w", err)
	}

	// 记录任务映射
	b.jobs[key] = job.ID

	log.Debugf("添加定时任务: %s %s - jobID: %s", symbol, interval, job.ID)
	return nil
}

// removeTimerJob 移除定时任务
func (b *BaseKlineCollector) removeTimerJob(symbol, interval string) error {
	key := makeSubscriptionKey(symbol, interval)

	b.jobsMu.Lock()
	defer b.jobsMu.Unlock()

	jobID, exists := b.jobs[key]
	if !exists {
		return nil // 不存在，无需移除
	}

	// 从定时器中移除
	if err := b.timer.RemoveJob(jobID); err != nil {
		return fmt.Errorf("failed to remove timer job: %w", err)
	}

	// 删除映射
	delete(b.jobs, key)

	log.Debugf("移除定时任务: %s %s - jobID: %s", symbol, interval, jobID)
	return nil
}

// makeSubscriptionKey 创建订阅键
func makeSubscriptionKey(symbol, interval string) string {
	return fmt.Sprintf("%s|%s", symbol, interval)
}

// parseSubscriptionKey 解析订阅键
func parseSubscriptionKey(key string) (symbol, interval string) {
	for i := 0; i < len(key); i++ {
		if key[i] == '|' {
			return key[:i], key[i+1:]
		}
	}
	return "", ""
}

// IntervalToCron 将时间间隔转换为cron表达式
func IntervalToCron(interval string) string {
	switch interval {
	case "1s":
		return "* * * * * *"        // 每秒
	case "1m":
		return "0 * * * * *"        // 每分钟
	case "3m":
		return "0 */3 * * * *"      // 每3分钟
	case "5m":
		return "0 */5 * * * *"      // 每5分钟
	case "15m":
		return "0 */15 * * * *"     // 每15分钟
	case "30m":
		return "0 */30 * * * *"     // 每30分钟
	case "1h":
		return "0 0 * * * *"        // 每小时
	case "2h":
		return "0 0 */2 * * *"      // 每2小时
	case "4h":
		return "0 0 */4 * * *"      // 每4小时
	case "6h":
		return "0 0 */6 * * *"      // 每6小时
	case "8h":
		return "0 0 */8 * * *"      // 每8小时
	case "12h":
		return "0 0 */12 * * *"     // 每12小时
	case "1d":
		return "0 0 0 * * *"        // 每天
	case "3d":
		return "0 0 0 */3 * *"      // 每3天
	case "1w":
		return "0 0 0 * * 0"        // 每周
	case "1M":
		return "0 0 0 1 * *"        // 每月
	default:
		return "0 * * * * *"        // 默认每分钟
	}
}


