# FormsFlow.ai Version Update Tool - Go Edition
# Makefile for building cross-platform binaries

.PHONY: build clean install test help

# Variables
APP_NAME = formsflow-version-updater
VERSION = 1.0.0
BUILD_DIR = bin
GO_FILES = main.go

# Default target
all: build

# Build for current platform
build:
	@echo "🔨 Building $(APP_NAME) for current platform..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "-X main.AppVersion=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME) $(GO_FILES)
	@echo "✅ Build complete: $(BUILD_DIR)/$(APP_NAME)"

# Build for all platforms
build-all: build-linux build-windows build-darwin
	@echo "🌍 All platform builds complete!"

# Build for Linux
build-linux:
	@echo "🐧 Building for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.AppVersion=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 $(GO_FILES)

# Build for Windows
build-windows:
	@echo "🪟 Building for Windows..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build -ldflags "-X main.AppVersion=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe $(GO_FILES)

# Build for macOS
build-darwin:
	@echo "🍎 Building for macOS..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.AppVersion=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 $(GO_FILES)
	GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.AppVersion=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 $(GO_FILES)

# Install to system PATH (Unix-like systems only)
install: build
	@echo "📦 Installing $(APP_NAME) to /usr/local/bin..."
	@sudo cp $(BUILD_DIR)/$(APP_NAME) /usr/local/bin/
	@echo "✅ Installed successfully!"

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@echo "✅ Clean complete!"

# Run tests (when we add them)
test:
	@echo "🧪 Running tests..."
	go test -v ./...

# Run with dry-run for testing
test-run: build
	@echo "🔍 Testing with dry-run..."
	./$(BUILD_DIR)/$(APP_NAME) --dry-run

# Run the tool
run: build
	@echo "🚀 Running version updater..."
	./$(BUILD_DIR)/$(APP_NAME)

# Show help
help:
	@echo "FormsFlow.ai Version Update Tool - Build Commands"
	@echo ""
	@echo "Available targets:"
	@echo "  build        - Build for current platform"
	@echo "  build-all    - Build for all platforms (Linux, Windows, macOS)"
	@echo "  build-linux  - Build for Linux"
	@echo "  build-windows- Build for Windows"
	@echo "  build-darwin - Build for macOS (Intel + Apple Silicon)"
	@echo "  install      - Install to system PATH (requires sudo)"
	@echo "  clean        - Remove build artifacts"
	@echo "  test         - Run tests"
	@echo "  test-run     - Build and run with --dry-run"
	@echo "  run          - Build and run the tool"
	@echo "  help         - Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make build                    # Build for current platform"
	@echo "  make build-all               # Build for all platforms"
	@echo "  make test-run                # Test with dry-run"
	@echo "  make run                     # Run the version updater" 