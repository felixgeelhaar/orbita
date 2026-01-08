.PHONY: all build build-worker build-mcp test test-unit test-integration coverage security coverage-check coverage-report coverage-badge migrate-up migrate-down migrate-create sqlc docker-up docker-down docker-logs dev worker clean help tools

# Variables
BINARY_NAME=orbita
GO=go
GOFLAGS=-race

# Default target
all: lint test build

# Build
build:
	$(GO) build -o bin/$(BINARY_NAME) ./cmd/orbita

build-worker:
	$(GO) build -o bin/worker ./cmd/worker

build-mcp:
	$(GO) build -o bin/mcp ./cmd/mcp

mcp-serve: build
	@echo "Starting MCP server via CLI command; press Ctrl+C to stop"
	./bin/$(BINARY_NAME) mcp serve

# Run tests
test:
	$(GO) test -v $(GOFLAGS) -cover ./...

test-unit:
	$(GO) test -v $(GOFLAGS) -short ./...

test-integration:
	$(GO) test -v $(GOFLAGS) -run Integration ./...

# Coverage
coverage:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Security & Coverage
security:
	verdict sast --path .
	verdict vuln --path .
	verdict secrets --path .

coverage-check:
	coverctl check --profile coverage.out

coverage-report:
	coverctl report --profile coverage.out

coverage-badge:
	coverctl badge --profile coverage.out --output coverage-badge.svg

# Database migrations
migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

migrate-force:
	@read -p "Version to force: " version; \
	migrate -path migrations -database "$(DATABASE_URL)" force $$version

# sqlc
sqlc:
	sqlc generate -f db/sqlc.yaml

sqlc-verify:
	sqlc verify -f db/sqlc.yaml

# Docker
docker-up:
	docker-compose -f deploy/docker-compose.yml up -d

docker-down:
	docker-compose -f deploy/docker-compose.yml down

docker-down-v:
	docker-compose -f deploy/docker-compose.yml down -v

docker-logs:
	docker-compose -f deploy/docker-compose.yml logs -f

docker-ps:
	docker-compose -f deploy/docker-compose.yml ps

# Development
dev: docker-up
	@echo "Waiting for services to be ready..."
	@sleep 3
	$(GO) run ./cmd/orbita

worker:
	$(GO) run ./cmd/worker

# Clean
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Install tools
tools:
	go install github.com/felixgeelhaar/verdictsec@latest
	go install github.com/felixgeelhaar/coverctl@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/vektra/mockery/v2@latest

# Help
help:
	@echo "Available targets:"
	@echo "  build           - Build the CLI binary"
	@echo "  build-worker    - Build the worker binary"
	@echo "  build-mcp       - Build the MCP server binary"
	@echo "  test            - Run all tests with race detection"
	@echo "  test-unit       - Run unit tests only"
	@echo "  test-integration- Run integration tests only"
	@echo "  coverage        - Generate test coverage report"
	@echo "  security        - Run security scans (SAST, vuln, secrets)"
	@echo "  coverage-check  - Check coverage against policy"
	@echo "  coverage-report - Generate coverage report"
	@echo "  coverage-badge  - Generate coverage badge SVG"
	@echo "  migrate-up      - Apply all migrations"
	@echo "  migrate-down    - Rollback last migration"
	@echo "  migrate-create  - Create new migration"
	@echo "  sqlc            - Generate sqlc code"
	@echo "  docker-up       - Start Docker services"
	@echo "  docker-down     - Stop Docker services"
	@echo "  docker-down-v   - Stop Docker services and remove volumes"
	@echo "  docker-logs     - Tail Docker logs"
	@echo "  dev             - Start services and run CLI"
	@echo "  worker          - Run background worker"
	@echo "  clean           - Remove build artifacts"
	@echo "  tools           - Install development tools"
