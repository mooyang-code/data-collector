# 量化数据采集器架构设计

## 项目概述

量化数据采集器是一个用于采集各种金融市场数据的系统，支持多数据源、多数据类型的实时数据采集和存储。

## 核心架构

### 1. 整体架构图

```
┌─────────────────────────────────────────────────────────────┐
│                    AppManager (顶层管理器)                    │
│  - 从配置文件读取App列表                                        │
│  - 管理App生命周期                                            │
│  - 提供统一的管理接口                                          │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      │ 管理多个App
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                     App (应用实例)                           │
│  - 与数据集绑定                                              │
│  - 包含采集器管理器和存储器                                    │
│  - 独立的生命周期管理                                         │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      │ 包含
                      ▼
┌─────────────────────────────────────────────────────────────┐
│              CollectorManager (采集器管理器)                  │
│  - 管理多种类型的数据采集器                                    │
│  - 协调采集器的启动和停止                                      │
│  - 监控采集器状态                                            │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      │ 管理多个Collector
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                Collector (数据采集器)                        │
│  - 包含数据类型定义                                          │
│  - 包含触发器                                               │
│  - 执行具体的数据采集逻辑                                      │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      │ 包含
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                   Trigger (触发器)                          │
│  - 定时触发器 (CronTrigger)                                  │
│  - 事件触发器 (EventTrigger)                                 │
│  - 手动触发器 (ManualTrigger)                                │
└─────────────────────────────────────────────────────────────┘
```

### 2. 数据流向

```
配置文件 → AppManager → App → CollectorManager → Collector → Trigger
                                    ↓
                              Storage (存储器)
```

## 核心组件详细设计

### 1. AppManager (顶层管理器)

**职责：**
- 从配置文件中读取系统中有哪些App
- 管理App的生命周期（创建、启动、停止、销毁）
- 提供App的统一管理接口
- 监控App的运行状态

**核心接口：**
```go
type AppManager interface {
    LoadConfig(configPath string) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    AddApp(appConfig *AppConfig) error
    RemoveApp(appID string) error
    GetApp(appID string) (App, bool)
    GetAllApps() []App
    GetStats() AppManagerStats
}
```

### 2. App (应用实例)

**职责：**
- 与一个数据集绑定
- 包含一个采集器管理器和存储器
- 管理自身的生命周期
- 提供数据集相关的操作接口

**核心接口：**
```go
type App interface {
    GetID() string
    GetDataset() Dataset
    GetCollectorManager() CollectorManager
    GetStorage() Storage
    Initialize(config *AppConfig) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    IsRunning() bool
    GetStatus() AppStatus
    GetMetrics() AppMetrics
}
```

### 3. CollectorManager (采集器管理器)

**职责：**
- 管理多种类型的数据采集器
- 协调采集器的启动和停止
- 监控采集器状态
- 处理采集器之间的依赖关系

**核心接口：**
```go
type CollectorManager interface {
    AddCollector(collector Collector) error
    RemoveCollector(collectorID string) error
    GetCollector(collectorID string) (Collector, bool)
    GetCollectors() []Collector
    GetCollectorsByType(dataType DataType) []Collector
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    IsRunning() bool
    GetMetrics() CollectorManagerMetrics
}
```

### 4. Collector (数据采集器)

**职责：**
- 包含数据类型和触发器
- 执行具体的数据采集逻辑
- 将采集到的数据发送到存储器
- 处理采集过程中的错误和重试

**核心接口：**
```go
type Collector interface {
    GetID() string
    GetType() string
    GetDataType() DataType
    GetTrigger() Trigger
    SetTrigger(trigger Trigger) error
    Initialize(config *CollectorConfig) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    IsRunning() bool
    GetStatus() CollectorStatus
    GetMetrics() CollectorMetrics
    Collect(ctx context.Context) error
}
```

### 5. Trigger (触发器)

**职责：**
- 控制采集器的执行时机
- 支持多种触发方式（定时、事件、手动）
- 提供触发状态监控

