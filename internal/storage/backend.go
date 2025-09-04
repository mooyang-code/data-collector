// Package storage 通用存储后端
package storage

import (
	"context"
	"time"
)

// StorageBackend 通用存储后端接口
type StorageBackend interface {
	// 初始化存储后端
	Initialize(config *Config) error
	
	// 存储单条数据
	Store(ctx context.Context, data *DataRecord) error
	
	// 批量存储数据
	StoreBatch(ctx context.Context, data []*DataRecord) error
	
	// 查询数据
	Query(ctx context.Context, query *QueryRequest) (*QueryResult, error)
	
	// 获取统计信息
	GetStats() *StorageStats
	
	// 健康检查
	HealthCheck() error
	
	// 关闭存储后端
	Close() error
}

// DataRecord 通用数据记录
type DataRecord struct {
	// 数据类型 (symbols, klines, depth, trades, etc.)
	Type string `json:"type"`
	
	// 交易所
	Exchange string `json:"exchange"`
	
	// 交易对
	Symbol string `json:"symbol"`
	
	// 时间戳
	Timestamp time.Time `json:"timestamp"`
	
	// 数据内容 (JSON格式)
	Data map[string]interface{} `json:"data"`
	
	// 元数据
	Metadata map[string]string `json:"metadata,omitempty"`
}

// QueryRequest 查询请求
type QueryRequest struct {
	// 数据类型过滤
	Type string `json:"type,omitempty"`
	
	// 交易所过滤
	Exchange string `json:"exchange,omitempty"`
	
	// 交易对过滤
	Symbol string `json:"symbol,omitempty"`
	
	// 时间范围
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	
	// 分页
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
	
	// 排序
	OrderBy   string `json:"order_by,omitempty"`
	OrderDesc bool   `json:"order_desc,omitempty"`
}

// QueryResult 查询结果
type QueryResult struct {
	// 数据记录
	Records []*DataRecord `json:"records"`
	
	// 总数
	Total int64 `json:"total"`
	
	// 是否有更多数据
	HasMore bool `json:"has_more"`
	
	// 查询耗时
	Duration time.Duration `json:"duration"`
}

// StorageStats 存储统计信息
type StorageStats struct {
	// 后端类型
	Backend string `json:"backend"`
	
	// 是否健康
	Healthy bool `json:"healthy"`
	
	// 总记录数
	TotalRecords int64 `json:"total_records"`
	
	// 存储大小 (字节)
	StorageSize int64 `json:"storage_size"`
	
	// 最后写入时间
	LastWriteTime time.Time `json:"last_write_time"`
	
	// 最后读取时间
	LastReadTime time.Time `json:"last_read_time"`
	
	// 写入速率 (记录/秒)
	WriteRate float64 `json:"write_rate"`
	
	// 读取速率 (记录/秒)
	ReadRate float64 `json:"read_rate"`
	
	// 错误计数
	ErrorCount int64 `json:"error_count"`
	
	// 最后错误
	LastError string `json:"last_error,omitempty"`
	
	// 连接状态
	Connected bool `json:"connected"`
	
	// 额外信息
	Extra map[string]interface{} `json:"extra,omitempty"`
}
