# 数据采集器云函数使用指南

## 概述

数据采集器现在支持两种运行模式：
- **独立运行模式（Standalone）**：传统的定时采集模式，适合长期运行的服务器环境
- **云函数模式（Serverless）**：基于事件驱动的采集模式，适合按需采集场景

## 云函数模式特性

1. **动态任务管理**：通过云函数事件动态创建、更新、删除采集任务
2. **按需采集**：只在需要时启动采集器，节省资源
3. **任务持久化**：使用Bleve嵌入式存储保存任务状态，无需外部依赖
4. **灵活的信令控制**：支持细粒度的采集配置

## 部署步骤

### 1. 构建云函数

```bash
# 构建云函数版本
make build-scf

# 构建完成后会生成 collector-scf.zip 文件
```

### 2. 部署到云函数

将生成的 `collector-scf.zip` 文件通过以下方式之一部署：

1. **腾讯云控制台**：
   - 登录腾讯云函数控制台
   - 创建新函数或更新已有函数
   - 上传 collector-scf.zip
   - 配置运行时为 Go1
   - 设置处理程序为 main

2. **腾讯云CLI**：
   ```bash
   # 使用腾讯云CLI部署（需要先安装配置CLI）
   tcf function deploy --name data-collector --runtime Go1 --handler main --code-source collector-scf.zip
   ```

### 3. 函数配置

在云函数控制台设置以下配置：
- **内存**: 512MB（推荐）或更高
- **超时时间**: 900秒
- **环境变量**: 
  - `TZ`: Asia/Shanghai
- **触发器**: 根据需要配置API网关或定时触发器

## 信令格式

云函数通过接收JSON格式的信令来控制采集器：

### 创建任务

```json
{
  "action": "create_task",
  "task_id": "task_001",
  "project_id": "moox_project_123",
  "dataset": "crypto_market",
  "data_type": "kline",
  "source": "binance",
  "symbols": ["BTCUSDT", "ETHUSDT"],
  "frequency": "5m",
  "start_time": "2024-01-01T00:00:00Z",
  "end_time": "2024-12-31T23:59:59Z",
  "config": {
    "batch_size": 100,
    "history_days": 30
  }
}
```

### 更新任务

```json
{
  "action": "update_task",
  "task_id": "task_001",
  "config": {
    "symbols": ["BTCUSDT", "ETHUSDT", "BNBUSDT"],
    "frequency": "1h"
  }
}
```

### 删除任务

```json
{
  "action": "delete_task",
  "task_id": "task_001"
}
```

### 查询任务列表

```json
{
  "action": "list_tasks",
  "project_id": "moox_project_123",
  "status": "running"
}
```

### 获取任务状态

```json
{
  "action": "get_task_status",
  "task_id": "task_001"
}
```

## 本地测试

### 1. 运行云函数模式

```bash
# 使用云函数配置文件运行
make run-serverless

# 或手动运行
go run main.go -config configs/config.serverless.yaml
```

### 2. 模拟云函数调用

```bash
# 创建测试事件文件
cat > test-event.json << EOF
{
  "action": "create_task",
  "task_id": "test_001",
  "project_id": "test_project",
  "dataset": "test_dataset",
  "data_type": "kline",
  "source": "binance",
  "symbols": ["BTCUSDT"],
  "frequency": "1h"
}
EOF

# 使用 curl 测试（如果启用了API网关）
curl -X POST https://your-api-gateway-url/task \
  -H "Content-Type: application/json" \
  -d @test-event.json
```

## 配置说明

### 主配置文件

云函数模式使用专门的配置文件 `configs/config.serverless.yaml`：

```yaml
runtime:
  mode: "serverless"  # 设置为云函数模式
  serverless:
    provider: "tencent"
    enable_local_timer: false  # 关闭本地定时器
    task_store:
      type: "bleve"
      path: "/tmp/data/tasks"  # 使用临时目录
```

### 任务存储

云函数使用Bleve作为嵌入式任务存储：
- 无需外部数据库依赖
- 支持全文搜索
- 任务状态自动持久化
- 存储位置：`/tmp/data/tasks`（云函数临时目录）

## 监控和日志

### 日志查看

1. **云函数控制台**：在腾讯云函数控制台查看实时日志
2. **日志服务**：配置CLS日志服务进行集中日志管理

### 监控指标

- 调用次数
- 执行时间
- 错误率
- 任务状态统计

## 最佳实践

1. **合理设置超时时间**：根据采集任务的复杂度设置函数超时时间（默认900秒）
2. **内存配置**：建议至少512MB内存，复杂任务可增加到1024MB
3. **并发控制**：通过配置限制同时运行的采集任务数
4. **错误处理**：任务失败会自动记录错误信息，可通过查询接口获取

## 故障排查

### 常见问题

1. **任务创建失败**
   - 检查信令格式是否正确
   - 确认必填字段是否完整
   - 查看云函数日志获取详细错误信息

2. **采集器无法启动**
   - 检查数据源配置是否正确
   - 确认API密钥等认证信息是否有效
   - 查看任务状态中的错误信息

3. **存储空间不足**
   - 云函数/tmp目录有空间限制
   - 定期清理过期任务
   - 考虑使用外部存储服务

## API 参考

### 响应格式

成功响应：
```json
{
  "success": true,
  "data": {
    "task_id": "task_001",
    "status": "running",
    "message": "任务创建成功"
  }
}
```

错误响应：
```json
{
  "success": false,
  "message": "任务信令验证失败",
  "error": "data_type不能为空"
}
```

### 任务状态

- `pending`: 等待执行
- `running`: 正在运行
- `completed`: 已完成
- `failed`: 执行失败
- `stopped`: 已停止