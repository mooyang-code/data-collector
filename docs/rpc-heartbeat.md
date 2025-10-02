# 心跳服务架构

## 概述

数据采集器使用高效的TRPC协议进行心跳上报，实现节点状态的实时监控。

## 技术优势

- **更高的传输效率**：二进制协议，数据传输量更小
- **更低的延迟**：避免HTTP协议的开销
- **更好的错误处理**：内置的错误码和重试机制
- **连接复用**：减少连接建立的开销

## 心跳机制

### 心跳间隔
- 基础间隔：5秒
- 随机延迟：0-2秒（防止心跳风暴）
- 超时检测：11秒

### 心跳数据结构
```protobuf
message HeartbeatReq {
  AppInfo app_info = 1;           // 应用认证信息
  string node_id = 2;             // 节点ID
  int64 timestamp = 3;            // 时间戳
  string status = 4;              // 节点状态: running/idle
  repeated RunningTaskInfo running_tasks = 5;  // 运行中的任务列表
}
```

## RPC服务配置

### 服务端点
- 服务名：`trpc.moox.server.CloudNodeAPI`
- 命名空间：`Production`
- 超时时间：10秒

### 认证信息
```go
AppInfo {
    AppId:  "data-collector",
    AppKey: "collector-key"  // TODO: 从配置读取
}
```

## 监控和调试

### 日志标识

心跳服务日志前缀：`[Heartbeat]`

### 常见问题

1. **RPC连接失败**
   - 检查moox服务是否启动了RPC端口
   - 确认网络连通性
   - 查看防火墙规则

2. **认证失败**
   - 检查AppId和AppKey配置
   - 确认RPC服务端的认证配置

3. **性能问题**
   - 监控心跳间隔是否正常（5秒）
   - 检查网络延迟

## 主要特性

| 特性 | 说明 |
|------|------|
| 平均延迟 | 10-30ms |
| 协议格式 | 二进制Protobuf |
| 连接管理 | 长连接复用 |
| 错误处理 | 内置重试机制 |

## 心跳流程

1. 节点启动时初始化RPC客户端
2. 立即发送第一次心跳
3. 启动定时器，每5秒（加随机延迟）发送心跳
4. 心跳包含节点状态和运行中的任务信息
5. 服务端更新节点状态，返回确认