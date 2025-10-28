package config

import (
	"sync"

	"github.com/mooyang-code/go-commlib/apicache"
)

// 缓存管理相关变量
var (
	cache        apicache.ConfigCacher
	compass      *apicache.CompassTargetParse
	compassOnce  sync.Once
	addressMap   map[string]string
	addressMutex sync.RWMutex
)

// InitCache 初始化缓存系统
// addressMap: compass地址映射，key为服务名，value为地址
func InitCache(addressMap map[string]string, target string, apiCachers ...apicache.APICacher) error {
	// 初始化compass解析器
	compassOnce.Do(func() {
		compass, _ = apicache.NewCompassWithOptions(addressMap)
	})

	// 如果有地址映射需要更新
	if len(addressMap) > 0 {
		compass.UpdateAddressMap(addressMap)
	}

	// 构造缓存选项
	options := []apicache.TypedCacheOption{
		apicache.WithRequestTarget(target),
		apicache.WithCompassTargetParse(compass),
	}

	// 初始化缓存
	var err error
	cache, err = apicache.NewCacheWithOptions(options, apiCachers...)
	return err
}

// GetCache 获取缓存实例
func GetCache() apicache.ConfigCacher {
	return cache
}

// Query 查询缓存数据
func Query(schemaID, searchKey string) any {
	if cache == nil {
		return nil
	}
	return cache.GetDataItem(schemaID, searchKey)
}

// GetAll 获取指定schema的所有数据
func GetAll(schemaID string) any {
	if cache == nil {
		return nil
	}
	return cache.GetAll(schemaID)
}

// GetKeys 获取指定schema的所有key
func GetKeys(schemaID string) []string {
	if cache == nil {
		return nil
	}
	return cache.GetKeys(schemaID)
}

// UpdateCompassMap 更新compass地址映射
func UpdateCompassMap(newMap map[string]string) {
	addressMutex.Lock()
	defer addressMutex.Unlock()

	// 更新内存中的地址映射
	if addressMap == nil {
		addressMap = make(map[string]string)
	}
	for k, v := range newMap {
		addressMap[k] = v
	}

	// 更新compass解析器
	if compass != nil {
		compass.UpdateAddressMap(newMap)
	}
}

// SetServiceAddress 设置单个服务的地址
func SetServiceAddress(serviceName, address string) {
	addressMutex.Lock()
	defer addressMutex.Unlock()

	if addressMap == nil {
		addressMap = make(map[string]string)
	}
	addressMap[serviceName] = address

	if compass != nil {
		compass.UpdateSingleAddress(serviceName, address)
	}
}

// GetCompassMap 获取当前地址映射的副本
func GetCompassMap() map[string]string {
	addressMutex.RLock()
	defer addressMutex.RUnlock()

	if addressMap == nil {
		return nil
	}

	// 返回副本
	result := make(map[string]string)
	for k, v := range addressMap {
		result[k] = v
	}
	return result
}

// GetCompassParser 获取compass解析器（用于高级操作）
func GetCompassParser() *apicache.CompassTargetParse {
	compassOnce.Do(func() {
		if compass == nil {
			compass, _ = apicache.NewCompassWithOptions(nil)
		}
	})
	return compass
}
