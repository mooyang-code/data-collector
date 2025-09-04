// Package storage 配置转换器
package storage

import (
	"fmt"
	"time"
)

// ConfigFromMap 从map创建存储配置
func ConfigFromMap(data map[string]interface{}) (*Config, error) {
	config := &Config{}
	
	// 基础配置
	if backend, ok := data["backend"].(string); ok {
		config.Backend = backend
	}
	
	if enabled, ok := data["enabled"].(bool); ok {
		config.Enabled = enabled
	}
	
	if batchSize, ok := data["batch_size"].(int); ok {
		config.BatchSize = batchSize
	}
	
	if bufferSize, ok := data["buffer_size"].(int); ok {
		config.BufferSize = bufferSize
	}
	
	if flushInterval, ok := data["flush_interval"].(string); ok {
		if duration, err := time.ParseDuration(flushInterval); err == nil {
			config.FlushInterval = duration
		} else {
			return nil, fmt.Errorf("无效的刷新间隔: %s", flushInterval)
		}
	}
	
	// 内存存储配置
	if memoryData, ok := data["memory"].(map[string]interface{}); ok {
		config.Memory = &MemoryConfig{}
		
		if maxEntries, ok := memoryData["max_entries"].(int); ok {
			config.Memory.MaxEntries = maxEntries
		}
		
		if ttl, ok := memoryData["ttl"].(string); ok {
			if duration, err := time.ParseDuration(ttl); err == nil {
				config.Memory.TTL = duration
			} else {
				return nil, fmt.Errorf("无效的TTL: %s", ttl)
			}
		}
	}
	
	// 文件存储配置
	if fileData, ok := data["file"].(map[string]interface{}); ok {
		config.File = &FileConfig{}
		
		if path, ok := fileData["path"].(string); ok {
			config.File.Path = path
		}
		
		if format, ok := fileData["format"].(string); ok {
			config.File.Format = format
		}
		
		if compress, ok := fileData["compress"].(bool); ok {
			config.File.Compress = compress
		}
		
		if rotateSize, ok := fileData["rotate_size"].(int); ok {
			config.File.RotateSize = rotateSize
		}
		
		if keepFiles, ok := fileData["keep_files"].(int); ok {
			config.File.KeepFiles = keepFiles
		}
	}
	

	
	// Redis存储配置
	if redisData, ok := data["redis"].(map[string]interface{}); ok {
		config.Redis = &RedisConfig{}
		
		if addr, ok := redisData["addr"].(string); ok {
			config.Redis.Addr = addr
		}
		
		if password, ok := redisData["password"].(string); ok {
			config.Redis.Password = password
		}
		
		if db, ok := redisData["db"].(int); ok {
			config.Redis.DB = db
		}
		
		if poolSize, ok := redisData["pool_size"].(int); ok {
			config.Redis.PoolSize = poolSize
		}
		
		if minIdleConns, ok := redisData["min_idle_conns"].(int); ok {
			config.Redis.MinIdleConns = minIdleConns
		}
		
		if keyPrefix, ok := redisData["key_prefix"].(string); ok {
			config.Redis.KeyPrefix = keyPrefix
		}
		
		if ttl, ok := redisData["ttl"].(string); ok {
			if duration, err := time.ParseDuration(ttl); err == nil {
				config.Redis.TTL = duration
			} else {
				return nil, fmt.Errorf("无效的TTL: %s", ttl)
			}
		}
	}
	
	return config, nil
}
