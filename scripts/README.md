# Data Collector 构建脚本说明

本目录包含了 Data Collector 项目的所有构建、部署和管理脚本。

## 脚本概览

### 核心构建脚本

#### `build.sh` - 主构建脚本
仿照 moox 项目的构建脚本设计，提供完整的构建功能。

**功能特性：**
- 自动构建所有组件（data-collector、symtool、klinedump、trpc-server、trpc-client）
- 生成启动和停止脚本
- 拷贝配置文件到正确位置
- 创建完整的目录结构
- 生成使用说明文档

**使用方法：**
```bash
# 基础构建
./scripts/build.sh

# 指定版本构建
./scripts/build.sh v1.2.0
```

**输出结构：**
```
release/
├── bin/                    # 二进制文件
├── configs/               # 配置文件
├── data/                  # 数据目录
├── log/                   # 日志目录
├── start.sh               # 启动脚本
├── stop.sh                # 停止脚本
└── README.md              # 使用说明
```

### 测试和验证脚本

#### `test_build.sh` - 构建测试脚本
验证构建结果的完整性和正确性。

**功能特性：**
- 检查二进制文件存在性和权限
- 验证配置文件格式
- 测试基本功能（版本信息、帮助信息）
- 生成测试报告

**使用方法：**
```bash
# 测试构建结果
./scripts/test_build.sh
```

### 部署脚本

#### `deploy.sh` - 自动化部署脚本
支持远程服务器部署，包含完整的部署流程。

**功能特性：**
- SSH 远程部署
- 自动备份现有版本
- 系统服务安装
- 部署验证
- Dry-run 模式

**使用方法：**
```bash
# 基础部署
./scripts/deploy.sh -t server.example.com -u deploy

# 安装为系统服务并备份
./scripts/deploy.sh -t 192.168.1.100 -s -b

# 预览部署操作
./scripts/deploy.sh --dry-run -t localhost
```

**参数说明：**
- `-t, --target`: 目标服务器地址
- `-u, --user`: SSH用户名
- `-p, --port`: SSH端口
- `-d, --deploy-dir`: 部署目录
- `-s, --service`: 安装为系统服务
- `-b, --backup`: 备份现有版本
- `--dry-run`: 预览模式

### 清理脚本

#### `clean.sh` - 项目清理脚本
清理各种构建文件、缓存和临时文件。

**功能特性：**
- 分类清理（构建文件、缓存、日志、数据、临时文件）
- Dry-run 预览模式
- 安全确认机制
- 详细的清理报告

**使用方法：**
```bash
# 清理构建文件
./scripts/clean.sh -b

# 清理所有文件
./scripts/clean.sh -a

# 预览清理操作
./scripts/clean.sh --dry-run -a

# 清理缓存和日志
./scripts/clean.sh -c -l
```

**参数说明：**
- `-a, --all`: 清理所有文件
- `-b, --build`: 清理构建文件
- `-c, --cache`: 清理Go缓存
- `-l, --logs`: 清理日志文件
- `-d, --data`: 清理数据文件（谨慎使用）
- `-t, --temp`: 清理临时文件
- `-r, --release`: 清理发布文件
- `--dry-run`: 预览模式
- `-f, --force`: 强制执行

## 文档

#### `BUILD.md` - 详细构建指南
完整的构建文档，包含：
- 环境要求
- 构建选项详解
- 开发工作流
- 跨平台构建
- 故障排除

#### `README.md` - 本文档
脚本使用说明和概览。

## 与 Makefile 的关系

项目同时提供了 Makefile 和构建脚本两套构建系统：

### Makefile 优势
- 声明式构建目标
- 依赖关系管理
- 并行构建支持
- IDE 集成友好

### 构建脚本优势
- 更灵活的逻辑控制
- 详细的输出信息
- 完整的部署流程
- 更好的错误处理

### 推荐使用场景

**开发阶段使用 Makefile：**
```bash
make deps          # 安装依赖
make test           # 运行测试
make build-all      # 构建所有组件
make dev            # 开发模式运行
```

**生产部署使用构建脚本：**
```bash
./scripts/build.sh v1.0.0      # 生产构建
./scripts/test_build.sh         # 验证构建
./scripts/deploy.sh -t prod     # 部署到生产
```

## 最佳实践

### 1. 开发工作流

```bash
# 1. 代码开发
make fmt && make check

# 2. 测试验证
make test

# 3. 本地构建
make build-all

# 4. 完整构建测试
./scripts/build.sh dev
./scripts/test_build.sh

# 5. 清理环境
./scripts/clean.sh -b
```

### 2. 发布流程

```bash
# 1. 版本构建
./scripts/build.sh v1.2.0

# 2. 构建验证
./scripts/test_build.sh

# 3. 跨平台构建
make release VERSION=v1.2.0

# 4. 部署测试
./scripts/deploy.sh --dry-run -t test-server

# 5. 生产部署
./scripts/deploy.sh -t prod-server -s -b
```

### 3. 维护清理

```bash
# 定期清理开发环境
./scripts/clean.sh -b -c -l -t

# 深度清理（包含数据）
./scripts/clean.sh -a

# 预览清理效果
./scripts/clean.sh --dry-run -a
```

## 脚本特性对比

| 特性 | build.sh | test_build.sh | deploy.sh | clean.sh |
|------|----------|---------------|-----------|----------|
| 颜色输出 | ✅ | ✅ | ✅ | ✅ |
| 错误处理 | ✅ | ✅ | ✅ | ✅ |
| Dry-run模式 | ❌ | ❌ | ✅ | ✅ |
| 帮助信息 | ❌ | ❌ | ✅ | ✅ |
| 参数解析 | ✅ | ❌ | ✅ | ✅ |
| 安全确认 | ❌ | ❌ | ✅ | ✅ |
| 详细日志 | ✅ | ✅ | ✅ | ✅ |

## 故障排除

### 常见问题

1. **权限问题**
   ```bash
   chmod +x scripts/*.sh
   ```

2. **构建失败**
   ```bash
   ./scripts/clean.sh -b -c
   make deps
   ./scripts/build.sh
   ```

3. **部署连接失败**
   ```bash
   # 检查SSH连接
   ssh user@server
   
   # 使用dry-run模式测试
   ./scripts/deploy.sh --dry-run -t server
   ```

4. **清理确认**
   ```bash
   # 先预览要删除的文件
   ./scripts/clean.sh --dry-run -a
   ```

### 调试技巧

1. **启用详细输出**
   ```bash
   set -x  # 在脚本开头添加
   ```

2. **检查脚本语法**
   ```bash
   bash -n scripts/build.sh
   ```

3. **逐步执行**
   ```bash
   bash -x scripts/build.sh
   ```

## 贡献指南

### 添加新脚本

1. 遵循现有的命名约定
2. 包含完整的帮助信息
3. 实现错误处理和颜色输出
4. 添加到本文档中

### 修改现有脚本

1. 保持向后兼容性
2. 更新相关文档
3. 测试所有功能分支

### 代码风格

- 使用 4 空格缩进
- 函数名使用下划线分隔
- 变量名使用大写字母
- 添加适当的注释

## 参考资料

- [Bash 脚本编程指南](https://www.gnu.org/software/bash/manual/)
- [Shell 脚本最佳实践](https://google.github.io/styleguide/shellguide.html)
- [Make 官方文档](https://www.gnu.org/software/make/manual/)
- [项目架构文档](../docs/ARCHITECTURE.md)
