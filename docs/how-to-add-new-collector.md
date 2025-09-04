# 如何添加新的采集器

本文档介绍如何使用自注册机制添加新的采集器，无需修改外层代码。

## 自注册机制概述

采集器使用自注册机制，通过 `init()` 函数在包被导入时自动注册到全局注册中心。这样可以实现：

- ✅ 新增采集器只需在自己的包中添加代码
- ✅ 无需修改外层的工厂代码
- ✅ 支持按交易所、数据类型、市场类型分类
- ✅ 自动发现和管理采集器

## 添加新采集器的步骤

### 1. 创建采集器目录结构

```
internal/app/
├── init.go                     # 采集器包导入
├── collector_registry.go       # 注册中心
├── collector_factory.go        # 工厂实现
└── {exchange}/{datatype}/
    ├── collector.go            # 采集器实现
    ├── config.go               # 配置结构
    └── register.go             # 自注册代码
```

例如，添加 Bybit 现货交易对采集器：

```
internal/app/bybit/symbols/
├── collector.go
├── config.go
└── register.go
```

### 2. 实现采集器核心逻辑

在 `collector.go` 中实现具体的采集器：

```go
// Package symbols Bybit交易对采集器
package symbols

import (
    "context"
    "github.com/mooyang-code/data-collector/configs"
)

// BybitSymbolCollector Bybit交易对采集器
type BybitSymbolCollector struct {
    exchange   string
    marketType string
    baseURL    string
    config     *configs.Collector
    running    bool
}

// NewBybitSymbolCollector 创建Bybit交易对采集器
func NewBybitSymbolCollector(config *BybitSymbolConfig) (*BybitSymbolCollector, error) {
    return &BybitSymbolCollector{
        exchange:   config.Exchange,
        marketType: config.MarketType,
        baseURL:    config.BaseURL,
        running:    false,
    }, nil
}

// Initialize 初始化采集器
func (c *BybitSymbolCollector) Initialize(ctx context.Context) error {
    // 实现初始化逻辑
    return nil
}

// StartCollection 启动采集
func (c *BybitSymbolCollector) StartCollection(ctx context.Context) error {
    // 实现采集逻辑
    c.running = true
    return nil
}

// StopCollection 停止采集
func (c *BybitSymbolCollector) StopCollection(ctx context.Context) error {
    // 实现停止逻辑
    c.running = false
    return nil
}

// IsRunning 检查是否运行中
func (c *BybitSymbolCollector) IsRunning() bool {
    return c.running
}
```

### 3. 定义配置结构

在 `config.go` 中定义采集器配置：

```go
// Package symbols Bybit交易对采集器配置
package symbols

import "time"

// BybitSymbolConfig Bybit交易对采集器配置
type BybitSymbolConfig struct {
    Exchange           string
    MarketType         string
    BaseURL           string
    BufferSize        int
    EnableAutoRefresh bool
    RefreshInterval   time.Duration
    EnableFiltering   bool
    AllowedQuotes     []string
}
```

### 4. 创建自注册代码

在 `register.go` 中实现自注册：

