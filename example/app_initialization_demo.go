// Package main 应用初始化演示（最好是中文注释！）
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mooyang-code/data-collector/internal/app"
)

func main() {
	fmt.Println("=== 数据采集器应用初始化演示 ===")

	// 1. 创建应用工厂
	factory := app.NewAppFactory()
	
	// 2. 创建应用管理器
	manager := app.NewAppManager(factory)

	// 3. 加载配置文件（这会自动初始化应用）
	configPath := "../example-config.yaml"
	fmt.Printf("加载配置文件: %s\n", configPath)
	
	if err := manager.LoadConfig(configPath); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 4. 显示已初始化的应用
	fmt.Println("\n=== 已初始化的应用 ===")
	allApps := manager.GetAllApps()
	fmt.Printf("总应用数: %d\n", len(allApps))
	
	for _, app := range allApps {
		fmt.Printf("应用ID: %s\n", app.GetID())
		fmt.Printf("  状态: %s\n", app.GetStatus().State)
		fmt.Printf("  运行中: %v\n", app.IsRunning())
		
		// 显示采集器信息
		collectorManager := app.GetCollectorManager()
		if dataCollectorManager, ok := collectorManager.(*app.DataCollectorManager); ok {
			collectors := dataCollectorManager.GetCollectors()
			fmt.Printf("  采集器数量: %d\n", len(collectors))
			
			for collectorID, collectorInfo := range collectors {
				fmt.Printf("    - %s:\n", collectorID)
				fmt.Printf("      数据类型: %s\n", collectorInfo.Config.DataType)
				fmt.Printf("      市场类型: %s\n", collectorInfo.Config.MarketType)
				fmt.Printf("      触发间隔: %s\n", collectorInfo.Config.Schedule.TriggerInterval)
				fmt.Printf("      运行状态: %v\n", collectorInfo.Running)
			}
		}
		fmt.Println()
	}

	// 5. 启动应用管理器
	fmt.Println("=== 启动应用管理器 ===")
	ctx := context.Background()
	
	if err := manager.Start(ctx); err != nil {
		log.Fatalf("启动应用管理器失败: %v", err)
	}

	// 6. 显示启动后的状态
	fmt.Println("\n=== 启动后状态 ===")
	stats := manager.GetStats()
	fmt.Printf("总应用数: %d\n", stats.TotalApps)
	fmt.Printf("运行中应用数: %d\n", stats.RunningApps)
	fmt.Printf("错误应用数: %d\n", stats.ErrorApps)

	// 显示每个应用的详细状态
	for _, app := range manager.GetAllApps() {
		status := app.GetStatus()
		metrics := app.GetMetrics()
		
		fmt.Printf("\n应用 %s:\n", app.GetID())
		fmt.Printf("  状态: %s\n", status.State)
		fmt.Printf("  启动时间: %s\n", status.StartTime.Format("2006-01-02 15:04:05"))
		fmt.Printf("  运行时长: %s\n", metrics.Uptime)
		fmt.Printf("  最后错误: %s\n", status.LastError)
		
		// 显示采集器运行状态
		collectorManager := app.GetCollectorManager()
		if dataCollectorManager, ok := collectorManager.(*app.DataCollectorManager); ok {
			collectors := dataCollectorManager.GetCollectors()
			runningCount := 0
			for _, collectorInfo := range collectors {
				if collectorInfo.Running {
					runningCount++
				}
			}
			fmt.Printf("  运行中采集器: %d/%d\n", runningCount, len(collectors))
		}
	}

	// 7. 运行一段时间
	fmt.Println("\n=== 运行中... ===")
	fmt.Println("应用将运行 10 秒钟...")
	time.Sleep(10 * time.Second)

	// 8. 停止应用管理器
	fmt.Println("\n=== 停止应用管理器 ===")
	if err := manager.Stop(ctx); err != nil {
		log.Printf("停止应用管理器失败: %v", err)
	}

	// 9. 显示最终状态
	fmt.Println("\n=== 最终状态 ===")
	finalStats := manager.GetStats()
	fmt.Printf("总应用数: %d\n", finalStats.TotalApps)
	fmt.Printf("运行中应用数: %d\n", finalStats.RunningApps)
	fmt.Printf("错误应用数: %d\n", finalStats.ErrorApps)

	fmt.Println("\n=== 演示完成 ===")
}
