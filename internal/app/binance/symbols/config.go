// Package symbols 币安交易对配置
package symbols

import (
	"fmt"
	"time"

	"github.com/mooyang-code/data-collector/internal/datatype/symbols"
)

// BinanceSymbolConfig 币安交易对采集配置
type BinanceSymbolConfig struct {
	// 基础配置
	Exchange string `yaml:"exchange" json:"exchange"` // 交易所名称
	BaseURL  string `yaml:"baseUrl" json:"baseUrl"`   // API基础URL
	
	// 采集配置
	RefreshInterval   time.Duration `yaml:"refreshInterval" json:"refreshInterval"`     // 刷新间隔
	EnableAutoRefresh bool          `yaml:"enableAutoRefresh" json:"enableAutoRefresh"` // 是否启用自动刷新
	
	// HTTP配置
	HTTPTimeout    time.Duration `yaml:"httpTimeout" json:"httpTimeout"`       // HTTP请求超时时间
	RequestLimit   int           `yaml:"requestLimit" json:"requestLimit"`     // 请求限制（每分钟）
	RetryCount     int           `yaml:"retryCount" json:"retryCount"`         // 重试次数
	RetryInterval  time.Duration `yaml:"retryInterval" json:"retryInterval"`   // 重试间隔
	
	// 过滤配置
	SymbolFilter   SymbolFilterConfig `yaml:"symbolFilter" json:"symbolFilter"`     // 交易对过滤配置
	EnableFiltering bool              `yaml:"enableFiltering" json:"enableFiltering"` // 是否启用过滤
	
	// 存储配置
	EnablePersistence bool `yaml:"enablePersistence" json:"enablePersistence"` // 是否启用持久化
	BufferSize        int  `yaml:"bufferSize" json:"bufferSize"`               // 缓冲区大小
	
	// 监控配置
	EnableMetrics bool `yaml:"enableMetrics" json:"enableMetrics"` // 是否启用指标监控
}

// SymbolFilterConfig 交易对过滤配置
type SymbolFilterConfig struct {
	// 状态过滤
	AllowedStatuses []symbols.SymbolStatus `yaml:"allowedStatuses" json:"allowedStatuses"` // 允许的状态
	
	// 类型过滤
	AllowedTypes []symbols.SymbolType `yaml:"allowedTypes" json:"allowedTypes"` // 允许的类型
	
	// 资产过滤
	AllowedBaseAssets  []string `yaml:"allowedBaseAssets" json:"allowedBaseAssets"`   // 允许的基础资产
	AllowedQuoteAssets []string `yaml:"allowedQuoteAssets" json:"allowedQuoteAssets"` // 允许的计价资产
	
	// 权限过滤
	RequiredPermissions []string `yaml:"requiredPermissions" json:"requiredPermissions"` // 必需的权限
	
	// 名称过滤
	IncludePatterns []string `yaml:"includePatterns" json:"includePatterns"` // 包含模式（正则表达式）
	ExcludePatterns []string `yaml:"excludePatterns" json:"excludePatterns"` // 排除模式（正则表达式）
	
	// 其他过滤条件
	MinNotionalThreshold float64 `yaml:"minNotionalThreshold" json:"minNotionalThreshold"` // 最小名义价值阈值
	OnlyActiveSymbols    bool    `yaml:"onlyActiveSymbols" json:"onlyActiveSymbols"`       // 仅活跃交易对
}

