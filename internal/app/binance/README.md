# 币安交易所数据采集器架构

## 架构概述

本项目采用了分层架构设计，将数据采集逻辑分为公共实现层和交易所特定实现层：

```
/internal/datatype/          # 公共实现层
├── klines/                  # K线数据公共实现
│   ├── base_collector.go    # 基础K线采集器（定时器、事件管理等）
│   ├── interface.go         # K线采集器接口定义
│   └── model.go            # K线数据模型
└── symbols/                 # 交易对数据公共实现
    ├── base_collector.go    # 基础交易对采集器
    ├── interface.go         # 交易对采集器接口定义
    └── model.go            # 交易对数据模型

/internal/app/binance/       # 币安特定实现层
├── klines/                  # K线数据采集
│   ├── adapter.go           # 币安K线适配器（HTTP接口实现）
│   ├── config.go           # 币安K线配置
│   ├── collector.go        # 币安K线采集器（基于公共实现）
│   ├── spot/               # 现货K线应用
│   └── futures/            # 合约K线应用
├── symbols/                 # 交易对数据采集
│   ├── adapter.go          # 币安交易对适配器
│   ├── config.go          # 币安交易对配置
│   └── collector.go       # 币安交易对采集器
└── example.go              # 使用示例
```

## 设计理念

### 1. 公共实现复用
- **定时器管理**：所有交易所的K线采集都使用相同的定时器逻辑
- **事件管理**：统一的事件发布/订阅机制
- **存储管理**：统一的数据存储接口
- **生命周期管理**：统一的启动/停止逻辑

### 2. 适配器模式
每个交易所只需要实现特定的适配器接口：
- **K线适配器**：实现 `KlineAdapter` 接口，提供HTTP API调用逻辑
- **交易对适配器**：实现 `SymbolAdapter` 接口，提供交易对信息获取逻辑

### 3. 配置驱动
- 通过配置文件控制采集行为
- 支持不同交易对、不同周期的灵活配置
- 支持定时器、重试、过滤等策略配置

## 核心组件

### K线采集器 (Klines)

#### 1. 基础采集器 (`BaseKlineCollector`)
- 提供定时器管理功能
- 提供事件发布机制
- 提供订阅管理功能
- 提供速率限制功能

#### 2. 币安适配器 (`BinanceKlineAdapter`)
- 实现币安API的HTTP调用
- 支持现货和合约API
- 处理数据格式转换
- 处理错误重试

#### 3. 币安采集器 (`BinanceKlineCollector`)
- 基于 `BaseKlineCollector`
- 使用 `BinanceKlineAdapter`
- 支持历史数据回补
- 支持多交易对、多周期采集

### 交易对采集器 (Symbols)

#### 1. 基础采集器 (`BaseSymbolsCollector`)
- 提供交易对存储和管理
- 提供增量更新和全量快照
- 提供事件发布机制

#### 2. 币安适配器 (`BinanceSymbolAdapter`)
- 实现币安交易对信息API调用
- 支持现货和合约交易对
- 处理过滤器和权限信息

#### 3. 币安采集器 (`BinanceSymbolCollector`)
- 基于 `BaseSymbolsCollector`
- 使用 `BinanceSymbolAdapter`
- 支持过滤配置
- 支持定时刷新

## 使用方式

### 1. 直接使用采集器

```go
// K线采集器
config := &binanceKlines.BinanceKlineConfig{
    Exchange: "binance_spot",
    BaseURL:  "https://api.binance.com",
    Symbols:  []string{"BTCUSDT", "ETHUSDT"},
    Intervals: []klines.Interval{klines.Interval1m, klines.Interval1h},
}

collector, err := binanceKlines.NewBinanceKlineCollector(config)
if err != nil {
    return err
}

err = collector.Initialize(ctx)
err = collector.StartCollection(ctx)

// 监听事件
events := collector.Events()
for event := range events {
    // 处理K线事件
}
```

### 2. 使用应用级封装

```go
// 现货K线应用
app := spot.NewBinanceSpotKlineApp("binance_spot")
err := app.Initialize(appConfig)
err := app.Start(ctx)

// 合约K线应用
futuresApp := futures.NewBinanceFuturesKlineApp("binance_futures")
err := futuresApp.Initialize(appConfig)
err := futuresApp.Start(ctx)
```

### 3. 直接使用适配器

```go
// 直接使用适配器获取数据
adapter := binanceKlines.NewBinanceKlineAdapter("https://api.binance.com")
klineData, err := adapter.FetchHistoryKlines(ctx, "BTCUSDT", klines.Interval1h, startTime, endTime, 100)
```

## 扩展新交易所

要添加新的交易所（如OKX），只需要：

1. **实现适配器接口**：
   ```go
   type OKXKlineAdapter struct {}
   func (a *OKXKlineAdapter) FetchHistoryKlines(...) ([]*klines.Kline, error) {
       // 实现OKX的API调用逻辑
   }
   ```

2. **创建配置结构**：
   ```go
   type OKXKlineConfig struct {
       // OKX特定的配置项
   }
   ```

3. **创建采集器**：
   ```go
   type OKXKlineCollector struct {
       *klines.BaseKlineCollector
       adapter *OKXKlineAdapter
       config  *OKXKlineConfig
   }
   ```

4. **注册适配器**：
   ```go
   klines.RegisterKlineAdapter("okx", okxAdapter)
   ```

## 配置示例

### K线采集配置
```yaml
exchange: "binance_spot"
baseUrl: "https://api.binance.com"
symbols: ["BTCUSDT", "ETHUSDT", "BNBUSDT"]
intervals: ["1m", "5m", "1h", "1d"]
timerConfigs:
  "1m":
    cronExpr: "0 * * * * *"
    timeout: "30s"
    maxRetries: 3
    enabled: true
httpTimeout: "30s"
requestLimit: 1200
enableBackfill: true
backfillDays: 7
```

### 交易对采集配置
```yaml
exchange: "binance"
baseUrl: "https://api.binance.com"
refreshInterval: "5m"
enableAutoRefresh: true
symbolFilter:
  allowedStatuses: ["TRADING"]
  allowedTypes: ["spot"]
  allowedQuoteAssets: ["USDT", "BTC", "ETH"]
  onlyActiveSymbols: true
enableFiltering: true
```

## 优势

1. **代码复用**：公共逻辑只需实现一次，所有交易所共享
2. **易于扩展**：新增交易所只需实现适配器接口
3. **配置灵活**：通过配置文件控制采集行为
4. **职责分离**：公共逻辑与交易所特定逻辑分离
5. **测试友好**：可以独立测试适配器和采集器
6. **维护简单**：修改公共逻辑时，所有交易所自动受益

## 注意事项

1. **API限制**：不同交易所有不同的API限制，需要在配置中设置合适的请求频率
2. **数据格式**：适配器负责将交易所特定的数据格式转换为统一的标准格式
3. **错误处理**：适配器应该处理网络错误、API错误等异常情况
4. **时区处理**：确保时间戳的时区处理正确
5. **资源管理**：及时释放HTTP连接等资源
