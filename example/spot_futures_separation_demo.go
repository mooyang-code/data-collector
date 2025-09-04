// Package main 现货合约采集器分离演示（最好是中文注释！）
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/mooyang-code/data-collector/internal/app"
)

func main() {
	fmt.Println("=== 现货和合约采集器分离演示 ===")

	// 1. 创建应用工厂和管理器
	factory := app.NewAppFactory()
	manager := app.NewAppManager(factory)

	// 2. 加载配置文件
	configPath := "../configs/config.yaml"
	fmt.Printf("加载配置文件: %s\n", configPath)
	
	if err := manager.LoadConfig(configPath); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 3. 分析采集器的分离情况
	fmt.Println("\n=== 采集器分离分析 ===")
	allApps := manager.GetAllApps()
	
	for _, app := range allApps {
		fmt.Printf("应用: %s\n", app.GetID())
		
		collectorManager := app.GetCollectorManager()
		if dataCollectorManager, ok := collectorManager.(*app.DataCollectorManager); ok {
			collectors := dataCollectorManager.GetCollectors()
			
			// 按市场类型分组统计
			spotCollectors := make(map[string]*app.CollectorInfo)
			futuresCollectors := make(map[string]*app.CollectorInfo)
			
			for collectorID, collectorInfo := range collectors {
				switch collectorInfo.Config.MarketType {
				case "spot":
					spotCollectors[collectorID] = collectorInfo
				case "futures":
					futuresCollectors[collectorID] = collectorInfo
				}
			}
			
			fmt.Printf("  总采集器数: %d\n", len(collectors))
			fmt.Printf("  现货采集器数: %d\n", len(spotCollectors))
			fmt.Printf("  合约采集器数: %d\n", len(futuresCollectors))
			
			// 显示现货采集器详情
			if len(spotCollectors) > 0 {
				fmt.Println("  \n  现货采集器:")
				for collectorID, collectorInfo := range spotCollectors {
					fmt.Printf("    - %s:\n", collectorID)
					fmt.Printf("      数据类型: %s\n", collectorInfo.Config.DataType)
					fmt.Printf("      市场类型: %s\n", collectorInfo.Config.MarketType)
					fmt.Printf("      API地址: %s\n", collectorInfo.Config.RestBaseURL)
					fmt.Printf("      启用状态: %v\n", collectorInfo.Config.Enabled)
					
					if collectorInfo.Config.DataType == "symbols" {
						fmt.Printf("      持久化文件: %s\n", collectorInfo.Config.Config.PersistenceFile)
					} else if collectorInfo.Config.DataType == "klines" {
						fmt.Printf("      时间间隔: %v\n", collectorInfo.Config.Config.Intervals)
						fmt.Printf("      缓冲区大小: %d\n", collectorInfo.Config.Config.BufferSize)
					}
				}
			}
			
			// 显示合约采集器详情
			if len(futuresCollectors) > 0 {
				fmt.Println("  \n  合约采集器:")
				for collectorID, collectorInfo := range futuresCollectors {
					fmt.Printf("    - %s:\n", collectorID)
					fmt.Printf("      数据类型: %s\n", collectorInfo.Config.DataType)
					fmt.Printf("      市场类型: %s\n", collectorInfo.Config.MarketType)
					fmt.Printf("      API地址: %s\n", collectorInfo.Config.RestBaseURL)
					fmt.Printf("      启用状态: %v\n", collectorInfo.Config.Enabled)
					
					if collectorInfo.Config.DataType == "symbols" {
						fmt.Printf("      持久化文件: %s\n", collectorInfo.Config.Config.PersistenceFile)
					} else if collectorInfo.Config.DataType == "klines" {
						fmt.Printf("      时间间隔: %v\n", collectorInfo.Config.Config.Intervals)
						fmt.Printf("      缓冲区大小: %d\n", collectorInfo.Config.Config.BufferSize)
					}
				}
			}
		}
		fmt.Println()
	}

	// 4. 启动应用管理器
	fmt.Println("=== 启动应用管理器 ===")
	ctx := context.Background()
	
	if err := manager.Start(ctx); err != nil {
		log.Fatalf("启动应用管理器失败: %v", err)
	}

	// 5. 显示采集器创建结果
	fmt.Println("\n=== 采集器创建结果 ===")
	for _, app := range manager.GetAllApps() {
		fmt.Printf("应用: %s\n", app.GetID())
		
		collectorManager := app.GetCollectorManager()
		if dataCollectorManager, ok := collectorManager.(*app.DataCollectorManager); ok {
			collectors := dataCollectorManager.GetCollectors()
			
			// 统计创建成功和失败的采集器
			spotSuccess := 0
			spotFailed := 0
			futuresSuccess := 0
			futuresFailed := 0
			
			for collectorID, collectorInfo := range collectors {
				isSuccess := collectorInfo.Instance != nil
				
				switch collectorInfo.Config.MarketType {
				case "spot":
					if isSuccess {
						spotSuccess++
					} else {
						spotFailed++
					}
				case "futures":
					if isSuccess {
						futuresSuccess++
					} else {
						futuresFailed++
					}
				}
				
				fmt.Printf("  - %s:\n", collectorID)
				fmt.Printf("    市场类型: %s\n", collectorInfo.Config.MarketType)
				fmt.Printf("    创建状态: %v\n", isSuccess)
				if isSuccess {
					fmt.Printf("    采集器类型: %s\n", collectorInfo.Instance.GetType())
					fmt.Printf("    运行状态: %v\n", collectorInfo.Instance.IsRunning())
				} else {
					fmt.Printf("    错误: 采集器实例创建失败\n")
				}
			}
			
			fmt.Printf("\n  创建结果统计:\n")
			fmt.Printf("    现货采集器: 成功=%d, 失败=%d\n", spotSuccess, spotFailed)
			fmt.Printf("    合约采集器: 成功=%d, 失败=%d\n", futuresSuccess, futuresFailed)
		}
		fmt.Println()
	}

	// 6. 显示支持的采集器类型
	fmt.Println("=== 支持的采集器类型 ===")
	collectorFactory := app.NewCollectorFactory()
	supportedTypes := collectorFactory.GetSupportedTypes()
	
	fmt.Printf("总共支持 %d 种采集器类型:\n", len(supportedTypes))
	for _, collectorType := range supportedTypes {
		fmt.Printf("  - %s\n", collectorType)
	}

	// 7. 按数据类型和市场类型分组显示
	fmt.Println("\n=== 按类型分组的采集器 ===")
	for _, app := range manager.GetAllApps() {
		collectorManager := app.GetCollectorManager()
		if dataCollectorManager, ok := collectorManager.(*app.DataCollectorManager); ok {
			
			// 现货交易对采集器
			spotSymbolsCollectors := dataCollectorManager.GetCollectorsByType("symbols")
			spotSymbols := 0
			for _, collector := range spotSymbolsCollectors {
				if collector.Config.MarketType == "spot" {
					spotSymbols++
				}
			}
			
			// 合约交易对采集器
			futuresSymbols := 0
			for _, collector := range spotSymbolsCollectors {
				if collector.Config.MarketType == "futures" {
					futuresSymbols++
				}
			}
			
			// K线采集器
			klinesCollectors := dataCollectorManager.GetCollectorsByType("klines")
			spotKlines := 0
			futuresKlines := 0
			for _, collector := range klinesCollectors {
				if collector.Config.MarketType == "spot" {
					spotKlines++
				} else if collector.Config.MarketType == "futures" {
					futuresKlines++
				}
			}
			
			fmt.Printf("应用 %s 的采集器分布:\n", app.GetID())
			fmt.Printf("  现货交易对采集器: %d 个\n", spotSymbols)
			fmt.Printf("  合约交易对采集器: %d 个\n", futuresSymbols)
			fmt.Printf("  现货K线采集器: %d 个\n", spotKlines)
			fmt.Printf("  合约K线采集器: %d 个\n", futuresKlines)
		}
	}

	// 8. 停止应用管理器
	fmt.Println("\n=== 停止应用管理器 ===")
	if err := manager.Stop(ctx); err != nil {
		log.Printf("停止应用管理器失败: %v", err)
	}

	fmt.Println("\n=== 演示完成 ===")
	fmt.Println("总结:")
	fmt.Println("1. 现货和合约采集器在配置文件中是完全分离的")
	fmt.Println("2. 它们使用不同的API端点（api.binance.com vs fapi.binance.com）")
	fmt.Println("3. 数据存储到不同的文件中")
	fmt.Println("4. 采集器工厂支持按 appName.dataType.marketType 创建具体实例")
	fmt.Println("5. 这种设计便于独立管理现货和合约数据的采集")
}
