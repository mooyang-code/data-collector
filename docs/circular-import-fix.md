# 循环导入问题修复

本文档说明了如何解决采集器自注册机制中的循环导入问题。

## 问题描述

原始的自注册机制出现了循环导入：

```
github.com/mooyang-code/data-collector/internal/app/binance/klines (register.go) ->
github.com/mooyang-code/data-collector/internal/app (collector_factory.go) ->
github.com/mooyang-code/data-collector/internal/app/collectors (import.go) ->
github.com/mooyang-code/data-collector/internal/app/binance/klines
```

## 参考解决方案

参考了 `xData-mini/storage/internal/services/adaptor/dao` 的设计模式：

1. **接口定义在上层包** (`dao/interface.go`)
2. **注册机制在上层包** (`dao/device.go`)
3. **具体实现在子包** (`dao/duckdb/`)
4. **子包通过 `init()` 函数自注册**
5. **上层包不直接导入子包**

## 解决方案

### 1. 删除导致循环导入的文件

- ❌ 删除: `internal/app/collectors/import.go`
- ❌ 删除: `internal/app/collectors/` 目录

### 2. 创建集中的初始化文件

创建 `internal/app/init.go`：

```go
package app

import (
    // 导入所有采集器包以触发自注册
    _ "github.com/mooyang-code/data-collector/internal/app/binance/klines"
    _ "github.com/mooyang-code/data-collector/internal/app/binance/symbols"
    _ "github.com/mooyang-code/data-collector/internal/app/okx/symbols"
)

// InitCollectors 初始化所有采集器
func InitCollectors() {
    // 这个函数确保了上面的导入语句被执行
    // 从而触发各个采集器包的 init() 函数
}
```

### 3. 在应用管理器中调用初始化

修改 `app_manager.go`：

```go
// NewAppManager 创建应用管理器
func NewAppManager(factory AppFactory) AppManager {
    // 初始化采集器注册
    InitCollectors()
    
    return &AppManagerImpl{
        factory: factory,
        apps:    make(map[string]App),
    }
}
```

### 4. 移除工厂中的循环导入

修改 `collector_factory.go`，删除对 `collectors` 包的导入：

```go
import (
    "fmt"
    "github.com/mooyang-code/data-collector/configs"
    // 删除: _ "github.com/mooyang-code/data-collector/internal/app/collectors"
)
```

## 修复后的代码结构

### 文件结构

```
internal/app/
├── init.go                     # 采集器包导入（避免循环导入）
├── collector_interface.go      # 采集器接口定义
├── collector_registry.go       # 采集器注册中心
├── collector_factory.go        # 采集器工厂实现
├── app_factory.go              # 应用工厂
├── app_manager.go              # 应用管理器
└── {exchange}/{datatype}/
    ├── collector.go            # 具体采集器实现
    ├── config.go               # 配置定义
    └── register.go             # 自注册代码
```

### 导入关系

```
app_manager.go -> init.go -> binance/klines/register.go -> collector_registry.go
                          -> binance/symbols/register.go -> collector_registry.go
                          -> okx/symbols/register.go -> collector_registry.go

collector_factory.go -> collector_registry.go

无循环导入！
```

## 工作原理

1. **应用启动时**：
   - `NewAppManager()` 被调用
   - `InitCollectors()` 被调用
   - 导入语句触发各个采集器包的 `init()` 函数

2. **采集器注册**：
   - 各个采集器包的 `init()` 函数执行
   - 调用 `app.RegisterCollectorCreator()` 注册自己
   - 注册信息存储在全局注册中心

3. **采集器创建**：
   - 工厂从注册中心获取创建函数
   - 创建具体的采集器实例

## 优势

1. **✅ 避免循环导入**：通过集中的初始化文件管理导入
2. **✅ 保持自注册**：采集器仍然可以自动注册
3. **✅ 易于扩展**：添加新采集器只需修改 `init.go`
4. **✅ 清晰的依赖关系**：避免了复杂的导入链

## 添加新采集器的步骤

1. **创建采集器实现**：在 `internal/app/{exchange}/{datatype}/` 目录下
2. **添加 register.go**：在 `init()` 函数中调用 `app.RegisterCollectorCreator()`
3. **更新 init.go**：添加新包的导入语句
4. **完成**：新采集器自动被系统识别

## 示例：添加 Bybit 采集器

### 1. 创建采集器实现

```go
// internal/app/bybit/symbols/register.go
func init() {
    app.RegisterCollectorCreator("bybit", "symbols", "spot", createBybitSymbolCollector)
}
```

### 2. 更新初始化文件

```go
// internal/app/init.go
import (
    _ "github.com/mooyang-code/data-collector/internal/app/binance/klines"
    _ "github.com/mooyang-code/data-collector/internal/app/binance/symbols"
    _ "github.com/mooyang-code/data-collector/internal/app/okx/symbols"
    _ "github.com/mooyang-code/data-collector/internal/app/bybit/symbols"  // 新增
)
```

### 3. 自动工作

新采集器会在应用启动时自动注册和识别。

## 总结

通过参考 `xData-mini` 的设计模式，我们成功解决了循环导入问题：

- 保持了自注册机制的优势
- 避免了循环导入的问题
- 代码结构更加清晰
- 易于维护和扩展

这种模式是处理 Go 语言中自注册机制和循环导入问题的最佳实践。
