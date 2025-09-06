package collector

import (
	"fmt"
	"sync"
)

type CollectorCreatorFunc func(config map[string]interface{}) (Collector, error)

type CollectorDescriptor struct {
	Source       string               // 数据源，如 "binance"
	SourceCN     string               // 数据源中文名
	DataType     string               // 数据类型，如 "kline"
	DataTypeCN   string               // 数据类型中文名
	MarketType   string               // 市场类型，如 "spot"
	MarketTypeCN string               // 市场类型中文名
	Description  string               // 描述
	Creator      CollectorCreatorFunc // 创建函数
}

type CollectorRegistry struct {
	collectors map[string]*CollectorDescriptor
	mu         sync.RWMutex
}

var globalRegistry = &CollectorRegistry{
	collectors: make(map[string]*CollectorDescriptor),
}

// RegisterWithDescriptor 使用描述符注册采集器
func RegisterWithDescriptor(descriptor *CollectorDescriptor) error {
	return globalRegistry.RegisterWithDescriptor(descriptor)
}

// Register 简化的注册方法
func Register(source, dataType string, creator CollectorCreatorFunc) error {
	return globalRegistry.RegisterWithDescriptor(&CollectorDescriptor{
		Source:   source,
		DataType: dataType,
		Creator:  creator,
	})
}

// GetRegistry 获取全局注册表
func GetRegistry() *CollectorRegistry {
	return globalRegistry
}

func (r *CollectorRegistry) RegisterWithDescriptor(descriptor *CollectorDescriptor) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	key := r.generateKey(descriptor.Source, descriptor.DataType)
	if _, exists := r.collectors[key]; exists {
		return fmt.Errorf("采集器 %s 已经注册", key)
	}
	
	r.collectors[key] = descriptor
	return nil
}

func (r *CollectorRegistry) Get(source, dataType string) (*CollectorDescriptor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	key := r.generateKey(source, dataType)
	descriptor, exists := r.collectors[key]
	if !exists {
		return nil, fmt.Errorf("采集器 %s 未注册", key)
	}
	
	return descriptor, nil
}

func (r *CollectorRegistry) List() []*CollectorDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	descriptors := make([]*CollectorDescriptor, 0, len(r.collectors))
	for _, desc := range r.collectors {
		descriptors = append(descriptors, desc)
	}
	
	return descriptors
}

func (r *CollectorRegistry) ListBySource(source string) []*CollectorDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var descriptors []*CollectorDescriptor
	for _, desc := range r.collectors {
		if desc.Source == source {
			descriptors = append(descriptors, desc)
		}
	}
	
	return descriptors
}

func (r *CollectorRegistry) CreateCollector(source, dataType string, config map[string]interface{}) (Collector, error) {
	descriptor, err := r.Get(source, dataType)
	if err != nil {
		return nil, err
	}
	
	if config == nil {
		config = make(map[string]interface{})
	}
	
	return descriptor.Creator(config)
}

func (r *CollectorRegistry) generateKey(source, dataType string) string {
	return fmt.Sprintf("%s:%s", source, dataType)
}

// CollectorCreatorBuilder 构建器模式，简化采集器注册
type CollectorCreatorBuilder struct {
	descriptor *CollectorDescriptor
}

func NewBuilder() *CollectorCreatorBuilder {
	return &CollectorCreatorBuilder{
		descriptor: &CollectorDescriptor{},
	}
}

func (b *CollectorCreatorBuilder) Source(source, sourceCN string) *CollectorCreatorBuilder {
	b.descriptor.Source = source
	b.descriptor.SourceCN = sourceCN
	return b
}

func (b *CollectorCreatorBuilder) DataType(dataType, dataTypeCN string) *CollectorCreatorBuilder {
	b.descriptor.DataType = dataType
	b.descriptor.DataTypeCN = dataTypeCN
	return b
}

func (b *CollectorCreatorBuilder) MarketType(marketType, marketTypeCN string) *CollectorCreatorBuilder {
	b.descriptor.MarketType = marketType
	b.descriptor.MarketTypeCN = marketTypeCN
	return b
}

func (b *CollectorCreatorBuilder) Description(description string) *CollectorCreatorBuilder {
	b.descriptor.Description = description
	return b
}

func (b *CollectorCreatorBuilder) Creator(creator CollectorCreatorFunc) *CollectorCreatorBuilder {
	b.descriptor.Creator = creator
	return b
}

func (b *CollectorCreatorBuilder) Register() error {
	return RegisterWithDescriptor(b.descriptor)
}