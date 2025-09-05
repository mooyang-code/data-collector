// Package app 采集器注册中心
package app

import (
	"fmt"
	"sort"
	"strings"
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

// CollectorDescriptor 采集器描述信息
type CollectorDescriptor struct {
	Exchange     string               // 交易所名称
	ExchangeCN   string               // 交易所中文名
	DataType     string               // 数据类型
	DataTypeCN   string               // 数据类型中文名
	MarketType   string               // 市场类型
	MarketTypeCN string               // 市场类型中文名
	Description  string               // 描述信息
	Creator      CollectorCreatorFunc // 创建函数
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

// 存储描述信息的全局映射
var descriptorStore = make(map[string]*CollectorDescriptor)

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

// RegisterCollectorWithDescriptor 注册采集器（带描述信息）
func RegisterCollectorWithDescriptor(desc *CollectorDescriptor) error {
	if desc == nil {
		return fmt.Errorf("采集器描述不能为空")
	}

	if desc.Exchange == "" || desc.DataType == "" || desc.MarketType == "" {
		return fmt.Errorf("交易所、数据类型、市场类型不能为空")
	}

	if desc.Creator == nil {
		return fmt.Errorf("创建函数不能为空")
	}

	// 注册到全局注册中心
	RegisterCollectorCreator(desc.Exchange, desc.DataType, desc.MarketType, desc.Creator)

	// 存储描述信息
	storeDescriptor(desc)

	return nil
}

// storeDescriptor 存储描述信息
func storeDescriptor(desc *CollectorDescriptor) {
	key := fmt.Sprintf("%s.%s.%s", desc.Exchange, desc.DataType, desc.MarketType)
	descriptorStore[key] = desc
}

// GetCollectorDescriptor 获取采集器描述信息
func GetCollectorDescriptor(exchange, dataType, marketType string) *CollectorDescriptor {
	key := fmt.Sprintf("%s.%s.%s", exchange, dataType, marketType)
	return descriptorStore[key]
}

// ListCollectorsByExchange 列出指定交易所的所有采集器
func ListCollectorsByExchange(exchange string) []*CollectorDescriptor {
	var result []*CollectorDescriptor
	for _, desc := range descriptorStore {
		if desc.Exchange == exchange {
			result = append(result, desc)
		}
	}
	// 按数据类型和市场类型排序
	sort.Slice(result, func(i, j int) bool {
		if result[i].DataType != result[j].DataType {
			return result[i].DataType < result[j].DataType
		}
		return result[i].MarketType < result[j].MarketType
	})
	return result
}

// PrintRegisteredCollectors 打印所有已注册的采集器
func PrintRegisteredCollectors() string {
	var lines []string

	// 按交易所分组
	exchangeMap := make(map[string][]*CollectorDescriptor)
	for _, desc := range descriptorStore {
		exchangeMap[desc.Exchange] = append(exchangeMap[desc.Exchange], desc)
	}

	// 排序交易所名称
	var exchanges []string
	for exchange := range exchangeMap {
		exchanges = append(exchanges, exchange)
	}
	sort.Strings(exchanges)

	// 构建输出
	for _, exchange := range exchanges {
		collectors := exchangeMap[exchange]
		exchangeCN := ""
		if len(collectors) > 0 && collectors[0].ExchangeCN != "" {
			exchangeCN = fmt.Sprintf("(%s)", collectors[0].ExchangeCN)
		}
		lines = append(lines, fmt.Sprintf("📊 %s %s", exchange, exchangeCN))

		// 按数据类型和市场类型排序
		sort.Slice(collectors, func(i, j int) bool {
			if collectors[i].DataType != collectors[j].DataType {
				return collectors[i].DataType < collectors[j].DataType
			}
			return collectors[i].MarketType < collectors[j].MarketType
		})

		for _, desc := range collectors {
			dataTypeCN := ""
			if desc.DataTypeCN != "" {
				dataTypeCN = fmt.Sprintf("(%s)", desc.DataTypeCN)
			}
			marketTypeCN := ""
			if desc.MarketTypeCN != "" {
				marketTypeCN = fmt.Sprintf("(%s)", desc.MarketTypeCN)
			}
			lines = append(lines, fmt.Sprintf("  ├─ %s %s - %s %s",
				desc.DataType, dataTypeCN, desc.MarketType, marketTypeCN))
			if desc.Description != "" {
				lines = append(lines, fmt.Sprintf("  │  %s", desc.Description))
			}
		}
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// CollectorCreatorBuilder 采集器创建器构建器
type CollectorCreatorBuilder struct {
	descriptor *CollectorDescriptor
}

// NewCollectorCreatorBuilder 创建新的构建器
func NewCollectorCreatorBuilder() *CollectorCreatorBuilder {
	return &CollectorCreatorBuilder{
		descriptor: &CollectorDescriptor{},
	}
}

// WithExchange 设置交易所
func (b *CollectorCreatorBuilder) WithExchange(exchange, exchangeCN string) *CollectorCreatorBuilder {
	b.descriptor.Exchange = exchange
	b.descriptor.ExchangeCN = exchangeCN
	return b
}

// WithDataType 设置数据类型
func (b *CollectorCreatorBuilder) WithDataType(dataType, dataTypeCN string) *CollectorCreatorBuilder {
	b.descriptor.DataType = dataType
	b.descriptor.DataTypeCN = dataTypeCN
	return b
}

// WithMarketType 设置市场类型
func (b *CollectorCreatorBuilder) WithMarketType(marketType, marketTypeCN string) *CollectorCreatorBuilder {
	b.descriptor.MarketType = marketType
	b.descriptor.MarketTypeCN = marketTypeCN
	return b
}

// WithDescription 设置描述
func (b *CollectorCreatorBuilder) WithDescription(description string) *CollectorCreatorBuilder {
	b.descriptor.Description = description
	return b
}

// WithCreator 设置创建函数
func (b *CollectorCreatorBuilder) WithCreator(creator CollectorCreatorFunc) *CollectorCreatorBuilder {
	b.descriptor.Creator = creator
	return b
}

// Register 注册采集器
func (b *CollectorCreatorBuilder) Register() error {
	return RegisterCollectorWithDescriptor(b.descriptor)
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
