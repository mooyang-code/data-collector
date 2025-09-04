// Package symbols 币安交易对采集器实现
package symbols

import (
	"context"
	"fmt"
	"time"

	"github.com/mooyang-code/data-collector/internal/datatype/symbols"
	"trpc.group/trpc-go/trpc-go/log"
)

// BinanceSymbolCollector 币安交易对采集器
type BinanceSymbolCollector struct {
	*symbols.BaseSymbolsCollector
	adapter *BinanceSymbolAdapter
	config  *BinanceSymbolConfig

	// 定时刷新控制
	ticker   *time.Ticker
	stopCh   chan struct{}
	running  bool
}

// NewBinanceSymbolCollector 创建币安交易对采集器
func NewBinanceSymbolCollector(config *BinanceSymbolConfig) (*BinanceSymbolCollector, error) {
	if config == nil {
		config = DefaultBinanceSymbolConfig()
	}
	
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}
	
	// 创建适配器
	adapter := NewBinanceSymbolAdapter(config.BaseURL, config.Exchange)
	
	// 创建基础采集器
	baseCollector := symbols.NewBaseSymbolsCollector(config.Exchange, config.BufferSize)

	collector := &BinanceSymbolCollector{
		BaseSymbolsCollector: baseCollector,
		adapter:              adapter,
		config:               config,
		stopCh:               make(chan struct{}),
	}
	
	// 注册适配器
	symbols.RegisterSymbolAdapter(config.Exchange, adapter)
	
	return collector, nil
}

// Initialize 初始化采集器
func (c *BinanceSymbolCollector) Initialize(ctx context.Context) error {
	log.Infof("初始化币安交易对采集器...")

	// 启动定时刷新
	if c.config.EnableAutoRefresh {
		c.ticker = time.NewTicker(c.config.RefreshInterval)
		log.Infof("启用定时刷新 - exchange: %s, interval: %s", c.config.Exchange, c.config.RefreshInterval)
	}

	log.Infof("币安交易对采集器初始化完成")
	return nil
}

// refreshLoop 刷新循环
func (c *BinanceSymbolCollector) refreshLoop(ctx context.Context) {
	defer log.Infof("交易对刷新循环退出 - exchange: %s", c.config.Exchange)

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-c.ticker.C:
			c.doRefresh(ctx)
		}
	}
}

// doRefresh 执行刷新
func (c *BinanceSymbolCollector) doRefresh(ctx context.Context) {
	log.Debugf("开始刷新交易对信息 - exchange: %s", c.config.Exchange)

	// 使用适配器获取所有交易对
	allSymbols, err := c.adapter.FetchAll(ctx)
	if err != nil {
		log.Errorf("获取交易对信息失败 - exchange: %s, error: %v", c.config.Exchange, err)
		return
	}

	// 应用过滤器
	filteredSymbols := c.applyFilters(allSymbols)

	// 应用全量快照
	c.ApplyFullSnapshot(filteredSymbols, "binance_timer")

	log.Debugf("完成刷新交易对信息 - exchange: %s, total: %d, filtered: %d",
		c.config.Exchange, len(allSymbols), len(filteredSymbols))
}

// applyFilters 应用过滤器
func (c *BinanceSymbolCollector) applyFilters(allSymbols []*symbols.SymbolMeta) []*symbols.SymbolMeta {
	if !c.config.EnableFiltering {
		return allSymbols
	}
	
	filtered := make([]*symbols.SymbolMeta, 0, len(allSymbols))
	for _, symbol := range allSymbols {
		if c.config.IsSymbolAllowed(symbol) {
			filtered = append(filtered, symbol)
		}
	}
	
	return filtered
}

// StartCollection 启动采集
func (c *BinanceSymbolCollector) StartCollection(ctx context.Context) error {
	log.Infof("启动币安交易对采集...")

	// 启动基础采集器
	if err := c.Start(ctx); err != nil {
		return fmt.Errorf("启动基础采集器失败: %w", err)
	}

	// 立即执行一次刷新
	if err := c.RefreshNow(ctx); err != nil {
		log.Warnf("初始刷新失败: %v", err)
	}

	// 启动定时刷新循环
	if c.config.EnableAutoRefresh && c.ticker != nil {
		c.running = true
		go c.refreshLoop(ctx)
	}

	log.Infof("币安交易对采集启动完成")
	return nil
}

