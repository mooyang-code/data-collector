# Data Collector Scripts Makefile
# 用于管理构建脚本的执行

.PHONY: help build test clean deploy install-tools
.PHONY: build-dev build-prod build-release
.PHONY: test-build test-deploy
.PHONY: clean-all clean-build clean-cache clean-logs clean-data
.PHONY: deploy-local deploy-test deploy-prod
.PHONY: setup check-env

# 默认目标
all: help

# 变量定义
VERSION ?= dev
TARGET_HOST ?= localhost
DEPLOY_USER ?= root
DEPLOY_DIR ?= /opt/data-collector

# 脚本路径
BUILD_SCRIPT := ./build.sh
TEST_SCRIPT := ./test_build.sh
CLEAN_SCRIPT := ./clean.sh
DEPLOY_SCRIPT := ./deploy.sh

# 帮助信息
help:
	@echo "🛠️  Data Collector 脚本管理工具"
	@echo ""
	@echo "📦 构建脚本:"
	@echo "  build-dev      - 开发版本构建 (使用 build.sh)"
	@echo "  build-prod     - 生产版本构建"
	@echo "  build-release  - 发布版本构建"
	@echo "  test-build     - 测试构建结果"
	@echo ""
	@echo "🧹 清理脚本:"
	@echo "  clean-build    - 清理构建文件"
	@echo "  clean-cache    - 清理Go缓存"
	@echo "  clean-logs     - 清理日志文件"
	@echo "  clean-data     - 清理数据文件 (谨慎使用)"
	@echo "  clean-all      - 清理所有文件"
	@echo ""
	@echo "🚀 部署脚本:"
	@echo "  deploy-local   - 本地部署测试"
	@echo "  deploy-test    - 测试环境部署"
	@echo "  deploy-prod    - 生产环境部署"
	@echo "  test-deploy    - 测试部署配置"
	@echo ""
	@echo "🔧 工具脚本:"
	@echo "  setup          - 初始化开发环境"
	@echo "  check-env      - 检查环境配置"
	@echo "  install-tools  - 安装开发工具"
	@echo ""
	@echo "💡 使用示例:"
	@echo "  make build-prod VERSION=v1.0.0"
	@echo "  make deploy-test TARGET_HOST=test.example.com"
	@echo "  make clean-all"
	@echo ""
	@echo "📋 环境变量:"
	@echo "  VERSION       - 构建版本 (默认: $(VERSION))"
	@echo "  TARGET_HOST   - 部署目标主机 (默认: $(TARGET_HOST))"
	@echo "  DEPLOY_USER   - 部署用户 (默认: $(DEPLOY_USER))"
	@echo "  DEPLOY_DIR    - 部署目录 (默认: $(DEPLOY_DIR))"

# 检查环境
check-env:
	@echo "🔍 检查环境配置..."
	@echo "Go版本: $(shell go version 2>/dev/null || echo '未安装')"
	@echo "Git版本: $(shell git --version 2>/dev/null || echo '未安装')"
	@echo "Docker版本: $(shell docker --version 2>/dev/null || echo '未安装')"
	@echo "当前目录: $(shell pwd)"
	@echo "脚本权限:"
	@ls -la *.sh 2>/dev/null || echo "  没有找到脚本文件"

# 设置脚本权限
setup:
	@echo "🔧 设置脚本权限..."
	@chmod +x *.sh
	@echo "✅ 脚本权限设置完成"
	@$(MAKE) check-env

# 构建脚本目标
build-dev:
	@echo "📦 开发版本构建..."
	@$(BUILD_SCRIPT) $(VERSION)

build-prod:
	@echo "📦 生产版本构建..."
	@$(BUILD_SCRIPT) $(VERSION)

build-release:
	@echo "📦 发布版本构建..."
	@if [ "$(VERSION)" = "dev" ]; then \
		echo "❌ 错误: 发布构建需要指定版本号"; \
		echo "使用: make build-release VERSION=v1.0.0"; \
		exit 1; \
	fi
	@$(BUILD_SCRIPT) $(VERSION)

# 测试脚本目标
test-build:
	@echo "🧪 测试构建结果..."
	@$(TEST_SCRIPT)

test-deploy:
	@echo "🧪 测试部署配置..."
	@$(DEPLOY_SCRIPT) --dry-run -t $(TARGET_HOST) -u $(DEPLOY_USER) -d $(DEPLOY_DIR)

# 清理脚本目标
clean-build:
	@echo "🧹 清理构建文件..."
	@$(CLEAN_SCRIPT) -b -f

clean-cache:
	@echo "🧹 清理Go缓存..."
	@$(CLEAN_SCRIPT) -c -f

clean-logs:
	@echo "🧹 清理日志文件..."
	@$(CLEAN_SCRIPT) -l -f

