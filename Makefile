.PHONY: build run clean test release build-all

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS = -ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE)"

# Build the binary
build:
	go build $(LDFLAGS) -o tfplan-commenter main.go

# Build for all platforms
build-all:
	# Linux x86_64
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/tfplan-commenter-linux-amd64 main.go
	# macOS Intel
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/tfplan-commenter-darwin-amd64 main.go
	# macOS Apple Silicon
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/tfplan-commenter-darwin-arm64 main.go

# Create release directory and build all platforms
release: clean-dist
	mkdir -p dist
	$(MAKE) build-all
	# Create checksums
	cd dist && sha256sum * > checksums.txt

# Run with the example plan.json file
run: build
	./tfplan-commenter plan.json

# Run and specify output file
run-output: build
	./tfplan-commenter plan.json terraform-plan-comment.md

# Show version
version: build
	./tfplan-commenter -version

# Clean build artifacts
clean:
	rm -f tfplan-commenter terraform-plan-comment.md test-output.md

# Clean distribution artifacts
clean-dist:
	rm -rf dist/

# Test the program
test: build
	./tfplan-commenter plan.json test-output.md
	@echo "Generated test output:"
	@cat test-output.md

# Install dependencies (if any are added later)
deps:
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Show help
help: build
	./tfplan-commenter -help
