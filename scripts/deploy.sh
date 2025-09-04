#!/bin/bash

# Data Collector 部署脚本
# 用于自动化部署到目标服务器

set -e

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

# 显示帮助信息
show_help() {
    cat << EOF
Data Collector 部署脚本

用法: $0 [选项]

选项:
    -h, --help              显示帮助信息
    -t, --target HOST       目标服务器地址
    -u, --user USER         SSH用户名 (默认: root)
    -p, --port PORT         SSH端口 (默认: 22)
    -d, --deploy-dir DIR    部署目录 (默认: /opt/data-collector)
    -v, --version VERSION   部署版本 (默认: 从构建目录)
    -s, --service           安装为系统服务
    -b, --backup            部署前备份现有版本
    --dry-run               只显示将要执行的操作，不实际执行

示例:
    $0 -t 192.168.1.100 -u deploy -d /opt/data-collector
    $0 -t server.example.com -s -b
    $0 --dry-run -t localhost

EOF
}

# 默认配置
TARGET_HOST=""
SSH_USER="root"
SSH_PORT="22"
DEPLOY_DIR="/opt/data-collector"
VERSION=""
INSTALL_SERVICE=false
BACKUP_EXISTING=false
DRY_RUN=false

# 获取脚本目录
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
PROJECT_ROOT=$(cd "$SCRIPT_DIR/.." && pwd)
RELEASE_DIR="$PROJECT_ROOT/release"

# 解析命令行参数
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -t|--target)
                TARGET_HOST="$2"
                shift 2
                ;;
            -u|--user)
                SSH_USER="$2"
                shift 2
                ;;
            -p|--port)
                SSH_PORT="$2"
                shift 2
                ;;
            -d|--deploy-dir)
                DEPLOY_DIR="$2"
                shift 2
                ;;
            -v|--version)
                VERSION="$2"
                shift 2
                ;;
            -s|--service)
                INSTALL_SERVICE=true
                shift
                ;;
            -b|--backup)
                BACKUP_EXISTING=true
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            *)
                print_error "未知选项: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# 验证参数
validate_args() {
    if [ -z "$TARGET_HOST" ]; then
        print_error "必须指定目标服务器地址 (-t|--target)"
        exit 1
    fi
    
    if [ ! -d "$RELEASE_DIR" ]; then
        print_error "构建目录不存在: $RELEASE_DIR"
        print_info "请先运行构建: make build-all 或 ./scripts/build.sh"
        exit 1
    fi
}

# 执行命令 (支持 dry-run)
execute_cmd() {
    local cmd="$1"
    local description="$2"
    
    if [ "$DRY_RUN" = true ]; then
        print_info "[DRY-RUN] $description"
        print_info "[DRY-RUN] 命令: $cmd"
    else
        print_info "$description"
        eval "$cmd"
    fi
}

# 执行远程命令
execute_remote_cmd() {
    local cmd="$1"
    local description="$2"
    
    local ssh_cmd="ssh -p $SSH_PORT $SSH_USER@$TARGET_HOST '$cmd'"
    execute_cmd "$ssh_cmd" "$description"
}

# 检查远程连接
check_remote_connection() {
    print_info "检查远程服务器连接..."
    
    if [ "$DRY_RUN" = true ]; then
        print_info "[DRY-RUN] 检查 SSH 连接到 $SSH_USER@$TARGET_HOST:$SSH_PORT"
        return 0
    fi
    
    if ssh -p "$SSH_PORT" -o ConnectTimeout=10 -o BatchMode=yes "$SSH_USER@$TARGET_HOST" exit 2>/dev/null; then
        print_success "远程连接正常"
    else
        print_error "无法连接到远程服务器"
        print_info "请检查："
        print_info "1. 服务器地址是否正确"
        print_info "2. SSH 密钥是否配置"
        print_info "3. 网络连接是否正常"
        exit 1
    fi
}

