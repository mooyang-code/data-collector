package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mooyang-code/data-collector/internal/core/collector"
)

// ExampleCollector 示例采集器，展示如何使用Cron功能
type ExampleCollector struct {
	*collector.BaseCollector
}

func NewExampleCollector() *ExampleCollector {
	return &ExampleCollector{
		BaseCollector: collector.NewBaseCollector(
			"example_collector",
			"demo",
			"example",
		),
	}
}

func (c *ExampleCollector) Initialize(ctx context.Context) error {
	log.Println("初始化示例采集器...")
	
	// 调用基类初始化
	if err := c.BaseCollector.Initialize(ctx); err != nil {
		return err
	}
	
	// 添加不同类型的定时器
	
	// 1. 每分钟整点执行（适合需要整点采集的数据）
	if err := c.AddTimer("minute_task", 1*time.Minute, c.minuteTask); err != nil {
		return err
	}
	
	// 2. 每5分钟整点执行
	if err := c.AddTimer("5min_task", 5*time.Minute, c.fiveMinuteTask); err != nil {
		return err
	}
	
	// 3. 每小时整点执行
	if err := c.AddTimer("hourly_task", 1*time.Hour, c.hourlyTask); err != nil {
		return err
	}
	
	// 4. 使用Cron表达式 - 每天上午9点执行
	if err := c.AddCronTimer("daily_report", "0 0 9 * * *", c.dailyReport); err != nil {
		return err
	}
	
	// 5. 使用Cron表达式 - 工作日每天下午6点执行
	if err := c.AddCronTimer("weekday_summary", "0 0 18 * * 1-5", c.weekdaySummary); err != nil {
		return err
	}
	
	// 6. 使用Cron表达式 - 每月1号凌晨2点执行
	if err := c.AddCronTimer("monthly_cleanup", "0 0 2 1 * *", c.monthlyCleanup); err != nil {
		return err
	}
	
	// 7. 测试用 - 每10秒执行（用于演示）
	if err := c.AddTimer("test_task", 10*time.Second, c.testTask); err != nil {
		return err
	}
	
	log.Println("示例采集器初始化完成")
	return nil
}

// 定时任务实现
func (c *ExampleCollector) minuteTask(ctx context.Context) error {
	log.Printf("[整分钟任务] 执行时间: %s", time.Now().Format("15:04:05"))
	// 这里执行实际的数据采集逻辑
	return nil
}

func (c *ExampleCollector) fiveMinuteTask(ctx context.Context) error {
	log.Printf("[5分钟任务] 执行时间: %s", time.Now().Format("15:04:05"))
	// 例如：采集5分钟K线数据
	return nil
}

func (c *ExampleCollector) hourlyTask(ctx context.Context) error {
	log.Printf("[整点任务] 执行时间: %s", time.Now().Format("15:04:05"))
	// 例如：生成小时报表
	return nil
}

func (c *ExampleCollector) dailyReport(ctx context.Context) error {
	log.Printf("[每日报告] 执行时间: %s", time.Now().Format("2006-01-02 15:04:05"))
	// 生成每日报告
	return nil
}

func (c *ExampleCollector) weekdaySummary(ctx context.Context) error {
	log.Printf("[工作日总结] 执行时间: %s", time.Now().Format("2006-01-02 15:04:05"))
	// 生成工作日总结
	return nil
}

func (c *ExampleCollector) monthlyCleanup(ctx context.Context) error {
	log.Printf("[月度清理] 执行时间: %s", time.Now().Format("2006-01-02 15:04:05"))
	// 执行月度数据清理
	return nil
}

func (c *ExampleCollector) testTask(ctx context.Context) error {
	log.Printf("[测试任务] 执行时间: %s （每10秒）", time.Now().Format("15:04:05"))
	return nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("启动Cron采集器示例...")
	
	// 创建采集器
	collector := NewExampleCollector()
	
	ctx := context.Background()
	
	// 初始化
	if err := collector.Initialize(ctx); err != nil {
		log.Fatalf("初始化失败: %v", err)
	}
	
	// 启动
	if err := collector.Start(ctx); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
	
	// 显示状态
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		
		for range ticker.C {
			status := collector.GetStatus()
			fmt.Printf("\n===== 采集器状态 =====\n")
			fmt.Printf("ID: %s, 运行中: %v\n", status.ID, status.IsRunning)
			fmt.Printf("定时器:\n")
			for name, timer := range status.Timers {
				fmt.Printf("  - %s: 下次执行 %s, 已执行 %d 次, 错误 %d 次\n",
					name,
					timer.NextRun.Format("15:04:05"),
					timer.RunCount,
					timer.ErrorCount,
				)
			}
			fmt.Println("====================")
		}
	}()
	
	// 等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	log.Println("采集器运行中，按 Ctrl+C 退出...")
	log.Println("提示：")
	log.Println("- 测试任务每10秒执行一次")
	log.Println("- 整分钟任务在每分钟的00秒执行")
	log.Println("- 5分钟任务在 :00, :05, :10 等时间执行")
	log.Println("- 查看下次执行时间了解调度计划")
	
	<-sigChan
	
	// 停止
	log.Println("\n正在停止采集器...")
	if err := collector.Stop(ctx); err != nil {
		log.Printf("停止失败: %v", err)
	}
	
	log.Println("示例程序结束")
}