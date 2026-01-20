# flow-cli Makefile

# Version info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build settings
BINARY_NAME = flow
MAIN_PACKAGE = .
GO = go

# Linker flags for version injection
LDFLAGS = -ldflags "-s -w \
	-X github.com/mateus/flow-cli/cmd.Version=$(VERSION) \
	-X github.com/mateus/flow-cli/cmd.Commit=$(COMMIT) \
	-X github.com/mateus/flow-cli/cmd.BuildDate=$(BUILD_DATE)"

# Platforms for cross-compilation
PLATFORMS = linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: all build clean test lint install uninstall release help

# Default target
all: build

# Build the binary
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	$(GO) build $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Built: $(BINARY_NAME)"

# Build for Windows
build-windows:
	@echo "Building $(BINARY_NAME).exe $(VERSION)..."
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BINARY_NAME).exe $(MAIN_PACKAGE)
	@echo "Built: $(BINARY_NAME).exe"

# Install to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME) to GOPATH/bin..."
ifeq ($(OS),Windows_NT)
	@cp $(BINARY_NAME).exe "$(shell go env GOPATH)/bin/$(BINARY_NAME).exe" 2>/dev/null || cp $(BINARY_NAME) "$(shell go env GOPATH)/bin/$(BINARY_NAME).exe"
else
	@cp $(BINARY_NAME) "$(shell go env GOPATH)/bin/$(BINARY_NAME)"
endif
	@echo "Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)"

# Uninstall
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
ifeq ($(OS),Windows_NT)
	rm -f "$(shell go env GOPATH)/bin/$(BINARY_NAME).exe"
else
	rm -f "$(shell go env GOPATH)/bin/$(BINARY_NAME)"
endif
	@echo "Uninstalled"

# Run tests
test:
	@echo "Running tests..."
	$(GO) test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Lint
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "Done"

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	$(GO) mod tidy
	@echo "Done"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe
	rm -f coverage.out coverage.html
	rm -rf dist/
	@echo "Clean"

# Build for all platforms
release: clean
	@echo "Building releases for $(VERSION)..."
	@mkdir -p dist
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} \
		$(GO) build $(LDFLAGS) -o dist/$(BINARY_NAME)_$(VERSION)_$${platform%/*}_$${platform#*/}$(if $(findstring windows,$${platform%/*}),.exe,) $(MAIN_PACKAGE); \
		echo "Built: dist/$(BINARY_NAME)_$(VERSION)_$${platform%/*}_$${platform#*/}"; \
	done
	@echo "Release builds complete"

# Package releases
package: release
	@echo "Packaging releases..."
	@cd dist && for f in $(BINARY_NAME)_*; do \
		if [ -f "$$f" ]; then \
			if echo "$$f" | grep -q '.exe$$'; then \
				zip "$${f%.exe}.zip" "$$f"; \
			else \
				tar -czf "$$f.tar.gz" "$$f"; \
			fi; \
		fi; \
	done
	@echo "Packages created in dist/"

# Development run
run:
	$(GO) run $(MAIN_PACKAGE) $(ARGS)

# Watch for changes and rebuild (requires entr)
watch:
	@if command -v entr >/dev/null 2>&1; then \
		find . -name '*.go' | entr -r make build; \
	else \
		echo "entr not installed. Install with your package manager."; \
	fi

# Show help
help:
	@echo "flow-cli Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make              Build the binary"
	@echo "  make build        Build the binary"
	@echo "  make install      Install to GOPATH/bin"
	@echo "  make uninstall    Remove from GOPATH/bin"
	@echo "  make test         Run tests"
	@echo "  make test-coverage Run tests with coverage report"
	@echo "  make lint         Run linter"
	@echo "  make fmt          Format code"
	@echo "  make tidy         Tidy dependencies"
	@echo "  make clean        Remove build artifacts"
	@echo "  make release      Build for all platforms"
	@echo "  make package      Build and package releases"
	@echo "  make run ARGS=... Run with arguments"
	@echo "  make help         Show this help"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION=$(VERSION)"
	@echo "  COMMIT=$(COMMIT)"
	@echo "  BUILD_DATE=$(BUILD_DATE)"
