.PHONY: help build test run clean migrate-up migrate-down seed

help:
	@echo "Samotsvety API — Makefile targets"
	@echo ""
	@echo "  make build          Build the application"
	@echo "  make test           Run all tests"
	@echo "  make run            Build and run the server"
	@echo "  make clean          Remove binary and temp files"
	@echo "  make migrate-up     Apply database migrations"
	@echo "  make migrate-down   Rollback database migrations"
	@echo "  make seed           Seed the database with sample data"
	@echo "  make deps           Download and verify dependencies"

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
	@if ! command -v migrate &> /dev/null; then \
		echo "Error: golang-migrate not installed. Install with:"; \
		echo "  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"; \
		exit 1; \
	fi
	migrate -path migrations -database "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)" up

migrate-down:
	@echo "Rolling back migrations..."
	@if ! command -v migrate &> /dev/null; then \
		echo "Error: golang-migrate not installed. Install with:"; \
		echo "  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"; \
		exit 1; \
	fi
	migrate -path migrations -database "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)" down

# Seed database
seed:
	@echo "Seeding database..."
	@if [ ! -d "seeds/minerals" ]; then \
		echo "Warning: seeds/minerals directory not found"; \
		exit 1; \
	fi
	@echo "Seed data location: seeds/minerals/"
	@echo "TODO: Implement seed command or use Go seed utility"

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
