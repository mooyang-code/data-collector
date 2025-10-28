package task

import (
	"context"
	"testing"
	"time"

	"github.com/mooyang-code/data-collector/pkg/logger"
	"github.com/mooyang-code/data-collector/pkg/model"
)

func TestTaskScheduler(t *testing.T) {
	// 创建调度器
	scheduler := NewTaskScheduler(logger.NewDefault())
	
	// 创建测试任务
	task := &model.Task{
		ID:       "test-task",
		Type:     model.TaskTypeKLine,
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Schedule: "*/1 * * * * *", // 每秒执行一次
	}
	
	// 计数器
	executedCount := 0
	
	// 创建处理器
	handler := func(ctx context.Context, t *model.Task) error {
		executedCount++
		return nil
	}
	
	// 启动调度器
	ctx := context.Background()
	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer scheduler.Stop(ctx)
	
	// 添加任务
	if err := scheduler.AddCronTask(task, task.Schedule, handler); err != nil {
		t.Fatalf("Failed to add cron task: %v", err)
	}
	
	// 等待任务执行几次
	time.Sleep(3 * time.Second)
	
	// 验证任务执行了
	if executedCount < 2 {
		t.Errorf("Expected at least 2 executions, got %d", executedCount)
	}
	
	// 获取任务状态
	status, err := scheduler.GetTaskStatus(task.ID)
	if err != nil {
		t.Fatalf("Failed to get task status: %v", err)
	}
	
	if status.TaskID != task.ID {
		t.Errorf("Expected task ID %s, got %s", task.ID, status.TaskID)
	}
	
	if status.RunCount == 0 {
		t.Error("Expected run count > 0")
	}
	
	// 移除任务
	if err := scheduler.RemoveTask(task.ID); err != nil {
		t.Fatalf("Failed to remove task: %v", err)
	}
	
	// 验证任务已被移除
	_, err = scheduler.GetTaskStatus(task.ID)
	if err == nil {
		t.Error("Expected error when getting status of removed task")
	}
}

func TestTaskSchedulerWithInterval(t *testing.T) {
	// 创建调度器
	scheduler := NewTaskScheduler(logger.NewDefault())
	
	// 创建测试任务
	task := &model.Task{
		ID:       "test-interval-task",
		Type:     model.TaskTypeKLine,
		Exchange: "binance",
		Symbol:   "ETHUSDT",
		Interval: "1s", // 每秒执行一次
	}
	
	// 计数器
	executedCount := 0
	
	// 创建处理器
	handler := func(ctx context.Context, t *model.Task) error {
		executedCount++
		return nil
	}
	
	// 启动调度器
	ctx := context.Background()
	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer scheduler.Stop(ctx)
	
	// 添加任务
	if err := scheduler.AddTask(task, time.Second, handler); err != nil {
		t.Fatalf("Failed to add interval task: %v", err)
	}
	
	// 等待任务执行几次
	time.Sleep(3 * time.Second)
	
	// 验证任务执行了
	if executedCount < 2 {
		t.Errorf("Expected at least 2 executions, got %d", executedCount)
	}
	
	// 清理
	scheduler.RemoveTask(task.ID)
}