// RefreshNow 立即刷新
func (c *BinanceSymbolCollector) RefreshNow(ctx context.Context) error {
	log.Infof("立即刷新交易对信息 - exchange: %s", c.config.Exchange)
	c.doRefresh(ctx)
	return nil
}

// StopCollection 停止采集
func (c *BinanceSymbolCollector) StopCollection(ctx context.Context) error {
	log.Infof("停止币安交易对采集...")

	// 停止刷新循环
	if c.running {
		close(c.stopCh)
		c.running = false
	}

	// 停止定时器
	if c.ticker != nil {
		c.ticker.Stop()
	}

	// 停止基础采集器
	if err := c.Close(); err != nil {
		return fmt.Errorf("停止基础采集器失败: %w", err)
	}

	log.Infof("币安交易对采集停止完成")
	return nil
}

// GetConfig 获取配置
func (c *BinanceSymbolCollector) GetConfig() *BinanceSymbolConfig {
	return c.config
}

// GetAdapter 获取适配器
func (c *BinanceSymbolCollector) GetAdapter() *BinanceSymbolAdapter {
	return c.adapter
}

// UpdateConfig 更新配置
func (c *BinanceSymbolCollector) UpdateConfig(config *BinanceSymbolConfig) error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}
	
	c.config = config
	
	// 更新适配器配置
	c.adapter = NewBinanceSymbolAdapter(config.BaseURL, config.Exchange)
	
	log.Infof("币安交易对采集器配置更新完成")
	return nil
}

// GetSymbolByName 根据名称获取交易对
func (c *BinanceSymbolCollector) GetSymbolByName(symbolName string) (*symbols.SymbolMeta, error) {
	return c.Symbol(symbolName)
}

// GetActiveSymbols 获取活跃交易对
func (c *BinanceSymbolCollector) GetActiveSymbols() []*symbols.SymbolMeta {
	allSymbols := c.Symbols()
	activeSymbols := make([]*symbols.SymbolMeta, 0, len(allSymbols))
	
	for _, symbol := range allSymbols {
		if symbol.IsActive() {
			activeSymbols = append(activeSymbols, symbol)
		}
	}
	
	return activeSymbols
}

// GetSymbolsByType 根据类型获取交易对
func (c *BinanceSymbolCollector) GetSymbolsByType(symbolType symbols.SymbolType) []*symbols.SymbolMeta {
	allSymbols := c.Symbols()
	filteredSymbols := make([]*symbols.SymbolMeta, 0, len(allSymbols))
	
	for _, symbol := range allSymbols {
		if symbol.Type == symbolType {
			filteredSymbols = append(filteredSymbols, symbol)
		}
	}
	
	return filteredSymbols
}

// GetSymbolsByQuoteAsset 根据计价资产获取交易对
func (c *BinanceSymbolCollector) GetSymbolsByQuoteAsset(quoteAsset string) []*symbols.SymbolMeta {
	allSymbols := c.Symbols()
	filteredSymbols := make([]*symbols.SymbolMeta, 0, len(allSymbols))
	
	for _, symbol := range allSymbols {
		if symbol.QuoteAsset == quoteAsset {
			filteredSymbols = append(filteredSymbols, symbol)
		}
	}
	
	return filteredSymbols
}

// GetStats 获取统计信息
func (c *BinanceSymbolCollector) GetStats() *symbols.SymbolStats {
	allSymbols := c.Symbols()
	
	stats := &symbols.SymbolStats{
		Exchange:    c.config.Exchange,
		TotalCount:  len(allSymbols),
		LastUpdate:  time.Now(),
	}
	
	for _, symbol := range allSymbols {
		if symbol.IsActive() {
			stats.ActiveCount++
		}
		
		switch symbol.Type {
		case symbols.TypeSpot:
			stats.SpotCount++
		case symbols.TypePerp, symbols.TypeDelivery:
			stats.ContractCount++
		case symbols.TypeOption:
			stats.OptionCount++
		}
	}
	
	return stats
}
