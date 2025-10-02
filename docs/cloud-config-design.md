# 数据采集器云端配置中心设计方案

## 1. 概述

本方案旨在将数据采集器的配置从本地文件迁移到云端中心存储，实现分布式部署和集中管理。通过中心配置，可以灵活分配各个机器上的采集任务，实现负载均衡和高可用。

## 2. 系统架构

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  管理后台/API   │────▶│   配置中心DB    │◀────│   采集节点1     │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                               ▲                           │
                               │                           │
                               │                  ┌─────────────────┐
                               └──────────────────│   采集节点2     │
                                                  └─────────────────┘
```

### 核心组件
- **配置中心DB**: 存储节点信息和采集器配置
- **配置管理API**: 提供配置查询、节点注册、心跳接收等接口
- **采集节点**: 运行数据采集器的机器，通过 apicache 组件拉取配置

## 3. 数据库表设计

### 3.1 云函数数据采集器节点信息表 (t_cloud_nodes)
```sql
-- ************ 创建云函数数据采集器节点信息表 ************
CREATE TABLE t_cloud_nodes (
    c_id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, -- 主键ID
    c_node_id TEXT NOT NULL DEFAULT '', -- 节点唯一标识
    c_node_name TEXT NOT NULL DEFAULT '', -- 节点名称
    c_ip_address TEXT NOT NULL DEFAULT '', -- IP地址
    c_hostname TEXT NOT NULL DEFAULT '', -- 主机名
    c_version TEXT NOT NULL DEFAULT '', -- 采集器版本
    c_status INTEGER NOT NULL DEFAULT 0, -- 节点状态（0=离线，1=在线，2=维护中）
    c_last_heartbeat DATETIME, -- 最后心跳时间
    c_metadata TEXT NOT NULL DEFAULT '', -- 节点额外信息（JSON格式）
    c_invalid INTEGER NOT NULL DEFAULT 0, -- 删除标记
    c_ctime DATETIME DEFAULT CURRENT_TIMESTAMP, -- 创建时间
    c_mtime DATETIME DEFAULT CURRENT_TIMESTAMP, -- 修改时间
    UNIQUE (c_node_id)
);

-- 创建索引
CREATE INDEX idx_t_scf_collector_nodes_status ON t_cloud_nodes (c_status);
CREATE INDEX idx_t_scf_collector_nodes_heartbeat ON t_cloud_nodes (c_last_heartbeat);
CREATE INDEX idx_t_scf_collector_nodes_ctime ON t_cloud_nodes (c_ctime);
CREATE INDEX idx_t_scf_collector_nodes_mtime ON t_cloud_nodes (c_mtime);

-- 创建触发器，更新修改时间
CREATE TRIGGER update_scf_collector_nodes_mtime AFTER UPDATE ON t_cloud_nodes
BEGIN
    UPDATE t_cloud_nodes SET c_mtime = CURRENT_TIMESTAMP WHERE rowid = NEW.rowid;
END;
```

### 3.2 节点采集器配置表 (t_node_collectors_conf)
```sql
-- ************ 创建节点采集器配置表 ************
CREATE TABLE t_node_collectors_conf (
    c_id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, -- 主键ID
    c_node_id TEXT NOT NULL DEFAULT '', -- 节点ID
    c_source_name TEXT NOT NULL DEFAULT '', -- 数据源名称(binance/okx等)
    c_collector_type TEXT NOT NULL DEFAULT '', -- 采集器类型(kline/ticker/orderbook/trade)
    c_data_type TEXT NOT NULL DEFAULT '', -- 数据类型
    c_cron_expression TEXT NOT NULL DEFAULT '', -- 定时器表达式(如: 0 */5 * * * ?)
    c_symbols TEXT NOT NULL DEFAULT '', -- 交易对列表（JSON格式：["BTCUSDT","ETHUSDT"]）
    c_intervals TEXT NOT NULL DEFAULT '', -- K线时间间隔（JSON格式：["1m","5m","1h"]）
    c_config TEXT NOT NULL DEFAULT '', -- 采集器详细配置（JSON格式）
    c_enabled INTEGER NOT NULL DEFAULT 1, -- 是否启用（-1否，1是）
    c_priority INTEGER NOT NULL DEFAULT 0, -- 优先级(用于负载均衡)
    c_invalid INTEGER NOT NULL DEFAULT 0, -- 删除标记
    c_ctime DATETIME DEFAULT CURRENT_TIMESTAMP, -- 创建时间
    c_mtime DATETIME DEFAULT CURRENT_TIMESTAMP, -- 修改时间
    UNIQUE (c_node_id, c_source_name, c_collector_type, c_invalid)
);

