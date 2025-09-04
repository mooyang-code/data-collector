// Package model 共享的数据结构定义（最好是中文注释！）
package model

import (
	"context"
	"time"
)

// ================================
// 状态结构定义
// ================================

// AppStatus App状态结构
type AppStatus struct {
	State       AppState  `json:"state"`
	Message     string    `json:"message"`
	LastUpdate  time.Time `json:"last_update"`
	StartTime   time.Time `json:"start_time"`
	ErrorCount  int       `json:"error_count"`
	LastError   string    `json:"last_error"`
}

// CollectorStatus 采集器状态结构
type CollectorStatus struct {
	State       CollectorState `json:"state"`
	Message     string         `json:"message"`
	LastUpdate  time.Time      `json:"last_update"`
	StartTime   time.Time      `json:"start_time"`
	DataCount   int64          `json:"data_count"`
	ErrorCount  int64          `json:"error_count"`
	LastError   string         `json:"last_error"`
	LastCollect time.Time      `json:"last_collect"`
}

// TriggerStatus 触发器状态结构
type TriggerStatus struct {
	State        TriggerState `json:"state"`
	Message      string       `json:"message"`
	LastUpdate   time.Time    `json:"last_update"`
	StartTime    time.Time    `json:"start_time"`
	TriggerCount int64        `json:"trigger_count"`
	ErrorCount   int64        `json:"error_count"`
	LastError    string       `json:"last_error"`
	LastTrigger  time.Time    `json:"last_trigger"`
}

// AppManagerStats App管理器统计结构
type AppManagerStats struct {
	TotalApps     int                    `json:"total_apps"`
	RunningApps   int                    `json:"running_apps"`
	ErrorApps     int                    `json:"error_apps"`
	EnabledApps   int                    `json:"enabled_apps"`
	TotalUptime   time.Duration          `json:"total_uptime"`
	Uptime        time.Duration          `json:"uptime"`
	LastUpdate    time.Time              `json:"last_update"`
	CustomMetrics map[string]interface{} `json:"custom_metrics"`
}

// ================================
// 数据记录结构定义
// ================================

// DataRecord 数据记录结构
type DataRecord struct {
	Type      DataType               `json:"type"`
	Exchange  string                 `json:"exchange"`
	Symbol    string                 `json:"symbol"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Metadata  map[string]string      `json:"metadata"`
}

// QueryRequest 查询请求结构
type QueryRequest struct {
	DataType  DataType               `json:"data_type"`
	Exchange  string                 `json:"exchange"`
	Symbol    string                 `json:"symbol"`
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time"`
	Limit     int                    `json:"limit"`
	Filters   map[string]interface{} `json:"filters"`
}

// QueryResult 查询结果结构
type QueryResult struct {
	Records    []*DataRecord `json:"records"`
	Total      int64         `json:"total"`
	HasMore    bool          `json:"has_more"`
	NextCursor string        `json:"next_cursor"`
}

// StorageStats 存储统计结构
type StorageStats struct {
	Backend     string                 `json:"backend"`
	Healthy     bool                   `json:"healthy"`
	Connected   bool                   `json:"connected"`
	RecordCount int64                  `json:"record_count"`
	StorageSize int64                  `json:"storage_size"`
	LastWrite   time.Time              `json:"last_write"`
	WriteRate   float64                `json:"write_rate"`   // 写入速率 (条/秒)
	ErrorRate   float64                `json:"error_rate"`   // 错误率
	Extra       map[string]interface{} `json:"extra"`
}

// HealthStatus 健康状态结构
type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]string `json:"checks"`
	Message   string            `json:"message"`
}

// ================================
// 数据源结构定义
// ================================

// DataSource 数据源结构
type DataSource struct {
	Type      string                 `json:"type"`
	Endpoint  string                 `json:"endpoint"`
	Symbols   []string               `json:"symbols"`
	Intervals []string               `json:"intervals"`
	Config    map[string]interface{} `json:"config"`
}

// ================================
// 回调函数类型
// ================================

// TriggerCallback 触发器回调函数
type TriggerCallback func(ctx context.Context) error
