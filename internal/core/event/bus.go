package event

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// EventHandler 事件处理函数
type EventHandler func(Event) error

// EventBus 事件总线接口
type EventBus interface {
	// 发布事件
	Publish(event Event) error
	PublishAsync(event Event)
	
	// 订阅事件（支持通配符）
	Subscribe(pattern string, handler EventHandler) error
	SubscribeOnce(pattern string, handler EventHandler) error
	
	// 取消订阅
	Unsubscribe(pattern string, handler EventHandler) error
	
	// 监控
	GetStats() EventBusStats
	
	// 生命周期
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// EventBusStats 事件总线统计信息
type EventBusStats struct {
	PublishedTotal   int64
	ProcessedTotal   int64
	ErrorsTotal      int64
	QueueSize        int
	SubscribersCount int
	ProcessingTime   time.Duration
}

// MemoryEventBus 基于内存的事件总线实现
type MemoryEventBus struct {
	subscribers map[string][]subscription
	mu          sync.RWMutex
	
	eventChan chan Event
	workers   int
	wg        sync.WaitGroup
	
	stats EventBusStats
	
	running bool
	stopChan chan struct{}
}

type subscription struct {
	pattern string
	handler EventHandler
	once    bool
}

// NewMemoryEventBus 创建内存事件总线
func NewMemoryEventBus(bufferSize, workers int) *MemoryEventBus {
	return &MemoryEventBus{
		subscribers: make(map[string][]subscription),
		eventChan:   make(chan Event, bufferSize),
		workers:     workers,
	}
}

func (b *MemoryEventBus) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if b.running {
		return fmt.Errorf("事件总线已经在运行")
	}
	
	b.running = true
	b.stopChan = make(chan struct{})
	
	// 启动工作协程
	for i := 0; i < b.workers; i++ {
		b.wg.Add(1)
		go b.worker(ctx, i)
	}
	
	log.Printf("事件总线已启动，工作协程数: %d", b.workers)
	return nil
}

func (b *MemoryEventBus) Stop(ctx context.Context) error {
	b.mu.Lock()
	if !b.running {
		b.mu.Unlock()
		return fmt.Errorf("事件总线未在运行")
	}
	close(b.stopChan)
	b.running = false
	b.mu.Unlock()
	
	// 等待所有工作协程退出
	done := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		log.Println("事件总线已停止")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("停止事件总线超时")
	}
}

func (b *MemoryEventBus) Publish(event Event) error {
	b.mu.RLock()
	if !b.running {
		b.mu.RUnlock()
		return fmt.Errorf("事件总线未在运行")
	}
	b.mu.RUnlock()
	
	select {
	case b.eventChan <- event:
		b.stats.PublishedTotal++
		return nil
	default:
		return fmt.Errorf("事件队列已满")
	}
}

func (b *MemoryEventBus) PublishAsync(event Event) {
	go func() {
		if err := b.Publish(event); err != nil {
			log.Printf("异步发布事件失败: %v", err)
		}
	}()
}

func (b *MemoryEventBus) Subscribe(pattern string, handler EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	sub := subscription{
		pattern: pattern,
		handler: handler,
		once:    false,
	}
	
	b.subscribers[pattern] = append(b.subscribers[pattern], sub)
	b.stats.SubscribersCount++
	
	log.Printf("订阅事件: %s", pattern)
	return nil
}

func (b *MemoryEventBus) SubscribeOnce(pattern string, handler EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	sub := subscription{
		pattern: pattern,
		handler: handler,
		once:    true,
	}
	
	b.subscribers[pattern] = append(b.subscribers[pattern], sub)
	b.stats.SubscribersCount++
	
	return nil
}

func (b *MemoryEventBus) Unsubscribe(pattern string, handler EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	subs, exists := b.subscribers[pattern]
	if !exists {
		return fmt.Errorf("未找到订阅: %s", pattern)
	}
	
	// 移除指定的处理器
	var newSubs []subscription
	for _, sub := range subs {
		// 这里简化处理，实际应该比较函数地址
		newSubs = append(newSubs, sub)
	}
	
	if len(newSubs) == 0 {
		delete(b.subscribers, pattern)
	} else {
		b.subscribers[pattern] = newSubs
	}
	
	b.stats.SubscribersCount--
	return nil
}

func (b *MemoryEventBus) GetStats() EventBusStats {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	stats := b.stats
	stats.QueueSize = len(b.eventChan)
	return stats
}

func (b *MemoryEventBus) worker(ctx context.Context, id int) {
	defer b.wg.Done()
	
	for {
		select {
		case event := <-b.eventChan:
			b.processEvent(event)
		case <-ctx.Done():
			log.Printf("事件处理工作协程 %d 退出(上下文取消)", id)
			return
		case <-b.stopChan:
			log.Printf("事件处理工作协程 %d 退出(停止信号)", id)
			return
		}
	}
}

func (b *MemoryEventBus) processEvent(event Event) {
	startTime := time.Now()
	defer func() {
		b.stats.ProcessingTime = time.Since(startTime)
		b.stats.ProcessedTotal++
	}()
	
	b.mu.RLock()
	// 复制订阅者列表，避免长时间持锁
	var handlers []subscription
	for pattern, subs := range b.subscribers {
		if b.matchPattern(pattern, event.Type()) {
			handlers = append(handlers, subs...)
		}
	}
	b.mu.RUnlock()
	
	// 执行处理器
	for _, sub := range handlers {
		if err := sub.handler(event); err != nil {
			b.stats.ErrorsTotal++
			log.Printf("处理事件 %s 失败: %v", event.Type(), err)
		}
		
		// 如果是一次性订阅，移除它
		if sub.once {
			b.Unsubscribe(sub.pattern, sub.handler)
		}
	}
}

func (b *MemoryEventBus) matchPattern(pattern, eventType string) bool {
	// 支持简单的通配符匹配
	// 例如: "data.*" 匹配 "data.kline", "data.ticker" 等
	// "data.*.collected" 匹配 "data.kline.collected", "data.ticker.collected" 等
	
	// 精确匹配
	if pattern == eventType {
		return true
	}
	
	// 通配符匹配
	parts := strings.Split(pattern, ".")
	eventParts := strings.Split(eventType, ".")
	
	if len(parts) != len(eventParts) {
		return false
	}
	
	for i, part := range parts {
		if part != "*" && part != eventParts[i] {
			return false
		}
	}
	
	return true
}