-- 创建索引
CREATE INDEX idx_t_node_collectors_conf_node_id ON t_node_collectors_conf (c_node_id);
CREATE INDEX idx_t_node_collectors_conf_enabled ON t_node_collectors_conf (c_enabled);
CREATE INDEX idx_t_node_collectors_conf_source_name ON t_node_collectors_conf (c_source_name);
CREATE INDEX idx_t_node_collectors_conf_collector_type ON t_node_collectors_conf (c_collector_type);
CREATE INDEX idx_t_node_collectors_conf_ctime ON t_node_collectors_conf (c_ctime);
CREATE INDEX idx_t_node_collectors_conf_mtime ON t_node_collectors_conf (c_mtime);

-- 创建触发器，更新修改时间
CREATE TRIGGER update_node_collectors_conf_mtime AFTER UPDATE ON t_node_collectors_conf
BEGIN
    UPDATE t_node_collectors_conf SET c_mtime = CURRENT_TIMESTAMP WHERE rowid = NEW.rowid;
END;
```

## 4. 基于 apicache 的配置拉取方案

### 4.1 缓存结构定义
```go
// NodeConfig 节点配置缓存项
type NodeConfig struct {
    AccessUrl string // 配置拉取API地址
    nodeID    string
}

// SchemaID 实现 apicache.APICacher 接口
func (n NodeConfig) SchemaID() string {
    return "node_config_" + n.nodeID
}

// GetAccessUrl 实现 apicache.APICacher 接口
func (n NodeConfig) GetAccessUrl() string {
    return n.AccessUrl
}
```

### 4.2 配置数据结构
```go
// ConfigResponse 配置API响应
type ConfigResponse struct {
    Version    int                `json:"version"`
    NodeID     string             `json:"node_id"`
    Collectors []CollectorConfig  `json:"collectors"`
    UpdatedAt  time.Time          `json:"updated_at"`
}

// CollectorConfig 采集器配置
type CollectorConfig struct {
    SourceName     string          `json:"source_name"`
    CollectorType  string          `json:"collector_type"`
    DataType       string          `json:"data_type"`
    CronExpression string          `json:"cron_expression"`
    Symbols        []string        `json:"symbols"`
    Intervals      []string        `json:"intervals,omitempty"`
    Config         json.RawMessage `json:"config"`
    Enabled        bool            `json:"enabled"`
}
```

### 4.3 初始化和使用
```go
// 初始化配置缓存
func InitCollectorConfig(nodeID string, configAPIUrl string) error {
    return cache.InitSingleDBCache(
        NodeConfig{
            AccessUrl: fmt.Sprintf("%s/api/v1/nodes/%s/config", configAPIUrl, nodeID),
            nodeID:    nodeID,
        },
    )
}

// GetNodeConfig 获取节点配置
func GetNodeConfig(nodeID string) (*ConfigResponse, error) {
    data := cache.QueryDataItem("node_config_"+nodeID, nodeID)
    if data == nil {
        return nil, fmt.Errorf("config not found for node: %s", nodeID)
    }
    
    config, ok := data.(*ConfigResponse)
    if !ok {
        return nil, fmt.Errorf("invalid config type")
    }
    
    return config, nil
}
```

## 5. API 接口设计

### 5.1 获取节点配置
**接口**: `GET /api/v1/nodes/{node_id}/config`

**响应示例**:
```json
{
    "version": 123,
    "node_id": "node-001",
    "collectors": [
        {
            "source_name": "binance",
            "collector_type": "kline",
            "data_type": "candlestick",
            "cron_expression": "0 */1 * * * ?",
            "symbols": ["BTCUSDT", "ETHUSDT"],
            "intervals": ["1m", "5m", "1h"],
            "config": {
                "api_url": "https://api.binance.com",
                "batch_size": 100,
                "history_days": 30
            },
            "enabled": true
        },
        {
            "source_name": "binance", 
            "collector_type": "ticker",
            "data_type": "market_ticker",
            "cron_expression": "*/5 * * * * ?",
            "symbols": ["BTCUSDT", "ETHUSDT", "BNBUSDT"],
            "config": {
                "update_interval": "5s"
            },
            "enabled": true
        }
    ],
    "updated_at": "2025-09-07T10:00:00Z"
}
```

### 5.2 节点心跳上报
**接口**: `POST /api/v1/nodes/{node_id}/heartbeat`

**请求体**:
```json
{
    "node_id": "node-001",
    "timestamp": "2025-09-07T10:00:00Z",
    "status": "online",
    "metrics": {
        "cpu_usage": 45.2,
        "memory_usage": 78.5,
        "active_collectors": 5
    }
}
```

### 5.3 节点注册
**接口**: `POST /api/v1/nodes/register`

**请求体**:
```json
{
    "node_id": "node-001",
    "node_name": "采集器-北京-01",
    "version": "2.0.0",
    "metadata": {
        "region": "beijing",
        "datacenter": "dc1"
    }
}
```

## 6. 采集器启动流程

```go
func main() {
    // 1. 读取本地基础配置（节点ID、配置中心地址等）
    localConfig := loadLocalConfig()
    
    // 2. 初始化配置缓存（使用 apicache）
    if err := InitCollectorConfig(localConfig.NodeID, localConfig.ConfigCenterURL); err != nil {
        log.Fatal("初始化配置缓存失败:", err)
    }
    
    // 3. 启动心跳上报协程
    go startHeartbeatReporter(localConfig.NodeID, localConfig.ConfigCenterURL)
    
    // 4. 启动配置监听和采集器管理
    collectorManager := NewCollectorManager(localConfig.NodeID)
    
    // 5. 定期检查配置更新（apicache会自动处理）
    ticker := time.NewTicker(30 * time.Second)
    for range ticker.C {
        config, err := GetNodeConfig(localConfig.NodeID)
        if err != nil {
            log.Error("获取配置失败:", err)
            continue
        }
        
        // 应用新配置
        collectorManager.UpdateCollectors(config.Collectors)
    }
}
```

## 7. 配置热更新机制

### 7.1 更新流程
1. 拉取最新配置
2. 对比版本号或配置内容
3. 如有更新：
   - 停止被移除的采集器
   - 更新已存在的采集器配置
   - 启动新增的采集器
4. 更新本地版本号

### 7.2 采集器管理器
```go
type CollectorManager struct {
    nodeID           string
    activeCollectors map[string]*Collector
    mu               sync.RWMutex
}