# 备份现有版本
backup_existing() {
    if [ "$BACKUP_EXISTING" = true ]; then
        print_info "备份现有版本..."
        
        local backup_dir="$DEPLOY_DIR.backup.$(date +%Y%m%d_%H%M%S)"
        execute_remote_cmd "if [ -d '$DEPLOY_DIR' ]; then cp -r '$DEPLOY_DIR' '$backup_dir' && echo '备份完成: $backup_dir'; else echo '没有现有版本需要备份'; fi" "创建备份"
    fi
}

# 停止现有服务
stop_existing_service() {
    print_info "停止现有服务..."
    
    execute_remote_cmd "if [ -f '$DEPLOY_DIR/stop.sh' ]; then cd '$DEPLOY_DIR' && ./stop.sh; else echo '没有找到停止脚本'; fi" "停止服务"
    
    # 等待服务完全停止
    execute_remote_cmd "sleep 3" "等待服务停止"
}

# 创建部署目录
create_deploy_directory() {
    print_info "创建部署目录..."
    
    execute_remote_cmd "mkdir -p '$DEPLOY_DIR'" "创建目录 $DEPLOY_DIR"
    execute_remote_cmd "mkdir -p '$DEPLOY_DIR/bin'" "创建 bin 目录"
    execute_remote_cmd "mkdir -p '$DEPLOY_DIR/configs'" "创建 configs 目录"
    execute_remote_cmd "mkdir -p '$DEPLOY_DIR/data'" "创建 data 目录"
    execute_remote_cmd "mkdir -p '$DEPLOY_DIR/log'" "创建 log 目录"
}