clean-data:
	@echo "⚠️  清理数据文件..."
	@echo "这将删除所有数据文件，确认请按回车，取消请按 Ctrl+C"
	@read
	@$(CLEAN_SCRIPT) -d -f

clean-all:
	@echo "🧹 清理所有文件..."
	@$(CLEAN_SCRIPT) -a -f

# 部署脚本目标
deploy-local:
	@echo "🚀 本地部署..."
	@$(DEPLOY_SCRIPT) -t localhost -u $(USER) -d /tmp/data-collector-test

deploy-test:
	@echo "🚀 测试环境部署..."
	@if [ "$(TARGET_HOST)" = "localhost" ]; then \
		echo "❌ 错误: 请指定测试服务器地址"; \
		echo "使用: make deploy-test TARGET_HOST=test.example.com"; \
		exit 1; \
	fi
	@$(DEPLOY_SCRIPT) -t $(TARGET_HOST) -u $(DEPLOY_USER) -d $(DEPLOY_DIR) -b

deploy-prod:
	@echo "🚀 生产环境部署..."
	@if [ "$(TARGET_HOST)" = "localhost" ]; then \
		echo "❌ 错误: 请指定生产服务器地址"; \
		echo "使用: make deploy-prod TARGET_HOST=prod.example.com"; \
		exit 1; \
	fi
	@if [ "$(VERSION)" = "dev" ]; then \
		echo "❌ 错误: 生产部署需要指定版本号"; \
		echo "使用: make deploy-prod VERSION=v1.0.0 TARGET_HOST=prod.example.com"; \
		exit 1; \
	fi
	@echo "⚠️  即将部署到生产环境: $(TARGET_HOST)"
	@echo "版本: $(VERSION)"
	@echo "确认请按回车，取消请按 Ctrl+C"
	@read
	@$(DEPLOY_SCRIPT) -t $(TARGET_HOST) -u $(DEPLOY_USER) -d $(DEPLOY_DIR) -s -b

# 安装开发工具
install-tools:
	@echo "🔧 安装开发工具..."
	@echo "安装 golangci-lint..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "安装 protoc-gen-go..."
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@echo "安装 protoc-gen-go-grpc..."
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "安装 mockgen..."
	@go install github.com/golang/mock/mockgen@latest
	@echo "✅ 开发工具安装完成"

# 组合目标
build: build-dev

test: test-build

clean: clean-build

deploy: deploy-local

# 完整工作流
ci: clean build test
	@echo "✅ CI 流程完成"

cd: build-prod test-build
	@echo "✅ CD 流程完成，准备部署"

# 快速开发流程
dev: clean-build build-dev test-build
	@echo "✅ 开发流程完成"

# 发布流程
release: clean-all build-release test-build
	@echo "✅ 发布流程完成"
	@echo "下一步: make deploy-prod VERSION=$(VERSION) TARGET_HOST=your-server"

# 显示当前状态
status:
	@echo "📊 当前状态:"
	@echo "版本: $(VERSION)"
	@echo "目标主机: $(TARGET_HOST)"
	@echo "部署用户: $(DEPLOY_USER)"
	@echo "部署目录: $(DEPLOY_DIR)"
	@echo ""
	@echo "构建文件:"
	@ls -la ../release/ 2>/dev/null || echo "  没有构建文件"
	@echo ""
	@echo "脚本文件:"
	@ls -la *.sh 2>/dev/null || echo "  没有脚本文件"

# 清理脚本缓存
clean-scripts:
	@echo "🧹 清理脚本缓存..."
	@rm -f ./*.log
	@rm -f ./*.tmp
	@echo "✅ 脚本缓存清理完成"

# 验证脚本
validate:
	@echo "🔍 验证脚本..."
	@for script in *.sh; do \
		if [ -f "$$script" ]; then \
			echo "检查 $$script..."; \
			bash -n "$$script" && echo "  ✅ 语法正确" || echo "  ❌ 语法错误"; \
		fi; \
	done

# 显示脚本帮助
script-help:
	@echo "📖 脚本帮助信息:"
	@echo ""
	@if [ -f "$(BUILD_SCRIPT)" ]; then \
		echo "=== build.sh ==="; \
		$(BUILD_SCRIPT) --help 2>/dev/null || echo "没有帮助信息"; \
		echo ""; \
	fi
	@if [ -f "$(CLEAN_SCRIPT)" ]; then \
		echo "=== clean.sh ==="; \
		$(CLEAN_SCRIPT) --help 2>/dev/null || echo "没有帮助信息"; \
		echo ""; \
	fi
	@if [ -f "$(DEPLOY_SCRIPT)" ]; then \
		echo "=== deploy.sh ==="; \
		$(DEPLOY_SCRIPT) --help 2>/dev/null || echo "没有帮助信息"; \
	fi
