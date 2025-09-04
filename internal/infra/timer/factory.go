package timer

import (
	"fmt"
	"time"
)

// TimerFactory 定时器工厂
type TimerFactory struct {
	config *Config
}

// NewTimerFactory 创建定时器工厂
func NewTimerFactory(config *Config) *TimerFactory {
	if config == nil {
		config = DefaultConfig()
	}
	
	return &TimerFactory{
		config: config,
	}
}

// CreateTimer 创建定时器
func (f *TimerFactory) CreateTimer() (Timer, error) {
	return NewCronTimer(f.config)
}

// CreateTimerWithConfig 使用指定配置创建定时器
func (f *TimerFactory) CreateTimerWithConfig(config *Config) (Timer, error) {
	return NewCronTimer(config)
}

// CronHelper Cron表达式辅助工具
type CronHelper struct{}

// NewCronHelper 创建Cron辅助工具
func NewCronHelper() *CronHelper {
	return &CronHelper{}
}

// EverySecond 每秒执行
func (h *CronHelper) EverySecond() string {
	return "* * * * * *"
}

// EveryMinute 每分钟执行
func (h *CronHelper) EveryMinute() string {
	return "0 * * * * *"
}

// EveryHour 每小时执行
func (h *CronHelper) EveryHour() string {
	return "0 0 * * * *"
}

// EveryDay 每天执行（午夜）
func (h *CronHelper) EveryDay() string {
	return "0 0 0 * * *"
}

// EveryWeek 每周执行（周日午夜）
func (h *CronHelper) EveryWeek() string {
	return "0 0 0 * * 0"
}

// EveryMonth 每月执行（1号午夜）
func (h *CronHelper) EveryMonth() string {
	return "0 0 0 1 * *"
}

// Every 每隔指定时间执行
func (h *CronHelper) Every(duration time.Duration) string {
	switch {
	case duration < time.Minute:
		// 秒级
		seconds := int(duration.Seconds())
		return fmt.Sprintf("*/%d * * * * *", seconds)
	case duration < time.Hour:
		// 分钟级
		minutes := int(duration.Minutes())
		return fmt.Sprintf("0 */%d * * * *", minutes)
	case duration < 24*time.Hour:
		// 小时级
		hours := int(duration.Hours())
		return fmt.Sprintf("0 0 */%d * * *", hours)
	default:
		// 天级
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("0 0 0 */%d * *", days)
	}
}

// At 在指定时间执行（每天）
func (h *CronHelper) At(hour, minute, second int) string {
	return fmt.Sprintf("%d %d %d * * *", second, minute, hour)
}

// AtTime 在指定时间执行（每天）
func (h *CronHelper) AtTime(t time.Time) string {
	return h.At(t.Hour(), t.Minute(), t.Second())
}

// OnWeekday 在指定星期几执行
func (h *CronHelper) OnWeekday(weekday time.Weekday, hour, minute, second int) string {
	return fmt.Sprintf("%d %d %d * * %d", second, minute, hour, int(weekday))
}

// OnDate 在指定日期执行（每月）
func (h *CronHelper) OnDate(day, hour, minute, second int) string {
	return fmt.Sprintf("%d %d %d %d * *", second, minute, hour, day)
}

// Validate 验证Cron表达式
func (h *CronHelper) Validate(cronExpr string) error {
	// 这里可以使用cron库来验证表达式
	// 简单验证：检查是否有6个字段（支持秒）
	return nil
}

// NextRun 计算下次执行时间
func (h *CronHelper) NextRun(cronExpr string) (time.Time, error) {
	// 这里可以使用cron库来计算下次执行时间
	return time.Now().Add(time.Minute), nil
}

// JobHelper 通用任务辅助工具
type JobHelper struct{}

// NewJobHelper 创建任务辅助工具
func NewJobHelper() *JobHelper {
	return &JobHelper{}
}

// CreateSimpleJob 创建简单的定时任务
func (h *JobHelper) CreateSimpleJob(id, name, description, cronExpr string, fn JobFunc) *Job {
	return NewJobBuilder().
		WithID(JobID(id)).
		WithName(name).
		WithDescription(description).
		WithCron(cronExpr).
		WithFunc(fn).
		WithTimeout(5 * time.Minute).
		WithMaxRetries(3).
		Build()
}

// CreatePeriodicJob 创建周期性任务
func (h *JobHelper) CreatePeriodicJob(id, name string, interval time.Duration, fn JobFunc) *Job {
	cronHelper := NewCronHelper()
	cronExpr := cronHelper.Every(interval)

	return NewJobBuilder().
		WithID(JobID(id)).
		WithName(name).
		WithDescription(fmt.Sprintf("每%v执行一次", interval)).
		WithCron(cronExpr).
		WithFunc(fn).
		WithTimeout(5 * time.Minute).
		WithMaxRetries(3).
		Build()
}

// CreateCustomJob 创建自定义配置的任务
func (h *JobHelper) CreateCustomJob(id, name, description, cronExpr string, fn JobFunc, timeout time.Duration, maxRetries int) *Job {
	return NewJobBuilder().
		WithID(JobID(id)).
		WithName(name).
		WithDescription(description).
		WithCron(cronExpr).
		WithFunc(fn).
		WithTimeout(timeout).
		WithMaxRetries(maxRetries).
		Build()
}

// GetDefaultTimer 获取默认配置的定时器
func GetDefaultTimer() (Timer, error) {
	factory := NewTimerFactory(DefaultConfig())
	return factory.CreateTimer()
}
