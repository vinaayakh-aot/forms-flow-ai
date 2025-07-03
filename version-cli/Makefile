# FormsFlow.ai Version Update Tool - Go Edition
# Production-ready Makefile for building cross-platform binaries

.PHONY: build clean install test help release checksums

# Variables
APP_NAME = formsflow-version-updater
VERSION = 1.0.0
BUILD_DIR = bin
RELEASE_DIR = release
GO_FILES = main.go

# Build metadata
BUILD_TIME = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH = $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

# Production build flags
LDFLAGS = -s -w \
	-X main.AppVersion=$(VERSION) \
	-X main.BuildTime=$(BUILD_TIME) \
	-X main.GitCommit=$(GIT_COMMIT) \
	-X main.GitBranch=$(GIT_BRANCH)

# Build flags for static binaries
BUILD_FLAGS = -trimpath -buildmode=exe
CGO_FLAGS = CGO_ENABLED=0

# Default target
all: build

# Build for current platform (optimized)
build:
	@echo "🔨 Building $(APP_NAME) for current platform (optimized)..."
	@mkdir -p $(BUILD_DIR)
	$(CGO_FLAGS) go build $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) $(GO_FILES)
	@echo "✅ Build complete: $(BUILD_DIR)/$(APP_NAME)"

# Build for all platforms (production release)
build-all: build-linux build-windows build-darwin build-freebsd
	@echo "🌍 All platform builds complete!"

# Build for Linux (AMD64 + ARM64)
build-linux:
	@echo "🐧 Building for Linux (AMD64 + ARM64)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(CGO_FLAGS) go build $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 $(GO_FILES)
	GOOS=linux GOARCH=arm64 $(CGO_FLAGS) go build $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 $(GO_FILES)
	GOOS=linux GOARCH=386 $(CGO_FLAGS) go build $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-linux-386 $(GO_FILES)

# Build for Windows (AMD64 + ARM64 + 386)
build-windows:
	@echo "🪟 Building for Windows (AMD64 + ARM64 + 386)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(CGO_FLAGS) go build $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe $(GO_FILES)
	GOOS=windows GOARCH=arm64 $(CGO_FLAGS) go build $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-windows-arm64.exe $(GO_FILES)
	GOOS=windows GOARCH=386 $(CGO_FLAGS) go build $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-windows-386.exe $(GO_FILES)

# Build for macOS (Intel + Apple Silicon)
build-darwin:
	@echo "🍎 Building for macOS (Intel + Apple Silicon)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(CGO_FLAGS) go build $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 $(GO_FILES)
	GOOS=darwin GOARCH=arm64 $(CGO_FLAGS) go build $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 $(GO_FILES)

# Build for FreeBSD
build-freebsd:
	@echo "🔥 Building for FreeBSD..."
	@mkdir -p $(BUILD_DIR)
	GOOS=freebsd GOARCH=amd64 $(CGO_FLAGS) go build $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-freebsd-amd64 $(GO_FILES)
	GOOS=freebsd GOARCH=arm64 $(CGO_FLAGS) go build $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-freebsd-arm64 $(GO_FILES)

# Production release with checksums
release: build-all checksums
	@echo "📦 Creating production release..."
	@mkdir -p $(RELEASE_DIR)
	@cp $(BUILD_DIR)/* $(RELEASE_DIR)/
	@cp checksums.txt $(RELEASE_DIR)/
	@echo "✅ Release ready in $(RELEASE_DIR)/"

# Generate checksums for all binaries
checksums: build-all
	@echo "🔐 Generating checksums..."
	@cd $(BUILD_DIR) && sha256sum * > ../checksums.txt
	@echo "✅ Checksums generated: checksums.txt"

# Install to system PATH (Unix-like systems only)
install: build
	@echo "📦 Installing $(APP_NAME) to /usr/local/bin..."
	@sudo cp $(BUILD_DIR)/$(APP_NAME) /usr/local/bin/
	@echo "✅ Installed successfully!"

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) $(RELEASE_DIR) checksums.txt
	@echo "✅ Clean complete!"

# Run tests with race detection
test:
	@echo "🧪 Running tests with race detection..."
	go test -race -v ./...

# Run linting and static analysis
lint:
	@echo "🔍 Running static analysis..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo "Installing golangci-lint..."; go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; }
	golangci-lint run

# Format code
fmt:
	@echo "🎨 Formatting code..."
	go fmt ./...
	go mod tidy

# Run security scan
security:
	@echo "🔒 Running security scan..."
	@command -v gosec >/dev/null 2>&1 || { echo "Installing gosec..."; go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; }
	gosec ./...

# Benchmark tests
bench:
	@echo "⚡ Running benchmarks..."
	go test -bench=. -benchmem ./...

# Run with dry-run for testing
test-run: build
	@echo "🔍 Testing with dry-run..."
	./$(BUILD_DIR)/$(APP_NAME) --dry-run

# Run the tool
run: build
	@echo "🚀 Running version updater..."
	./$(BUILD_DIR)/$(APP_NAME)

# Development build with debug info
dev-build:
	@echo "🔧 Building development version..."
	@mkdir -p $(BUILD_DIR)
	go build -gcflags="all=-N -l" -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-dev $(GO_FILES)
	@echo "✅ Dev build complete: $(BUILD_DIR)/$(APP_NAME)-dev"

# Show build info
info:
	@echo "📊 Build Information:"
	@echo "  App Name:    $(APP_NAME)"
	@echo "  Version:     $(VERSION)"
	@echo "  Build Time:  $(BUILD_TIME)"
	@echo "  Git Commit:  $(GIT_COMMIT)"
	@echo "  Git Branch:  $(GIT_BRANCH)"
	@echo "  Go Version:  $(shell go version)"

# Show help
help:
	@echo "FormsFlow.ai Version Update Tool - Production Build Commands"
	@echo ""
	@echo "🏗️  BUILD TARGETS:"
	@echo "  build         - Build optimized binary for current platform"
	@echo "  build-all     - Build for all platforms (Linux, Windows, macOS, FreeBSD)"
	@echo "  build-linux   - Build for Linux (AMD64, ARM64, 386)"
	@echo "  build-windows - Build for Windows (AMD64, ARM64, 386)"
	@echo "  build-darwin  - Build for macOS (Intel + Apple Silicon)"
	@echo "  build-freebsd - Build for FreeBSD (AMD64, ARM64)"
	@echo "  dev-build     - Build development version with debug info"
	@echo ""
	@echo "📦 RELEASE TARGETS:"
	@echo "  release       - Create production release with checksums"
	@echo "  checksums     - Generate SHA256 checksums for all binaries"
	@echo "  install       - Install to system PATH (requires sudo)"
	@echo ""
	@echo "🧪 QUALITY TARGETS:"
	@echo "  test          - Run tests with race detection"
	@echo "  lint          - Run static analysis (golangci-lint)"
	@echo "  fmt           - Format code and tidy modules"
	@echo "  security      - Run security scan (gosec)"
	@echo "  bench         - Run benchmark tests"
	@echo ""
	@echo "🔧 UTILITY TARGETS:"
	@echo "  test-run      - Build and test with --dry-run"
	@echo "  run           - Build and run the tool"
	@echo "  clean         - Remove all build artifacts"
	@echo "  info          - Show build information"
	@echo "  help          - Show this help message"
	@echo ""
	@echo "🚀 PRODUCTION EXAMPLES:"
	@echo "  make release                  # Full production release"
	@echo "  make build-all && make checksums  # Build all + verify"
	@echo "  make lint test security       # Full quality check"
	@echo "  make fmt && make build        # Format and build" 