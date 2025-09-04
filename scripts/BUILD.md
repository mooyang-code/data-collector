# Data Collector 构建指南

## 概述

本文档描述了如何构建和部署 Data Collector 项目。项目提供了多种构建方式，包括 Makefile 和构建脚本。

## 环境要求

### 基础环境
- Go 1.21 或更高版本
- Git
- Make (可选)

### 开发工具 (可选)
- golangci-lint (代码检查)
- protoc (Protocol Buffers 编译器)
- Docker (容器化部署)

## 快速开始

### 1. 使用 Makefile (推荐)

```bash
# 查看所有可用命令
make help

# 完整构建 (包含依赖安装、测试、构建)
make all

# 快速构建 (跳过测试)
make quick-build

# 只构建主程序
make build-collector

# 构建所有组件
make build-all
```

### 2. 使用构建脚本

```bash
# 基础构建
./scripts/build.sh

# 指定版本构建
./scripts/build.sh v1.2.0
```

## 详细构建选项

### Makefile 目标

#### 构建目标
- `build-collector` - 构建数据采集器
- `build-symtool` - 构建符号工具
- `build-klinedump` - 构建K线导出工具
- `build-trpc-server` - 构建TRPC服务器
- `build-trpc-client` - 构建TRPC客户端
- `build-all` - 构建所有二进制程序

#### 开发工具
- `deps` - 安装Go依赖
- `proto` - 生成protobuf代码
- `check` - 代码检查(lint + vet)
- `test` - 运行测试
- `coverage` - 生成测试覆盖率报告
- `fmt` - 格式化代码
- `tidy` - 整理依赖

#### 数据管理
- `init-data` - 初始化数据目录
- `clean-data` - 清理数据文件
- `dev-data` - 开发模式数据设置

#### 运行和部署
- `dev` - 开发模式运行
- `run` - 在构建目录运行服务
- `stop` - 停止运行的服务
- `install` - 完整构建并安装到release目录

#### 发布和容器化
- `release` - 跨平台发布构建
- `docker` - 构建Docker镜像
- `docker-push` - 推送Docker镜像

### 构建脚本选项

构建脚本提供了简化的构建流程：

```bash
# 基础用法
./scripts/build.sh [版本号]

# 示例
./scripts/build.sh v1.0.0
./scripts/build.sh dev
```

## 构建输出

### 目录结构

构建完成后，会在 `release/` 目录下生成以下结构：

```
release/
├── bin/                    # 二进制文件
│   ├── data-collector     # 主程序
│   ├── symtool           # 符号工具
│   ├── klinedump         # K线导出工具
│   ├── trpc-server       # TRPC服务器
│   └── trpc-client       # TRPC客户端
├── configs/              # 配置文件
│   ├── config.yaml       # 主配置
│   └── trpc_go.yaml      # TRPC配置
├── data/                 # 数据目录
├── log/                  # 日志目录
├── start.sh              # 启动脚本
├── stop.sh               # 停止脚本
└── README.md             # 使用说明
```

### 二进制文件

- **data-collector**: 主要的数据采集器程序
- **symtool**: 交易对符号管理工具
- **klinedump**: K线数据导出工具
- **trpc-server**: TRPC服务器端程序
- **trpc-client**: TRPC客户端程序

## 开发工作流

### 1. 初始化开发环境

```bash
# 安装开发工具
make install-tools

# 初始化项目
make init

# 安装依赖
make deps
```

### 2. 开发过程

```bash
# 代码格式化
make fmt

# 代码检查
make check

# 运行测试
make test

# 生成覆盖率报告
make coverage
```

### 3. 构建和测试

```bash
# 快速构建
make quick-build

# 开发模式运行
make dev

# 完整构建
make install
```

## 跨平台构建

### 发布构建

```bash
# 构建所有平台的发布版本
make release
```

支持的平台：
- linux/amd64
- linux/arm64
- darwin/amd64
- darwin/arm64
- windows/amd64

### Docker 构建

```bash
# 构建Docker镜像
make docker

# 推送到仓库
make docker-push
```

## 测试

### 单元测试

```bash
# 运行所有测试
make test

# 运行特定模块测试
make test-collector
make test-storage
make test-services
```

### 集成测试

```bash
# 运行集成测试
make integration-test

# 运行性能测试
make perf-test
```

### 测试覆盖率

```bash
# 生成覆盖率报告
make coverage

# 查看覆盖率报告
open coverage.html
```

## 部署

### 本地部署

```bash
# 构建并安装
make install

# 启动服务
cd release && ./start.sh

# 停止服务
cd release && ./stop.sh
```

### 生产部署

1. 使用发布构建：
```bash
make release VERSION=v1.0.0
```

2. 解压到目标服务器：
```bash
tar -xzf data-collector-v1.0.0-linux-amd64.tar.gz
cd data-collector-v1.0.0-linux-amd64
```

3. 配置和启动：
```bash
# 编辑配置文件
vim configs/config.yaml

# 启动服务
./start.sh
```

## 故障排除

### 常见问题

1. **Go版本过低**
   - 确保使用 Go 1.21 或更高版本

2. **依赖下载失败**
   - 检查网络连接
   - 配置 Go 代理：`export GOPROXY=https://goproxy.cn,direct`

3. **构建失败**
   - 清理缓存：`make clean`
   - 重新安装依赖：`make deps`

4. **测试失败**
   - 检查测试环境配置
   - 查看详细错误信息

### 调试技巧

1. 使用详细输出：
```bash
make test -v
```

2. 检查构建日志：
```bash
make build-all 2>&1 | tee build.log
```

3. 验证二进制文件：
```bash
./release/bin/data-collector --version
```

## 贡献指南

### 代码提交前检查

```bash
# 完整的代码质量检查
make check

# 运行所有测试
make test-all

# 确保构建成功
make build-all
```

### 发布流程

1. 更新版本号
2. 运行完整测试：`make test-all`
3. 构建发布版本：`make release VERSION=vX.Y.Z`
4. 测试发布包
5. 创建Git标签和发布

## 参考资料

- [Go 官方文档](https://golang.org/doc/)
- [Make 教程](https://www.gnu.org/software/make/manual/)
- [Docker 文档](https://docs.docker.com/)
- [项目架构文档](../docs/ARCHITECTURE.md)
