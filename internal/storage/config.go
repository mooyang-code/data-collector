// Package storage 存储配置
package storage

import (
	"time"
)

// Config 存储配置
type Config struct {
	// 存储后端类型
	Backend string `yaml:"backend" json:"backend"`
	
	// 是否启用存储
	Enabled bool `yaml:"enabled" json:"enabled"`
	
	// 批处理大小
	BatchSize int `yaml:"batch_size" json:"batch_size"`
	
	// 缓冲区大小
	BufferSize int `yaml:"buffer_size" json:"buffer_size"`
	
	// 刷新间隔
	FlushInterval time.Duration `yaml:"flush_interval" json:"flush_interval"`
	
	// 内存存储配置
	Memory *MemoryConfig `yaml:"memory,omitempty" json:"memory,omitempty"`
	
	// 文件存储配置
	File *FileConfig `yaml:"file,omitempty" json:"file,omitempty"`
	

	
	// Redis存储配置
	Redis *RedisConfig `yaml:"redis,omitempty" json:"redis,omitempty"`
}

// MemoryConfig 内存存储配置
type MemoryConfig struct {
	// 最大条目数
	MaxEntries int `yaml:"max_entries" json:"max_entries"`
	
	// 过期时间
	TTL time.Duration `yaml:"ttl" json:"ttl"`
}

// FileConfig 文件存储配置
type FileConfig struct {
	// 文件路径
	Path string `yaml:"path" json:"path"`
	
	// 文件格式 (json, csv, parquet)
	Format string `yaml:"format" json:"format"`
	
	// 是否压缩
	Compress bool `yaml:"compress" json:"compress"`
	
	// 文件轮转大小 (MB)
	RotateSize int `yaml:"rotate_size" json:"rotate_size"`
	
	// 保留文件数
	KeepFiles int `yaml:"keep_files" json:"keep_files"`
}



// RedisConfig Redis存储配置
type RedisConfig struct {
	// 地址
	Addr string `yaml:"addr" json:"addr"`
	
	// 密码
	Password string `yaml:"password" json:"password"`
	
	// 数据库
	DB int `yaml:"db" json:"db"`
	
	// 连接池大小
	PoolSize int `yaml:"pool_size" json:"pool_size"`
	
	// 最小空闲连接数
	MinIdleConns int `yaml:"min_idle_conns" json:"min_idle_conns"`
	
	// 键前缀
	KeyPrefix string `yaml:"key_prefix" json:"key_prefix"`
	
	// 过期时间
	TTL time.Duration `yaml:"ttl" json:"ttl"`
}

// DefaultConfig 返回默认存储配置
func DefaultConfig() *Config {
	return &Config{
		Backend:       "memory",
		Enabled:       true,
		BatchSize:     100,
		BufferSize:    1000,
		FlushInterval: 5 * time.Second,
		Memory: &MemoryConfig{
			MaxEntries: 10000,
			TTL:        1 * time.Hour,
		},
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	
	if c.Backend == "" {
		c.Backend = "memory"
	}
	
	if c.BatchSize <= 0 {
		c.BatchSize = 100
	}
	
	if c.BufferSize <= 0 {
		c.BufferSize = 1000
	}
	
	if c.FlushInterval <= 0 {
		c.FlushInterval = 5 * time.Second
	}
	
	return nil
}
