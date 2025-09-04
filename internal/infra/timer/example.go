package timer

import (
	"context"
	"fmt"
	"time"

	"trpc.group/trpc-go/trpc-go/log"
)

// Example 定时器使用示例
type Example struct {
	timer Timer
}

// NewExample 创建示例
func NewExample() (*Example, error) {
	timer, err := GetDefaultTimer()
	if err != nil {
		return nil, err
	}
	
	return &Example{
		timer: timer,
	}, nil
}

// RunBasicExample 运行基础示例
func (e *Example) RunBasicExample(ctx context.Context) error {
	log.Info("Starting timer basic example...")
	
	// 启动定时器
	if err := e.timer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start timer: %w", err)
	}
	defer e.timer.Stop(ctx)
	
	// 创建简单的定时任务
	job := NewJobBuilder().
		WithID("example_job_1").
		WithName("示例任务1").
		WithDescription("每10秒执行一次的示例任务").
		WithCron("*/10 * * * * *"). // 每10秒执行一次
		WithFunc(func(ctx context.Context) error {
			log.Info("Example job 1 executed")
			return nil
		}).
		WithTimeout(5 * time.Second).
		Build()
	
	// 添加任务
	if err := e.timer.AddJob(job); err != nil {
		return fmt.Errorf("failed to add job: %w", err)
	}
	
	// 等待一段时间观察执行
	time.Sleep(35 * time.Second)
	
	// 列出所有任务
	jobs, err := e.timer.ListJobs()
	if err != nil {
		return fmt.Errorf("failed to list jobs: %w", err)
	}
	
	log.Infof("Total jobs: %d", len(jobs))
	for _, job := range jobs {
		log.Infof("Job: %s, Status: %s, RunCount: %d", 
			job.Name, job.Status.String(), job.RunCount)
	}
	
	return nil
}

// RunPeriodicExample 运行周期性任务示例
func (e *Example) RunPeriodicExample(ctx context.Context) error {
	log.Info("Starting periodic timer example...")

	// 启动定时器
	if err := e.timer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start timer: %w", err)
	}
	defer e.timer.Stop(ctx)

	helper := NewJobHelper()

	// 创建周期性任务
	periodicJob := helper.CreatePeriodicJob("data_sync", "数据同步任务", 30*time.Second, func(ctx context.Context) error {
		log.Info("Executing data sync task...")

		// 模拟数据同步
		time.Sleep(2 * time.Second)

		log.Info("Data sync task completed successfully")
		return nil
	})

	// 添加周期性任务
	if err := e.timer.AddJob(periodicJob); err != nil {
		return fmt.Errorf("failed to add periodic job: %w", err)
	}

	// 创建自定义任务
	customJob := helper.CreateCustomJob(
		"cleanup_task",
		"清理任务",
		"定期清理临时文件和过期数据",
		"0 0 */2 * * *", // 每2小时执行一次
		func(ctx context.Context) error {
			log.Info("Executing cleanup task...")

			// 模拟清理操作
			time.Sleep(1 * time.Second)

			log.Info("Cleanup task completed")
			return nil
		},
		10*time.Minute, // 超时时间
		2,              // 最大重试次数
	)

	// 添加自定义任务
	if err := e.timer.AddJob(customJob); err != nil {
		return fmt.Errorf("failed to add custom job: %w", err)
	}

	// 运行一段时间
	time.Sleep(2 * time.Minute)

	// 显示执行统计
	jobs, _ := e.timer.ListJobs()
	for _, job := range jobs {
		executions, _ := e.timer.GetJobExecutions(job.ID, 5)
		log.Infof("Job %s: RunCount=%d, FailCount=%d, Recent executions=%d",
			job.Name, job.RunCount, job.FailCount, len(executions))
	}

	return nil
}

// RunAdvancedExample 运行高级示例
func (e *Example) RunAdvancedExample(ctx context.Context) error {
	log.Info("Starting advanced timer example...")
	
	// 启动定时器
	if err := e.timer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start timer: %w", err)
	}
	defer e.timer.Stop(ctx)
	
	// 创建一个会失败的任务来演示重试机制
	failingJob := NewJobBuilder().
		WithID("failing_job").
		WithName("失败任务").
		WithDescription("演示重试机制的任务").
		WithCron("*/15 * * * * *"). // 每15秒执行一次
		WithFunc(func(ctx context.Context) error {
			log.Info("Executing failing job...")
			return fmt.Errorf("simulated failure")
		}).
		WithTimeout(5 * time.Second).
		WithMaxRetries(2).
		Build()
	
	// 添加失败任务
	if err := e.timer.AddJob(failingJob); err != nil {
		return fmt.Errorf("failed to add failing job: %w", err)
	}
	
	// 创建一个成功的任务
	successJob := NewJobBuilder().
		WithID("success_job").
		WithName("成功任务").
		WithDescription("总是成功的任务").
		WithCron("*/20 * * * * *"). // 每20秒执行一次
		WithFunc(func(ctx context.Context) error {
			log.Info("Executing success job...")
			time.Sleep(1 * time.Second)
			log.Info("Success job completed")
			return nil
		}).
		WithTimeout(10 * time.Second).
		Build()
	
	// 添加成功任务
	if err := e.timer.AddJob(successJob); err != nil {
		return fmt.Errorf("failed to add success job: %w", err)
	}
	
	// 运行一段时间观察重试机制
	time.Sleep(1 * time.Minute)
	
	// 手动触发任务
	log.Info("Manually triggering success job...")
	if err := e.timer.TriggerJob("success_job"); err != nil {
		log.Errorf("Failed to trigger job: %v", err)
	}
	
	// 禁用失败任务
	log.Info("Disabling failing job...")
	if err := e.timer.DisableJob("failing_job"); err != nil {
		log.Errorf("Failed to disable job: %v", err)
	}
	
	// 再运行一段时间
	time.Sleep(30 * time.Second)
	
	// 显示最终统计
	jobs, _ := e.timer.ListJobs()
	for _, job := range jobs {
		log.Infof("Final stats - Job: %s, Enabled: %t, RunCount: %d, FailCount: %d, LastError: %s", 
			job.Name, job.Enabled, job.RunCount, job.FailCount, job.LastError)
	}
	
	return nil
}

// RunAllExamples 运行所有示例
func RunAllExamples() error {
	example, err := NewExample()
	if err != nil {
		return fmt.Errorf("failed to create example: %w", err)
	}
	
	ctx := context.Background()
	
	log.Info("=== Running Basic Example ===")
	if err := example.RunBasicExample(ctx); err != nil {
		log.Errorf("Basic example failed: %v", err)
	}
	
	log.Info("=== Running Periodic Example ===")
	if err := example.RunPeriodicExample(ctx); err != nil {
		log.Errorf("Periodic example failed: %v", err)
	}
	
	log.Info("=== Running Advanced Example ===")
	if err := example.RunAdvancedExample(ctx); err != nil {
		log.Errorf("Advanced example failed: %v", err)
	}
	
	log.Info("All examples completed")
	return nil
}
