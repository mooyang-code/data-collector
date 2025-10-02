package types

import "time"

// TaskConfig 任务配置
type TaskConfig struct {
	TaskID        string                 `json:"task_id"`
	CollectorType string                 `json:"collector_type"`
	Source        string                 `json:"source"`
	Interval      string                 `json:"interval"`
	Config        map[string]interface{} `json:"config"`
}

// RunningTask 运行中的任务
type RunningTask struct {
	TaskConfig   TaskConfig
	CollectorID  string
	StartTime    time.Time
	LastExecTime time.Time
	ExecCount    int64
	ErrorCount   int64
}

// RunningTaskInfo 运行中任务信息（用于心跳上报）
type RunningTaskInfo struct {
	TaskID        string    `json:"task_id"`
	CollectorType string    `json:"collector_type"`
	Source        string    `json:"source"`
	StartTime     time.Time `json:"start_time"`
	LastExecTime  time.Time `json:"last_exec_time"`
	ExecCount     int64     `json:"exec_count"`
	ErrorCount    int64     `json:"error_count"`
}