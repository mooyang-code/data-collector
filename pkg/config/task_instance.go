package config

import "encoding/json"

const TBTaskInstance = "t_collector_task_instances"

// CollectorTaskInstanceCache 采集任务实例缓存结构
type CollectorTaskInstanceCache struct {
	// ID 主键ID
	ID int `json:"ID"`
	// TaskID 任务唯一标识
	TaskID string `json:"TaskID"`
	// RuleID 规则ID（关联配置表）
	RuleID string `json:"RuleID"`
	// NodeID 执行节点ID
	NodeID string `json:"NodeID"`
	// TaskParams 任务执行参数（原始JSON字符串）
	TaskParams string `json:"TaskParams"`
	// Invalid 任务删除标记
	Invalid int `json:"Invalid"`
	// AccessUrl 访问该表的接口url
	AccessUrl string

	// === 以下为 TaskParams 解析后的结构化字段 ===
	// DataType 数据类型（如 kline, ticker, depth 等）
	DataType string `json:"-"`
	// DataSource 数据源（如 binance, okx 等）
	DataSource string `json:"-"`
	// InstType 产品类型: SPOT(现货), SWAP(永续合约), FUTURES(交割合约)
	InstType string `json:"-"`
	// Symbol 交易对（如 BTC-USDT）
	Symbol string `json:"-"`
	// Intervals K线周期列表（如 ["1m", "3m", "5m"]）
	Intervals []string `json:"-"`
}

// taskParamsJSON TaskParams 的 JSON 解析结构
type taskParamsJSON struct {
	DataType   string   `json:"data_type"`
	DataSource string   `json:"data_source"`
	InstType   string   `json:"inst_type"`
	Symbol     string   `json:"symbol"`
	Intervals  []string `json:"intervals"`
}

// ParseTaskParams 解析 TaskParams JSON 字符串并填充结构化字段
func (c *CollectorTaskInstanceCache) ParseTaskParams() error {
	if c.TaskParams == "" {
		return nil
	}

	var params taskParamsJSON
	if err := json.Unmarshal([]byte(c.TaskParams), &params); err != nil {
		return err
	}

	c.DataType = params.DataType
	c.DataSource = params.DataSource
	c.InstType = params.InstType
	c.Symbol = params.Symbol
	c.Intervals = params.Intervals
	return nil
}

// SchemaID 实现接口APICacher
func (CollectorTaskInstanceCache) SchemaID() string {
	return TBTaskInstance
}

// URL 实现接口APICacher
func (c CollectorTaskInstanceCache) URL() string {
	return c.AccessUrl
}

// SearchFields 实现接口APICacher
func (CollectorTaskInstanceCache) SearchFields() map[string]string {
	return map[string]string{TBTaskInstance: "ID"}
}

// FilterKey 实现接口APICacher
func (CollectorTaskInstanceCache) FilterKey() string {
	return "Invalid=0"
}

// GetAllTaskInstanceList 获取所有采集任务实例缓存
func GetAllTaskInstanceList() []*CollectorTaskInstanceCache {
	instances, ok := GetAll(TBTaskInstance).([]*CollectorTaskInstanceCache)
	if !ok {
		return nil
	}
	// 解析每个实例的 TaskParams
	for _, instance := range instances {
		_ = instance.ParseTaskParams()
	}
	return instances
}

// GetTaskInstancesByNode 根据节点ID获取任务实例列表
func GetTaskInstancesByNode(nodeID string) []*CollectorTaskInstanceCache {
	if nodeID == "" {
		return nil
	}

	allInstances := GetAllTaskInstanceList()
	if allInstances == nil {
		return nil
	}

	var result []*CollectorTaskInstanceCache
	for _, instance := range allInstances {
		if instance.NodeID == nodeID {
			result = append(result, instance)
		}
	}
	return result
}
