# 量化数据采集器 - 重构版本

## 项目概述

这是一个重构后的量化数据采集器项目，采用清晰的分层架构设计，支持多数据源、多数据类型的实时数据采集和存储。

## 核心特性

- 🏗️ **清晰的架构设计**: 分层架构，职责明确
- 🔧 **可扩展性**: 支持插件化的数据源和存储后端
- 📊 **多数据类型**: 支持K线、标的、深度、成交、行情等数据
- ⚡ **高性能**: 并发采集，批量存储
- 🔍 **监控完善**: 内置健康检查和指标监控
- 🎛️ **配置灵活**: 支持热更新和动态配置

## 架构设计

### 核心组件

```
AppManager (顶层管理器)
    ├── App (应用实例)
    │   ├── Dataset (数据集)
    │   ├── CollectorManager (采集器管理器)
    │   │   └── Collector (数据采集器)
    │   │       └── Trigger (触发器)
    │   └── Storage (存储器)
    └── AppFactory (应用工厂)
```

### 组件说明

1. **AppManager**: 顶层管理器，从配置文件读取App列表并管理App生命周期
2. **App**: 与数据集绑定的应用单元，包含采集器管理器和存储器
3. **Dataset**: 数据集定义，描述一组类似数据的集合
4. **CollectorManager**: 采集器管理器，管理多种类型的数据采集器
5. **Collector**: 数据采集器，包含数据类型和触发器
6. **Trigger**: 触发器，控制采集器的执行时机（定时、事件、手动等）
7. **Storage**: 存储器，提供统一的数据存储接口

## 目录结构

```
src/github.com/mooyang-code/data-collector/
├── cmd/                             # 主程序入口
│   └── main.go                      # 统一的主程序
├── configs/                         # 配置文件和配置结构定义
│   ├── config.go                    # 配置结构定义和配置加载
│   ├── global.yaml                  # 全局配置
│   ├── apps.yaml                    # App配置
│   └── datasets.yaml                # 数据集配置
├── model/                           # 共享数据模型
│   ├── enums.go                     # 枚举类型定义
│   └── types.go                     # 数据结构和状态定义
├── internal/                        # 内部实现
│   ├── app/                         # App实现
│   │   ├── interfaces.go            # 核心接口定义
│   │   ├── app.go                   # 统一App实现
│   │   ├── app_factory.go           # App工厂
│   │   └── app_manager.go           # App管理器
│   ├── collector/                   # 采集器实现
│   │   ├── manager.go               # 采集器管理器
│   │   └── base_collector.go        # 基础采集器
│   ├── trigger/                     # 触发器实现
│   │   └── trigger.go               # 触发器实现
│   ├── storage/                     # 存储实现
│   │   └── memory.go                # 内存存储
│   └── datatype/                    # 数据类型实现
│       ├── klines/                  # K线相关
│       └── symbols/                 # 交易对相关
├── docs/                            # 文档
│   └── ARCHITECTURE.md              # 架构设计文档
└── README.md                        # 本文件
```

## 快速开始

### 1. 编译项目

```bash
cd src/github.com/mooyang-code/data-collector
go build -o data-collector cmd/main.go
```

### 2. 配置文件

项目使用三个主要配置文件：

- `configs/global.yaml`: 全局配置（日志、监控、性能等）
- `configs/apps.yaml`: App配置（定义系统中的应用实例）
- `configs/datasets.yaml`: 数据集配置（定义数据源和采集器）

### 3. 运行程序

```bash
# 启动采集器
./data-collector -config=configs

# 查看App列表
./data-collector -config=configs -list-apps

# 查看统计信息
./data-collector -config=configs -stats

# 查看健康状态
./data-collector -config=configs -health

# 调试模式
./data-collector -config=configs -debug
```

## 配置说明

### App配置 (apps.yaml)

```yaml
apps:
  - id: "binance-spot-app"
    name: "Binance现货数据采集器"
    type: "binance_spot"
    dataset_id: "binance_spot_main"
    enabled: true
    description: "采集Binance现货市场数据"
```

### 数据集配置 (datasets.yaml)

```yaml
datasets:
  - id: "binance_spot_main"
    name: "Binance现货主要数据集"
    source:
      type: "binance_spot"
      symbols: ["BTCUSDT", "ETHUSDT"]
      intervals: ["1m", "5m", "1h"]
    collectors:
      - type: "kline"
        trigger:
          type: "cron"
          schedule: "*/1 * * * *"
    storage:
      type: "clickhouse"
      dsn: "tcp://localhost:9000/market_data"
```

## 支持的数据类型

- **K线数据 (kline)**: OHLCV数据，包含开高低收价格和成交量
- **标的数据 (symbol)**: 交易对信息，包含基础货币、计价货币等
- **深度数据 (depth)**: 订单簿数据
- **成交数据 (trade)**: 实时成交记录
- **行情数据 (ticker)**: 24小时统计数据

## 支持的触发器类型

- **定时触发器 (cron)**: 基于Cron表达式的定时触发
- **事件触发器 (event)**: 基于外部事件的触发
- **手动触发器 (manual)**: 手动控制的触发
- **Webhook触发器 (webhook)**: 基于HTTP回调的触发

## 支持的存储后端

- **内存存储 (memory)**: 适用于测试和临时数据
- **文件存储 (file)**: 本地文件存储
- **ClickHouse**: 高性能列式数据库
- **MySQL**: 关系型数据库
- **Redis**: 缓存和队列存储

## 监控和健康检查

### 健康检查端点

```bash
# 检查整体健康状态
curl http://localhost:8081/health

# 检查指标
curl http://localhost:8080/metrics
```

### 监控指标

- App运行状态和数量
- 采集器运行状态和性能
- 存储器状态和统计
- 错误率和延迟统计

## 扩展开发

### 添加新的数据源

1. 实现 `app.App` 接口
2. 在 `AppFactory` 中注册新类型
3. 添加相应的配置支持

### 添加新的存储后端

1. 实现 `app.Storage` 接口
2. 在存储工厂中注册新类型
3. 添加配置支持

### 添加新的触发器

1. 实现 `app.Trigger` 接口
2. 在触发器工厂中注册新类型
3. 添加配置支持

## 性能优化

- 使用连接池管理数据库连接
- 批量写入减少I/O开销
- 并发采集提高吞吐量
- 内存缓存减少重复计算
- 压缩存储节省空间

## 故障排除

### 常见问题

1. **配置文件找不到**: 检查 `-config` 参数路径
2. **App启动失败**: 检查数据集配置和依赖服务
3. **采集器异常**: 查看日志中的错误信息
4. **存储失败**: 检查存储后端连接和权限

### 日志级别

- `debug`: 详细的调试信息
- `info`: 一般信息（默认）
- `warn`: 警告信息
- `error`: 错误信息

## 贡献指南

1. Fork 项目
2. 创建特性分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request

## 许可证

本项目采用 MIT 许可证。

## 联系方式

如有问题或建议，请提交 Issue 或联系项目维护者。
