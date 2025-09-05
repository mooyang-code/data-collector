// Package app 注册表工具函数
package app

import (
	"fmt"
	"sort"
	"strings"

	"trpc.group/trpc-go/trpc-go/log"
)

// PrintAllRegistrations 打印所有已注册的应用和采集器
func PrintAllRegistrations() {
	log.Info("===== 已注册的应用和采集器 =====")
	
	// 打印已注册的应用
	PrintRegisteredApps()
	
	// 打印已注册的采集器
	PrintRegisteredCollectorsDetails()
	
	log.Info("================================")
}

// PrintRegisteredApps 打印所有已注册的应用
func PrintRegisteredApps() {
	registry := GetAppRegistry()
	apps := registry.GetAllApps()
	
	if len(apps) == 0 {
		log.Info("📦 没有已注册的应用")
		return
	}
	
	log.Info("📦 已注册的应用:")
	
	// 排序应用名称
	var appNames []string
	for name := range apps {
		appNames = append(appNames, name)
	}
	sort.Strings(appNames)
	
	for _, name := range appNames {
		app := apps[name]
		log.Infof("  • %s (%s) - %s", name, app.DisplayName, app.Description)
	}
}

// PrintRegisteredCollectorsDetails 打印所有已注册的采集器（详细版）
func PrintRegisteredCollectorsDetails() {
	output := PrintRegisteredCollectors()
	if output != "" {
		log.Info("📊 已注册的采集器:")
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if line != "" {
				log.Info(line)
			}
		}
	} else {
		log.Info("📊 没有已注册的采集器")
	}
}

// ValidateRegistrations 验证注册的应用和采集器
func ValidateRegistrations() error {
	registry := GetAppRegistry()
	apps := registry.GetAllApps()
	
	if len(apps) == 0 {
		return fmt.Errorf("没有注册任何应用")
	}
	
	// 检查每个应用是否有对应的采集器
	collectorRegistry := GetGlobalRegistry()
	for appName := range apps {
		collectors := collectorRegistry.GetCollectorsByExchange(appName)
		if len(collectors) == 0 {
			log.Warnf("应用 %s 没有注册任何采集器", appName)
		}
	}
	
	return nil
}

// GetRegistrationSummary 获取注册摘要信息
func GetRegistrationSummary() string {
	var lines []string
	
	// 统计应用数量
	appRegistry := GetAppRegistry()
	apps := appRegistry.GetAllApps()
	lines = append(lines, fmt.Sprintf("已注册应用数: %d", len(apps)))
	
	// 统计采集器数量
	collectorRegistry := GetGlobalRegistry()
	collectors := collectorRegistry.GetRegisteredCollectors()
	lines = append(lines, fmt.Sprintf("已注册采集器数: %d", len(collectors)))
	
	// 按交易所统计
	exchangeStats := make(map[string]int)
	for _, entry := range collectors {
		exchangeStats[entry.Exchange]++
	}
	
	if len(exchangeStats) > 0 {
		lines = append(lines, "\n按交易所统计:")
		for exchange, count := range exchangeStats {
			appDesc, exists := apps[exchange]
			displayName := exchange
			if exists && appDesc.DisplayName != "" {
				displayName = fmt.Sprintf("%s (%s)", exchange, appDesc.DisplayName)
			}
			lines = append(lines, fmt.Sprintf("  • %s: %d 个采集器", displayName, count))
		}
	}
	
	// 按数据类型统计
	dataTypeStats := make(map[string]int)
	for _, entry := range collectors {
		dataTypeStats[entry.DataType]++
	}
	
	if len(dataTypeStats) > 0 {
		lines = append(lines, "\n按数据类型统计:")
		for dataType, count := range dataTypeStats {
			lines = append(lines, fmt.Sprintf("  • %s: %d 个采集器", dataType, count))
		}
	}
	
	return strings.Join(lines, "\n")
}

// InitializeRegistries 初始化注册表（在应用启动时调用）
func InitializeRegistries() {
	// 打印所有注册信息
	PrintAllRegistrations()
	
	// 验证注册
	if err := ValidateRegistrations(); err != nil {
		log.Warnf("注册验证警告: %v", err)
	}
	
	// 打印摘要
	summary := GetRegistrationSummary()
	log.Info("\n" + summary)
}