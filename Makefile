.PHONY: run build migrate-up migrate-down migrate-new seed test test-int lint

run: ## Hot reload with air
	air

build: ## Build binary
	go build -o bin/api ./cmd/api

migrate-up: ## Run all pending migrations
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down: ## Roll back one migration
	migrate -path migrations -database "$(DATABASE_URL)" down 1

migrate-new: ## Create a new migration: make migrate-new name=create_foo
	migrate create -ext sql -dir migrations -seq $(name)

seed: ## Seed demo data and admin user
	go run ./seed

test: ## Run unit tests with race detector
	go test ./... -race -count=1

test-int: ## Run integration tests (requires running DB)
	go test ./... -tags=integration -race -count=1

lint: ## Lint with golangci-lint
	golangci-lint run

docker-up: ## Start Postgres + API via docker-compose
	docker compose up --build

docker-db: ## Start only Postgres
	docker compose up db -d
