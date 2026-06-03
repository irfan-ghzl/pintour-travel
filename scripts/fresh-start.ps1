# ─────────────────────────────────────────────────────────────────────────────
# Pintour Travel — Fresh Start Script
# Usage: pwsh ./scripts/fresh-start.ps1
#
# Script ini handle:
#   1. Stop & wipe semua container + volume
#   2. Rebuild image dari source terbaru
#   3. Start container (DB → Redis → API → Web)
#   4. Tunggu DB benar2 healthy (bukan cuma "started")
#   5. Apply migration tambahan (004 & 005)
#   6. Seed dummy data
#   7. Verifikasi semua endpoint
# ─────────────────────────────────────────────────────────────────────────────

$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot/..

function Step($n, $msg) {
  Write-Host ""
  Write-Host "═══════ STEP $n ═══════ $msg" -ForegroundColor Cyan
}

function Ok($msg) { Write-Host "  ✅ $msg" -ForegroundColor Green }
function Err($msg) { Write-Host "  ❌ $msg" -ForegroundColor Red }
function Info($msg) { Write-Host "  ℹ  $msg" -ForegroundColor Gray }

# ─── 1. Stop & wipe ─────────────────────────────────────────────────────────
Step 1 "Stop & wipe semua container + volume"
docker compose down -v 2>&1 | Out-Host
Ok "Containers & volumes removed"

# ─── 2. Rebuild ─────────────────────────────────────────────────────────────
Step 2 "Rebuild images (api + web) dari source terbaru"
docker compose build --no-cache 2>&1 | Out-Host
if ($LASTEXITCODE -ne 0) { Err "Build failed"; exit 1 }
Ok "Images rebuilt"

# ─── 3. Start db & redis dulu, tunggu healthy ───────────────────────────────
Step 3 "Start db + redis"
docker compose up -d db redis 2>&1 | Out-Host

Info "Menunggu DB benar2 healthy (apply migration 001-005)..."
$maxWait = 90  # second
$elapsed = 0
while ($elapsed -lt $maxWait) {
  $status = docker inspect --format='{{.State.Health.Status}}' pintour_db 2>$null
  if ($status -eq "healthy") { Ok "DB healthy setelah $elapsed detik"; break }
  Start-Sleep -Seconds 3
  $elapsed += 3
  Write-Host "    ($elapsed/$maxWait s) status: $status" -ForegroundColor DarkGray
}
if ($elapsed -ge $maxWait) {
  Err "DB tidak healthy setelah $maxWait detik"
  docker compose logs db --tail 30 | Out-Host
  exit 1
}

# ─── 4. Apply migration tambahan (kalau 004 & 005 belum auto-apply) ─────────
Step 4 "Apply migration 004 (soft delete) + 005 (facilities)"
@("004_soft_delete.sql", "005_facilities_and_extras.sql") | ForEach-Object {
  Info "Applying $_"
  Get-Content "db/migrations/$_" | docker exec -i pintour_db psql -U pintour -d pintour_db 2>&1 | Out-Host
}
Ok "Migration 004 & 005 applied"

# ─── 5. Start api + web ──────────────────────────────────────────────────────
Step 5 "Start api + web"
docker compose up -d api web 2>&1 | Out-Host
Start-Sleep -Seconds 5
Ok "Containers started"

# ─── 6. Seed dummy data ──────────────────────────────────────────────────────
Step 6 "Seed dummy data (6 users + 5 packages + 7 leads + ...)"
go run ./cmd/seed-demo 2>&1 | Out-Host
if ($LASTEXITCODE -ne 0) { Err "Seed gagal"; exit 1 }

# ─── 7. Verifikasi ───────────────────────────────────────────────────────────
Step 7 "Verifikasi semua service"
docker compose ps | Out-Host

Write-Host ""
Info "Test health endpoint..."
try {
  $health = Invoke-RestMethod -Uri "http://localhost:8080/health" -TimeoutSec 5
  Ok "API healthy: $($health | ConvertTo-Json -Compress)"
} catch {
  Err "API tidak respond di :8080 — $($_.Exception.Message)"
}

Info "Test katalog packages..."
try {
  $pkg = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/packages" -TimeoutSec 5
  Ok "Katalog: $($pkg.meta.total) paket"
} catch {
  Err "Katalog API gagal — $($_.Exception.Message)"
}

Info "Test web index..."
try {
  $web = Invoke-WebRequest -Uri "http://localhost" -TimeoutSec 5 -UseBasicParsing
  if ($web.StatusCode -eq 200) {
    Ok "Web responding (status 200, $($web.Content.Length) bytes)"
  } else {
    Err "Web status: $($web.StatusCode)"
  }
} catch {
  Err "Web tidak respond — $($_.Exception.Message)"
}

# ─── Done ────────────────────────────────────────────────────────────────────
Write-Host ""
Write-Host "═══════════════════════════════════════════════════════════════" -ForegroundColor Green
Write-Host "  🎉 SETUP COMPLETE" -ForegroundColor Green
Write-Host "═══════════════════════════════════════════════════════════════" -ForegroundColor Green
Write-Host ""
Write-Host "  Frontend  : http://localhost" -ForegroundColor White
Write-Host "  API       : http://localhost:8080/api/v1" -ForegroundColor White
Write-Host "  Swagger   : http://localhost:8080/swagger/index.html" -ForegroundColor White
Write-Host ""
Write-Host "  Admin     : admin@pintour.com / admin123" -ForegroundColor Yellow
Write-Host "  Peserta   : 628111000006 / peserta123" -ForegroundColor Yellow
Write-Host ""
Write-Host "  ⚠  Hard refresh browser: Ctrl+Shift+R" -ForegroundColor Yellow
