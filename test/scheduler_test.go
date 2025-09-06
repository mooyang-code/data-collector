package test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mooyang-code/data-collector/internal/core/scheduler"
)

// TestCronScheduler 测试Cron调度器功能
func TestCronScheduler(t *testing.T) {
	s := scheduler.NewCronScheduler()
	ctx := context.Background()

	// 启动调度器
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer s.Stop()

	t.Run("测试整点执行", func(t *testing.T) {
		var count int32
		
		// 添加每秒执行的任务（用于快速测试）
		err := s.AddCronTask("test-second", "*/2 * * * * *", func(ctx context.Context) error {
			atomic.AddInt32(&count, 1)
			fmt.Printf("任务执行: %v, 计数: %d\n", time.Now().Format("15:04:05"), count)
			return nil
		})
		if err != nil {
			t.Fatalf("Failed to add task: %v", err)
		}

		// 等待任务执行
		time.Sleep(5 * time.Second)
		
		finalCount := atomic.LoadInt32(&count)
		if finalCount < 2 {
			t.Errorf("Expected at least 2 executions, got %d", finalCount)
		}
	})

	t.Run("测试整分钟执行", func(t *testing.T) {
		// 添加整分钟执行的任务
		err := s.AddCronTask("test-minute", "0 * * * * *", func(ctx context.Context) error {
			fmt.Printf("整分钟任务执行: %v\n", time.Now().Format("15:04:05"))
			return nil
		})
		if err != nil {
			t.Fatalf("Failed to add minute task: %v", err)
		}

		// 获取任务状态
		status, err := s.GetTaskStatus("test-minute")
		if err != nil {
			t.Fatalf("Failed to get task status: %v", err)
		}
		
		fmt.Printf("任务状态: %+v\n", status)
		fmt.Printf("下次执行时间: %v\n", status.NextRun.Format("15:04:05"))
	})

	t.Run("测试时间间隔转换", func(t *testing.T) {
		// 测试不同的时间间隔
		intervals := []time.Duration{
			30 * time.Second,
			5 * time.Minute,
			1 * time.Hour,
			24 * time.Hour,
		}

		for _, interval := range intervals {
			taskName := fmt.Sprintf("interval-%v", interval)
			err := s.AddTask(taskName, interval, func(ctx context.Context) error {
				return nil
			})
			if err != nil {
				t.Errorf("Failed to add interval task %v: %v", interval, err)
				continue
			}

			status, _ := s.GetTaskStatus(taskName)
			fmt.Printf("间隔 %v 转换为 Cron: %s, 下次执行: %v\n", 
				interval, status.CronExpr, status.NextRun.Format("15:04:05"))
		}
	})

	// 列出所有任务
	tasks := s.ListTasks()
	fmt.Println("\n所有任务:")
	for name, status := range tasks {
		fmt.Printf("- %s: %s, 下次执行: %v\n", 
			name, status.CronExpr, status.NextRun.Format("15:04:05"))
	}
}

// TestCronExpression 测试Cron表达式
func TestCronExpression(t *testing.T) {
	testCases := []struct {
		name     string
		cronExpr string
		desc     string
	}{
		{"每分钟整点", "0 * * * * *", "每分钟的第0秒执行"},
		{"每小时整点", "0 0 * * * *", "每小时的第0分0秒执行"},
		{"每天午夜", "0 0 0 * * *", "每天0点0分0秒执行"},
		{"工作日9点", "0 0 9 * * 1-5", "周一到周五9点执行"},
		{"每15分钟", "0 */15 * * * *", "每15分钟的整点执行"},
		{"每月1号", "0 0 0 1 * *", "每月1号0点执行"},
	}

	s := scheduler.NewCronScheduler()
	ctx := context.Background()
	s.Start(ctx)
	defer s.Stop()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := s.AddCronTask(tc.name, tc.cronExpr, func(ctx context.Context) error {
				return nil
			})
			if err != nil {
				t.Errorf("Failed to add cron task %s: %v", tc.name, err)
				return
			}

			status, _ := s.GetTaskStatus(tc.name)
			fmt.Printf("%s (%s): 下次执行时间: %v\n", 
				tc.name, tc.desc, status.NextRun.Format("2006-01-02 15:04:05"))
		})
	}
}