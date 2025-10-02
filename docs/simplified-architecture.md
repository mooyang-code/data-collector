# 简化的数据采集器架构

## 概述

新的简化架构取消了云函数本地的嵌入式存储，改为通过缓存组件统一获取任务配置，实现了更好的解耦和简化。

## 核心组件

### 1. 任务管理器 (TaskManager)
- 管理数据采集任务的生命周期
- 维护运行中任务的状态
- 提供任务的启动、停止、查询功能

### 2. 任务同步器 (TaskSynchronizer)
- 每2秒从缓存组件拉取任务配置
- 对比本地运行任务，执行增删操作
- 自动处理任务配置变更

### 3. 心跳服务 (HeartbeatService)
- 每5秒上报节点状态（带0-2秒随机延迟）
- 上报运行中任务的状态信息
- 处理Moox的恢复请求

### 4. 缓存客户端 (CacheClient)
- 从缓存服务获取节点任务配置
- 缓存服务负责定时从Moox拉取数据

## 数据流程

```
1. 任务配置流程
   Moox DB -> 缓存服务 -> 云函数
   
2. 心跳流程
   云函数 -> Moox (5秒间隔)
   
3. 恢复流程
   Moox检测超时 -> 发送探测(仅moox_url) -> 云函数恢复
```

## 配置示例

```yaml
runtime:
  serverless:
    node_id: "${SCF_FUNCTION_NAME}"
    moox_url: "${MOOX_SERVICE_URL}"
    cache_endpoint: "${CACHE_SERVICE_URL}"
```

## 优势

1. **无状态设计**：云函数不保存任何状态
2. **自动恢复**：2秒内自动同步最新配置
3. **解耦合**：任务管理与云函数分离
4. **易于维护**：逻辑简单，组件职责清晰

## API接口

### Moox端

1. **心跳接收**
   ```
   POST /api/v1/heartbeat/simple
   {
     "node_id": "function-001",
     "timestamp": "2024-01-01T00:00:00Z",
     "status": "running",
     "running_tasks": [...]
   }
   ```

2. **节点任务查询**（供缓存服务调用）
   ```
   GET /api/v1/tasks/node/{nodeID}
   Response: {
     "success": true,
     "data": [任务配置列表]
   }
   ```

### 云函数端

1. **恢复请求处理**
   ```json
   {
     "action": "recovery",
     "data": {
       "moox_url": "https://moox.example.com"
     }
   }
   ```

2. **状态查询**
   ```json
   {
     "action": "status"
   }
   ```

## 部署注意事项

1. 确保环境变量配置正确
   - `MOOX_SERVICE_URL`：Moox服务地址
   - `CACHE_SERVICE_URL`：缓存服务地址
   - `SCF_FUNCTION_NAME`：云函数名称

2. 缓存服务需要能访问Moox API

3. 云函数需要有足够的执行时长限制（建议>60秒）

## 故障处理

1. **缓存服务不可用**：云函数将保持当前任务继续运行
2. **Moox服务不可用**：心跳失败但任务继续执行
3. **云函数迁移**：2秒内自动从缓存恢复任务配置