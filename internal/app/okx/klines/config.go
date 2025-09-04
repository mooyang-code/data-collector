// Package klines OKX K线采集器配置
package klines

import (
	"fmt"
	"time"

	"github.com/mooyang-code/data-collector/internal/datatype/klines"
)

// OKXKlineConfig OKX K线采集配置
type OKXKlineConfig struct {
	// 基础配置
	Exchange string `yaml:"exchange" json:"exchange"` // 交易所名称
	BaseURL  string `yaml:"baseUrl" json:"baseUrl"`   // API基础URL
	
	// 采集配置
	Symbols        []string          `yaml:"symbols" json:"symbols"`               // 交易对列表
	Intervals      []klines.Interval `yaml:"intervals" json:"intervals"`           // K线间隔列表
	BufferSize     int               `yaml:"bufferSize" json:"bufferSize"`         // 缓冲区大小
	HistoryLimit   int               `yaml:"historyLimit" json:"historyLimit"`     // 历史数据限制
	EnableBackfill bool              `yaml:"enableBackfill" json:"enableBackfill"` // 启用回补
	BackfillDays   int               `yaml:"backfillDays" json:"backfillDays"`     // 回补天数
	
	// HTTP配置
	HTTPTimeout    time.Duration `yaml:"httpTimeout" json:"httpTimeout"`       // HTTP请求超时时间
	RequestLimit   int           `yaml:"requestLimit" json:"requestLimit"`     // 请求限制（每分钟）
	RetryCount     int           `yaml:"retryCount" json:"retryCount"`         // 重试次数
	RetryInterval  time.Duration `yaml:"retryInterval" json:"retryInterval"`   // 重试间隔
	
	// 存储配置
	EnablePersistence bool `yaml:"enablePersistence" json:"enablePersistence"` // 是否启用持久化
	
	// 监控配置
	EnableMetrics bool `yaml:"enableMetrics" json:"enableMetrics"` // 是否启用指标监控
}

// DefaultOKXKlineConfig 返回默认的OKX K线配置
func DefaultOKXKlineConfig() *OKXKlineConfig {
	return &OKXKlineConfig{
		Exchange: "okx",
		BaseURL:  "https://www.okx.com/api/v5",
		
		// 采集配置
		Symbols: []string{
			"BTC-USDT", "ETH-USDT", "BNB-USDT",
		},
		Intervals: []klines.Interval{
			klines.Interval1m,
			klines.Interval5m,
			klines.Interval15m,
			klines.Interval1h,
			klines.Interval1d,
		},
		BufferSize:     1000,
		HistoryLimit:   10000,
		EnableBackfill: true,
		BackfillDays:   7,
		
		// HTTP配置
		HTTPTimeout:   30 * time.Second,
		RequestLimit:  20, // OKX API限制
		RetryCount:    3,
		RetryInterval: 1 * time.Second,
		
		// 存储配置
		EnablePersistence: true,
		
		// 监控配置
		EnableMetrics: true,
	}
}

// Validate 验证配置
func (c *OKXKlineConfig) Validate() error {
	if c.Exchange == "" {
		return fmt.Errorf("exchange不能为空")
	}
	
	if c.BaseURL == "" {
		return fmt.Errorf("baseURL不能为空")
	}
	
	if len(c.Symbols) == 0 {
		return fmt.Errorf("symbols不能为空")
	}
	
	if len(c.Intervals) == 0 {
		return fmt.Errorf("intervals不能为空")
	}
	
	if c.BufferSize <= 0 {
		return fmt.Errorf("bufferSize必须大于0")
	}
	
	if c.HTTPTimeout <= 0 {
		return fmt.Errorf("httpTimeout必须大于0")
	}
	
	if c.RequestLimit <= 0 {
		return fmt.Errorf("requestLimit必须大于0")
	}
	
	return nil
}