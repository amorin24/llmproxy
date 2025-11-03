.PHONY: test test-unit test-integration test-race test-coverage clean build lint help

help:
	@echo "Available targets:"
	@echo "  test              - Run all tests"
	@echo "  test-unit         - Run unit tests only"
	@echo "  test-integration  - Run integration tests"
	@echo "  test-race         - Run tests with race detector"
	@echo "  test-coverage     - Run tests with coverage report"
	@echo "  lint              - Run linters"
	@echo "  build             - Build the application"
	@echo "  clean             - Clean build artifacts"

test:
	go test ./... -v -timeout 10m

test-unit:
	go test ./... -v -short -timeout 5m

test-integration:
	go test ./... -v -run Integration -timeout 10m

test-race:
	go test ./... -race -timeout 15m

test-coverage:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint not found, installing..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

build:
	go build -o bin/llmproxy ./cmd/server

clean:
	rm -rf bin/ coverage.out coverage.html
