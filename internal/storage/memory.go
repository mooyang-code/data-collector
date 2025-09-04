// Package storage 内存存储实现
package storage

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"trpc.group/trpc-go/trpc-go/log"
)

// MemoryBackend 内存存储后端
type MemoryBackend struct {
	config *Config
	
	// 数据存储 (按类型分组)
	data map[string][]*DataRecord
	
	// 统计信息
	stats *StorageStats
	
	// 互斥锁
	mu sync.RWMutex
	
	// 是否已初始化
	initialized bool
}

// NewMemoryBackend 创建内存存储后端
func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		data: make(map[string][]*DataRecord),
		stats: &StorageStats{
			Backend:   "memory",
			Healthy:   true,
			Connected: true,
			Extra:     make(map[string]interface{}),
		},
	}
}

// Initialize 初始化存储后端
func (m *MemoryBackend) Initialize(config *Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.initialized {
		return fmt.Errorf("内存存储后端已经初始化")
	}
	
	m.config = config
	m.initialized = true
	
	log.Infof("内存存储后端初始化完成")
	return nil
}

// Store 存储单条数据
func (m *MemoryBackend) Store(ctx context.Context, data *DataRecord) error {
	if !m.initialized {
		return fmt.Errorf("内存存储后端未初始化")
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// 检查容量限制
	if m.config != nil && m.config.Memory != nil {
		totalRecords := m.getTotalRecordsLocked()
		if totalRecords >= int64(m.config.Memory.MaxEntries) {
			// 删除最旧的记录
			m.removeOldestRecordLocked()
		}
	}
	
	// 存储数据
	dataType := data.Type
	if m.data[dataType] == nil {
		m.data[dataType] = make([]*DataRecord, 0)
	}
	
	// 复制数据记录
	record := &DataRecord{
		Type:      data.Type,
		Exchange:  data.Exchange,
		Symbol:    data.Symbol,
		Timestamp: data.Timestamp,
		Data:      make(map[string]interface{}),
		Metadata:  make(map[string]string),
	}
	
	// 深拷贝数据
	for k, v := range data.Data {
		record.Data[k] = v
	}
	for k, v := range data.Metadata {
		record.Metadata[k] = v
	}
	
	m.data[dataType] = append(m.data[dataType], record)
	
	// 更新统计信息
	m.stats.TotalRecords++
	m.stats.LastWriteTime = time.Now()
	
	return nil
}

// StoreBatch 批量存储数据
func (m *MemoryBackend) StoreBatch(ctx context.Context, data []*DataRecord) error {
	if !m.initialized {
		return fmt.Errorf("内存存储后端未初始化")
	}
	
	for _, record := range data {
		if err := m.Store(ctx, record); err != nil {
			return fmt.Errorf("批量存储失败: %w", err)
		}
	}
	
	return nil
}

// Query 查询数据
func (m *MemoryBackend) Query(ctx context.Context, query *QueryRequest) (*QueryResult, error) {
	if !m.initialized {
		return nil, fmt.Errorf("内存存储后端未初始化")
	}
	
	startTime := time.Now()
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var allRecords []*DataRecord
	
	// 收集匹配的记录
	if query.Type != "" {
		// 查询特定类型
		if records, exists := m.data[query.Type]; exists {
			allRecords = append(allRecords, records...)
		}
	} else {
		// 查询所有类型
		for _, records := range m.data {
			allRecords = append(allRecords, records...)
		}
	}
	
	// 过滤记录
	var filteredRecords []*DataRecord
	for _, record := range allRecords {
		if m.matchesFilter(record, query) {
			filteredRecords = append(filteredRecords, record)
		}
	}
	
	// 排序
	if query.OrderBy != "" {
		m.sortRecords(filteredRecords, query.OrderBy, query.OrderDesc)
	}
	
	// 分页
	total := int64(len(filteredRecords))
	offset := query.Offset
	limit := query.Limit
	
	if limit <= 0 {
		limit = 100 // 默认限制
	}
	
	if offset < 0 {
		offset = 0
	}
	
	end := offset + limit
	if end > len(filteredRecords) {
		end = len(filteredRecords)
	}
	
	var resultRecords []*DataRecord
	if offset < len(filteredRecords) {
		resultRecords = filteredRecords[offset:end]
	}
	
	// 更新统计信息
	m.stats.LastReadTime = time.Now()
	
	return &QueryResult{
		Records:  resultRecords,
		Total:    total,
		HasMore:  end < len(filteredRecords),
		Duration: time.Since(startTime),
	}, nil
}

// GetStats 获取统计信息
func (m *MemoryBackend) GetStats() *StorageStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// 计算存储大小（估算）
	storageSize := int64(0)
	for _, records := range m.data {
		storageSize += int64(len(records) * 1024) // 估算每条记录1KB
	}
	
	stats := *m.stats
	stats.StorageSize = storageSize
	stats.Extra["data_types"] = len(m.data)
	
	return &stats
}

