# 如何添加新的数据采集器

本文档介绍如何在新架构下添加新的数据采集器。

## 1. 架构概述

新架构采用插件化设计，支持自注册机制：
- **App层**：代表一个数据源（如 Binance 交易所）
- **Collector层**：具体的数据采集器（如 K线采集器）
- **事件驱动**：通过事件总线解耦数据流

## 2. 添加新数据源的步骤

### 2.1 创建目录结构

```bash
internal/source/
└── market/              # 数据源类别
    └── newexchange/     # 新交易所
        ├── api/         # API 客户端
        │   └── client.go
        ├── collectors/  # 采集器实现
        │   ├── kline.go
        │   └── ticker.go
        └── app.go       # App 实现
```

### 2.2 实现 App

```go
// internal/source/market/newexchange/app.go
package newexchange

import (
    "context"
    "github.com/mooyang-code/data-collector/internal/core/app"
)

type NewExchangeApp struct {
    *app.BaseApp
    client *api.Client
}

// init 函数实现自注册
func init() {
    err := app.RegisterCreator(
        "newexchange",           // 唯一标识
        "新交易所",              // 显示名称
        "新交易所数据采集应用",   // 描述
        app.SourceTypeMarket,    // 数据源类型
        NewApp,                  // 创建函数
    )
    if err != nil {
        panic(err)
    }
}

func NewApp(config *app.AppConfig) (app.App, error) {
    // 创建 API 客户端
    client := api.NewClient(config.Settings)
    
    // 创建 App 实例
    return &NewExchangeApp{
        BaseApp: app.NewBaseApp("newexchange", "新交易所", app.SourceTypeMarket),
        client:  client,
    }, nil
}

func (a *NewExchangeApp) Initialize(ctx context.Context) error {
    // 测试 API 连接
    if err := a.client.Ping(); err != nil {
        return err
    }
    
    // 注册采集器
    a.registerCollectors()
    
    // 调用基类初始化
    return a.BaseApp.Initialize(ctx)
}
```

### 2.3 实现采集器

```go
// internal/source/market/newexchange/collectors/kline.go
package collectors

import (
    "github.com/mooyang-code/data-collector/internal/core/collector"
    "github.com/mooyang-code/data-collector/internal/model/market"
)

type KlineCollector struct {
    *collector.BaseCollector
    client *api.Client
}

// init 函数实现自注册
func init() {
    err := collector.NewBuilder().
        Source("newexchange", "新交易所").
        DataType("kline", "K线").
        MarketType("spot", "现货").
        Description("新交易所K线数据采集器").
        Creator(NewKlineCollector).
        Register()
    
    if err != nil {
        panic(err)
    }
}

func NewKlineCollector(config map[string]interface{}) (collector.Collector, error) {
    // 解析配置
    client := config["client"].(*api.Client)
    
    return &KlineCollector{
        BaseCollector: collector.NewBaseCollector("newexchange_kline", "market", "kline"),
        client:        client,
    }, nil
}

func (c *KlineCollector) Initialize(ctx context.Context) error {
    // 调用基类初始化
    if err := c.BaseCollector.Initialize(ctx); err != nil {
        return err
    }
    
    // 添加定时器
    c.AddTimer("collect_1m", 1*time.Minute, c.collectKlines)
    
    return nil
}

func (c *KlineCollector) collectKlines(ctx context.Context) error {
    // 采集数据
    klines, err := c.client.GetKlines()
    if err != nil {
        return err
    }
    
    // 发布事件
    return c.PublishEvent(&DataEvent{
        Type: "data.kline.collected",
        Data: klines,
    })
}
```

## 3. 配置文件

在 `configs/config.yaml` 中添加新数据源的配置：

```yaml
# 数据源配置
sources:
  market:
    - name: "newexchange"
      enabled: true
      config: "configs/sources/market/newexchange.yaml"
```

创建数据源配置文件 `configs/sources/market/newexchange.yaml`：

```yaml
# 新交易所数据源配置
app:
  id: "newexchange"
  name: "新交易所"
  description: "新交易所数据采集"
  type: "market"

api:
  base_url: "https://api.newexchange.com"

auth:
  api_key: "${NEWEXCHANGE_API_KEY}"
  api_secret: "${NEWEXCHANGE_API_SECRET}"

collectors:
  kline:
    enabled: true
    symbols: 
      - "BTCUSDT"
      - "ETHUSDT"
    intervals:
      - "1m"
      - "5m"
```

## 4. 导入包触发自注册

在主程序中导入新的数据源包：

```go
// cmd/collector/main.go
import (
    // ... 其他导入
    
    // 导入新交易所，触发自注册
    _ "github.com/mooyang-code/data-collector/internal/source/market/newexchange"
    _ "github.com/mooyang-code/data-collector/internal/source/market/newexchange/collectors"
)
```

## 5. 最佳实践

1. **错误处理**：始终处理 API 调用的错误
2. **日志记录**：记录关键操作和错误信息
3. **配置验证**：在初始化时验证配置的完整性
4. **资源管理**：正确管理连接、定时器等资源
5. **单元测试**：为采集器编写单元测试

## 6. 测试新采集器

```bash
# 编译项目
make build

# 使用测试配置运行
./bin/collector --config configs/config.yaml

# 查看日志确认采集器正常工作
tail -f logs/collector.log
```

## 7. 常见问题

### Q: 采集器没有被加载？
A: 检查是否在主程序中导入了采集器包，init 函数需要被触发。

### Q: 如何调试采集器？
A: 使用日志级别 debug 运行，查看详细的执行日志。

### Q: 如何处理 API 限流？
A: 使用内置的速率限制器或在 API 客户端中实现。

## 8. 参考示例

查看 Binance 实现作为参考：
- App 实现：`internal/source/market/binance/app.go`
- K线采集器：`internal/source/market/binance/collectors/kline.go`
- 行情采集器：`internal/source/market/binance/collectors/ticker.go`