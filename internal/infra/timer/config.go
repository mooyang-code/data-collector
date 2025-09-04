// Package timer 定时器基础组件
package timer

import (
	"time"
)

// Config 定时器配置
type Config struct {
	// 是否启用定时器
	Enabled bool `yaml:"enabled" json:"enabled"`
	
	// 默认时区
	Timezone string `yaml:"timezone" json:"timezone"`
	
	// 最大并发任务数
	MaxConcurrentJobs int `yaml:"maxConcurrentJobs" json:"maxConcurrentJobs"`
	
	// 任务执行超时时间
	JobTimeout time.Duration `yaml:"jobTimeout" json:"jobTimeout"`
	
	// 是否启用任务恢复（重启后恢复未完成的任务）
	EnableRecovery bool `yaml:"enableRecovery" json:"enableRecovery"`
	
	// 任务历史保留时间
	HistoryRetention time.Duration `yaml:"historyRetention" json:"historyRetention"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Enabled:           true,
		Timezone:          "UTC",
		MaxConcurrentJobs: 10,
		JobTimeout:        30 * time.Minute,
		EnableRecovery:    false,
		HistoryRetention:  24 * time.Hour,
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.MaxConcurrentJobs <= 0 {
		c.MaxConcurrentJobs = 10
	}
	
	if c.JobTimeout <= 0 {
		c.JobTimeout = 30 * time.Minute
	}
	
	if c.HistoryRetention <= 0 {
		c.HistoryRetention = 24 * time.Hour
	}
	
	if c.Timezone == "" {
		c.Timezone = "UTC"
	}
	
	return nil
}

// GetLocation 获取时区位置
func (c *Config) GetLocation() (*time.Location, error) {
	return time.LoadLocation(c.Timezone)
}
