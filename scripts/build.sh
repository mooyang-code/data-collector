#!/bin/bash

# Data Collector 构建脚本
# 使用方法: ./build.sh [版本号]

set -e  # 遇到错误立即退出

# 颜色输出函数
print_info() {
    echo -e "\033[34m[INFO]\033[0m $1"
}

print_success() {
    echo -e "\033[32m[SUCCESS]\033[0m $1"
}

print_error() {
    echo -e "\033[31m[ERROR]\033[0m $1"
}

print_warning() {
    echo -e "\033[33m[WARNING]\033[0m $1"
}

# 获取版本号
VERSION=${1:-"dev"}
BUILD_TIME=$(date +"%Y-%m-%d %H:%M:%S")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

print_info "开始构建 Data Collector..."
print_info "版本: $VERSION"
print_info "构建时间: $BUILD_TIME"
print_info "Git提交: $GIT_COMMIT"

# 定义路径变量
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
PROJECT_ROOT=$(cd "$SCRIPT_DIR/.." && pwd)
RELEASE_DIR="$PROJECT_ROOT/release"
APP_NAME="data-collector"
BUILD_DIR="$RELEASE_DIR"

print_info "脚本目录: $SCRIPT_DIR"
print_info "项目根目录: $PROJECT_ROOT"
print_info "构建目录: $BUILD_DIR"

# 切换到项目根目录
cd "$PROJECT_ROOT"

# 创建构建目录
print_info "创建构建目录..."
mkdir -p "$BUILD_DIR"
mkdir -p "$BUILD_DIR/bin"
mkdir -p "$BUILD_DIR/configs"
mkdir -p "$BUILD_DIR/data"
mkdir -p "$BUILD_DIR/log"

