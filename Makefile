.PHONY: help run build test migrate migrate-up migrate-down docker-up docker-down docker-build clean

# Variables
APP_NAME=chattergo
MAIN_PATH=./cmd/server
BINARY_NAME=bin/$(APP_NAME)
MIGRATIONS_PATH=./migrations

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

run: ## Run the application locally
	@echo "Running application..."
	go run $(MAIN_PATH)/main.go

build: ## Build the application binary
	@echo "Building application..."
	@mkdir -p bin
	go build -o $(BINARY_NAME) $(MAIN_PATH)/main.go
	@echo "Binary built: $(BINARY_NAME)"

test: ## Run tests
	@echo "Running tests..."
	go test -v -race -cover ./...

migrate: ## Install golang-migrate tool
	@echo "Installing golang-migrate..."
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

migrate-up: ## Run database migrations up
	@echo "Running migrations up..."
	migrate -path $(MIGRATIONS_PATH) -database "postgresql://postgres:postgres@localhost:5432/chattergo?sslmode=disable" up

migrate-down: ## Run database migrations down
	@echo "Running migrations down..."
	migrate -path $(MIGRATIONS_PATH) -database "postgresql://postgres:postgres@localhost:5432/chattergo?sslmode=disable" down

docker-build: ## Build docker images
	@echo "Building docker images..."
	docker-compose build

docker-up: ## Start docker containers
	@echo "Starting docker containers..."
	docker-compose up -d
	@echo "Waiting for database to be ready..."
	@sleep 5
	@echo "Running migrations..."
	docker-compose exec backend sh -c "if command -v migrate >/dev/null 2>&1; then migrate -path /root/migrations -database 'postgresql://postgres:postgres@postgres:5432/chattergo?sslmode=disable' up; else echo 'Migrate tool not available in container. Run migrations manually.'; fi"

docker-down: ## Stop docker containers
	@echo "Stopping docker containers..."
	docker-compose down

docker-logs: ## Show docker logs
	@echo "Showing docker logs..."
	docker-compose logs -f

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f $(APP_NAME)

.DEFAULT_GOAL := help