**触发器类型：**
- **定时触发器 (CronTrigger)**: 基于Cron表达式的定时触发
- **事件触发器 (EventTrigger)**: 基于外部事件的触发
- **手动触发器 (ManualTrigger)**: 手动控制的触发

**核心接口：**
```go
type Trigger interface {
    GetID() string
    GetType() TriggerType
    Initialize(config interface{}) error
    Start(ctx context.Context, callback TriggerCallback) error
    Stop(ctx context.Context) error
    IsRunning() bool
    GetStatus() TriggerStatus
    GetMetrics() TriggerMetrics
}
```

### 6. Storage (存储器)

**职责：**
- 提供统一的数据存储接口
- 支持多种存储后端（内存、文件、数据库等）
- 处理数据的批量写入和查询

**核心接口：**
```go
type Storage interface {
    GetID() string
    GetType() string
    Initialize(config interface{}) error
    Store(ctx context.Context, data *DataRecord) error
    StoreBatch(ctx context.Context, data []*DataRecord) error
    Query(ctx context.Context, query *QueryRequest) (*QueryResult, error)
    Close() error
    GetStats() *StorageStats
}
```

## 数据类型定义

### 1. 支持的数据类型

- **K线数据 (Kline)**: OHLCV数据，包含开高低收价格和成交量
- **标的Symbol数据 (Symbol)**: 交易对信息，包含基础货币、计价货币等
- **深度数据 (Depth)**: 订单簿数据
- **成交数据 (Trade)**: 实时成交记录
- **行情数据 (Ticker)**: 24小时统计数据

### 2. 数据结构

```go
type DataType string

const (
    DataTypeKline  DataType = "kline"
    DataTypeSymbol DataType = "symbol"
    DataTypeDepth  DataType = "depth"
    DataTypeTrade  DataType = "trade"
    DataTypeTicker DataType = "ticker"
)
```

## 配置文件结构

### 1. 全局配置 (config.yaml)

```yaml
app:
  name: "data-collector"
  version: "2.0.0"
  environment: "production"

logging:
  level: "info"
  format: "json"

monitoring:
  enabled: true
  port: 8080
```

### 2. App配置 (apps.yaml)

```yaml
apps:
  - id: "binance-spot-app"
    name: "Binance现货数据采集"
    type: "binance_spot"
    dataset_id: "binance_spot_main"
    enabled: true
    
  - id: "binance-futures-app"
    name: "Binance合约数据采集"
    type: "binance_futures"
    dataset_id: "binance_futures_main"
    enabled: true
```

### 3. 数据集配置 (datasets.yaml)

```yaml
datasets:
  - id: "binance_spot_main"
    name: "Binance现货主要数据集"
    description: "采集Binance现货市场的主要交易对数据"
    source:
      type: "binance_spot"
      symbols: ["BTCUSDT", "ETHUSDT", "BNBUSDT"]
      intervals: ["1m", "5m", "1h", "1d"]
    collectors:
      - type: "kline"
        trigger:
          type: "cron"
          schedule: "*/1 * * * *"
      - type: "symbol"
        trigger:
          type: "cron"
          schedule: "0 */6 * * *"
    storage:
      type: "clickhouse"
      config:
        dsn: "tcp://localhost:9000/market_data"
    enabled: true
```

## 实现计划

1. **第一阶段**: 核心接口定义和基础数据结构
2. **第二阶段**: 基础组件实现（App、CollectorManager、Collector）
3. **第三阶段**: 触发器系统实现
4. **第四阶段**: 存储器系统实现
5. **第五阶段**: AppManager实现和配置管理
6. **第六阶段**: 工厂模式和依赖注入
7. **第七阶段**: 示例和测试代码

## 扩展性设计

- **插件化架构**: 支持动态加载新的数据源适配器
- **配置热更新**: 支持运行时配置更新
- **水平扩展**: 支持分布式部署
- **监控集成**: 内置Prometheus指标和健康检查
