// Package app 采集器注册中心（最好是中文注释！）
package app

import (
	"fmt"
	"sync"

	"github.com/mooyang-code/data-collector/configs"
	"trpc.group/trpc-go/trpc-go/log"
)

// CollectorCreatorFunc 采集器创建函数类型
type CollectorCreatorFunc func(appName, collectorName string, config *configs.Collector) (Collector, error)

// CollectorRegistryEntry 采集器注册条目
type CollectorRegistryEntry struct {
	Exchange   string               // 交易所名称，如 "binance", "okx"
	DataType   string               // 数据类型，如 "symbols", "klines"
	MarketType string               // 市场类型，如 "spot", "futures"
	Creator    CollectorCreatorFunc // 创建函数
}

// CollectorRegistry 采集器注册中心
type CollectorRegistry struct {
	entries map[string]*CollectorRegistryEntry
	mutex   sync.RWMutex
}

// 全局注册中心实例
var globalRegistry = &CollectorRegistry{
	entries: make(map[string]*CollectorRegistryEntry),
}

// RegisterCollectorCreator 注册采集器创建器
func RegisterCollectorCreator(exchange, dataType, marketType string, creator CollectorCreatorFunc) {
	globalRegistry.Register(exchange, dataType, marketType, creator)
}

// Register 注册采集器创建器
func (r *CollectorRegistry) Register(exchange, dataType, marketType string, creator CollectorCreatorFunc) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	key := fmt.Sprintf("%s.%s.%s", exchange, dataType, marketType)
	if _, exists := r.entries[key]; exists {
		log.Warnf("采集器创建器已存在，将被覆盖: %s", key)
	}

	r.entries[key] = &CollectorRegistryEntry{
		Exchange:   exchange,
		DataType:   dataType,
		MarketType: marketType,
		Creator:    creator,
	}
	log.Infof("注册采集器创建器: %s", key)
}

// CreateCollector 创建采集器
func (r *CollectorRegistry) CreateCollector(appName, collectorName string, config *configs.Collector) (Collector, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	key := fmt.Sprintf("%s.%s.%s", appName, config.DataType, config.MarketType)
	entry, exists := r.entries[key]
	if !exists {
		return nil, fmt.Errorf("未找到采集器创建器: %s", key)
	}

	collector, err := entry.Creator(appName, collectorName, config)
	if err != nil {
		return nil, fmt.Errorf("创建采集器失败: %w", err)
	}
	log.Infof("创建采集器成功: %s", key)
	return collector, nil
}

// GetSupportedTypes 获取支持的采集器类型
func (r *CollectorRegistry) GetSupportedTypes() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	types := make([]string, 0, len(r.entries))
	for key := range r.entries {
		types = append(types, key)
	}
	return types
}

// GetRegisteredCollectors 获取已注册的采集器信息
func (r *CollectorRegistry) GetRegisteredCollectors() map[string]*CollectorRegistryEntry {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make(map[string]*CollectorRegistryEntry)
	for key, entry := range r.entries {
		result[key] = &CollectorRegistryEntry{
			Exchange:   entry.Exchange,
			DataType:   entry.DataType,
			MarketType: entry.MarketType,
			Creator:    entry.Creator,
		}
	}
	return result
}

// GetCollectorsByExchange 根据交易所获取采集器
func (r *CollectorRegistry) GetCollectorsByExchange(exchange string) map[string]*CollectorRegistryEntry {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make(map[string]*CollectorRegistryEntry)
	for key, entry := range r.entries {
		if entry.Exchange == exchange {
			result[key] = entry
		}
	}
	return result
}

// GetCollectorsByDataType 根据数据类型获取采集器
func (r *CollectorRegistry) GetCollectorsByDataType(dataType string) map[string]*CollectorRegistryEntry {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make(map[string]*CollectorRegistryEntry)
	for key, entry := range r.entries {
		if entry.DataType == dataType {
			result[key] = entry
		}
	}
	return result
}

// GetCollectorsByMarketType 根据市场类型获取采集器
func (r *CollectorRegistry) GetCollectorsByMarketType(marketType string) map[string]*CollectorRegistryEntry {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make(map[string]*CollectorRegistryEntry)
	for key, entry := range r.entries {
		if entry.MarketType == marketType {
			result[key] = entry
		}
	}
	return result
}

// IsSupported 检查是否支持指定的采集器类型
func (r *CollectorRegistry) IsSupported(exchange, dataType, marketType string) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	key := fmt.Sprintf("%s.%s.%s", exchange, dataType, marketType)
	_, exists := r.entries[key]
	return exists
}

// GetGlobalRegistry 获取全局注册中心
func GetGlobalRegistry() *CollectorRegistry {
	return globalRegistry
}

// CreateCollector 直接使用全局注册中心
func CreateCollector(appName, collectorName string, config *configs.Collector) (Collector, error) {
	return globalRegistry.CreateCollector(appName, collectorName, config)
}

func GetSupportedTypes() []string {
	return globalRegistry.GetSupportedTypes()
}

func IsSupported(exchange, dataType, marketType string) bool {
	return globalRegistry.IsSupported(exchange, dataType, marketType)
}

// MockCollectorConfig 模拟采集器配置（用于演示）
type MockCollectorConfig struct {
	DataType   string
	MarketType string
	Schedule   MockScheduleConfig
	Config     MockConfig
}

// MockScheduleConfig 模拟调度配置
type MockScheduleConfig struct {
	TriggerInterval   string
	EnableAutoRefresh bool
	MaxRetries        int
	RetryInterval     string
}

// MockConfig 模拟配置
type MockConfig struct {
	Filters    []string
	BufferSize int
}
