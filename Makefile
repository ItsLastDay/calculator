# Convenience targets for local development (requires Go 1.23+ and Node 20+).
# Without local toolchains, use the Docker commands in README.md instead.

.PHONY: help install dev-backend dev-frontend test lint coverage build docker-build docker-run

help: ## Show available targets
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "  %-16s %s\n", $$1, $$2}'

install: ## Install frontend dependencies
	cd frontend && npm ci

dev-backend: ## Run the API on :8080
	cd backend && go run ./cmd/server

dev-frontend: ## Run the Vite dev server on :5173 (proxies /api to :8080)
	cd frontend && npm run dev

test: ## Run all unit tests
	cd backend && go test ./...
	cd frontend && npm test

lint: ## Run linters and format checks
	cd backend && golangci-lint run ./...
	cd backend/api && npx --yes @redocly/cli@1.25.0 lint openapi.yaml
	cd frontend && npm run lint && npm run format:check && npm run typecheck

coverage: ## Generate coverage reports
	cd backend && go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html
	cd frontend && npm run coverage

build: ## Build the frontend bundle and backend binary
	cd frontend && npm run build
	cd backend && CGO_ENABLED=0 go build -o calculator ./cmd/server

docker-build: ## Build the full-stack image
	docker build -t calculator:latest .

docker-run: ## Run the full-stack image on :8080
	docker run --rm -p 8080:8080 calculator:latest
