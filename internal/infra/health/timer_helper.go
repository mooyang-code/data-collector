// Package health 健康检查相关的定时任务辅助工具
package health

import (
	"fmt"
	"time"

	"github.com/mooyang-code/data-collector/internal/infra/timer"
)

// HealthJobHelper 健康检查任务辅助工具
type HealthJobHelper struct {
	cronHelper *timer.CronHelper
}

// NewHealthJobHelper 创建健康检查任务辅助工具
func NewHealthJobHelper() *HealthJobHelper {
	return &HealthJobHelper{
		cronHelper: timer.NewCronHelper(),
	}
}

// CreateHealthCheckJob 创建健康检查任务
func (h *HealthJobHelper) CreateHealthCheckJob(name string, fn timer.JobFunc) *timer.Job {
	jobID := timer.JobID(fmt.Sprintf("health_check_%s", name))
	
	return timer.NewJobBuilder().
		WithID(jobID).
		WithName(fmt.Sprintf("健康检查-%s", name)).
		WithDescription(fmt.Sprintf("%s组件健康检查", name)).
		WithCron("0 */5 * * * *"). // 每5分钟执行一次
		WithFunc(fn).
		WithTimeout(2 * time.Minute).
		WithMaxRetries(1).
		Build()
}

// CreateDatabaseHealthCheckJob 创建数据库健康检查任务
func (h *HealthJobHelper) CreateDatabaseHealthCheckJob(dbName string, fn timer.JobFunc) *timer.Job {
	jobID := timer.JobID(fmt.Sprintf("health_check_db_%s", dbName))
	
	return timer.NewJobBuilder().
		WithID(jobID).
		WithName(fmt.Sprintf("数据库健康检查-%s", dbName)).
		WithDescription(fmt.Sprintf("%s数据库连接和性能检查", dbName)).
		WithCron("0 */3 * * * *"). // 每3分钟执行一次
		WithFunc(fn).
		WithTimeout(1 * time.Minute).
		WithMaxRetries(2).
		Build()
}

// CreateAPIHealthCheckJob 创建API健康检查任务
func (h *HealthJobHelper) CreateAPIHealthCheckJob(apiName string, fn timer.JobFunc) *timer.Job {
	jobID := timer.JobID(fmt.Sprintf("health_check_api_%s", apiName))
	
	return timer.NewJobBuilder().
		WithID(jobID).
		WithName(fmt.Sprintf("API健康检查-%s", apiName)).
		WithDescription(fmt.Sprintf("%s API可用性和响应时间检查", apiName)).
		WithCron("0 */2 * * * *"). // 每2分钟执行一次
		WithFunc(fn).
		WithTimeout(30 * time.Second).
		WithMaxRetries(3).
		Build()
}

// CreateStorageHealthCheckJob 创建存储健康检查任务
func (h *HealthJobHelper) CreateStorageHealthCheckJob(storageName string, fn timer.JobFunc) *timer.Job {
	jobID := timer.JobID(fmt.Sprintf("health_check_storage_%s", storageName))
	
	return timer.NewJobBuilder().
		WithID(jobID).
		WithName(fmt.Sprintf("存储健康检查-%s", storageName)).
		WithDescription(fmt.Sprintf("%s存储系统健康状态检查", storageName)).
		WithCron("0 */10 * * * *"). // 每10分钟执行一次
		WithFunc(fn).
		WithTimeout(5 * time.Minute).
		WithMaxRetries(2).
		Build()
}

// CreateMemoryHealthCheckJob 创建内存使用检查任务
func (h *HealthJobHelper) CreateMemoryHealthCheckJob(fn timer.JobFunc) *timer.Job {
	jobID := timer.JobID("health_check_memory")
	
	return timer.NewJobBuilder().
		WithID(jobID).
		WithName("内存使用检查").
		WithDescription("系统内存使用情况监控").
		WithCron("0 */1 * * * *"). // 每1分钟执行一次
		WithFunc(fn).
		WithTimeout(30 * time.Second).
		WithMaxRetries(1).
		Build()
}

// CreateDiskHealthCheckJob 创建磁盘空间检查任务
func (h *HealthJobHelper) CreateDiskHealthCheckJob(fn timer.JobFunc) *timer.Job {
	jobID := timer.JobID("health_check_disk")
	
	return timer.NewJobBuilder().
		WithID(jobID).
		WithName("磁盘空间检查").
		WithDescription("系统磁盘空间使用情况监控").
		WithCron("0 */5 * * * *"). // 每5分钟执行一次
		WithFunc(fn).
		WithTimeout(1 * time.Minute).
		WithMaxRetries(1).
		Build()
}

// CreateCustomHealthCheckJob 创建自定义健康检查任务
func (h *HealthJobHelper) CreateCustomHealthCheckJob(name, description, cronExpr string, fn timer.JobFunc, timeout time.Duration) *timer.Job {
	jobID := timer.JobID(fmt.Sprintf("health_check_custom_%s", name))
	
	return timer.NewJobBuilder().
		WithID(jobID).
		WithName(fmt.Sprintf("自定义健康检查-%s", name)).
		WithDescription(description).
		WithCron(cronExpr).
		WithFunc(fn).
		WithTimeout(timeout).
		WithMaxRetries(2).
		Build()
}

// CreateBatchHealthCheckJobs 批量创建健康检查任务
func (h *HealthJobHelper) CreateBatchHealthCheckJobs(components []string, fnFactory func(component string) timer.JobFunc) []*timer.Job {
	var jobs []*timer.Job
	
	for _, component := range components {
		fn := fnFactory(component)
		job := h.CreateHealthCheckJob(component, fn)
		jobs = append(jobs, job)
	}
	
	return jobs
}

// GetRecommendedHealthCheckInterval 获取推荐的健康检查间隔
func (h *HealthJobHelper) GetRecommendedHealthCheckInterval(checkType string) string {
	switch checkType {
	case "critical":
		return "0 */1 * * * *" // 每1分钟
	case "important":
		return "0 */3 * * * *" // 每3分钟
	case "normal":
		return "0 */5 * * * *" // 每5分钟
	case "low":
		return "0 */10 * * * *" // 每10分钟
	default:
		return "0 */5 * * * *" // 默认每5分钟
	}
}

// ValidateHealthCheckConfig 验证健康检查配置
func (h *HealthJobHelper) ValidateHealthCheckConfig(name, cronExpr string, timeout time.Duration) error {
	if name == "" {
		return fmt.Errorf("health check name cannot be empty")
	}
	
	if cronExpr == "" {
		return fmt.Errorf("cron expression cannot be empty")
	}
	
	if timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	
	if timeout > 10*time.Minute {
		return fmt.Errorf("timeout should not exceed 10 minutes for health checks")
	}
	
	return nil
}
