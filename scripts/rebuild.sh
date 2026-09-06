#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# Pintour Travel — Docker rebuild automation
#
# Usage:
#   ./scripts/rebuild.sh             # rebuild + restart (data dipertahankan)
#   ./scripts/rebuild.sh --fresh     # WIPE DB + rebuild + seed admin
#   ./scripts/rebuild.sh --skip-build
#   ./scripts/rebuild.sh --api-only
#   ./scripts/rebuild.sh --no-cache
# ─────────────────────────────────────────────────────────────────────────────

set -e
cd "$(dirname "$0")/.."

FRESH=false
SKIP_BUILD=false
API_ONLY=false
NO_CACHE=false

for arg in "$@"; do
  case $arg in
    --fresh)      FRESH=true ;;
    --skip-build) SKIP_BUILD=true ;;
    --api-only)   API_ONLY=true ;;
    --no-cache)   NO_CACHE=true ;;
    *) echo "Unknown arg: $arg"; exit 1 ;;
  esac
done

CYAN='\033[0;36m'; GREEN='\033[0;32m'; YELLOW='\033[0;33m'; RED='\033[0;31m'; NC='\033[0m'

section() { echo -e "\n${CYAN}─── $1 ───${NC}"; }
ok()      { echo -e "${GREEN}✓ $1${NC}"; }
warn()    { echo -e "${YELLOW}⚠ $1${NC}"; }
err()     { echo -e "${RED}✗ $1${NC}"; }

[ ! -f .env ] && { err ".env tidak ditemukan. cp .env.example .env"; exit 1; }

# Semua perintah di skrip ini memakai overlay dev. Berkas dasar memaksa
# APP_ENV=production supaya Validate() benar-benar berjalan di server;
# menjalankannya apa adanya di laptop membuat API menolak start karena
# PORTAL_BASE_URL memang localhost dan Midtrans memang sandbox.
COMPOSE="docker compose -f docker-compose.yml -f docker-compose.dev.yml"

section "1. Stopping containers"
$COMPOSE down > /dev/null 2>&1
ok "stopped"

if $FRESH; then
  section "2. WIPE database volume"
  read -p "Yakin reset DB? Semua data akan hilang. [y/N] " confirm
  [[ ! "$confirm" =~ ^[Yy]$ ]] && { warn "Aborted"; exit 0; }
  docker volume rm pintour-travel_db_data > /dev/null 2>&1 || true
  ok "DB volume removed — migrations 001-006 akan dijalankan ulang"
fi

if ! $SKIP_BUILD; then
  section "3. Rebuilding images"
  BUILD_ARGS=""
  $NO_CACHE && BUILD_ARGS="--no-cache"
  if $API_ONLY; then
    $COMPOSE build $BUILD_ARGS api
  else
    $COMPOSE build $BUILD_ARGS
  fi
  ok "images rebuilt"
fi

section "4. Starting services"
$COMPOSE up -d
ok "services started"

section "5. Waiting for DB & API"
timeout=60
elapsed=0
api_ready=false
while [ $elapsed -lt $timeout ]; do
  status=$(docker inspect --format='{{.State.Health.Status}}' pintour_db 2>/dev/null || echo "")
  if [ "$status" = "healthy" ] && ! $api_ready; then
    ok "DB healthy"
  fi
  if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
    ok "API responding"
    api_ready=true
    break
  fi
  sleep 2
  elapsed=$((elapsed + 2))
  echo "  waiting... (${elapsed}/${timeout}s)"
done

if $FRESH && $api_ready; then
  section "6. Seeding default admin"
  DATABASE_URL="postgres://pintour:pintour_pass@localhost:5432/pintour_db?sslmode=disable" \
    go run ./cmd/seed
fi

section "Done!"
echo ""
echo "  Web app   : http://localhost"
echo "  API       : http://localhost:8080/api/v1"
echo "  Swagger   : http://localhost:8080/swagger/index.html"
echo ""
echo "  Tail logs : $COMPOSE logs -f"
echo "  Stop all  : $COMPOSE down"
echo ""