```go
// Package symbols Bybit交易对采集器自注册
package symbols

import (
    "context"
    "fmt"
    "time"

    "github.com/mooyang-code/data-collector/configs"
    "github.com/mooyang-code/data-collector/internal/app"
    "trpc.group/trpc-go/trpc-go/log"
)

// init 函数在包被导入时自动执行，注册采集器创建器
func init() {
    // 注册现货交易对采集器
    app.RegisterCollectorCreator("bybit", "symbols", "spot", createBybitSpotSymbolCollector)
    
    // 注册合约交易对采集器
    app.RegisterCollectorCreator("bybit", "symbols", "futures", createBybitFuturesSymbolCollector)
    
    log.Info("Bybit交易对采集器注册完成")
}

// createBybitSpotSymbolCollector 创建Bybit现货交易对采集器
func createBybitSpotSymbolCollector(appName, collectorName string, config *configs.Collector) (app.Collector, error) {
    // 转换配置
    bybitConfig := &BybitSymbolConfig{
        Exchange:           "bybit",
        MarketType:         "spot",
        BaseURL:           "https://api.bybit.com",
        BufferSize:        1000,
        EnableAutoRefresh: config.Schedule.EnableAutoRefresh,
        RefreshInterval:   5 * time.Minute,
        EnableFiltering:   len(config.Config.Filters) > 0,
        AllowedQuotes:     config.Config.Filters,
    }

    // 创建采集器实例
    collector, err := NewBybitSymbolCollector(bybitConfig)
    if err != nil {
        return nil, fmt.Errorf("创建Bybit现货交易对采集器失败: %w", err)
    }

    // 包装为通用接口
    wrapper := &BybitSymbolCollectorWrapper{
        collector:     collector,
        id:           fmt.Sprintf("%s.%s", appName, collectorName),
        collectorType: "bybit.symbols.spot",
        dataType:     config.DataType,
    }

    return wrapper, nil
}

// createBybitFuturesSymbolCollector 创建Bybit合约交易对采集器
func createBybitFuturesSymbolCollector(appName, collectorName string, config *configs.Collector) (app.Collector, error) {
    // 类似的实现，但使用合约API地址
    // ...
}

// BybitSymbolCollectorWrapper 包装器，实现通用接口
type BybitSymbolCollectorWrapper struct {
    collector     *BybitSymbolCollector
    id           string
    collectorType string
    dataType     string
    running      bool
}

// 实现 app.Collector 接口的所有方法
func (w *BybitSymbolCollectorWrapper) Initialize(ctx context.Context) error {
    return w.collector.Initialize(ctx)
}

func (w *BybitSymbolCollectorWrapper) StartCollection(ctx context.Context) error {
    err := w.collector.StartCollection(ctx)
    if err == nil {
        w.running = true
    }
    return err
}

func (w *BybitSymbolCollectorWrapper) StopCollection(ctx context.Context) error {
    err := w.collector.StopCollection(ctx)
    if err == nil {
        w.running = false
    }
    return err
}

func (w *BybitSymbolCollectorWrapper) IsRunning() bool {
    return w.running
}

func (w *BybitSymbolCollectorWrapper) GetID() string {
    return w.id
}

func (w *BybitSymbolCollectorWrapper) GetType() string {
    return w.collectorType
}

func (w *BybitSymbolCollectorWrapper) GetDataType() string {
    return w.dataType
}
```

### 5. 添加包导入

在 `internal/app/init.go` 中添加新包的导入：

```go
package app

import (
    // 现有的导入
    _ "github.com/mooyang-code/data-collector/internal/app/binance/klines"
    _ "github.com/mooyang-code/data-collector/internal/app/binance/symbols"
    _ "github.com/mooyang-code/data-collector/internal/app/okx/symbols"

    // 新增的导入
    _ "github.com/mooyang-code/data-collector/internal/app/bybit/symbols"
    _ "github.com/mooyang-code/data-collector/internal/app/bybit/klines"
)
```

## 注册机制说明

### 注册函数

```go
app.RegisterCollectorCreator(exchange, dataType, marketType, creator)
```

参数说明：
- `exchange`: 交易所名称，如 "binance", "okx", "bybit"
- `dataType`: 数据类型，如 "symbols", "klines", "trades"
- `marketType`: 市场类型，如 "spot", "futures", "options"
- `creator`: 创建函数，类型为 `CollectorCreatorFunc`

### 采集器类型命名规则

采集器类型按照 `{exchange}.{dataType}.{marketType}` 的格式命名，例如：
- `binance.symbols.spot` - 币安现货交易对采集器
- `binance.klines.futures` - 币安合约K线采集器
- `okx.symbols.spot` - OKX现货交易对采集器

## 测试新采集器

添加完成后，可以通过以下方式测试：

```go
// 检查是否支持
supported := app.IsSupported("bybit", "symbols", "spot")

// 获取支持的类型
types := app.GetSupportedTypes()

// 创建采集器实例
factory := app.NewCollectorFactory()
collector, err := factory.CreateCollector("bybit", "symbols_collector", config)
```

## 最佳实践

1. **命名规范**: 采集器包名使用数据类型名称，如 `symbols`, `klines`
2. **配置转换**: 在创建函数中将通用配置转换为采集器特定配置
3. **错误处理**: 提供详细的错误信息，便于调试
4. **日志记录**: 在关键步骤添加日志，便于监控
5. **接口实现**: 确保包装器完整实现 `app.Collector` 接口
6. **测试覆盖**: 为新采集器编写单元测试

## 总结

通过自注册机制，添加新采集器变得非常简单：

1. 在自己的包中实现采集器逻辑
2. 在 `register.go` 中注册创建函数
3. 在导入文件中添加包导入
4. 新采集器自动被系统识别和管理

这种设计遵循了开闭原则，使系统易于扩展而无需修改现有代码。
