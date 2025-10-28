package config

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
	// TaskParams 任务执行参数
	TaskParams string `json:"TaskParams"`
	// AccessUrl 访问该表的接口url
	AccessUrl string
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
	return "" // 不需要过滤条件，获取所有数据
}

// GetAllTaskInstanceList 获取所有采集任务实例缓存
func GetAllTaskInstanceList() []*CollectorTaskInstanceCache {
	instances, ok := GetAll(TBTaskInstance).([]*CollectorTaskInstanceCache)
	if !ok {
		return nil
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
