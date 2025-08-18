.PHONY: help build test clean install

# Default target
help:
	@echo "Available targets:"
	@echo "  build     - Build the provider binary"
	@echo "  test      - Run tests"
	@echo "  clean     - Clean build artifacts"
	@echo "  install   - Install provider locally for testing"
	@echo "  release   - Build and create release artifacts"

# Build the provider
build:
	@echo "Building provider..."
	go build -o terraform-provider-kubevirt .

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...
	go vet ./...
	go fmt ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f terraform-provider-kubevirt
	rm -rf bin/

# Install provider locally for testing
install: build
	@echo "Installing provider locally..."
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/terraform-dev/kubevirt/0.1.0/linux_amd64/
	cp terraform-provider-kubevirt ~/.terraform.d/plugins/registry.terraform.io/terraform-dev/kubevirt/0.1.0/linux_amd64/

# Create release artifacts
release: clean
	@echo "Creating release artifacts..."
	mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -o bin/terraform-provider-kubevirt_0.1.0 .
	GOOS=darwin GOARCH=amd64 go build -o bin/terraform-provider-kubevirt_0.1.0_darwin .
	@echo "Release artifacts created in bin/ directory"

# Development setup
dev-setup:
	@echo "Setting up development environment..."
	go mod download
	go mod verify
	@echo "Development environment ready!"

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run

# Generate documentation
docs:
	@echo "Generating documentation..."
	go generate ./...
