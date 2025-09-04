package symbols

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// BaseSymbolsCollector 基础交易对采集器实现
type BaseSymbolsCollector struct {
	exchange string
	outCh    chan SymbolEvent

	started atomic.Bool
	closed  atomic.Bool

	mu      sync.RWMutex
	symbols map[string]*SymbolMeta

	muRL      sync.RWMutex
	rateLimit *RateLimit
}

// NewBaseSymbolsCollector 创建基础交易对采集器
func NewBaseSymbolsCollector(exchange string, buf int) *BaseSymbolsCollector {
	return &BaseSymbolsCollector{
		exchange: exchange,
		outCh:    make(chan SymbolEvent, buf),
		symbols:  make(map[string]*SymbolMeta),
	}
}

// Start 启动采集器
func (b *BaseSymbolsCollector) Start(ctx context.Context) error {
	if b.closed.Load() {
		return ErrClosed
	}
	if !b.started.CompareAndSwap(false, true) {
		return ErrAlreadyStarted
	}
	return nil
}

// Close 关闭采集器
func (b *BaseSymbolsCollector) Close() error {
	if b.closed.CompareAndSwap(false, true) {
		close(b.outCh)
	}
	return nil
}

// Events 事件通道
func (b *BaseSymbolsCollector) Events() <-chan SymbolEvent {
	return b.outCh
}

// GetRateLimit 获取速率限制信息
func (b *BaseSymbolsCollector) GetRateLimit() *RateLimit {
	b.muRL.RLock()
	rl := b.rateLimit
	b.muRL.RUnlock()
	return rl
}

// SetRateLimit 设置速率限制
func (b *BaseSymbolsCollector) SetRateLimit(rl *RateLimit) {
	b.muRL.Lock()
	b.rateLimit = rl
	b.muRL.Unlock()
}

// Symbols 获取所有交易对
func (b *BaseSymbolsCollector) Symbols() []*SymbolMeta {
	b.mu.RLock()
	res := make([]*SymbolMeta, 0, len(b.symbols))
	for _, sm := range b.symbols {
		res = append(res, sm)
	}
	b.mu.RUnlock()
	return res
}

// Symbol 获取单个交易对
func (b *BaseSymbolsCollector) Symbol(symbol string) (*SymbolMeta, error) {
	b.mu.RLock()
	sm, ok := b.symbols[symbol]
	b.mu.RUnlock()
	if !ok {
		return nil, ErrNotFound
	}
	return sm, nil
}

// Refresh 默认刷新实现：不支持
func (b *BaseSymbolsCollector) Refresh(ctx context.Context) error {
	return ErrNotSupported
}

// AddOrUpdateSymbol 增量添加/更新交易对
func (b *BaseSymbolsCollector) AddOrUpdateSymbol(meta *SymbolMeta, source string) {
	if meta == nil || meta.Symbol == "" {
		return
	}
	if meta.Exchange == "" {
		meta.Exchange = b.exchange
	}
	
	var evType SymbolEventType
	b.mu.Lock()
	old, exists := b.symbols[meta.Symbol]
	if exists {
		b.symbols[meta.Symbol] = meta
		if !equalSymbolShallow(old, meta) {
			evType = SymbolUpdated
		} else {
			b.mu.Unlock()
			return // 没有变化，不发送事件
		}
	} else {
		b.symbols[meta.Symbol] = meta
		evType = SymbolAdded
	}
	b.mu.Unlock()

	b.emit(SymbolEvent{
		Type:   evType,
		Symbol: meta,
		Ts:     time.Now(),
		Source: source,
	})
}

// RemoveSymbol 删除交易对
func (b *BaseSymbolsCollector) RemoveSymbol(symbol string, source string) {
	if symbol == "" {
		return
	}
	
	var old *SymbolMeta
	b.mu.Lock()
	if v, ok := b.symbols[symbol]; ok {
		old = v
		delete(b.symbols, symbol)
	}
	b.mu.Unlock()
	
	if old == nil {
		return // 交易对不存在
	}
	
	b.emit(SymbolEvent{
		Type:   SymbolRemoved,
		Symbol: old,
		Ts:     time.Now(),
		Source: source,
	})
}

