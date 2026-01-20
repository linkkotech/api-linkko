.PHONY: help dev migrate cleanup logs test build clean down

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

dev: ## Start development environment with Docker Compose
	docker-compose up --build

migrate: ## Run database migrations
	docker-compose run --rm migrate

cleanup: ## Run idempotency keys cleanup
	docker-compose run --rm api cleanup

logs: ## Follow API logs
	docker-compose logs -f api

test: ## Run tests
	go test -v -race ./...

build: ## Build Docker image
	docker build -t linkko-api:latest .

clean: ## Clean up Go build cache
	go clean -cache -modcache -testcache

down: ## Stop and remove all containers
	docker-compose down -v

install: ## Install Go dependencies
	go mod download
	go mod tidy

lint: ## Run linter
	golangci-lint run

format: ## Format code
	go fmt ./...
	goimports -w .
