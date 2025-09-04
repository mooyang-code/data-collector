# Data Collector

## 构建信息
- 版本: dev
- 构建时间: 2025-09-03 23:52:42
- Git提交: unknown

## 目录结构
```
release/
├── bin/                    # 二进制文件目录
│   └── data-collector          # 主程序
├── start.sh               # 启动脚本
├── stop.sh                # 停止脚本
├── configs/               # 配置文件目录
│   ├── config.yaml        # 主配置文件
│   └── trpc_go.yaml       # TRPC框架配置
├── log/                   # 日志目录
└── data/                  # 数据目录
```

## 使用方法

### 直接运行
```bash
cd bin && ./data-collector
```

### 使用启动脚本（推荐）
```bash
# 启动服务
./start.sh

# 停止服务
./stop.sh
```

### 配置文件
根据部署环境选择相应的配置文件：
- 主配置：使用 `configs/config.yaml`
- TRPC配置：使用 `configs/trpc_go.yaml`

### 程序使用
```bash
# 查看版本信息
./bin/data-collector --version

# 查看帮助信息
./bin/data-collector --help

# 使用指定配置文件运行
./bin/data-collector -config=./configs/config.yaml
```

### 环境变量
生产环境建议设置以下环境变量：
```bash
export DATA_COLLECTOR_CONFIG="./configs/config.yaml"
export LOG_LEVEL="info"
export DATA_DIR="./data"
```

### 日志
服务日志存储在 `log/app.log` 文件中。

## 注意事项
1. 确保端口没有被占用
2. 确保配置文件正确
3. 定期备份数据目录
4. 监控日志文件大小
