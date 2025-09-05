// Package model 共享的枚举类型定义
package model

// DataType 数据类型枚举
type DataType string

const (
	DataTypeKline  DataType = "kline"  // K线数据
	DataTypeSymbol DataType = "symbol" // 交易对数据
	DataTypeDepth  DataType = "depth"  // 深度数据
	DataTypeTrade  DataType = "trade"  // 成交数据
	DataTypeTicker DataType = "ticker" // 行情数据
)

// TriggerType 触发器类型枚举
type TriggerType string

const (
	TriggerTypeCron    TriggerType = "cron"    // 定时触发器
	TriggerTypeEvent   TriggerType = "event"   // 事件触发器
	TriggerTypeManual  TriggerType = "manual"  // 手动触发器
	TriggerTypeWebhook TriggerType = "webhook" // Webhook触发器
)

// AppState App状态枚举
type AppState string

const (
	AppStateUnknown     AppState = "unknown"     // 未知状态
	AppStateInitialized AppState = "initialized" // 已初始化
	AppStateStarting    AppState = "starting"    // 启动中
	AppStateRunning     AppState = "running"     // 运行中
	AppStateStopping    AppState = "stopping"    // 停止中
	AppStateStopped     AppState = "stopped"     // 已停止
	AppStateError       AppState = "error"       // 错误状态
)

// CollectorState 采集器状态枚举
type CollectorState string

const (
	CollectorStateUnknown     CollectorState = "unknown"     // 未知状态
	CollectorStateInitialized CollectorState = "initialized" // 已初始化
	CollectorStateStarting    CollectorState = "starting"    // 启动中
	CollectorStateRunning     CollectorState = "running"     // 运行中
	CollectorStateStopping    CollectorState = "stopping"    // 停止中
	CollectorStateStopped     CollectorState = "stopped"     // 已停止
	CollectorStateError       CollectorState = "error"       // 错误状态
)

// TriggerState 触发器状态枚举
type TriggerState string

const (
	TriggerStateUnknown TriggerState = "unknown" // 未知状态
	TriggerStateIdle    TriggerState = "idle"    // 空闲状态
	TriggerStateRunning TriggerState = "running" // 运行中
	TriggerStateStopped TriggerState = "stopped" // 已停止
	TriggerStateError   TriggerState = "error"   // 错误状态
)

// StorageType 存储类型枚举
type StorageType string

const (
	StorageTypeMemory     StorageType = "memory"     // 内存存储
	StorageTypeFile       StorageType = "file"       // 文件存储
	StorageTypeClickHouse StorageType = "clickhouse" // ClickHouse存储
	StorageTypeMySQL      StorageType = "mysql"      // MySQL存储
	StorageTypeRedis      StorageType = "redis"      // Redis存储
)
