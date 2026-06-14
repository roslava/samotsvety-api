.PHONY: help build test run clean migrate-up migrate-down seed db-up db-down db-reset dev-install deps

# Load .env variables
-include .env

help:
	@echo "Samotsvety API — Makefile targets"
	@echo ""
	@echo "Development:"
	@echo "  make build          Build the application"
	@echo "  make test           Run all tests"
	@echo "  make run            Build and run the server"
	@echo "  make clean          Remove binary and temp files"
	@echo ""
	@echo "Database (Docker):"
	@echo "  make db-up          Start PostgreSQL in Docker"
	@echo "  make db-down        Stop PostgreSQL container"
	@echo "  make db-reset       Stop and remove PostgreSQL (⚠️  deletes data)"
	@echo "  make migrate-up     Apply database migrations"
	@echo "  make migrate-down   Rollback database migrations"
	@echo ""
	@echo "Other:"
	@echo "  make seed           Seed the database with sample data"
	@echo "  make deps           Download and verify dependencies"
	@echo "  make dev-install    Install development tools"

# Build the application
build:
	@echo "Building samotsvety-api..."
	go build -o bin/server cmd/server/main.go
	@echo "Build complete: bin/server"

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	@echo "Test coverage report: coverage.out"

# Run the server (requires .env file)
run: build
	@if [ ! -f .env ]; then cp .env.example .env; fi
	@echo "Starting server..."
	./bin/server

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out
	@echo "Clean complete"

# Database migrations
migrate-up:
	@echo "Applying migrations..."
	@if [ ! -f "$(HOME)/go/bin/migrate" ]; then \
		echo "Error: golang-migrate not installed. Install with:"; \
		echo "  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"; \
		exit 1; \
	fi
	@echo "Database URL: postgres://$(DB_USER):***@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)"
	$(HOME)/go/bin/migrate -path migrations -database "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)" up

migrate-down:
	@echo "Rolling back migrations..."
	@if [ ! -f "$(HOME)/go/bin/migrate" ]; then \
		echo "Error: golang-migrate not installed. Install with:"; \
		echo "  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"; \
		exit 1; \
	fi
	@echo "Database URL: postgres://$(DB_USER):***@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)"
	$(HOME)/go/bin/migrate -path migrations -database "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)" down

# Seed database
seed:
	@echo "Seeding data..."
	@if [ ! -d "seeds/minerals" ]; then \
		echo "Error: seeds/minerals directory not found"; \
		exit 1; \
	fi
	@go run cmd/seed/main.go
	@echo "✅ Seeding completed!"

# Docker database operations
db-up:
	@echo "Starting PostgreSQL in Docker..."
	docker-compose up -d postgres
	@echo "PostgreSQL is starting... wait for health check (10s)"
	@sleep 10
	@echo "Database ready at postgres://postgres:postgres@localhost:5432/samotsvety"

db-down:
	@echo "Stopping PostgreSQL..."
	docker-compose down

db-reset:
	@echo "⚠️  Removing PostgreSQL container and data..."
	docker-compose down -v
	@echo "Data deleted. Run 'make db-up' to start fresh"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod verify
	@echo "Dependencies downloaded and verified"

# Development: install tools
dev-install:
	@echo "Installing development tools..."
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	@echo "Development tools installed"

# Generate Swagger documentation
swag:
	@echo "Generating Swagger docs..."
	swag init -g cmd/server/main.go --parseDependency --parseInternal --parseDepth 1
	@echo "Swagger docs generated successfully!"