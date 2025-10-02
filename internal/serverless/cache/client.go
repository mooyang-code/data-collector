package cache

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/mooyang-code/data-collector/internal/serverless/types"
	"github.com/mooyang-code/go-commlib/apicache"
)

// CacheClient 缓存客户端接口
type CacheClient interface {
	// GetNodeTasks 获取节点的任务列表
	GetNodeTasks(nodeID string) ([]types.TaskConfig, error)
}

// CollectorTaskInstance 采集器任务实例（与moox服务中的结构对应）
type CollectorTaskInstance struct {
	ID              int        `json:"id"`
	InstanceID      string     `json:"instance_id"`
	TaskID          string     `json:"task_id"`
	ProjectID       string     `json:"project_id"`
	DatasetID       string     `json:"dataset_id"`
	NodeID          string     `json:"node_id"`
	TargetObjects   string     `json:"target_objects"`
	ExecutionParams string     `json:"execution_params"`
	Status          int        `json:"status"`
	StartTime       *time.Time `json:"start_time"`
	EndTime         *time.Time `json:"end_time"`
	Result          string     `json:"result"`
	CreateTime      time.Time  `json:"create_time"`
	ModifyTime      time.Time  `json:"modify_time"`
}

// TaskInstanceCache 任务实例缓存
type TaskInstanceCache struct {
	Instances []CollectorTaskInstance `json:"instances"`
	AccessUrl string                  `json:"-"`
}

// SchemaID 获取缓存schema标识
func (TaskInstanceCache) SchemaID() string {
	return "t_collector_task_instances"
}

// URL 获取数据接口url
func (t TaskInstanceCache) URL() string {
	return t.AccessUrl
}

// SearchFields 获取查询field值
func (TaskInstanceCache) SearchFields() map[string]string {
	return map[string]string{"t_collector_task_instances": "node_id"}
}

// FilterKey 获取过滤key值
func (TaskInstanceCache) FilterKey() string {
	return "status IN (0,1)" // 只缓存待执行(0)和执行中(1)的任务
}

// APICacheClient 使用apicache的缓存客户端实现
type APICacheClient struct {
	cacher      apicache.ConfigCacher
	cacheURL    string
	initialized bool
	mu          sync.RWMutex
}

// NewAPICacheClient 创建基于apicache的缓存客户端
func NewAPICacheClient(cacheURL string) CacheClient {
	client := &APICacheClient{
		cacheURL: cacheURL,
	}

	// 初始化缓存
	if err := client.initCache(); err != nil {
		log.Printf("[Cache] Warning: Failed to initialize cache: %v", err)
	}
	return client
}

// initCache 初始化缓存
func (c *APICacheClient) initCache() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 创建缓存实例
	cacher, err := apicache.NewCache(
		TaskInstanceCache{
			AccessUrl: c.cacheURL + "/moox-api/t_collector_task_instances", // moox的任务实例接口
		},
	)
	if err != nil {
		return err
	}

	c.cacher = cacher
	c.initialized = true
	return nil
}

// GetNodeTasks 获取节点的任务列表
func (c *APICacheClient) GetNodeTasks(nodeID string) ([]types.TaskConfig, error) {
	c.mu.RLock()
	initialized := c.initialized
	cacher := c.cacher
	c.mu.RUnlock()

	if !initialized || cacher == nil {
		log.Printf("[Cache] Cache not initialized, returning empty task list")
		return []types.TaskConfig{}, nil
	}

	// 从缓存中获取所有任务实例
	allData := cacher.GetAll("t_collector_task_instances")
	if allData == nil {
		return []types.TaskConfig{}, nil
	}

	// 过滤出该节点的任务并转换格式
	var tasks []types.TaskConfig
	if instances, ok := allData.([]interface{}); ok {
		for _, item := range instances {
			if instance, ok := item.(*CollectorTaskInstance); ok && instance != nil && instance.NodeID == nodeID {
				// 将任务实例转换为TaskConfig
				taskConfig := c.convertToTaskConfig(instance)
				tasks = append(tasks, taskConfig)
			}
		}
	}

	return tasks, nil
}

// convertToTaskConfig 将CollectorTaskInstance转换为TaskConfig
func (c *APICacheClient) convertToTaskConfig(instance *CollectorTaskInstance) types.TaskConfig {
	// 解析执行参数
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(instance.ExecutionParams), &params); err != nil {
		log.Printf("[Cache] Failed to parse execution params for instance %s: %v", instance.InstanceID, err)
		params = make(map[string]interface{})
	}

	// 构建TaskConfig
	taskConfig := types.TaskConfig{
		TaskID: instance.InstanceID, // 使用实例ID作为任务ID
		Config: params,
	}

	// 从执行参数中提取必要字段
	if collectorType, ok := params["collector_type"].(string); ok {
		taskConfig.CollectorType = collectorType
	}
	if sourceName, ok := params["source_name"].(string); ok {
		taskConfig.Source = sourceName
	}
	if interval, ok := params["interval"].(string); ok {
		taskConfig.Interval = interval
	}

	return taskConfig
}

// HTTPCacheClient 简单的HTTP缓存客户端（作为备选方案）
type HTTPCacheClient struct {
	client *APICacheClient
}

// NewHTTPCacheClient 创建HTTP缓存客户端（向后兼容）
func NewHTTPCacheClient(endpoint string) CacheClient {
	return NewAPICacheClient(endpoint)
}

// GetAllNodeTasks 获取所有节点的任务（辅助函数）
func GetAllNodeTasks(client CacheClient) map[string][]types.TaskConfig {
	// 如果是APICacheClient，可以获取所有缓存数据
	if apiClient, ok := client.(*APICacheClient); ok {
		apiClient.mu.RLock()
		defer apiClient.mu.RUnlock()

		if !apiClient.initialized || apiClient.cacher == nil {
			return make(map[string][]types.TaskConfig)
		}

		// 获取所有任务实例
		allData := apiClient.cacher.GetAll("t_collector_task_instances")
		if allData == nil {
			return make(map[string][]types.TaskConfig)
		}

		// 按节点分组转换
		result := make(map[string][]types.TaskConfig)
		if instances, ok := allData.([]interface{}); ok {
			for _, item := range instances {
				if instance, ok := item.(*CollectorTaskInstance); ok && instance != nil {
					taskConfig := apiClient.convertToTaskConfig(instance)
					result[instance.NodeID] = append(result[instance.NodeID], taskConfig)
				}
			}
		}

		return result
	}

	// 默认返回空map
	return make(map[string][]types.TaskConfig)
}