func (m *CollectorManager) UpdateCollectors(newConfigs []CollectorConfig) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    // 1. 构建新配置映射
    newConfigMap := make(map[string]CollectorConfig)
    for _, cfg := range newConfigs {
        key := fmt.Sprintf("%s_%s", cfg.SourceName, cfg.CollectorType)
        newConfigMap[key] = cfg
    }
    
    // 2. 停止被移除的采集器
    for key, collector := range m.activeCollectors {
        if _, exists := newConfigMap[key]; !exists {
            collector.Stop()
            delete(m.activeCollectors, key)
        }
    }
    
    // 3. 更新或启动采集器
    for key, cfg := range newConfigMap {
        if !cfg.Enabled {
            continue
        }
        
        if existingCollector, exists := m.activeCollectors[key]; exists {
            // 更新现有采集器
            existingCollector.UpdateConfig(cfg)
        } else {
            // 启动新采集器
            collector := NewCollector(cfg)
            collector.Start()
            m.activeCollectors[key] = collector
        }
    }
}
```

## 8. 故障处理策略

### 8.1 本地缓存机制
- apicache 组件内置本地缓存功能
- 配置中心不可用时，使用本地缓存继续运行
- 本地缓存定期持久化到文件

### 8.2 心跳与健康检查
- 节点每30秒上报一次心跳
- 超过5分钟无心跳，节点状态标记为 offline
- 管理后台可实时监控节点状态

### 8.3 网络异常处理
- apicache 内置指数退避重试机制
- 支持配置重试次数和超时时间
- 网络恢复后自动同步最新配置

## 9. 安全考虑

### 9.1 认证与授权
- 节点使用唯一 node_id 进行身份识别
- API 接口支持 Token 认证
- 每个节点只能获取分配给自己的配置

### 9.2 配置加密
- 敏感信息（如 API 密钥）在数据库中加密存储
- 传输过程使用 HTTPS 加密
- 支持配置项级别的权限控制

## 10. 实施计划

### 第一阶段：基础设施搭建（1周）
- [ ] 部署配置中心数据库
- [ ] 开发配置管理 API 服务
- [ ] 实现节点注册和心跳接口

### 第二阶段：采集器改造（2周）
- [ ] 集成 apicache 组件
- [ ] 改造采集器启动逻辑
- [ ] 实现配置热更新机制
- [ ] 添加本地缓存和故障恢复

### 第三阶段：管理功能完善（1周）
- [ ] 开发配置管理后台
- [ ] 实现节点监控大屏
- [ ] 添加告警通知功能

### 第四阶段：测试与上线（1周）
- [ ] 功能测试
- [ ] 性能测试
- [ ] 灰度发布
- [ ] 全量上线

## 11. 收益总结

- **集中管理**: 所有采集器配置集中管理，便于维护
- **动态调度**: 支持动态分配采集任务，实现负载均衡
- **高可用性**: 故障节点的任务可快速迁移到其他节点
- **灵活扩展**: 新增节点只需注册即可，无需手动配置
- **版本控制**: 配置变更可追溯，支持回滚
- **实时监控**: 实时掌握所有节点和采集器状态