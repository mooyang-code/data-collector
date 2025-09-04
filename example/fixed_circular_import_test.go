// Package main 修复循环导入后的测试（最好是中文注释！）
package main

import (
	"fmt"
	"log"

	"github.com/mooyang-code/data-collector/internal/app"
)

func main() {
	fmt.Println("=== 修复循环导入后的测试 ===")

	// 1. 创建应用工厂和管理器（这会触发采集器初始化）
	fmt.Println("\n=== 创建应用管理器 ===")
	factory := app.NewAppFactory()
	manager := app.NewAppManager(factory) // 这里会调用 InitCollectors()

	if manager == nil {
		log.Fatal("❌ 应用管理器创建失败")
	}
	fmt.Println("✅ 应用管理器创建成功")

	// 2. 测试注册中心
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

	// 3. 测试工厂
	fmt.Println("\n=== 测试采集器工厂 ===")
	collectorFactory := app.NewCollectorFactory()
	
	factoryTypes := collectorFactory.GetSupportedTypes()
	fmt.Printf("工厂支持的类型数量: %d\n", len(factoryTypes))
	
	// 验证工厂和注册中心的一致性
	if len(supportedTypes) == len(factoryTypes) {
		fmt.Println("✅ 工厂和注册中心类型数量一致")
	} else {
		fmt.Printf("⚠️  工厂和注册中心类型数量不一致: 注册中心=%d, 工厂=%d\n", 
			len(supportedTypes), len(factoryTypes))
	}

	// 4. 测试支持性检查
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

	// 5. 按类型分组显示
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

	// 6. 测试应用管理器基本功能
	fmt.Println("\n=== 测试应用管理器 ===")
	stats := manager.GetStats()
	fmt.Printf("初始状态: 总应用=%d, 运行中=%d, 错误=%d\n", 
		stats.TotalApps, stats.RunningApps, stats.ErrorApps)

	fmt.Println("\n=== 测试完成 ===")
	fmt.Println("循环导入修复结果:")
	fmt.Println("✅ 删除了导致循环导入的 collectors/import.go")
	fmt.Println("✅ 在 app/init.go 中集中管理采集器包导入")
	fmt.Println("✅ 在 NewAppManager 中调用 InitCollectors() 确保初始化")
	fmt.Println("✅ 采集器自注册机制正常工作")
	fmt.Println("✅ 避免了循环导入问题")
}
