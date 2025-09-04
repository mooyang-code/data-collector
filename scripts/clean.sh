#!/bin/bash

# Data Collector 清理脚本
# 用于清理构建文件、缓存和临时文件

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
Data Collector 清理脚本

用法: $0 [选项]

选项:
    -h, --help          显示帮助信息
    -a, --all           清理所有文件 (构建文件 + 缓存 + 日志)
    -b, --build         清理构建文件
    -c, --cache         清理Go缓存
    -l, --logs          清理日志文件
    -d, --data          清理数据文件 (谨慎使用)
    -t, --temp          清理临时文件
    -r, --release       清理发布文件
    --dry-run           只显示将要删除的文件，不实际删除
    -f, --force         强制删除，不询问确认

示例:
    $0 -b               # 只清理构建文件
    $0 -a               # 清理所有文件
    $0 --dry-run -a     # 预览将要清理的所有文件
    $0 -c -l            # 清理缓存和日志

EOF
}

# 默认配置
CLEAN_ALL=false
CLEAN_BUILD=false
CLEAN_CACHE=false
CLEAN_LOGS=false
CLEAN_DATA=false
CLEAN_TEMP=false
CLEAN_RELEASE=false
DRY_RUN=false
FORCE=false

# 获取脚本目录
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
PROJECT_ROOT=$(cd "$SCRIPT_DIR/.." && pwd)

# 解析命令行参数
parse_args() {
    if [ $# -eq 0 ]; then
        show_help
        exit 0
    fi
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -a|--all)
                CLEAN_ALL=true
                shift
                ;;
            -b|--build)
                CLEAN_BUILD=true
                shift
                ;;
            -c|--cache)
                CLEAN_CACHE=true
                shift
                ;;
            -l|--logs)
                CLEAN_LOGS=true
                shift
                ;;
            -d|--data)
                CLEAN_DATA=true
                shift
                ;;
            -t|--temp)
                CLEAN_TEMP=true
                shift
                ;;
            -r|--release)
                CLEAN_RELEASE=true
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            -f|--force)
                FORCE=true
                shift
                ;;
            *)
                print_error "未知选项: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    # 如果指定了 --all，则启用所有清理选项
    if [ "$CLEAN_ALL" = true ]; then
        CLEAN_BUILD=true
        CLEAN_CACHE=true
        CLEAN_LOGS=true
        CLEAN_TEMP=true
        CLEAN_RELEASE=true
    fi
}

# 执行删除操作
remove_path() {
    local path="$1"
    local description="$2"
    
    if [ -e "$path" ]; then
        if [ "$DRY_RUN" = true ]; then
            print_info "[DRY-RUN] 将删除: $path ($description)"
            if [ -d "$path" ]; then
                local size=$(du -sh "$path" 2>/dev/null | cut -f1 || echo "未知")
                print_info "[DRY-RUN]   目录大小: $size"
                local count=$(find "$path" -type f 2>/dev/null | wc -l || echo "未知")
                print_info "[DRY-RUN]   文件数量: $count"
            elif [ -f "$path" ]; then
                local size=$(du -sh "$path" 2>/dev/null | cut -f1 || echo "未知")
                print_info "[DRY-RUN]   文件大小: $size"
            fi
        else
            print_info "删除: $path ($description)"
            rm -rf "$path"
            print_success "已删除: $description"
        fi
    else
        print_info "跳过: $path (不存在)"
    fi
}

# 清理构建文件
clean_build_files() {
    if [ "$CLEAN_BUILD" = true ]; then
        print_info "清理构建文件..."
        
        cd "$PROJECT_ROOT"
        
        # 清理主要构建目录
        remove_path "build" "构建目录"
        remove_path "bin" "二进制目录"
        
        # 清理测试覆盖率文件
        remove_path "coverage.out" "测试覆盖率数据"
        remove_path "coverage.html" "测试覆盖率报告"
        
        # 清理构建产物
        find . -name "*.test" -type f 2>/dev/null | while read -r file; do
            remove_path "$file" "测试二进制文件"
        done
        
        # 清理可能的二进制文件
        local binaries=("data-collector" "symtool" "klinedump" "trpc-server" "trpc-client")
        for binary in "${binaries[@]}"; do
            if [ -f "$binary" ]; then
                remove_path "$binary" "二进制文件"
            fi
        done
    fi
}

# 清理发布文件
clean_release_files() {
    if [ "$CLEAN_RELEASE" = true ]; then
        print_info "清理发布文件..."
        
        cd "$PROJECT_ROOT"
        
        remove_path "release" "发布目录"
        remove_path "release-dist" "发布分发目录"
        
        # 清理压缩包
        find . -name "data-collector-*.tar.gz" -type f 2>/dev/null | while read -r file; do
            remove_path "$file" "发布压缩包"
        done
        
        find . -name "data-collector-*.zip" -type f 2>/dev/null | while read -r file; do
            remove_path "$file" "发布压缩包"
        done
    fi
}

# 清理Go缓存
clean_go_cache() {
    if [ "$CLEAN_CACHE" = true ]; then
        print_info "清理Go缓存..."
        
        if [ "$DRY_RUN" = true ]; then
            print_info "[DRY-RUN] 将执行: go clean -cache"
            print_info "[DRY-RUN] 将执行: go clean -modcache"
            print_info "[DRY-RUN] 将执行: go clean -testcache"
        else
            cd "$PROJECT_ROOT"
            go clean -cache
            print_success "Go构建缓存已清理"
            
            go clean -testcache
            print_success "Go测试缓存已清理"
            
            # 询问是否清理模块缓存 (这会影响其他项目)
            if [ "$FORCE" = true ]; then
                go clean -modcache
                print_success "Go模块缓存已清理"
            else
                read -p "是否清理Go模块缓存? 这会影响所有Go项目 (y/N): " -n 1 -r
                echo
                if [[ $REPLY =~ ^[Yy]$ ]]; then
                    go clean -modcache
                    print_success "Go模块缓存已清理"
                else
                    print_info "跳过Go模块缓存清理"
                fi
            fi
        fi
    fi
}

