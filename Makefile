.PHONY: help build run test clean docker-up docker-down migrate-up migrate-down

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the application
	go build -o bin/nextd cmd/nextd/main.go

run: ## Run the application
	go run cmd/nextd/main.go

test: ## Run tests
	go test -v ./...

test-coverage: ## Run tests with coverage
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out

docker-up: ## Start Docker Compose
	docker compose up --build --force-recreate -d

docker-down: ## Stop Docker Compose
	docker compose down

migrate-up: ## Run database migrations up
	migrate -path migrations -database "postgres://nextd:nextd@localhost:5436/nextd?sslmode=disable" up

migrate-down: ## Run database migrations down
	migrate -path migrations -database "postgres://nextd:nextd@localhost:5436/nextd?sslmode=disable" down

migrate-create: ## Create new migration (usage: make migrate-create name=create_users_table)
	migrate create -ext sql -dir migrations -seq $(name)

lint: ## Run linter
	golangci-lint run

fmt: ## Format code
	go fmt ./...

tidy: ## Tidy go modules
	go mod tidy
