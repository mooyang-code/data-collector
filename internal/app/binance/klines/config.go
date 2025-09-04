// Package klines 币安K线配置
package klines

import (
	"fmt"
	"time"

	"github.com/mooyang-code/data-collector/internal/datatype/klines"
)

// BinanceKlineConfig 币安K线采集配置
type BinanceKlineConfig struct {
	// 基础配置
	Exchange string `yaml:"exchange" json:"exchange"` // 交易所名称
	BaseURL  string `yaml:"baseUrl" json:"baseUrl"`   // API基础URL
	
	// 采集配置
	Symbols   []string          `yaml:"symbols" json:"symbols"`     // 要采集的交易对列表
	Intervals []klines.Interval `yaml:"intervals" json:"intervals"` // 要采集的周期列表
	
	// 定时器配置
	TimerConfigs map[string]TimerConfig `yaml:"timerConfigs" json:"timerConfigs"` // 定时器配置，key为interval
	
	// HTTP配置
	HTTPTimeout    time.Duration `yaml:"httpTimeout" json:"httpTimeout"`       // HTTP请求超时时间
	RequestLimit   int           `yaml:"requestLimit" json:"requestLimit"`     // 请求限制（每分钟）
	RetryCount     int           `yaml:"retryCount" json:"retryCount"`         // 重试次数
	RetryInterval  time.Duration `yaml:"retryInterval" json:"retryInterval"`   // 重试间隔
	
	// 数据配置
	HistoryLimit   int           `yaml:"historyLimit" json:"historyLimit"`     // 历史数据获取限制
	BackfillDays   int           `yaml:"backfillDays" json:"backfillDays"`     // 回补天数
	EnableBackfill bool          `yaml:"enableBackfill" json:"enableBackfill"` // 是否启用回补
	
	// 存储配置
	EnablePersistence bool `yaml:"enablePersistence" json:"enablePersistence"` // 是否启用持久化
	BufferSize        int  `yaml:"bufferSize" json:"bufferSize"`               // 缓冲区大小
}

// TimerConfig 定时器配置
type TimerConfig struct {
	CronExpr    string        `yaml:"cronExpr" json:"cronExpr"`       // cron表达式
	Timeout     time.Duration `yaml:"timeout" json:"timeout"`         // 任务超时时间
	MaxRetries  int           `yaml:"maxRetries" json:"maxRetries"`   // 最大重试次数
	Enabled     bool          `yaml:"enabled" json:"enabled"`         // 是否启用
	Description string        `yaml:"description" json:"description"` // 描述
}

// DefaultBinanceKlineConfig 返回默认的币安K线配置
func DefaultBinanceKlineConfig() *BinanceKlineConfig {
	return &BinanceKlineConfig{
		Exchange: "binance",
		BaseURL:  "https://api.binance.com",
		
		// 默认采集主流交易对
		Symbols: []string{
			"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "XRPUSDT",
			"SOLUSDT", "DOTUSDT", "DOGEUSDT", "AVAXUSDT", "SHIBUSDT",
		},
		
		// 默认采集常用周期
		Intervals: []klines.Interval{
			klines.Interval1m,
			klines.Interval5m,
			klines.Interval15m,
			klines.Interval1h,
			klines.Interval4h,
			klines.Interval1d,
		},
		
		// 定时器配置
		TimerConfigs: map[string]TimerConfig{
			string(klines.Interval1m): {
				CronExpr:    "0 * * * * *",     // 每分钟
				Timeout:     30 * time.Second,
				MaxRetries:  3,
				Enabled:     true,
				Description: "1分钟K线定时采集",
			},
			string(klines.Interval5m): {
				CronExpr:    "0 */5 * * * *",   // 每5分钟
				Timeout:     30 * time.Second,
				MaxRetries:  3,
				Enabled:     true,
				Description: "5分钟K线定时采集",
			},
			string(klines.Interval15m): {
				CronExpr:    "0 */15 * * * *",  // 每15分钟
				Timeout:     30 * time.Second,
				MaxRetries:  3,
				Enabled:     true,
				Description: "15分钟K线定时采集",
			},
			string(klines.Interval1h): {
				CronExpr:    "0 0 * * * *",     // 每小时
				Timeout:     60 * time.Second,
				MaxRetries:  3,
				Enabled:     true,
				Description: "1小时K线定时采集",
			},
			string(klines.Interval4h): {
				CronExpr:    "0 0 */4 * * *",   // 每4小时
				Timeout:     60 * time.Second,
				MaxRetries:  3,
				Enabled:     true,
				Description: "4小时K线定时采集",
			},
			string(klines.Interval1d): {
				CronExpr:    "0 0 0 * * *",     // 每天
				Timeout:     120 * time.Second,
				MaxRetries:  3,
				Enabled:     true,
				Description: "1天K线定时采集",
			},
		},
		
		// HTTP配置
		HTTPTimeout:   30 * time.Second,
		RequestLimit:  1200, // 币安API限制
		RetryCount:    3,
		RetryInterval: 1 * time.Second,
		
		// 数据配置
		HistoryLimit:   1000,
		BackfillDays:   7,
		EnableBackfill: true,
		
		// 存储配置
		EnablePersistence: true,
		BufferSize:        1000,
	}
}

// GetTimerConfig 获取指定周期的定时器配置
func (c *BinanceKlineConfig) GetTimerConfig(interval klines.Interval) (TimerConfig, bool) {
	config, exists := c.TimerConfigs[string(interval)]
	return config, exists
}

// IsSymbolEnabled 检查交易对是否启用
func (c *BinanceKlineConfig) IsSymbolEnabled(symbol string) bool {
	for _, s := range c.Symbols {
		if s == symbol {
			return true
		}
	}
	return false
}

// IsIntervalEnabled 检查周期是否启用
func (c *BinanceKlineConfig) IsIntervalEnabled(interval klines.Interval) bool {
	for _, i := range c.Intervals {
		if i == interval {
			return true
		}
	}
	return false
}

// Validate 验证配置
func (c *BinanceKlineConfig) Validate() error {
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
	
	if c.HTTPTimeout <= 0 {
		return fmt.Errorf("httpTimeout必须大于0")
	}
	
	if c.RequestLimit <= 0 {
		return fmt.Errorf("requestLimit必须大于0")
	}
	
	return nil
}
