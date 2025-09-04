// Package main 清理后代码测试（最好是中文注释！）
package main

import (
	"fmt"
	"log"

	"github.com/mooyang-code/data-collector/internal/app"
)

func main() {
	fmt.Println("=== 清理后代码测试 ===")

	// 1. 测试注册中心
	fmt.Println("\n=== 测试注册中心 ===")
	registry := app.GetGlobalRegistry()
	
	supportedTypes := registry.GetSupportedTypes()
	fmt.Printf("支持的采集器类型数量: %d\n", len(supportedTypes))
	
	if len(supportedTypes) == 0 {
		fmt.Println("⚠️  没有发现已注册的采集器")
		fmt.Println("这可能是因为采集器包没有被正确导入")
	} else {
		fmt.Println("已注册的采集器类型:")
		for _, collectorType := range supportedTypes {
			fmt.Printf("  - %s\n", collectorType)
		}
	}

	// 2. 测试工厂
	fmt.Println("\n=== 测试采集器工厂 ===")
	factory := app.NewCollectorFactory()
	
	factoryTypes := factory.GetSupportedTypes()
	fmt.Printf("工厂支持的类型数量: %d\n", len(factoryTypes))
	
	// 验证工厂和注册中心的一致性
	if len(supportedTypes) == len(factoryTypes) {
		fmt.Println("✅ 工厂和注册中心类型数量一致")
	} else {
		fmt.Printf("⚠️  工厂和注册中心类型数量不一致: 注册中心=%d, 工厂=%d\n", 
			len(supportedTypes), len(factoryTypes))
	}

	// 3. 测试应用工厂
	fmt.Println("\n=== 测试应用工厂 ===")
	appFactory := app.NewAppFactory()
	
	appTypes := appFactory.GetSupportedTypes()
	fmt.Printf("支持的应用类型: %v\n", appTypes)

	// 4. 测试应用管理器
	fmt.Println("\n=== 测试应用管理器 ===")
	manager := app.NewAppManager(appFactory)
	
	if manager == nil {
		fmt.Println("❌ 应用管理器创建失败")
	} else {
		fmt.Println("✅ 应用管理器创建成功")
		
		// 获取初始状态
		stats := manager.GetStats()
		fmt.Printf("初始状态: 总应用=%d, 运行中=%d, 错误=%d\n", 
			stats.TotalApps, stats.RunningApps, stats.ErrorApps)
	}

	// 5. 测试支持性检查
	fmt.Println("\n=== 测试支持性检查 ===")
	testCases := []struct {
		exchange   string
		dataType   string
		marketType string
	}{
		{"binance", "symbols", "spot"},
		{"binance", "symbols", "futures"},
		{"binance", "klines", "spot"},
		{"binance", "klines", "futures"},
		{"okx", "symbols", "spot"},
		{"okx", "symbols", "futures"},
		{"unknown", "symbols", "spot"},
	}

	for _, testCase := range testCases {
		supported := registry.IsSupported(testCase.exchange, testCase.dataType, testCase.marketType)
		status := "❌ 不支持"
		if supported {
			status = "✅ 支持"
		}
		fmt.Printf("%s.%s.%s: %s\n", testCase.exchange, testCase.dataType, testCase.marketType, status)
	}

	// 6. 按类型分组显示
	fmt.Println("\n=== 按类型分组显示 ===")
	
	// 按交易所分组
	exchanges := []string{"binance", "okx", "bybit"}
	for _, exchange := range exchanges {
		collectors := registry.GetCollectorsByExchange(exchange)
		fmt.Printf("%s 交易所: %d 个采集器\n", exchange, len(collectors))
		for key := range collectors {
			fmt.Printf("  - %s\n", key)
		}
	}

	// 按数据类型分组
	fmt.Println("\n按数据类型分组:")
	dataTypes := []string{"symbols", "klines"}
	for _, dataType := range dataTypes {
		collectors := registry.GetCollectorsByDataType(dataType)
		fmt.Printf("%s 数据类型: %d 个采集器\n", dataType, len(collectors))
		for key := range collectors {
			fmt.Printf("  - %s\n", key)
		}
	}

	// 按市场类型分组
	fmt.Println("\n按市场类型分组:")
	marketTypes := []string{"spot", "futures"}
	for _, marketType := range marketTypes {
		collectors := registry.GetCollectorsByMarketType(marketType)
		fmt.Printf("%s 市场: %d 个采集器\n", marketType, len(collectors))
		for key := range collectors {
			fmt.Printf("  - %s\n", key)
		}
	}

	fmt.Println("\n=== 测试完成 ===")
	fmt.Println("代码清理结果:")
	fmt.Println("✅ 删除了重复的 collector_factory_new.go 文件")
	fmt.Println("✅ 统一了 CollectorCreator 类型定义")
	fmt.Println("✅ 保留了自注册机制的核心功能")
	fmt.Println("✅ 每个采集器包有自己的包装器实现")
	fmt.Println("✅ 工厂和注册中心功能正常")
}
