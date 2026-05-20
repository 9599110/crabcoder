# CrabCoder Makefile

.PHONY: build test lint clean install run dev fmt vet

# 项目信息
BINARY_NAME=crabcoder
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%dT%H:%M:%S')
GO_LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

# Go 参数
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# 目录
BUILD_DIR=./build
CMD_DIR=./cmd/cli

# 默认目标
all: build

# 构建
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(GO_LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)

# 构建 Linux 版本
build-linux:
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(GO_LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)

# 构建 macOS 版本
build-darwin:
	@echo "Building for macOS..."
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(GO_LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(GO_LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)

# 构建 Windows 版本
build-windows:
	@echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(GO_LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME).exe $(CMD_DIR)

# 运行
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

# 开发模式
dev: build
	@echo "Running in dev mode..."
	./$(BUILD_DIR)/$(BINARY_NAME) --dev

# 测试
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -cover ./...

# 测试覆盖率
test-cover:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# 静态分析
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

# 代码格式化
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# 依赖管理
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# 依赖更新
update:
	@echo "Updating dependencies..."
	$(GOMOD) update
	$(GOMOD) tidy

# 清理
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

# 安装
install: build
	@echo "Installing $(BINARY_NAME)..."
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

# 卸载
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	rm -f /usr/local/bin/$(BINARY_NAME)

# Docker 构建
docker-build:
	@echo "Building Docker image..."
	docker build -t crabcoder:latest .

docker-run:
	@echo "Running Docker container..."
	docker run -it --rm crabcoder:latest

# 帮助
help:
	@echo "Available targets:"
	@echo "  build        - Build the binary"
	@echo "  build-linux  - Build for Linux"
	@echo "  build-darwin - Build for macOS"
	@echo "  build-windows - Build for Windows"
	@echo "  run          - Build and run"
	@echo "  dev          - Build and run in dev mode"
	@echo "  test         - Run tests"
	@echo "  test-cover   - Run tests with coverage"
	@echo "  vet          - Run static analysis"
	@echo "  fmt          - Format code"
	@echo "  deps         - Download dependencies"
	@echo "  update       - Update dependencies"
	@echo "  clean        - Clean build artifacts"
	@echo "  install      - Install binary"
	@echo "  uninstall    - Uninstall binary"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"
	@echo "  help         - Show this help"
