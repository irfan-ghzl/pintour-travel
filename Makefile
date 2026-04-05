# ─────────────────────────────────────────────────────────────────────────────
# Pintour Travel – Makefile
# ─────────────────────────────────────────────────────────────────────────────

.PHONY: help dev build test lint clean \
        docker-up docker-down docker-build docker-logs \
        sqlc swag proto tidy seed-admin

# Default target
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ── Go ────────────────────────────────────────────────────────────────────────

tidy: ## Tidy Go modules
	go mod tidy

build: ## Build the Go API binary
	go build -o bin/pintour-server ./cmd/server

run: ## Run the Go API server (requires local DB & Redis)
	go run ./cmd/server

test: ## Run Go tests
	go test ./... -v -race -timeout 30s

lint: ## Lint Go code (requires golangci-lint)
	golangci-lint run ./...

# ── Code generation ───────────────────────────────────────────────────────────

sqlc: ## Generate type-safe Go code from SQL (requires sqlc)
	sqlc generate

swag: ## Generate Swagger docs (requires swaggo/swag)
	swag init -g cmd/server/main.go --output docs --parseDependency

proto: ## Compile Protobuf schemas (requires protoc + protoc-gen-go)
	protoc \
	  --go_out=. --go_opt=paths=source_relative \
	  api/proto/schema.proto

# ── Docker ────────────────────────────────────────────────────────────────────

docker-build: ## Build all Docker images
	docker compose build

docker-up: ## Start all services in the background
	docker compose up -d

docker-down: ## Stop and remove all containers
	docker compose down

docker-logs: ## Tail logs for all services
	docker compose logs -f

docker-clean: ## Stop containers and remove volumes
	docker compose down -v

# ── Frontend ──────────────────────────────────────────────────────────────────

web-install: ## Install npm dependencies for the web app
	cd web && npm install

web-dev: ## Start the Vite dev server
	cd web && npm run dev

web-build: ## Build the React app for production
	cd web && npm run build

# ── Utilities ─────────────────────────────────────────────────────────────────

clean: ## Remove build artifacts
	rm -rf bin/ web/dist

seed-admin: ## Insert a default admin user (bcrypt-hashed password: admin123)
	psql "$${DATABASE_URL:-postgres://pintour:pintour_pass@localhost:5432/pintour_db?sslmode=disable}" \
	  -c "INSERT INTO users (name, email, password, role) VALUES \
	      ('Admin', 'admin@pintour.com', \
	       '\$$2a\$$10\$$exampleHashPlaceholder.changeMeInProduction', \
	       'admin') ON CONFLICT (email) DO NOTHING;"
