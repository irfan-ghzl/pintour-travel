# ─────────────────────────────────────────────────────────────────────────────
# Pintour Travel — Docker rebuild automation
#
# Usage:
#   .\scripts\rebuild.ps1              # rebuild + restart (DB data dipertahankan)
#   .\scripts\rebuild.ps1 -Fresh       # WIPE DB + rebuild + migrations + seed admin
#   .\scripts\rebuild.ps1 -SkipBuild   # restart pakai image yang sudah ada
#   .\scripts\rebuild.ps1 -ApiOnly     # rebuild API saja (skip web)
# ─────────────────────────────────────────────────────────────────────────────

param(
  [switch]$Fresh,
  [switch]$SkipBuild,
  [switch]$ApiOnly,
  [switch]$NoCache
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

function Section($msg) {
  Write-Host ""
  Write-Host "─── $msg ───" -ForegroundColor Cyan
}

function Ok($msg) { Write-Host "✓ $msg" -ForegroundColor Green }
function Warn($msg) { Write-Host "⚠ $msg" -ForegroundColor Yellow }
function Err($msg) { Write-Host "✗ $msg" -ForegroundColor Red }

# Semua perintah di skrip ini memakai overlay dev. Berkas dasar memaksa
# APP_ENV=production supaya Validate() benar-benar berjalan di server;
# menjalankannya apa adanya di laptop membuat API menolak start karena
# PORTAL_BASE_URL memang localhost dan Midtrans memang sandbox.
$C = @('-f', 'docker-compose.yml', '-f', 'docker-compose.dev.yml')

# 0. Pre-check
if (-not (Test-Path ".env")) {
  Err ".env tidak ditemukan. Copy dari .env.example dulu:"
  Write-Host "    Copy-Item .env.example .env" -ForegroundColor Gray
  exit 1
}

Section "1. Stopping running containers"
docker compose @C down 2>&1 | Out-Null
Ok "Containers stopped"

if ($Fresh) {
  Section "2. WIPE database volume (semua data akan hilang!)"
  $confirm = Read-Host "Yakin reset DB? Semua data peserta/leads/invoice akan terhapus. [y/N]"
  if ($confirm -ne 'y' -and $confirm -ne 'Y') {
    Warn "Aborted"
    exit 0
  }
  docker volume rm pintour-travel_db_data 2>&1 | Out-Null
  Ok "DB volume removed — migrations 001-005 akan otomatis dijalankan ulang"
} else {
  Ok "Skip DB reset (pakai data lama)"
}

if (-not $SkipBuild) {
  Section "3. Rebuilding images"
  $buildArgs = @()
  if ($NoCache) { $buildArgs += "--no-cache" }
  if ($ApiOnly) {
    docker compose @C build @buildArgs api
  } else {
    docker compose @C build @buildArgs
  }
  if ($LASTEXITCODE -ne 0) {
    Err "Build gagal — cek error di atas"
    exit 1
  }
  Ok "Images rebuilt"
} else {
  Ok "Skip build (pakai image yang sudah ada)"
}

Section "4. Starting services"
docker compose @C up -d
if ($LASTEXITCODE -ne 0) {
  Err "Start gagal"
  exit 1
}
Ok "Services started"

Section "5. Waiting for DB & API to be healthy"
$timeout = 60
$elapsed = 0
$dbHealthy = $false
$apiReady = $false
while ($elapsed -lt $timeout) {
  $dbStatus = docker inspect --format='{{.State.Health.Status}}' pintour_db 2>$null
  if ($dbStatus -eq 'healthy' -and -not $dbHealthy) {
    Ok "Database healthy"
    $dbHealthy = $true
  }
  try {
    $resp = Invoke-WebRequest -Uri "http://localhost:8080/health" -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop 2>$null
    if ($resp.StatusCode -eq 200) {
      Ok "API responding at :8080"
      $apiReady = $true
      break
    }
  } catch {}
  Start-Sleep -Seconds 2
  $elapsed += 2
  Write-Host "  waiting... ($elapsed/${timeout}s)" -ForegroundColor DarkGray
}

if (-not $apiReady) {
  Warn "API belum responding setelah ${timeout}s — cek log:"
  Write-Host "    docker compose -f docker-compose.yml -f docker-compose.dev.yml logs -f api" -ForegroundColor Gray
}

if ($Fresh -and $dbHealthy) {
  Section "6. Seeding default admin"
  # cmd/seed connect via host (5432 exposed)
  $env:DATABASE_URL = "postgres://pintour:pintour_pass@localhost:5432/pintour_db?sslmode=disable"
  go run ./cmd/seed
  if ($LASTEXITCODE -eq 0) {
    Ok "Default admin created: admin@pintour.com / admin123"
  } else {
    Warn "Seed gagal — coba manual: go run ./cmd/seed"
  }
}

Section "Done!"
Write-Host ""
Write-Host "  Web app   : " -NoNewline; Write-Host "http://localhost" -ForegroundColor Yellow
Write-Host "  API       : " -NoNewline; Write-Host "http://localhost:8080/api/v1" -ForegroundColor Yellow
Write-Host "  Swagger   : " -NoNewline; Write-Host "http://localhost:8080/swagger/index.html" -ForegroundColor Yellow
Write-Host "  Postgres  : " -NoNewline; Write-Host "postgres://pintour:pintour_pass@localhost:5432/pintour_db" -ForegroundColor Yellow
Write-Host ""
Write-Host "  Tail logs : " -NoNewline; Write-Host "docker compose -f docker-compose.yml -f docker-compose.dev.yml logs -f" -ForegroundColor Gray
Write-Host "  Stop all  : " -NoNewline; Write-Host "docker compose -f docker-compose.yml -f docker-compose.dev.yml down" -ForegroundColor Gray
Write-Host ""
