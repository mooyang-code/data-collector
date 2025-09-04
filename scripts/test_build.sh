#!/bin/bash

# Data Collector 构建测试脚本
# 用于验证构建结果和功能

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

# 获取脚本目录
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
PROJECT_ROOT=$(cd "$SCRIPT_DIR/.." && pwd)
RELEASE_DIR="$PROJECT_ROOT/release"

print_info "开始测试构建结果..."
print_info "项目根目录: $PROJECT_ROOT"
print_info "发布目录: $RELEASE_DIR"

# 检查构建目录是否存在
check_build_directory() {
    print_info "检查构建目录..."
    
    if [ ! -d "$RELEASE_DIR" ]; then
        print_error "构建目录不存在: $RELEASE_DIR"
        print_info "请先运行构建: make build-all 或 ./scripts/build.sh"
        exit 1
    fi
    
    print_success "构建目录存在"
}

# 检查二进制文件
check_binaries() {
    print_info "检查二进制文件..."
    
    local binaries=(
        "data-collector"
        "symtool"
        "klinedump"
        "trpc-server"
        "trpc-client"
    )
    
    local missing_count=0
    
    for binary in "${binaries[@]}"; do
        local binary_path="$RELEASE_DIR/bin/$binary"
        if [ -f "$binary_path" ]; then
            print_success "✓ $binary 存在"
            
            # 检查文件权限
            if [ -x "$binary_path" ]; then
                print_success "  - 可执行权限正确"
            else
                print_warning "  - 缺少可执行权限"
                chmod +x "$binary_path"
                print_info "  - 已修复可执行权限"
            fi
            
            # 检查文件大小
            local file_size=$(du -h "$binary_path" | cut -f1)
            print_info "  - 文件大小: $file_size"
            
        else
            print_warning "✗ $binary 不存在"
            ((missing_count++))
        fi
    done
    
    if [ $missing_count -eq 0 ]; then
        print_success "所有二进制文件检查完成"
    else
        print_warning "有 $missing_count 个二进制文件缺失"
    fi
}

# 检查配置文件
check_configs() {
    print_info "检查配置文件..."
    
    local configs=(
        "config.yaml"
        "trpc_go.yaml"
    )
    
    for config in "${configs[@]}"; do
        local config_path="$RELEASE_DIR/configs/$config"
        if [ -f "$config_path" ]; then
            print_success "✓ $config 存在"
        else
            print_warning "✗ $config 不存在"
        fi
    done
    
    # 检查 bin 目录下的 trpc_go.yaml
    if [ -f "$RELEASE_DIR/bin/trpc_go.yaml" ]; then
        print_success "✓ bin/trpc_go.yaml 存在"
    else
        print_warning "✗ bin/trpc_go.yaml 不存在"
    fi
}

# 检查脚本文件
check_scripts() {
    print_info "检查脚本文件..."
    
    local scripts=(
        "start.sh"
        "stop.sh"
    )
    
    for script in "${scripts[@]}"; do
        local script_path="$RELEASE_DIR/$script"
        if [ -f "$script_path" ]; then
            print_success "✓ $script 存在"
            
            if [ -x "$script_path" ]; then
                print_success "  - 可执行权限正确"
            else
                print_warning "  - 缺少可执行权限"
                chmod +x "$script_path"
                print_info "  - 已修复可执行权限"
            fi
        else
            print_warning "✗ $script 不存在"
        fi
    done
}

# 检查目录结构
check_directories() {
    print_info "检查目录结构..."
    
    local directories=(
        "bin"
        "configs"
        "data"
        "log"
    )
    
    for dir in "${directories[@]}"; do
        local dir_path="$RELEASE_DIR/$dir"
        if [ -d "$dir_path" ]; then
            print_success "✓ $dir/ 目录存在"
        else
            print_warning "✗ $dir/ 目录不存在"
            mkdir -p "$dir_path"
            print_info "  - 已创建目录"
        fi
    done
}

