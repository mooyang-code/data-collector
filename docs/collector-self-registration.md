# 采集器自注册机制使用指南

## 概述

本项目实现了采集器的自注册机制，使得添加新的采集器时代码更加内聚，不需要修改其他模块的代码。

## 架构设计

### 目录结构

```
internal/app/
├── {exchange}/              # 交易所目录（如 binance, okx, bybit）
│   └── {datatype}/         # 数据类型目录（如 klines, symbols）
│       ├── register.go     # 自注册代码（必需）
│       ├── collector.go    # 采集器实现
│       └── config.go       # 配置定义
│
```

### 已实现的采集器

**Binance 交易所:**
- `internal/app/binance/klines/` - 币安K线采集器
- `internal/app/binance/symbols/` - 币安交易对采集器

**OKX 交易所:**
- `internal/app/okx/klines/` - OKX K线采集器
- `internal/app/okx/symbols/` - OKX交易对采集器

## 自注册工作原理

### 1. 采集器包的 init() 函数

每个采集器包都有一个 `register.go` 文件，包含 `init()` 函数：

```go
// internal/app/{exchange}/{datatype}/register.go
func init() {
    // 注册现货采集器
    app.RegisterCollectorCreator("exchange", "datatype", "spot", createSpotCollector)
    
    // 注册合约采集器
    app.RegisterCollectorCreator("exchange", "datatype", "futures", createFuturesCollector)
    
    log.Info("采集器注册完成")
}
```

### 2. 主程序直接导入

`main.go` 直接导入所有采集器包以触发自注册：

```go
import (
    // 其他导入...
    
    // 导入采集器包以触发自注册
    _ "github.com/mooyang-code/data-collector/internal/app/binance/klines"
    _ "github.com/mooyang-code/data-collector/internal/app/binance/symbols"
    _ "github.com/mooyang-code/data-collector/internal/app/okx/klines"
    _ "github.com/mooyang-code/data-collector/internal/app/okx/symbols"
)

func main() {
    // 导入语句已经触发了自注册
    // ... 初始化代码
}
```

## 添加新采集器的步骤

### 步骤 1: 创建采集器目录结构

```bash
mkdir -p internal/app/{exchange}/{datatype}
```

例如，添加 Bybit 的 K线采集器：
```bash
mkdir -p internal/app/bybit/klines
```

### 步骤 2: 实现采集器注册文件

创建 `internal/app/{exchange}/{datatype}/register.go`：

```go
// Package {datatype} {Exchange} {DataType}采集器自注册
package {datatype}

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
    // 注册现货采集器
    app.RegisterCollectorCreator("{exchange}", "{datatype}", "spot", create{Exchange}Spot{DataType}Collector)
    
    // 注册合约采集器
    app.RegisterCollectorCreator("{exchange}", "{datatype}", "futures", create{Exchange}Futures{DataType}Collector)
    
    log.Info("{Exchange}{DataType}采集器注册完成")
}

// create{Exchange}Spot{DataType}Collector 创建现货采集器
func create{Exchange}Spot{DataType}Collector(appName, collectorName string, config *configs.Collector) (app.Collector, error) {
    // 实现采集器创建逻辑
    // ...
}

// create{Exchange}Futures{DataType}Collector 创建合约采集器  
func create{Exchange}Futures{DataType}Collector(appName, collectorName string, config *configs.Collector) (app.Collector, error) {
    // 实现采集器创建逻辑
    // ...
}
```

### 步骤 3: 实现采集器核心逻辑

创建 `collector.go`、`config.go` 等文件实现具体的采集逻辑。

### 步骤 4: 添加导入语句

在 `main.go` 中添加新包的导入：

```go
import (
    // 现有导入...
    
    // 新添加的采集器
    _ "github.com/mooyang-code/data-collector/internal/app/{exchange}/{datatype}"
)
```

### 步骤 5: 重新构建

```bash
go build -o main main.go
```

新采集器会自动被系统识别和注册。

## 验证自注册机制

你可以创建一个简单的测试程序来验证：

```go
package main

import (
    "fmt"
    "github.com/mooyang-code/data-collector/internal/app"
    
    // 导入采集器包以触发自注册
    _ "github.com/mooyang-code/data-collector/internal/app/binance/klines"
    _ "github.com/mooyang-code/data-collector/internal/app/binance/symbols"
    _ "github.com/mooyang-code/data-collector/internal/app/okx/klines"
    _ "github.com/mooyang-code/data-collector/internal/app/okx/symbols"
)

func main() {
    // 获取注册中心
    registry := app.GetGlobalRegistry()
    
    // 检查已注册的采集器
    types := registry.GetSupportedTypes()
    fmt.Printf("已注册 %d 个采集器:\n", len(types))
    for i, t := range types {
        fmt.Printf("  %d. %s\n", i+1, t)
    }
}
```

## 配置文件对应

每个采集器需要在 `configs/config.yaml` 中有相应的配置：

```yaml
apps:
  {exchange}:
    name: "{exchange}-app"
    enabled: true
    collectors:
      spot_{datatype}_collector:
        name: "spot-{datatype}-collector"
        enabled: true
        dataType: "{datatype}"
        marketType: "spot"
        # ... 其他配置
      
      futures_{datatype}_collector:
        name: "futures-{datatype}-collector"
        enabled: true
        dataType: "{datatype}"
        marketType: "futures"
        # ... 其他配置
```

## 最佳实践

1. **命名规范**: 
   - 交易所名称使用小写（binance, okx, bybit）
   - 数据类型使用小写（klines, symbols, trades）
   - 注册时的类型字符串格式：`{exchange}.{datatype}.{marketType}`

2. **错误处理**: 
   - 在创建函数中进行完整的错误处理
   - 验证配置参数的有效性

3. **日志记录**: 
   - 在 init() 函数中记录注册完成信息
   - 在创建函数中记录采集器创建信息

4. **代码组织**: 
   - 保持每个采集器包的独立性
   - 使用统一的接口和包装器模式

## 故障排查

如果采集器没有被注册，检查：

1. `main.go` 中是否添加了导入语句
2. `register.go` 文件中的 init() 函数是否正确调用了注册函数
3. 包名和目录名是否一致
4. 是否有编译错误阻止了包的加载

通过这种自注册机制，添加新的采集器变得非常简单，代码更加内聚，维护成本更低。