# Pintour Travel – Sistem Informasi Agen Tour & Travel

Monolith system for a Tour & Travel consultant agency.

## Tech Stack

| Layer     | Technology                                      |
|-----------|-------------------------------------------------|
| Backend   | Go (Golang), Echo router, JWT auth              |
| Database  | PostgreSQL + sqlc (type-safe SQL)               |
| Caching   | Redis                                           |
| Schema    | Protobuf (`api/proto/schema.proto`)             |
| API Docs  | OpenAPI / Swagger (`swaggo/swag`)               |
| Frontend  | React 18, TypeScript, Vite, Tailwind CSS        |
| Container | Docker (multi-stage), Docker Compose            |

## Project Structure

```
pintour-travel/
├── cmd/server/          # Go entry point (main.go)
├── internal/
│   ├── config/          # App configuration (env vars)
│   ├── handler/         # HTTP handlers (tour, inquiry, quotation, user)
│   ├── middleware/       # JWT & RBAC middleware
│   ├── service/         # Business logic (user auth, WhatsApp link builder)
│   ├── repository/      # sqlc-generated DB layer (after `make sqlc`)
│   └── cache/           # Redis client wrapper
├── api/proto/           # Protobuf schema definitions
├── db/
│   ├── migrations/      # SQL migration files (001_init.sql)
│   └── queries/         # sqlc SQL query files
├── docs/                # Swagger docs (after `make swag`)
├── web/                 # React + TypeScript + Vite frontend
│   └── src/
│       ├── components/  # Reusable UI components
│       ├── pages/       # Route pages (public + admin)
│       ├── types/       # TypeScript interfaces
│       └── utils/       # API client, auth helpers
├── Dockerfile           # Multi-stage Go build
├── Dockerfile.web       # Multi-stage React/Nginx build
├── docker-compose.yml   # Full stack: App + DB + Redis + Web
├── Makefile             # Developer shortcuts
├── sqlc.yaml            # sqlc code generation config
└── nginx.conf           # Nginx reverse-proxy config for web container
```

## Quick Start

### With Docker (recommended)

```bash
# Build & start all services
make docker-up

# Tail logs
make docker-logs

# Stop
make docker-down
```

Services:
- **API**: http://localhost:8080
- **Swagger UI**: http://localhost:8080/swagger/index.html
- **Web**: http://localhost:80

### Local Development

```bash
# 1. Start DB & Redis via Docker
docker compose up db redis -d

# 2. Run the Go API
make run

# 3. Start the React dev server (in another terminal)
make web-dev  # → http://localhost:3000
```

## Code Generation

```bash
# Generate type-safe DB code from SQL
make sqlc

# Generate / refresh Swagger docs
make swag

# Compile Protobuf schemas
make proto
```

## API Endpoints

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET    | /health | Health check | Public |
| GET    | /swagger/* | Swagger UI | Public |
| POST   | /api/v1/auth/login | Login (JWT) | Public |
| GET    | /api/v1/packages | List tour packages | Public |
| GET    | /api/v1/packages/:slug | Package detail + itinerary | Public |
| GET    | /api/v1/destinations | List destinations | Public |
| GET    | /api/v1/testimonials | List testimonials | Public |
| POST   | /api/v1/inquiries | Submit Build-My-Trip form | Public |
| GET    | /api/v1/admin/auth/me | Current user info | 🔒 JWT |
| GET    | /api/v1/admin/dashboard/stats | Dashboard stats | 🔒 JWT |
| POST   | /api/v1/admin/packages | Create package | 🔒 JWT |
| PUT    | /api/v1/admin/packages/:id | Update package | 🔒 JWT |
| DELETE | /api/v1/admin/packages/:id | Delete package | 🔒 JWT |
| GET    | /api/v1/admin/inquiries | List inquiries/leads | 🔒 JWT |
| PATCH  | /api/v1/admin/inquiries/:id/status | Update inquiry status | 🔒 JWT |
| POST   | /api/v1/admin/quotations | Create quotation | 🔒 JWT |
| GET    | /api/v1/admin/quotations | List quotations | 🔒 JWT |
| GET    | /api/v1/admin/quotations/:id | Quotation detail + items | 🔒 JWT |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `8080` | HTTP port |
| `APP_ENV` | `development` | Environment name |
| `DATABASE_URL` | *(local)* | PostgreSQL DSN |
| `REDIS_ADDR` | `localhost:6379` | Redis address |
| `REDIS_PASSWORD` | *(empty)* | Redis password |
| `JWT_SECRET` | *(dev only)* | JWT signing key |
| `JWT_EXPIRATION_HOURS` | `72` | Token TTL in hours |
