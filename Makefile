# WindZ Monitor Build Configuration

# Go build flags for optimized binaries
LDFLAGS = -ldflags="-s -w"

# Default target
.PHONY: all
all: windz

# Local development build
.PHONY: dev
dev:
	go build -o windz-dev

# Production build for local platform
.PHONY: windz
windz:
	go build $(LDFLAGS) -o windz

# ARM Linux build for deployment
.PHONY: linux-arm64
linux-arm64:
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o windz

# All Linux architectures
.PHONY: linux
linux: linux-arm64 linux-amd64

.PHONY: linux-amd64
linux-amd64:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o windz-linux-amd64

# Cross-platform builds
.PHONY: darwin
darwin:
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o windz-darwin-arm64
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o windz-darwin-amd64

.PHONY: windows
windows:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o windz.exe

# Development tools
.PHONY: test
test:
	go test ./...

.PHONY: fmt
fmt:
	gofmt -w .

.PHONY: vet
vet:
	go vet ./...

.PHONY: clean
clean:
	rm -f windz windz-dev windz-linux-* windz-darwin-* windz.exe

.PHONY: deploy-build
deploy-build: linux-arm64
	@echo "ARM Linux binary 'windz' ready for deployment"

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  dev          - Development build (windz-dev)"
	@echo "  windz        - Production build for local platform"
	@echo "  linux-arm64  - ARM Linux build (windz) - for deployment"
	@echo "  linux-amd64  - x64 Linux build (windz-linux-amd64)"
	@echo "  linux        - All Linux builds"
	@echo "  darwin       - macOS builds (arm64 + amd64)"
	@echo "  windows      - Windows build (windz.exe)"
	@echo "  deploy-build - Build ARM Linux binary for deployment"
	@echo "  test         - Run tests"
	@echo "  fmt          - Format code with gofmt"
	@echo "  vet          - Run go vet"
	@echo "  clean        - Remove built binaries"