// HealthCheck 健康检查
func (m *MemoryBackend) HealthCheck() error {
	if !m.initialized {
		return fmt.Errorf("内存存储后端未初始化")
	}
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	m.stats.Healthy = true
	m.stats.Connected = true
	
	return nil
}

// Close 关闭存储后端
func (m *MemoryBackend) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !m.initialized {
		return nil
	}
	
	// 清空数据
	m.data = make(map[string][]*DataRecord)
	m.stats.Connected = false
	m.initialized = false
	
	log.Infof("内存存储后端已关闭")
	return nil
}

// 辅助方法

// getTotalRecordsLocked 获取总记录数（需要持有锁）
func (m *MemoryBackend) getTotalRecordsLocked() int64 {
	total := int64(0)
	for _, records := range m.data {
		total += int64(len(records))
	}
	return total
}

// removeOldestRecordLocked 删除最旧的记录（需要持有锁）
func (m *MemoryBackend) removeOldestRecordLocked() {
	var oldestType string
	var oldestIndex int
	var oldestTime time.Time
	
	first := true
	for dataType, records := range m.data {
		if len(records) == 0 {
			continue
		}
		
		for i, record := range records {
			if first || record.Timestamp.Before(oldestTime) {
				oldestType = dataType
				oldestIndex = i
				oldestTime = record.Timestamp
				first = false
			}
		}
	}
	
	if !first {
		// 删除最旧的记录
		records := m.data[oldestType]
		m.data[oldestType] = append(records[:oldestIndex], records[oldestIndex+1:]...)
		m.stats.TotalRecords--
	}
}

// matchesFilter 检查记录是否匹配过滤条件
func (m *MemoryBackend) matchesFilter(record *DataRecord, query *QueryRequest) bool {
	if query.Exchange != "" && record.Exchange != query.Exchange {
		return false
	}
	
	if query.Symbol != "" && record.Symbol != query.Symbol {
		return false
	}
	
	if query.StartTime != nil && record.Timestamp.Before(*query.StartTime) {
		return false
	}
	
	if query.EndTime != nil && record.Timestamp.After(*query.EndTime) {
		return false
	}
	
	return true
}

// sortRecords 排序记录
func (m *MemoryBackend) sortRecords(records []*DataRecord, orderBy string, desc bool) {
	sort.Slice(records, func(i, j int) bool {
		var less bool
		
		switch orderBy {
		case "timestamp":
			less = records[i].Timestamp.Before(records[j].Timestamp)
		case "exchange":
			less = records[i].Exchange < records[j].Exchange
		case "symbol":
			less = records[i].Symbol < records[j].Symbol
		case "type":
			less = records[i].Type < records[j].Type
		default:
			less = records[i].Timestamp.Before(records[j].Timestamp)
		}
		
		if desc {
			return !less
		}
		return less
	})
}
