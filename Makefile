# CodeContext Makefile for Building and Distribution

VERSION ?= 2.0.0
BINARY_NAME = codecontext
BUILD_DIR = dist
LDFLAGS = -ldflags "-X main.version=$(VERSION) -X main.buildDate=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')"

# Default target
all: clean build

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f $(BINARY_NAME)

# Create build directory
$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

# Build for current platform
build: $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/codecontext

# Build for multiple platforms
build-all: $(BUILD_DIR)
	# macOS (Intel)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/codecontext
	# macOS (Apple Silicon)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/codecontext
	# Linux (Intel)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/codecontext
	# Linux (ARM)
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/codecontext
	# Windows (Intel)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/codecontext

# Create release tarballs
release: build-all
	cd $(BUILD_DIR) && \
	tar -czf $(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64 && \
	tar -czf $(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64 && \
	tar -czf $(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64 && \
	tar -czf $(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64 && \
	zip $(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe

# Generate checksums for release files
checksums: release
	cd $(BUILD_DIR) && \
	shasum -a 256 *.tar.gz *.zip > checksums.txt

# Install locally (for testing)
install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

# Uninstall
uninstall:
	rm -f /usr/local/bin/$(BINARY_NAME)

# Run tests
test:
	go test ./...

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Prepare for Homebrew (build universal binary for macOS)
homebrew: $(BUILD_DIR)
	# Build both architectures
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-amd64 ./cmd/codecontext
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-arm64 ./cmd/codecontext
	# Create universal binary using lipo
	lipo -create -output $(BUILD_DIR)/$(BINARY_NAME) $(BUILD_DIR)/$(BINARY_NAME)-amd64 $(BUILD_DIR)/$(BINARY_NAME)-arm64
	# Create tarball for Homebrew
	cd $(BUILD_DIR) && tar -czf $(BINARY_NAME)-$(VERSION)-darwin-universal.tar.gz $(BINARY_NAME)
	# Generate checksum
	cd $(BUILD_DIR) && shasum -a 256 $(BINARY_NAME)-$(VERSION)-darwin-universal.tar.gz > $(BINARY_NAME)-$(VERSION)-darwin-universal.tar.gz.sha256

# Development build (with debug symbols)
dev-build: $(BUILD_DIR)
	go build -race -o $(BUILD_DIR)/$(BINARY_NAME)-dev ./cmd/codecontext

# Show help
help:
	@echo "Available targets:"
	@echo "  all         - Clean and build for current platform"
	@echo "  build       - Build for current platform"
	@echo "  build-all   - Build for all supported platforms"
	@echo "  release     - Create release tarballs for all platforms"
	@echo "  checksums   - Generate checksums for release files"
	@echo "  homebrew    - Build universal macOS binary for Homebrew"
	@echo "  install     - Install binary locally"
	@echo "  uninstall   - Remove installed binary"
	@echo "  test        - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  fmt         - Format code"
	@echo "  lint        - Lint code"
	@echo "  clean       - Clean build artifacts"
	@echo "  help        - Show this help"

.PHONY: all clean build build-all release checksums install uninstall test test-coverage fmt lint homebrew dev-build help