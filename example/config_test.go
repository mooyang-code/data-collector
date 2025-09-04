// Package example 配置加载示例（最好是中文注释！）
package main

import (
	"fmt"
	"log"

	"github.com/mooyang-code/data-collector/internal/app"
)

func main() {
	// 创建应用管理器
	manager := app.NewAppManager(nil)

	// 加载配置文件
	configPath := "../example-config.yaml"
	if err := manager.LoadConfig(configPath); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 获取配置并显示信息
	if appManager, ok := manager.(*app.AppManagerImpl); ok {
		config := appManager.GetConfig()
		if config != nil {
			fmt.Printf("系统配置:\n")
			fmt.Printf("  名称: %s\n", config.System.Name)
			fmt.Printf("  版本: %s\n", config.System.Version)
			fmt.Printf("  环境: %s\n", config.System.Environment)
			fmt.Printf("  调试模式: %v\n", config.System.Debug)

			fmt.Printf("\n启用的应用:\n")
			enabledApps := appManager.GetEnabledApps()
			for appName, appConfig := range enabledApps {
				fmt.Printf("  %s:\n", appName)
				fmt.Printf("    名称: %s\n", appConfig.Name)
				fmt.Printf("    启用状态: %v\n", appConfig.Enabled)
				fmt.Printf("    REST URL: %s\n", appConfig.BaseConfig.RestBaseURL)
				fmt.Printf("    WebSocket URL: %s\n", appConfig.BaseConfig.WsBaseURL)
				fmt.Printf("    日志级别: %s\n", appConfig.BaseConfig.LogLevel)

				fmt.Printf("    采集器:\n")
				for collectorName, collector := range appConfig.Collectors {
					if collector.Enabled {
						fmt.Printf("      %s:\n", collectorName)
						fmt.Printf("        数据类型: %s\n", collector.DataType)
						fmt.Printf("        市场类型: %s\n", collector.MarketType)
						fmt.Printf("        触发间隔: %s\n", collector.Schedule.TriggerInterval)
						fmt.Printf("        最大重试: %d\n", collector.Schedule.MaxRetries)
						if collector.DataType == "klines" {
							fmt.Printf("        时间间隔: %v\n", collector.Config.Intervals)
							fmt.Printf("        缓冲区大小: %d\n", collector.Config.BufferSize)
						}
					}
				}
			}

			fmt.Printf("\n持久化配置:\n")
			fmt.Printf("  后端: %s\n", config.Persistence.Backend)
			if config.Persistence.Backend == "parquet" {
				fmt.Printf("  Parquet配置:\n")
				fmt.Printf("    基础路径: %s\n", config.Persistence.Parquet.BasePath)
				fmt.Printf("    压缩方式: %s\n", config.Persistence.Parquet.Compression)
				fmt.Printf("    行组大小: %d\n", config.Persistence.Parquet.RowGroupSize)
				fmt.Printf("    页面大小: %d\n", config.Persistence.Parquet.PageSize)
				fmt.Printf("    刷新间隔: %s\n", config.Persistence.Parquet.FlushInterval)
			}
		}
	}

	// 测试获取特定应用配置
	fmt.Printf("\n测试获取Binance应用配置:\n")
	if appManager, ok := manager.(*app.AppManagerImpl); ok {
		binanceConfig, err := appManager.GetAppConfig("binance")
		if err != nil {
			fmt.Printf("获取Binance配置失败: %v\n", err)
		} else {
			fmt.Printf("Binance应用名称: %s\n", binanceConfig.Name)
			fmt.Printf("Binance启用状态: %v\n", binanceConfig.Enabled)

			// 获取启用的采集器
			if config := appManager.GetConfig(); config != nil {
				enabledCollectors, err := config.GetEnabledCollectors("binance")
				if err != nil {
					fmt.Printf("获取启用采集器失败: %v\n", err)
				} else {
					fmt.Printf("启用的采集器数量: %d\n", len(enabledCollectors))
					for name := range enabledCollectors {
						fmt.Printf("  - %s\n", name)
					}
				}
			}
		}
	}
}