# 清理旧文件
print_info "清理旧的构建文件..."
rm -f "$BUILD_DIR/bin/$APP_NAME"
rm -f "$BUILD_DIR/$APP_NAME"  # 清理旧的根目录下的二进制文件
rm -f "$BUILD_DIR/configs"/*.yaml

# 构建二进制文件
print_info "开始编译二进制文件..."
export CGO_ENABLED=1

# 添加构建信息
LDFLAGS="-X 'main.Version=$VERSION' -X 'main.BuildTime=$BUILD_TIME' -X 'main.GitCommit=$GIT_COMMIT'"

if go build -ldflags "$LDFLAGS" -o "$BUILD_DIR/bin/$APP_NAME" ./cmd/collector; then
    print_success "二进制文件编译成功: $BUILD_DIR/bin/$APP_NAME"
else
    print_error "二进制文件编译失败"
    exit 1
fi

# 只构建主程序，不构建其他工具
print_info "主程序构建完成"

# 拷贝配置文件
print_info "拷贝配置文件..."
if [ -d "configs" ]; then
    # 清理旧的配置文件
    rm -rf "$BUILD_DIR/configs"/*
    
    # 拷贝所有配置文件和子目录
    cp -r configs/* "$BUILD_DIR/configs/" 2>/dev/null || true
    if [ $? -eq 0 ]; then
        print_success "配置文件拷贝到configs目录成功"
        find "$BUILD_DIR/configs/" -type f | head -10
    else
        print_warning "没有找到配置文件"
    fi

    # 特别拷贝 trpc_go.yaml 到 bin 目录（如果存在）
    if [ -f "configs/trpc_go.yaml" ]; then
        cp configs/trpc_go.yaml "$BUILD_DIR/bin/"
        print_success "trpc_go.yaml 拷贝到bin目录成功"
    else
        print_warning "没有找到 trpc_go.yaml 文件"
    fi
else
    print_warning "configs目录不存在"
fi

# 创建启动脚本
print_info "创建启动脚本..."
cat > "$BUILD_DIR/start.sh" << EOF
#!/bin/bash

# Data Collector 启动脚本

APP_NAME="$APP_NAME"
PID_FILE="./\$APP_NAME.pid"

# 检查并停止已存在的进程
echo "检查已存在的进程..."

# 检查PID文件中的进程
if [ -f "\$PID_FILE" ]; then
    OLD_PID=\$(cat "\$PID_FILE")
    if ps -p "\$OLD_PID" > /dev/null 2>&1; then
        echo "发现运行中的服务 (PID: \$OLD_PID)，正在停止..."
        kill "\$OLD_PID"

        # 等待进程结束
        for i in {1..10}; do
            if ! ps -p "\$OLD_PID" > /dev/null 2>&1; then
                echo "旧进程已停止"
                break
            fi
            sleep 1
        done

        # 如果还在运行，强制杀死
        if ps -p "\$OLD_PID" > /dev/null 2>&1; then
            echo "强制停止旧进程..."
            kill -9 "\$OLD_PID"
        fi
    fi
    rm -f "\$PID_FILE"
fi

# 通过进程名查找并停止可能的进程
RUNNING_PIDS=\$(pgrep -f "\$APP_NAME" 2>/dev/null || true)
if [ ! -z "\$RUNNING_PIDS" ]; then
    echo "发现其他运行中的 \$APP_NAME 进程: \$RUNNING_PIDS"
    echo "正在停止这些进程..."
    echo "\$RUNNING_PIDS" | xargs kill 2>/dev/null || true
    sleep 2

    # 强制杀死仍在运行的进程
    STILL_RUNNING=\$(pgrep -f "\$APP_NAME" 2>/dev/null || true)
    if [ ! -z "\$STILL_RUNNING" ]; then
        echo "强制停止残留进程: \$STILL_RUNNING"
        echo "\$STILL_RUNNING" | xargs kill -9 2>/dev/null || true
    fi
fi

# 启动服务
echo "启动 \$APP_NAME..."
cd ./bin
nohup ./\$APP_NAME > ../log/app.log 2>&1 &
echo \$! > "../\$PID_FILE"
echo "服务已启动 (PID: \$(cat ../\$PID_FILE))"
echo "日志文件: log/app.log"
EOF

# 创建停止脚本
cat > "$BUILD_DIR/stop.sh" << EOF
#!/bin/bash

# Data Collector 停止脚本

APP_NAME="$APP_NAME"
PID_FILE="./\$APP_NAME.pid"

if [ ! -f "\$PID_FILE" ]; then
    echo "PID文件不存在，服务可能没有运行"
    exit 1
fi

PID=\$(cat "\$PID_FILE")

if ps -p "\$PID" > /dev/null 2>&1; then
    echo "停止服务 (PID: \$PID)..."
    kill "\$PID"

    # 等待进程结束
    for i in {1..10}; do
        if ! ps -p "\$PID" > /dev/null 2>&1; then
            echo "服务已停止"
            rm -f "\$PID_FILE"
            exit 0
        fi
        sleep 1
    done

    # 强制杀死进程
    echo "强制停止服务..."
    kill -9 "\$PID"
    rm -f "\$PID_FILE"
else
    echo "服务没有运行"
    rm -f "\$PID_FILE"
fi
EOF

# 设置脚本执行权限
chmod +x "$BUILD_DIR/start.sh"
chmod +x "$BUILD_DIR/stop.sh"
chmod +x "$BUILD_DIR/bin/$APP_NAME"

# 创建README文件
print_info "创建使用说明..."
cat > "$BUILD_DIR/README.md" << EOF
# Data Collector

## 构建信息
- 版本: $VERSION
- 构建时间: $BUILD_TIME
- Git提交: $GIT_COMMIT

## 目录结构
\`\`\`
release/
├── bin/                    # 二进制文件目录
│   └── $APP_NAME          # 主程序
├── start.sh               # 启动脚本
├── stop.sh                # 停止脚本
├── configs/               # 配置文件目录
│   ├── config.yaml        # 主配置文件
│   └── trpc_go.yaml       # TRPC框架配置
├── log/                   # 日志目录
└── data/                  # 数据目录
\`\`\`

## 使用方法

### 直接运行
\`\`\`bash
cd bin && ./$APP_NAME
\`\`\`

### 使用启动脚本（推荐）
\`\`\`bash
# 启动服务
./start.sh

# 停止服务
./stop.sh
\`\`\`

### 配置文件
根据部署环境选择相应的配置文件：
- 主配置：使用 \`configs/config.yaml\`
- TRPC配置：使用 \`configs/trpc_go.yaml\`

### 程序使用
\`\`\`bash
# 查看版本信息
./bin/$APP_NAME --version

# 查看帮助信息
./bin/$APP_NAME --help

# 使用指定配置文件运行
./bin/$APP_NAME -config=./configs/config.yaml
\`\`\`

### 环境变量
生产环境建议设置以下环境变量：
\`\`\`bash
export DATA_COLLECTOR_CONFIG="./configs/config.yaml"
export LOG_LEVEL="info"
export DATA_DIR="./data"
\`\`\`

### 日志
服务日志存储在 \`log/app.log\` 文件中。

## 注意事项
1. 确保端口没有被占用
2. 确保配置文件正确
3. 定期备份数据目录
4. 监控日志文件大小
EOF

# 显示构建结果
print_success "构建完成！"
echo ""
echo "构建结果："
echo "----------------------------------------"
echo "二进制文件: $BUILD_DIR/bin/$APP_NAME"
echo "配置目录:   $BUILD_DIR/configs/"
echo "日志目录:   $BUILD_DIR/log/"
echo "数据目录:   $BUILD_DIR/data/"
echo "启动脚本:   $BUILD_DIR/start.sh"
echo "停止脚本:   $BUILD_DIR/stop.sh"
echo "使用说明:   $BUILD_DIR/README.md"
echo ""
echo "文件列表："
ls -la "$BUILD_DIR/"
echo ""
echo "二进制文件："
ls -la "$BUILD_DIR/bin/"
echo ""
print_info "进入构建目录: cd $BUILD_DIR"
print_info "启动服务: ./start.sh"
print_info "停止服务: ./stop.sh"


