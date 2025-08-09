.PHONY: test lint lint-fix docs help migrate-up migrate-down migrate-status migrate-create migrate-force migrate-drop migrate-goto migrate-up-by migrate-down-by

# Load environment variables from .env file if it exists
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# Database configuration - these will use values from .env if present, otherwise defaults
DB_HOST ?= localhost
DB_PORT ?= 5432
DB_USER ?= postgres
DB_PASSWORD ?= postgres
DB_NAME ?= postgres
DB_SSL_MODE ?= disable

# Construct DATABASE_URL from individual components
DATABASE_URL = postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSL_MODE)

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
		-d .,\
./internal/jobs,\
./internal/company,\
./internal/technology,\
./internal/jobtech,\
./internal/techalias \
		-o ./docs
	@echo "✅ Swagger docs generated successfully"

# Migration commands
migrate-up:
	@echo "Applying all migrations..."
	@migrate -path migrations -database "$(DATABASE_URL)" up
	@echo "✅ All migrations applied successfully"

migrate-down:
	@echo "Rolling back all migrations..."
	@migrate -path migrations -database "$(DATABASE_URL)" down
	@echo "✅ All migrations rolled back successfully"

migrate-status:
	@echo "Checking migration status..."
	@migrate -path migrations -database "$(DATABASE_URL)" version

migrate-create:
	@if [ -z "$(NAME)" ]; then \
		echo "❌ Error: NAME is required. Usage: make migrate-create NAME=your_migration_name"; \
		exit 1; \
	fi
	@echo "Creating new migration: $(NAME)"
	@migrate create -ext sql -dir migrations -seq $(NAME)
	@echo "✅ Migration files created successfully"

migrate-force:
	@if [ -z "$(VERSION)" ]; then \
		echo "❌ Error: VERSION is required. Usage: make migrate-force VERSION=001"; \
		exit 1; \
	fi
	@echo "⚠️  Forcing migration to version $(VERSION)..."
	@migrate -path migrations -database "$(DATABASE_URL)" force $(VERSION)
	@echo "✅ Migration forced to version $(VERSION)"

migrate-drop:
	@echo "⚠️  WARNING: This will drop all tables and data!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
    echo; \
    if [ "$$REPLY" = "y" ] || [ "$$REPLY" = "Y" ]; then \
        migrate -path migrations -database "$(DATABASE_URL)" drop; \
        echo "✅ Database dropped successfully"; \
    else \
        echo "❌ Operation cancelled"; \
    fi

migrate-goto:
	@if [ -z "$(VERSION)" ]; then \
		echo "❌ Error: VERSION is required. Usage: make migrate-goto VERSION=3"; \
		exit 1; \
	fi
	@echo "Migrating to version $(VERSION)..."
	@migrate -path migrations -database "$(DATABASE_URL)" goto $(VERSION)
	@echo "✅ Migrated to version $(VERSION)"

migrate-up-by:
	@if [ -z "$(STEPS)" ]; then \
		echo "❌ Error: STEPS is required. Usage: make migrate-up-by STEPS=2"; \
		exit 1; \
	fi
	@echo "Applying $(STEPS) migrations..."
	@migrate -path migrations -database "$(DATABASE_URL)" up $(STEPS)
	@echo "✅ Applied $(STEPS) migrations successfully"

migrate-down-by:
	@if [ -z "$(STEPS)" ]; then \
		echo "❌ Error: STEPS is required. Usage: make migrate-down-by STEPS=1"; \
		exit 1; \
	fi
	@echo "Rolling back $(STEPS) migrations..."
	@migrate -path migrations -database "$(DATABASE_URL)" down $(STEPS)
	@echo "✅ Rolled back $(STEPS) migrations successfully"

# Show help
help:
	@echo "Available commands:"
	@echo ""
	@echo "Development:"
	@echo "  test           - Run all unit tests"
	@echo "  lint           - Run golangci-lint"
	@echo "  lint-fix       - Run golangci-lint with fix"
	@echo "  docs           - Generate swagger documentation"
	@echo ""
	@echo "Database Migrations:"
	@echo "  migrate-up     - Apply all pending migrations"
	@echo "  migrate-down   - Rollback all migrations"
	@echo "  migrate-status - Check current migration version"
	@echo "  migrate-create NAME=<name> - Create new migration files"
	@echo "  migrate-force VERSION=<ver> - Force migration to specific version"
	@echo "  migrate-drop   - Drop all tables (with confirmation)"
	@echo "  migrate-goto VERSION=<ver>  - Migrate to specific version"
	@echo "  migrate-up-by STEPS=<n>     - Apply N migrations"
	@echo "  migrate-down-by STEPS=<n>   - Rollback N migrations"
	@echo ""
	@echo "Examples:"
	@echo "  make migrate-create NAME=add_user_table"
	@echo "  make migrate-up-by STEPS=2"
	@echo "  make migrate-force VERSION=001"
	@echo ""
	@echo "Note: Database configuration is read from .env file or environment variables"