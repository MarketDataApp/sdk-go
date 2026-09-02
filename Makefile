# MarketData Go SDK v2 Makefile

.PHONY: all build test test-v test-cover test-integration lint fmt tidy clean help

# Default target
all: lint test build

# Build the package
build:
	go build ./...

# Run unit tests (default, no token needed)
test:
	go test ./...

# Run unit tests with verbose output
test-v:
	go test -v ./...

# Run unit tests with coverage
test-cover:
	go test -cover ./...

# Run unit tests with coverage report
test-coverage-report:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run integration tests (requires MARKETDATA_TOKEN)
test-integration:
	go test ./integration/... -tags=integration -v

# Run integration tests with race detection
test-integration-race:
	go test ./integration/... -tags=integration -v -race

# Run all tests (unit + integration)
test-all: test test-integration

# Lint the code (requires golangci-lint)
lint:
	golangci-lint run

# Format the code
fmt:
	go fmt ./...

# Tidy dependencies
tidy:
	go mod tidy

# Clean build artifacts
clean:
	rm -f coverage.out coverage.html
	go clean ./...

# Show help
help:
	@echo "MarketData Go SDK v2 Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make              - Lint, test, and build"
	@echo "  make build        - Build the package"
	@echo "  make test         - Run unit tests"
	@echo "  make test-v       - Run unit tests with verbose output"
	@echo "  make test-cover   - Run unit tests with coverage"
	@echo "  make test-integration - Run integration tests (requires MARKETDATA_TOKEN)"
	@echo "  make test-all     - Run all tests (unit + integration)"
	@echo "  make lint         - Run golangci-lint"
	@echo "  make fmt          - Format code"
	@echo "  make tidy         - Tidy dependencies"
	@echo "  make clean        - Clean build artifacts"
	@echo ""
	@echo "Integration tests require:"
	@echo "  export MARKETDATA_TOKEN=your-token"