# 上传文件
upload_files() {
    print_info "上传文件到远程服务器..."
    
    # 上传二进制文件
    if [ "$DRY_RUN" = true ]; then
        print_info "[DRY-RUN] 上传二进制文件"
        print_info "[DRY-RUN] scp -P $SSH_PORT $RELEASE_DIR/bin/* $SSH_USER@$TARGET_HOST:$DEPLOY_DIR/bin/"
    else
        scp -P "$SSH_PORT" "$RELEASE_DIR/bin"/* "$SSH_USER@$TARGET_HOST:$DEPLOY_DIR/bin/"
        print_success "二进制文件上传完成"
    fi
    
    # 上传配置文件
    if [ -d "$RELEASE_DIR/configs" ]; then
        if [ "$DRY_RUN" = true ]; then
            print_info "[DRY-RUN] 上传配置文件"
            print_info "[DRY-RUN] scp -P $SSH_PORT -r $RELEASE_DIR/configs/* $SSH_USER@$TARGET_HOST:$DEPLOY_DIR/configs/"
        else
            scp -P "$SSH_PORT" -r "$RELEASE_DIR/configs"/* "$SSH_USER@$TARGET_HOST:$DEPLOY_DIR/configs/"
            print_success "配置文件上传完成"
        fi
    fi
    
    # 上传脚本文件
    local scripts=("start.sh" "stop.sh")
    for script in "${scripts[@]}"; do
        if [ -f "$RELEASE_DIR/$script" ]; then
            if [ "$DRY_RUN" = true ]; then
                print_info "[DRY-RUN] 上传 $script"
                print_info "[DRY-RUN] scp -P $SSH_PORT $RELEASE_DIR/$script $SSH_USER@$TARGET_HOST:$DEPLOY_DIR/"
            else
                scp -P "$SSH_PORT" "$RELEASE_DIR/$script" "$SSH_USER@$TARGET_HOST:$DEPLOY_DIR/"
                print_success "$script 上传完成"
            fi
        fi
    done
    
    # 上传 README
    if [ -f "$RELEASE_DIR/README.md" ]; then
        if [ "$DRY_RUN" = true ]; then
            print_info "[DRY-RUN] 上传 README.md"
        else
            scp -P "$SSH_PORT" "$RELEASE_DIR/README.md" "$SSH_USER@$TARGET_HOST:$DEPLOY_DIR/"
            print_success "README.md 上传完成"
        fi
    fi
}

# 设置文件权限
set_permissions() {
    print_info "设置文件权限..."
    
    execute_remote_cmd "chmod +x '$DEPLOY_DIR/bin'/*" "设置二进制文件权限"
    execute_remote_cmd "chmod +x '$DEPLOY_DIR'/*.sh" "设置脚本文件权限"
    execute_remote_cmd "chown -R $SSH_USER:$SSH_USER '$DEPLOY_DIR'" "设置文件所有者"
}

# 安装系统服务
install_system_service() {
    if [ "$INSTALL_SERVICE" = true ]; then
        print_info "安装系统服务..."
        
        local service_content="[Unit]
Description=Data Collector Service
After=network.target

[Service]
Type=forking
User=$SSH_USER
WorkingDirectory=$DEPLOY_DIR
ExecStart=$DEPLOY_DIR/start.sh
ExecStop=$DEPLOY_DIR/stop.sh
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target"

        if [ "$DRY_RUN" = true ]; then
            print_info "[DRY-RUN] 创建系统服务文件"
            print_info "[DRY-RUN] 服务内容:"
            echo "$service_content"
        else
            execute_remote_cmd "echo '$service_content' | sudo tee /etc/systemd/system/data-collector.service" "创建服务文件"
            execute_remote_cmd "sudo systemctl daemon-reload" "重新加载系统服务"
            execute_remote_cmd "sudo systemctl enable data-collector" "启用服务"
            print_success "系统服务安装完成"
        fi
    fi
}

# 启动服务
start_service() {
    print_info "启动服务..."
    
    if [ "$INSTALL_SERVICE" = true ]; then
        execute_remote_cmd "sudo systemctl start data-collector" "启动系统服务"
        execute_remote_cmd "sudo systemctl status data-collector --no-pager" "检查服务状态"
    else
        execute_remote_cmd "cd '$DEPLOY_DIR' && ./start.sh" "启动服务"
    fi
}

# 验证部署
verify_deployment() {
    print_info "验证部署..."
    
    execute_remote_cmd "ls -la '$DEPLOY_DIR'" "检查部署目录"
    execute_remote_cmd "ls -la '$DEPLOY_DIR/bin'" "检查二进制文件"
    
    if [ "$INSTALL_SERVICE" = true ]; then
        execute_remote_cmd "sudo systemctl is-active data-collector" "检查服务状态"
    else
        execute_remote_cmd "if [ -f '$DEPLOY_DIR/data-collector.pid' ]; then echo '服务运行中'; else echo '服务未运行'; fi" "检查进程状态"
    fi
}

# 主函数
main() {
    echo "========================================"
    echo "  Data Collector 部署脚本"
    echo "========================================"
    echo ""
    
    parse_args "$@"
    validate_args
    
    print_info "部署配置："
    print_info "目标服务器: $SSH_USER@$TARGET_HOST:$SSH_PORT"
    print_info "部署目录: $DEPLOY_DIR"
    print_info "安装服务: $INSTALL_SERVICE"
    print_info "备份现有: $BACKUP_EXISTING"
    print_info "Dry Run: $DRY_RUN"
    echo ""
    
    if [ "$DRY_RUN" = false ]; then
        read -p "确认部署? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_info "部署已取消"
            exit 0
        fi
    fi
    
    check_remote_connection
    backup_existing
    stop_existing_service
    create_deploy_directory
    upload_files
    set_permissions
    install_system_service
    start_service
    verify_deployment
    
    echo ""
    echo "========================================"
    print_success "部署完成！"
    echo "========================================"
    
    print_info "部署信息："
    print_info "服务器: $TARGET_HOST"
    print_info "目录: $DEPLOY_DIR"
    if [ "$INSTALL_SERVICE" = true ]; then
        print_info "管理命令: sudo systemctl {start|stop|restart|status} data-collector"
    else
        print_info "启动: ssh $SSH_USER@$TARGET_HOST 'cd $DEPLOY_DIR && ./start.sh'"
        print_info "停止: ssh $SSH_USER@$TARGET_HOST 'cd $DEPLOY_DIR && ./stop.sh'"
    fi
}

# 执行主函数
main "$@"
