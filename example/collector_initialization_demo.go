// Package main 采集器初始化演示（最好是中文注释！）
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mooyang-code/data-collector/internal/app"
)

func main() {
	fmt.Println("=== 采集器初始化和启动演示 ===")

	// 1. 创建应用工厂和管理器
	factory := app.NewAppFactory()
	manager := app.NewAppManager(factory)

	// 2. 加载配置文件（自动初始化应用和采集器）
	configPath := "../example-config.yaml"
	fmt.Printf("加载配置文件: %s\n", configPath)
	
	if err := manager.LoadConfig(configPath); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 3. 显示已注册的采集器
	fmt.Println("\n=== 已注册的采集器 ===")
	allApps := manager.GetAllApps()
	
	for _, app := range allApps {
		fmt.Printf("应用: %s\n", app.GetID())
		
		collectorManager := app.GetCollectorManager()
		if dataCollectorManager, ok := collectorManager.(*app.DataCollectorManager); ok {
			collectors := dataCollectorManager.GetCollectors()
			fmt.Printf("  采集器总数: %d\n", len(collectors))
			
			for collectorID, collectorInfo := range collectors {
				fmt.Printf("  - %s:\n", collectorID)
				fmt.Printf("    数据类型: %s\n", collectorInfo.Config.DataType)
				fmt.Printf("    市场类型: %s\n", collectorInfo.Config.MarketType)
				fmt.Printf("    启用状态: %v\n", collectorInfo.Config.Enabled)
				fmt.Printf("    触发间隔: %s\n", collectorInfo.Config.Schedule.TriggerInterval)
				fmt.Printf("    最大重试: %d\n", collectorInfo.Config.Schedule.MaxRetries)
				fmt.Printf("    实例状态: %v\n", collectorInfo.Instance != nil)
				fmt.Printf("    运行状态: %v\n", collectorInfo.Running)
				
				// 显示采集器特定配置
				if collectorInfo.Config.DataType == "klines" {
					fmt.Printf("    K线配置:\n")
					fmt.Printf("      时间间隔: %v\n", collectorInfo.Config.Config.Intervals)
					fmt.Printf("      缓冲区大小: %d\n", collectorInfo.Config.Config.BufferSize)
					fmt.Printf("      最大K线数: %d\n", collectorInfo.Config.Config.MaxKlinesPerSymbol)
					fmt.Printf("      启用回补: %v\n", collectorInfo.Config.Config.EnableBackfill)
				} else if collectorInfo.Config.DataType == "symbols" {
					fmt.Printf("    交易对配置:\n")
					fmt.Printf("      过滤器: %v\n", collectorInfo.Config.Config.Filters)
					fmt.Printf("      自动刷新: %v\n", collectorInfo.Config.Schedule.EnableAutoRefresh)
				}
				fmt.Println()
			}
		}
	}

	// 4. 启动应用管理器（这会初始化和启动所有采集器）
	fmt.Println("=== 启动应用管理器 ===")
	ctx := context.Background()
	
	if err := manager.Start(ctx); err != nil {
		log.Fatalf("启动应用管理器失败: %v", err)
	}

	// 5. 显示启动后的采集器状态
	fmt.Println("\n=== 采集器启动状态 ===")
	for _, app := range manager.GetAllApps() {
		fmt.Printf("应用: %s (运行中: %v)\n", app.GetID(), app.IsRunning())
		
		collectorManager := app.GetCollectorManager()
		if dataCollectorManager, ok := collectorManager.(*app.DataCollectorManager); ok {
			collectors := dataCollectorManager.GetCollectors()
			runningCount := 0
			errorCount := 0
			
			for collectorID, collectorInfo := range collectors {
				if collectorInfo.Instance != nil {
					if collectorInfo.Instance.IsRunning() {
						runningCount++
					}
				} else {
					errorCount++
				}
				
				fmt.Printf("  - %s:\n", collectorID)
				fmt.Printf("    实例创建: %v\n", collectorInfo.Instance != nil)
				if collectorInfo.Instance != nil {
					fmt.Printf("    采集器ID: %s\n", collectorInfo.Instance.GetID())
					fmt.Printf("    采集器类型: %s\n", collectorInfo.Instance.GetType())
					fmt.Printf("    数据类型: %s\n", collectorInfo.Instance.GetDataType())
					fmt.Printf("    运行状态: %v\n", collectorInfo.Instance.IsRunning())
				} else {
					fmt.Printf("    错误: 采集器实例创建失败\n")
				}
			}
			
			fmt.Printf("  运行中采集器: %d/%d\n", runningCount, len(collectors))
			if errorCount > 0 {
				fmt.Printf("  错误采集器: %d\n", errorCount)
			}
		}
		fmt.Println()
	}

	// 6. 按数据类型显示采集器
	fmt.Println("=== 按数据类型分组的采集器 ===")
	for _, app := range manager.GetAllApps() {
		collectorManager := app.GetCollectorManager()
		if dataCollectorManager, ok := collectorManager.(*app.DataCollectorManager); ok {
			// 显示交易对采集器
			symbolsCollectors := dataCollectorManager.GetCollectorsByType("symbols")
			if len(symbolsCollectors) > 0 {
				fmt.Printf("交易对采集器 (%d个):\n", len(symbolsCollectors))
				for _, collectorInfo := range symbolsCollectors {
					fmt.Printf("  - %s.%s (运行: %v)\n", 
						collectorInfo.AppName, collectorInfo.CollectorName, collectorInfo.Running)
				}
			}
			
			// 显示K线采集器
			klinesCollectors := dataCollectorManager.GetCollectorsByType("klines")
			if len(klinesCollectors) > 0 {
				fmt.Printf("K线采集器 (%d个):\n", len(klinesCollectors))
				for _, collectorInfo := range klinesCollectors {
					fmt.Printf("  - %s.%s (运行: %v)\n", 
						collectorInfo.AppName, collectorInfo.CollectorName, collectorInfo.Running)
				}
			}
		}
	}

	// 7. 运行一段时间观察采集器工作
	fmt.Println("\n=== 采集器运行中... ===")
	fmt.Println("采集器将运行 30 秒钟，观察其工作状态...")
	
	// 每5秒显示一次状态
	for i := 0; i < 6; i++ {
		time.Sleep(5 * time.Second)
		fmt.Printf("运行时间: %d秒\n", (i+1)*5)
		
		// 显示简要状态
		stats := manager.GetStats()
		fmt.Printf("  应用状态: 总数=%d, 运行中=%d, 错误=%d\n", 
			stats.TotalApps, stats.RunningApps, stats.ErrorApps)
	}

	// 8. 停止应用管理器
	fmt.Println("\n=== 停止应用管理器 ===")
	if err := manager.Stop(ctx); err != nil {
		log.Printf("停止应用管理器失败: %v", err)
	}

	// 9. 显示最终状态
	fmt.Println("\n=== 最终状态 ===")
	finalStats := manager.GetStats()
	fmt.Printf("应用状态: 总数=%d, 运行中=%d, 错误=%d\n", 
		finalStats.TotalApps, finalStats.RunningApps, finalStats.ErrorApps)

	for _, app := range manager.GetAllApps() {
		fmt.Printf("应用 %s: 运行中=%v\n", app.GetID(), app.IsRunning())
		
		collectorManager := app.GetCollectorManager()
		if dataCollectorManager, ok := collectorManager.(*app.DataCollectorManager); ok {
			collectors := dataCollectorManager.GetCollectors()
			runningCount := 0
			for _, collectorInfo := range collectors {
				if collectorInfo.Instance != nil && collectorInfo.Instance.IsRunning() {
					runningCount++
				}
			}
			fmt.Printf("  采集器运行状态: %d/%d\n", runningCount, len(collectors))
		}
	}

	fmt.Println("\n=== 演示完成 ===")
}
