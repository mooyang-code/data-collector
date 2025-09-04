# Timer 定时器基础组件

## 概述

Timer 是一个基于 cron 的定时器基础组件，为数据采集器提供统一的定时任务管理功能。它支持秒级精度的定时任务调度、任务状态监控、执行历史记录等功能。

## 特性

- **灵活的调度**: 支持标准 cron 表达式，包括秒级调度
- **并发控制**: 可配置最大并发任务数
- **任务管理**: 支持任务的启用/禁用、手动触发
- **执行监控**: 记录任务执行历史和状态
- **错误处理**: 支持任务重试机制
- **优雅关闭**: 支持优雅停止所有正在执行的任务

## 核心组件

### 1. Timer 接口

```go
type Timer interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    AddJob(job *Job) error
    RemoveJob(jobID JobID) error
    GetJob(jobID JobID) (*Job, error)
    ListJobs() ([]*Job, error)
    EnableJob(jobID JobID) error
    DisableJob(jobID JobID) error
    TriggerJob(jobID JobID) error
    GetJobExecutions(jobID JobID, limit int) ([]*JobExecution, error)
    IsRunning() bool
}
```

### 2. Job 任务定义

```go
type Job struct {
    ID          JobID
    Name        string
    Description string
    CronExpr    string
    Func        JobFunc
    Status      JobStatus
    Timeout     time.Duration
    MaxRetries  int
    Enabled     bool
    // ... 其他字段
}
```

### 3. JobBuilder 任务构建器

```go
job := NewJobBuilder().
    WithID("my_job").
    WithName("我的任务").
    WithCron("*/30 * * * * *").
    WithFunc(myJobFunc).
    WithTimeout(5 * time.Minute).
    Build()
```

## 使用示例

### 基本使用

```go
// 1. 创建定时器
timer, err := timer.GetDefaultTimer()
if err != nil {
    log.Fatal(err)
}

// 2. 启动定时器
ctx := context.Background()
if err := timer.Start(ctx); err != nil {
    log.Fatal(err)
}
defer timer.Stop(ctx)

// 3. 创建任务
job := timer.NewJobBuilder().
    WithID("example_job").
    WithName("示例任务").
    WithCron("*/10 * * * * *"). // 每10秒执行一次
    WithFunc(func(ctx context.Context) error {
        log.Info("任务执行中...")
        return nil
    }).
    Build()

// 4. 添加任务
if err := timer.AddJob(job); err != nil {
    log.Fatal(err)
}
```

### 周期性任务

```go
helper := timer.NewJobHelper()

// 创建周期性任务
periodicJob := helper.CreatePeriodicJob("data_sync", "数据同步任务", 30*time.Second, func(ctx context.Context) error {
    // 数据同步逻辑
    log.Info("执行数据同步...")
    return nil
})

timer.AddJob(periodicJob)
```

### 高级功能

```go
// 手动触发任务
timer.TriggerJob("my_job")

// 禁用任务
timer.DisableJob("my_job")

// 获取执行历史
executions, _ := timer.GetJobExecutions("my_job", 10)
for _, exec := range executions {
    log.Printf("执行: %s, 状态: %s, 耗时: %v", 
        exec.ID, exec.Status, exec.Duration)
}

// 列出所有任务
jobs, _ := timer.ListJobs()
for _, job := range jobs {
    log.Printf("任务: %s, 状态: %s, 执行次数: %d", 
        job.Name, job.Status, job.RunCount)
}
```

## 配置选项

```go
config := &timer.Config{
    Enabled:           true,
    Timezone:          "UTC",
    MaxConcurrentJobs: 10,
    JobTimeout:        30 * time.Minute,
    EnableRecovery:    false,
    HistoryRetention:  24 * time.Hour,
}
```

## Cron 表达式

支持 6 位 cron 表达式（包含秒）：

```
秒 分 时 日 月 周
*  *  *  *  *  *
```

常用表达式：
- `* * * * * *` - 每秒执行
- `0 * * * * *` - 每分钟执行
- `0 0 * * * *` - 每小时执行
- `0 0 0 * * *` - 每天执行
- `*/30 * * * * *` - 每30秒执行
- `0 */5 * * * *` - 每5分钟执行

## 业务特定的辅助工具

基础定时器组件保持通用性，业务相关的任务创建通过专门的辅助工具实现：

### K线任务辅助工具

```go
// 使用K线任务辅助工具
import "github.com/mooyang-code/data-collector/internal/core/klines"

klineHelper := klines.NewKlineJobHelper()

// 创建K线数据获取任务
klineJob := klineHelper.CreateKlineJob("binance", "BTCUSDT", "1m", func(ctx context.Context) error {
    // K线数据获取逻辑
    return nil
})

timer.AddJob(klineJob)
```

### 健康检查任务辅助工具

```go
// 使用健康检查任务辅助工具
import "github.com/mooyang-code/data-collector/internal/infra/health"

healthHelper := health.NewHealthJobHelper()

// 创建数据库健康检查任务
dbHealthJob := healthHelper.CreateDatabaseHealthCheckJob("clickhouse", func(ctx context.Context) error {
    // 数据库健康检查逻辑
    return nil
})

timer.AddJob(dbHealthJob)
```

### K线采集器（内置定时器）

```go
// 创建配置
config := &klines.CollectorConfig{
    EnableTimer:       true,
    Timezone:          "UTC",
    MaxConcurrentJobs: 5,
    JobTimeout:        2 * time.Minute,
    MaxRetries:        3,
}

// 创建K线采集器（内置定时器功能）
collector, err := binance.NewBinanceKlineCollector("", 1000, config)

// 启动采集器
collector.Start(ctx)

// WebSocket订阅（实时数据流）
collector.Subscribe("BTCUSDT", "1m")
collector.Subscribe("ETHUSDT", "5m")

// 定时器任务（定时获取数据）
collector.AddKlineTimerJob("BTCUSDT", "1m")    // 自动转换为cron表达式
collector.AddKlineTimerJob("ETHUSDT", "5m")

// 自定义定时任务
fetchFunc := func(ctx context.Context) error {
    // 自定义数据获取逻辑
    return nil
}
collector.AddTimerJob("BNBUSDT", "1h", "0 0 * * * *", fetchFunc)
```

## 最佳实践

1. **任务超时**: 为每个任务设置合理的超时时间
2. **并发控制**: 根据系统资源设置最大并发任务数
3. **错误处理**: 在任务函数中妥善处理错误
4. **资源清理**: 确保在应用关闭时调用 Stop() 方法
5. **监控**: 定期检查任务执行状态和历史记录

## 注意事项

- 任务函数应该是幂等的
- 避免在任务中执行长时间阻塞操作
- 合理设置重试次数和间隔
- 定期清理执行历史记录
- 在高并发场景下注意内存使用

## 故障排除

### 常见问题

1. **任务不执行**: 检查 cron 表达式是否正确
2. **内存泄漏**: 检查是否正确调用 Stop() 方法
3. **任务超时**: 调整任务超时时间或优化任务逻辑
4. **并发问题**: 检查最大并发任务数设置

### 调试技巧

```go
// 启用详细日志
log.SetLevel(log.DebugLevel)

// 检查任务状态
jobs, _ := timer.ListJobs()
for _, job := range jobs {
    if job.Status == timer.JobStatusFailed {
        log.Errorf("任务失败: %s, 错误: %s", job.Name, job.LastError)
    }
}
```
