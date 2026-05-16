.PHONY: build test lint clean install

BUILD_DIR := bin
MAIN_PATH := ./cmd/cli/
BINARY := $(BUILD_DIR)/crabcoder

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)

GO := go
GOFLAGS := -ldflags="$(LDFLAGS)"

all: build

build:
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BINARY) $(MAIN_PATH)

build-linux:
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/crabcoder-linux-amd64 $(MAIN_PATH)

build-darwin:
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/crabcoder-darwin-arm64 $(MAIN_PATH)

build-windows:
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/crabcoder-windows-amd64.exe $(MAIN_PATH)

test:
	$(GO) test -v -race -cover ./...

test-short:
	$(GO) test -short -cover ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BUILD_DIR)

install: build
	install -m 755 $(BINARY) /usr/local/bin/

run: build
	./$(BINARY)