// ApplyFullSnapshot 全量替换快照 + diff 事件 + SnapshotEnd
func (b *BaseSymbolsCollector) ApplyFullSnapshot(list []*SymbolMeta, source string) {
	now := time.Now()
	newMap := make(map[string]*SymbolMeta, len(list))
	
	// 构建新的映射
	for _, meta := range list {
		if meta == nil || meta.Symbol == "" {
			continue
		}
		if meta.Exchange == "" {
			meta.Exchange = b.exchange
		}
		newMap[meta.Symbol] = meta
	}

	b.mu.Lock()
	oldMap := b.symbols
	b.symbols = newMap
	b.mu.Unlock()

	// 发送删除事件
	for sym, oldMeta := range oldMap {
		if _, ok := newMap[sym]; !ok {
			b.emit(SymbolEvent{
				Type:   SymbolRemoved,
				Symbol: oldMeta,
				Ts:     now,
				Source: source,
			})
		}
	}
	
	// 发送添加/更新事件
	for sym, newMeta := range newMap {
		if oldMeta, ok := oldMap[sym]; ok {
			if !equalSymbolShallow(oldMeta, newMeta) {
				b.emit(SymbolEvent{
					Type:   SymbolUpdated,
					Symbol: newMeta,
					Ts:     now,
					Source: source,
				})
			}
		} else {
			b.emit(SymbolEvent{
				Type:   SymbolAdded,
				Symbol: newMeta,
				Ts:     now,
				Source: source,
			})
		}
	}

	// 发送快照结束事件
	b.emit(SymbolEvent{
		Type:   SnapshotEnd,
		Ts:     time.Now(),
		Source: source,
	})
}

// emit 发送事件
func (b *BaseSymbolsCollector) emit(evt SymbolEvent) {
	if b.closed.Load() {
		return
	}
	select {
	case b.outCh <- evt:
	default:
		// 丢弃事件，可扩展为阻塞或统计
	}
}

// equalSymbolShallow 浅比较交易对是否相等
func equalSymbolShallow(a, b *SymbolMeta) bool {
	if a == nil || b == nil {
		return false
	}
	return a.Status == b.Status &&
		a.PricePrecision == b.PricePrecision &&
		a.QuantityPrecision == b.QuantityPrecision &&
		a.MinQty == b.MinQty &&
		a.MaxQty == b.MaxQty &&
		a.MinNotional == b.MinNotional &&
		a.TickSize == b.TickSize &&
		a.StepSize == b.StepSize &&
		a.Multiplier == b.Multiplier
}

// GetExchange 获取交易所名称
func (b *BaseSymbolsCollector) GetExchange() string {
	return b.exchange
}

// IsStarted 检查是否已启动
func (b *BaseSymbolsCollector) IsStarted() bool {
	return b.started.Load()
}

// IsClosed 检查是否已关闭
func (b *BaseSymbolsCollector) IsClosed() bool {
	return b.closed.Load()
}

// GetSymbolCount 获取交易对数量
func (b *BaseSymbolsCollector) GetSymbolCount() int {
	b.mu.RLock()
	count := len(b.symbols)
	b.mu.RUnlock()
	return count
}

// GetActiveSymbolCount 获取活跃交易对数量
func (b *BaseSymbolsCollector) GetActiveSymbolCount() int {
	b.mu.RLock()
	count := 0
	for _, symbol := range b.symbols {
		if symbol.IsActive() {
			count++
		}
	}
	b.mu.RUnlock()
	return count
}

// GetSymbolsByType 按类型获取交易对
func (b *BaseSymbolsCollector) GetSymbolsByType(symbolType SymbolType) []*SymbolMeta {
	b.mu.RLock()
	var result []*SymbolMeta
	for _, symbol := range b.symbols {
		if symbol.Type == symbolType {
			result = append(result, symbol)
		}
	}
	b.mu.RUnlock()
	return result
}

// GetSymbolsByStatus 按状态获取交易对
func (b *BaseSymbolsCollector) GetSymbolsByStatus(status SymbolStatus) []*SymbolMeta {
	b.mu.RLock()
	var result []*SymbolMeta
	for _, symbol := range b.symbols {
		if symbol.Status == status {
			result = append(result, symbol)
		}
	}
	b.mu.RUnlock()
	return result
}

// FilterSymbols 过滤交易对
func (b *BaseSymbolsCollector) FilterSymbols(filter *SymbolFilter) []*SymbolMeta {
	if filter == nil {
		return b.Symbols()
	}
	
	b.mu.RLock()
	var result []*SymbolMeta
	count := 0
	for _, symbol := range b.symbols {
		if filter.Match(symbol) {
			result = append(result, symbol)
			count++
			if filter.Limit > 0 && count >= filter.Limit {
				break
			}
		}
	}
	b.mu.RUnlock()
	return result
}

// ClearSymbols 清空所有交易对
func (b *BaseSymbolsCollector) ClearSymbols() {
	b.mu.Lock()
	b.symbols = make(map[string]*SymbolMeta)
	b.mu.Unlock()
}

// UpdateRateLimit 更新速率限制计数
func (b *BaseSymbolsCollector) UpdateRateLimit() {
	b.muRL.Lock()
	if b.rateLimit != nil {
		b.rateLimit.LastRequest = time.Now()
		b.rateLimit.RequestCount++
	}
	b.muRL.Unlock()
}
