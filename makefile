.PHONY: test lint lint-fix docs help

# Default target
.DEFAULT_GOAL := help

# Run all unit tests
test:
	@echo "Running all unit tests..."
	@go test ./...
	@echo "✅ Tests completed successfully"

# Run linter
lint:
	@echo "Running golangci-lint..."
	@golangci-lint run
	@echo "✅ Linting completed successfully"

# Run linter with fix
lint-fix:
	@echo "Running golangci-lint with fix..."
	@golangci-lint run --fix
	@echo "✅ Linting with fixes completed successfully"

# Generate swagger documentation
docs:
	@echo "Generating swagger documentation..."
	@swag init \
		-g main.go \
		-d ./cmd/server,\
./internal/jobs,\
./internal/company,\
./internal/technology,\
./internal/jobtech,\
./internal/techalias \
		-o ./docs
	@echo "✅ Swagger docs generated successfully"

# Show help
help:
	@echo "Available commands:"
	@echo "  test      - Run all unit tests"
	@echo "  lint      - Run golangci-lint"
	@echo "  lint-fix  - Run golangci-lint with fix"
	@echo "  docs      - Generate swagger documentation"
	@echo "  help      - Show this help message"