# 测试二进制文件基本功能
test_binaries() {
    print_info "测试二进制文件基本功能..."
    
    cd "$RELEASE_DIR/bin"
    
    # 测试主程序版本信息
    if [ -f "data-collector" ]; then
        print_info "测试 data-collector..."
        if ./data-collector --version 2>/dev/null || ./data-collector -version 2>/dev/null || ./data-collector version 2>/dev/null; then
            print_success "  - 版本信息正常"
        else
            print_warning "  - 无法获取版本信息"
        fi
        
        if ./data-collector --help 2>/dev/null || ./data-collector -help 2>/dev/null || ./data-collector help 2>/dev/null; then
            print_success "  - 帮助信息正常"
        else
            print_warning "  - 无法获取帮助信息"
        fi
    fi
    
    # 测试其他工具
    local tools=("symtool" "klinedump")
    for tool in "${tools[@]}"; do
        if [ -f "$tool" ]; then
            print_info "测试 $tool..."
            if ./"$tool" --help 2>/dev/null || ./"$tool" -help 2>/dev/null || ./"$tool" help 2>/dev/null; then
                print_success "  - 帮助信息正常"
            else
                print_warning "  - 无法获取帮助信息"
            fi
        fi
    done
    
    cd "$PROJECT_ROOT"
}

# 验证配置文件格式
validate_configs() {
    print_info "验证配置文件格式..."
    
    # 检查 YAML 文件格式
    if command -v python3 >/dev/null 2>&1; then
        for yaml_file in "$RELEASE_DIR/configs"/*.yaml; do
            if [ -f "$yaml_file" ]; then
                local filename=$(basename "$yaml_file")
                if python3 -c "import yaml; yaml.safe_load(open('$yaml_file'))" 2>/dev/null; then
                    print_success "✓ $filename 格式正确"
                else
                    print_error "✗ $filename 格式错误"
                fi
            fi
        done
    else
        print_warning "Python3 未安装，跳过 YAML 格式验证"
    fi
}

# 生成测试报告
generate_report() {
    print_info "生成测试报告..."
    
    local report_file="$RELEASE_DIR/build_test_report.txt"
    
    cat > "$report_file" << EOF
# Data Collector 构建测试报告

生成时间: $(date)
测试脚本: $0

## 目录结构
$(ls -la "$RELEASE_DIR/")

## 二进制文件
$(ls -la "$RELEASE_DIR/bin/" 2>/dev/null || echo "bin目录不存在")

## 配置文件
$(ls -la "$RELEASE_DIR/configs/" 2>/dev/null || echo "configs目录不存在")

## 磁盘使用
总大小: $(du -sh "$RELEASE_DIR" | cut -f1)

## 文件统计
二进制文件数量: $(find "$RELEASE_DIR/bin" -type f -executable 2>/dev/null | wc -l)
配置文件数量: $(find "$RELEASE_DIR/configs" -name "*.yaml" 2>/dev/null | wc -l)
脚本文件数量: $(find "$RELEASE_DIR" -maxdepth 1 -name "*.sh" 2>/dev/null | wc -l)

EOF
    
    print_success "测试报告已生成: $report_file"
}

# 主函数
main() {
    echo "========================================"
    echo "  Data Collector 构建测试"
    echo "========================================"
    echo ""
    
    check_build_directory
    check_directories
    check_binaries
    check_configs
    check_scripts
    test_binaries
    validate_configs
    generate_report
    
    echo ""
    echo "========================================"
    print_success "构建测试完成！"
    echo "========================================"
    
    print_info "下一步操作："
    print_info "1. 启动服务: cd $RELEASE_DIR && ./start.sh"
    print_info "2. 查看日志: tail -f $RELEASE_DIR/log/app.log"
    print_info "3. 停止服务: cd $RELEASE_DIR && ./stop.sh"
}

# 执行主函数
main "$@"
