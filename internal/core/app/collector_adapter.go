package app

import (
	"github.com/mooyang-code/data-collector/internal/core/collector"
)

// CollectorAdapter 将 collector.Collector 适配到 app.Collector
type CollectorAdapter struct {
	collector.Collector
}

// NewCollectorAdapter 创建采集器适配器
func NewCollectorAdapter(c collector.Collector) Collector {
	return &CollectorAdapter{
		Collector: c,
	}
}

// GetStatus 获取状态（适配器方法）
func (a *CollectorAdapter) GetStatus() CollectorStatus {
	// 从 collector.CollectorStatus 转换到 app.CollectorStatus
	cStatus := a.Collector.GetStatus()
	
	timers := make(map[string]TimerStatus)
	for name, timer := range cStatus.Timers {
		timers[name] = TimerStatus{
			Name:       timer.Name,
			Interval:   timer.Interval,
			LastRun:    timer.LastRun,
			NextRun:    timer.NextRun,
			RunCount:   timer.RunCount,
			ErrorCount: timer.ErrorCount,
		}
	}
	
	return CollectorStatus{
		ID:        cStatus.ID,
		Type:      cStatus.Type,
		DataType:  cStatus.DataType,
		IsRunning: cStatus.IsRunning,
		StartTime: cStatus.StartTime,
		Timers:    timers,
	}
}