// DefaultBinanceSymbolConfig 返回默认的币安交易对配置
func DefaultBinanceSymbolConfig() *BinanceSymbolConfig {
	return &BinanceSymbolConfig{
		Exchange: "binance",
		BaseURL:  "https://api.binance.com",
		
		// 采集配置
		RefreshInterval:   5 * time.Minute,
		EnableAutoRefresh: true,
		
		// HTTP配置
		HTTPTimeout:   30 * time.Second,
		RequestLimit:  1200, // 币安API限制
		RetryCount:    3,
		RetryInterval: 1 * time.Second,
		
		// 过滤配置
		SymbolFilter: SymbolFilterConfig{
			AllowedStatuses: []symbols.SymbolStatus{
				symbols.StatusTrading,
			},
			AllowedTypes: []symbols.SymbolType{
				symbols.TypeSpot,
			},
			AllowedQuoteAssets: []string{
				"USDT", "BUSD", "BTC", "ETH", "BNB",
			},
			OnlyActiveSymbols: true,
		},
		EnableFiltering: true,
		
		// 存储配置
		EnablePersistence: true,
		BufferSize:        256,
		
		// 监控配置
		EnableMetrics: true,
	}
}

// DefaultBinanceFuturesSymbolConfig 返回默认的币安合约交易对配置
func DefaultBinanceFuturesSymbolConfig() *BinanceSymbolConfig {
	config := DefaultBinanceSymbolConfig()
	config.Exchange = "binance_futures"
	config.BaseURL = "https://fapi.binance.com"
	config.SymbolFilter.AllowedTypes = []symbols.SymbolType{
		symbols.TypePerp,
		symbols.TypeDelivery,
	}
	return config
}

// Validate 验证配置
func (c *BinanceSymbolConfig) Validate() error {
	if c.Exchange == "" {
		return fmt.Errorf("exchange不能为空")
	}
	
	if c.BaseURL == "" {
		return fmt.Errorf("baseURL不能为空")
	}
	
	if c.RefreshInterval <= 0 {
		return fmt.Errorf("refreshInterval必须大于0")
	}
	
	if c.HTTPTimeout <= 0 {
		return fmt.Errorf("httpTimeout必须大于0")
	}
	
	if c.RequestLimit <= 0 {
		return fmt.Errorf("requestLimit必须大于0")
	}
	
	if c.BufferSize <= 0 {
		return fmt.Errorf("bufferSize必须大于0")
	}
	
	return nil
}

// IsSymbolAllowed 检查交易对是否被允许
func (c *BinanceSymbolConfig) IsSymbolAllowed(symbol *symbols.SymbolMeta) bool {
	if !c.EnableFiltering {
		return true
	}
	
	filter := c.SymbolFilter
	
	// 检查状态
	if len(filter.AllowedStatuses) > 0 {
		allowed := false
		for _, status := range filter.AllowedStatuses {
			if symbol.Status == status {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}
	
	// 检查类型
	if len(filter.AllowedTypes) > 0 {
		allowed := false
		for _, symbolType := range filter.AllowedTypes {
			if symbol.Type == symbolType {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}
	
	// 检查基础资产
	if len(filter.AllowedBaseAssets) > 0 {
		allowed := false
		for _, asset := range filter.AllowedBaseAssets {
			if symbol.BaseAsset == asset {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}
	
	// 检查计价资产
	if len(filter.AllowedQuoteAssets) > 0 {
		allowed := false
		for _, asset := range filter.AllowedQuoteAssets {
			if symbol.QuoteAsset == asset {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}
	
	// 检查权限
	if len(filter.RequiredPermissions) > 0 {
		for _, requiredPerm := range filter.RequiredPermissions {
			found := false
			for _, perm := range symbol.Permissions {
				if perm == requiredPerm {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	
	// 检查是否仅活跃交易对
	if filter.OnlyActiveSymbols && !symbol.IsActive() {
		return false
	}
	
	// TODO: 实现正则表达式模式匹配和最小名义价值阈值检查
	
	return true
}

// GetCollectorConfig 转换为公共采集器配置
func (c *BinanceSymbolConfig) GetCollectorConfig() *symbols.CollectorConfig {
	return &symbols.CollectorConfig{
		RefreshInterval:   c.RefreshInterval,
		EnableAutoRefresh: c.EnableAutoRefresh,
		MaxRetries:        c.RetryCount,
		RetryInterval:     c.RetryInterval,
		EnablePersistence: c.EnablePersistence,
		EnableMetrics:     c.EnableMetrics,
		BufferSize:        c.BufferSize,
	}
}
