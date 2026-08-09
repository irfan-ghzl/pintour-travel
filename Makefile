# ─────────────────────────────────────────────────────────────────────────────
# Pintour Travel – Makefile
# ─────────────────────────────────────────────────────────────────────────────

.PHONY: help dev build test test-cover cover-report lint clean \
        docker-up docker-down docker-build docker-logs \
        sqlc swag proto tidy seed-admin migrate migrate-down

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

# ── Coverage (exit criteria §21.10) ──────────────────────────────────────────
#
# Two flags carry this target, and dropping either makes the number wrong:
#
#   -coverpkg  Without it each package is measured only against its own tests,
#              so a package with no test file is left out of the DENOMINATOR
#              entirely and the number reads far higher than it is. With it,
#              every internal package counts whether or not it has tests —
#              which is what "coverage backend menyeluruh" means.
#
#   -count=1   Defeats the test cache. A cached test result carries the coverage
#              profile it was recorded with, keyed on the line numbers of the
#              source at that time. Mix those with a freshly compiled package
#              and the same code is counted twice under two numberings — the
#              denominator inflates and the percentage drifts. Measured once
#              without it here and it read 58,8% against a true 60,4%.

migrate: ## Apply pending SQL migrations to DATABASE_URL (safe to re-run)
	go run ./cmd/migrate

migrate-down: ## Roll back the most recent migration (DISCARDS DATA — read its .down.sql first)
	go run ./cmd/migrate -down

test-cover: ## Measure backend coverage over every internal package (§21.10)
	go test ./internal/... -count=1 -coverpkg=./internal/... -coverprofile=coverage.out
	@go tool cover -func=coverage.out | tail -1

cover-report: test-cover ## Coverage per package + a browsable HTML report
	@go tool cover -html=coverage.out -o coverage.html
	@echo "laporan HTML: coverage.html"
	@awk 'NR>1 {k=$$1; n[k]=$$2; if($$3>0) hit[k]=1} \
	  END {for (k in n) {split(k,a,":"); p=a[1]; sub(/\/[^\/]*$$/,"",p); \
	    gsub(/github.com\/irfan-ghzl\/pintour-travel\//,"",p); \
	    tot[p]+=n[k]; T+=n[k]; if(k in hit){cov[p]+=n[k]; C+=n[k]}} \
	  for (p in tot) printf "%6.1f%%  %5d stmts  %s\n", 100*cov[p]/tot[p], tot[p], p; \
	  printf "\n%6.1f%%  %5d stmts  TOTAL\n", 100*C/T, T}' coverage.out | sort -n

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

rebuild: ## Rebuild containers (keep DB data)
	@pwsh -File scripts/rebuild.ps1 || bash scripts/rebuild.sh

rebuild-fresh: ## WIPE DB + rebuild + run migrations + seed admin
	@pwsh -File scripts/rebuild.ps1 -Fresh || bash scripts/rebuild.sh --fresh

rebuild-api: ## Rebuild API saja (skip web)
	@pwsh -File scripts/rebuild.ps1 -ApiOnly || bash scripts/rebuild.sh --api-only

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