# 清理日志文件
clean_log_files() {
    if [ "$CLEAN_LOGS" = true ]; then
        print_info "清理日志文件..."
        
        cd "$PROJECT_ROOT"
        
        # 清理项目日志目录
        remove_path "log" "日志目录"
        remove_path "logs" "日志目录"
        
        # 清理发布目录中的日志
        remove_path "release/log" "发布日志目录"
        
        # 清理各种日志文件
        find . -name "*.log" -type f 2>/dev/null | while read -r file; do
            remove_path "$file" "日志文件"
        done
        
        find . -name "*.log.*" -type f 2>/dev/null | while read -r file; do
            remove_path "$file" "轮转日志文件"
        done
        
        # 清理nohup输出
        remove_path "nohup.out" "nohup输出文件"
    fi
}

# 清理数据文件
clean_data_files() {
    if [ "$CLEAN_DATA" = true ]; then
        print_warning "清理数据文件 - 这将删除所有数据！"
        
        if [ "$FORCE" = false ] && [ "$DRY_RUN" = false ]; then
            read -p "确认删除所有数据文件? 这个操作不可恢复! (y/N): " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                print_info "跳过数据文件清理"
                return
            fi
        fi
        
        cd "$PROJECT_ROOT"
        
        remove_path "data" "数据目录"
        remove_path "release/data" "发布数据目录"
        
        # 清理数据库文件
        find . -name "*.db" -type f 2>/dev/null | while read -r file; do
            remove_path "$file" "数据库文件"
        done
        
        find . -name "*.sqlite" -type f 2>/dev/null | while read -r file; do
            remove_path "$file" "SQLite数据库文件"
        done
    fi
}

# 清理临时文件
clean_temp_files() {
    if [ "$CLEAN_TEMP" = true ]; then
        print_info "清理临时文件..."
        
        cd "$PROJECT_ROOT"
        
        # 清理临时目录
        remove_path "tmp" "临时目录"
        remove_path "temp" "临时目录"
        
        # 清理各种临时文件
        find . -name "*.tmp" -type f 2>/dev/null | while read -r file; do
            remove_path "$file" "临时文件"
        done
        
        find . -name "*.temp" -type f 2>/dev/null | while read -r file; do
            remove_path "$file" "临时文件"
        done
        
        find . -name ".DS_Store" -type f 2>/dev/null | while read -r file; do
            remove_path "$file" "macOS系统文件"
        done
        
        find . -name "Thumbs.db" -type f 2>/dev/null | while read -r file; do
            remove_path "$file" "Windows系统文件"
        done
        
        # 清理编辑器临时文件
        find . -name "*~" -type f 2>/dev/null | while read -r file; do
            remove_path "$file" "编辑器备份文件"
        done
        
        find . -name "*.swp" -type f 2>/dev/null | while read -r file; do
            remove_path "$file" "Vim交换文件"
        done
        
        find . -name "*.swo" -type f 2>/dev/null | while read -r file; do
            remove_path "$file" "Vim交换文件"
        done
        
        # 清理PID文件
        find . -name "*.pid" -type f 2>/dev/null | while read -r file; do
            remove_path "$file" "PID文件"
        done
    fi
}

# 显示清理摘要
show_summary() {
    print_info "清理摘要："
    
    local operations=()
    [ "$CLEAN_BUILD" = true ] && operations+=("构建文件")
    [ "$CLEAN_RELEASE" = true ] && operations+=("发布文件")
    [ "$CLEAN_CACHE" = true ] && operations+=("Go缓存")
    [ "$CLEAN_LOGS" = true ] && operations+=("日志文件")
    [ "$CLEAN_DATA" = true ] && operations+=("数据文件")
    [ "$CLEAN_TEMP" = true ] && operations+=("临时文件")
    
    if [ ${#operations[@]} -eq 0 ]; then
        print_warning "没有指定清理操作"
        return
    fi
    
    for op in "${operations[@]}"; do
        print_info "- $op"
    done
    
    if [ "$DRY_RUN" = true ]; then
        print_warning "这是预览模式，没有实际删除文件"
        print_info "要执行实际清理，请移除 --dry-run 参数"
    fi
}

# 主函数
main() {
    echo "========================================"
    echo "  Data Collector 清理脚本"
    echo "========================================"
    echo ""
    
    parse_args "$@"
    
    print_info "项目根目录: $PROJECT_ROOT"
    echo ""
    
    show_summary
    echo ""
    
    if [ "$DRY_RUN" = false ] && [ "$FORCE" = false ]; then
        read -p "确认执行清理操作? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_info "清理操作已取消"
            exit 0
        fi
        echo ""
    fi
    
    clean_build_files
    clean_release_files
    clean_go_cache
    clean_log_files
    clean_data_files
    clean_temp_files
    
    echo ""
    echo "========================================"
    if [ "$DRY_RUN" = true ]; then
        print_info "清理预览完成！"
    else
        print_success "清理完成！"
    fi
    echo "========================================"
}

# 执行主函数
main "$@"
