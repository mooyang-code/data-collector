# Cron 调度器迁移指南

## 概述

本文档介绍如何将现有的基于 `time.Ticker` 的定时器迁移到新的 Cron 调度器，以实现整点触发功能。

## 主要改进

### 1. 整点执行
- **原实现**：定时器基于程序启动时间，例如程序在 14:23:45 启动，5分钟定时器会在 14:28:45、14:33:45 执行
- **新实现**：使用 Cron 表达式，5分钟定时器会在 14:25:00、14:30:00、14:35:00 等整点执行

### 2. 更灵活的调度
- 支持标准 Cron 表达式（6位，包含秒）
- 支持工作日、特定时间点等复杂调度需求
- 自动将时间间隔转换为合适的 Cron 表达式

## 迁移步骤

### 方案一：最小化修改（推荐用于快速迁移）

保持现有 `BaseCollector` 接口不变，只需修改内部实现：

```go
// 1. 修改 BaseCollector 的构造函数，添加调度器
type BaseCollector struct {
    // ... 其他字段
    scheduler scheduler.Scheduler
}

func NewBaseCollector(id, typ, dataType string) *BaseCollector {
    return &BaseCollector{
        // ... 其他初始化
        scheduler: scheduler.NewCronScheduler(),
    }
}

// 2. 修改 Start 方法，启动调度器
func (c *BaseCollector) Start(ctx context.Context) error {
    // ... 其他逻辑
    
    // 启动调度器
    if err := c.scheduler.Start(ctx); err != nil {
        return fmt.Errorf("启动调度器失败: %w", err)
    }
    
    // 将现有定时器添加到调度器
    for name, timer := range c.timers {
        taskName := fmt.Sprintf("%s.%s", c.id, name)
        c.scheduler.AddTask(taskName, timer.Interval, timer.Handler)
    }
    
    return nil
}

// 3. 修改 Stop 方法
func (c *BaseCollector) Stop(ctx context.Context) error {
    // 停止调度器
    if err := c.scheduler.Stop(); err != nil {
        log.Printf("停止调度器失败: %v", err)
    }
    // ... 其他逻辑
}
```

### 方案二：使用优化版本（推荐用于新项目）

使用 `BaseCollectorOptimized`，它完全集成了 Cron 调度器：

```go
// 1. 替换基类
type MyCollector struct {
    *collector.BaseCollectorOptimized  // 替换原来的 BaseCollector
}

// 2. 使用新的构造函数
func NewMyCollector() *MyCollector {
    return &MyCollector{
        BaseCollectorOptimized: collector.NewBaseCollectorOptimized(
            "my_collector", "type", "dataType",
        ),
    }
}

// 3. 添加定时器时可以选择使用 Cron 表达式
func (c *MyCollector) Initialize(ctx context.Context) error {
    // 使用时间间隔（会自动转换为整点执行）
    c.AddTimer("5min", 5*time.Minute, c.collect5Min)
    
    // 使用 Cron 表达式（更精确的控制）
    c.AddCronTimer("daily", "0 0 9 * * *", c.dailyTask)  // 每天9点
    
    return nil
}
```

## 时间间隔到 Cron 的转换规则

调度器会智能地将时间间隔转换为合适的 Cron 表达式：

| 时间间隔 | Cron 表达式 | 说明 |
|---------|------------|------|
| 30秒 | `*/30 * * * * *` | 每30秒执行 |
| 5分钟 | `0 */5 * * * *` | 每5分钟的00秒执行 |
| 1小时 | `0 0 */1 * * *` | 每小时的00分00秒执行 |
| 24小时 | `0 0 0 * * *` | 每天0点0分0秒执行 |

## 常用 Cron 表达式示例

```go
// 每分钟整点
"0 * * * * *"

// 每小时的第30分钟
"0 30 * * * *"

// 每天上午9点和下午6点
"0 0 9,18 * * *"

// 工作日上午9点
"0 0 9 * * 1-5"

// 每月1号和15号的凌晨2点
"0 0 2 1,15 * *"

// 每季度第一天
"0 0 0 1 1,4,7,10 *"
```

## 监控和调试

### 查看任务状态

```go
// 获取所有任务状态
tasks := scheduler.ListTasks()
for name, status := range tasks {
    log.Printf("任务: %s, 下次执行: %v, 已执行: %d次", 
        name, status.NextRun, status.RunCount)
}
```

### 日志输出

新的调度器会输出详细的日志：
```
[调度器] 2025/01/01 09:00:00 timer.go:135: 成功添加任务: collector.timer - Cron: 0 */5 * * * *
[调度器] 2025/01/01 09:05:00 timer.go:310: 任务 collector.timer 执行成功，耗时: 125ms
```

## 注意事项

1. **首次执行**：Cron 调度器不会立即执行任务，而是等待下一个调度时间点
2. **时区**：默认使用系统时区，可通过 `cron.WithLocation()` 选项指定时区
3. **并发控制**：调度器会避免同一任务的重复执行
4. **优雅停止**：调度器会等待正在执行的任务完成（最多30秒）

## 性能考虑

- Cron 调度器的开销很小，每个任务只占用很少的内存
- 调度精度为秒级，适合大多数数据采集场景
- 对于需要毫秒级精度的场景，仍可使用原生 `time.Ticker`

## 迁移检查清单

- [ ] 确认所有定时器都已迁移到新的调度器
- [ ] 验证整点执行是否符合预期
- [ ] 检查日志中的任务执行时间
- [ ] 测试停止和重启后的行为
- [ ] 更新相关文档和注释