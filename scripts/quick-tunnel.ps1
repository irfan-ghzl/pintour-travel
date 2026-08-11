# Menyalakan stack beserta Cloudflare quick tunnel, lalu menyelaraskan .env
# dengan URL publik yang baru diberikan.
#
#   .\scripts\quick-tunnel.ps1
#
# Kenapa perlu skrip dan bukan sekadar `docker compose up`: URL quick tunnel
# baru ada setelah cloudflared menyala, dan API perlu mengetahuinya sebelum bisa
# menyusun tautan portal yang benar. Tanpa langkah ini, peserta menerima tautan
# yang menunjuk localhost — kegagalan yang tidak terlihat sampai ada yang
# mencoba membukanya dari HP.
#
# Untuk mematikan:
#
#   docker compose -f docker-compose.yml -f docker-compose.quicktunnel.yml down

$ErrorActionPreference = 'Stop'

$repo = Split-Path -Parent $PSScriptRoot
Set-Location $repo

$composeArgs = @('-f', 'docker-compose.yml', '-f', 'docker-compose.quicktunnel.yml')
$envPath = Join-Path $repo '.env'

if (-not (Test-Path $envPath)) {
	Write-Error ".env tidak ada. Salin dari .env.example lebih dulu."
}

Write-Host "==> Menyalakan stack" -ForegroundColor Green
docker compose @composeArgs up -d
if (-not $?) { Write-Error "docker compose up gagal" }

Write-Host "==> Menunggu URL dari cloudflared" -ForegroundColor Green
$url = $null
for ($i = 1; $i -le 40; $i++) {
	$logs = (docker compose @composeArgs logs cloudflared 2>&1) | Out-String
	if ($logs -match 'https://[a-z0-9-]+\.trycloudflare\.com') {
		$url = $Matches[0]
		break
	}
	Start-Sleep -Seconds 3
}

if (-not $url) {
	Write-Host "URL tidak muncul dalam 2 menit. Log cloudflared:" -ForegroundColor Yellow
	docker compose @composeArgs logs cloudflared --tail=30
	Write-Error "gagal mendapatkan URL tunnel"
}

Write-Host "    $url" -ForegroundColor Cyan

# ── Selaraskan .env ──────────────────────────────────────────────────────────
# Hanya tiga kunci yang disentuh; sisanya dibiarkan apa adanya. Ditulis tanpa
# BOM karena docker compose membaca baris pertama secara harfiah, dan tiga byte
# tak terlihat di depannya membuat kunci pertama tidak pernah cocok.
Write-Host "==> Menyelaraskan .env" -ForegroundColor Green

$host_only = $url -replace '^https://', ''
$wanted = [ordered]@{
	'APP_URL'         = $url
	'PORTAL_BASE_URL' = $url
	'SITE_DOMAIN'     = $host_only
}

$lines = [System.Collections.ArrayList]@(Get-Content $envPath)
foreach ($key in $wanted.Keys) {
	$value = $wanted[$key]
	$found = $false
	for ($i = 0; $i -lt $lines.Count; $i++) {
		if ($lines[$i] -match "^\s*$key=") {
			$lines[$i] = "$key=$value"
			$found = $true
			break
		}
	}
	if (-not $found) { [void]$lines.Add("$key=$value") }
	Write-Host "    $key=$value"
}

$utf8NoBom = New-Object System.Text.UTF8Encoding($false)
[System.IO.File]::WriteAllLines($envPath, $lines, $utf8NoBom)

# API membaca .env saat start, jadi perubahan barusan baru berlaku setelah
# kontainernya dibuat ulang.
Write-Host "==> Memuat ulang API" -ForegroundColor Green
docker compose @composeArgs up -d --force-recreate api
if (-not $?) { Write-Error "gagal membuat ulang api" }

Write-Host "==> Memastikan aplikasi benar-benar melayani" -ForegroundColor Green
$ok = $false
for ($i = 1; $i -le 20; $i++) {
	try {
		$r = Invoke-WebRequest -Uri "$url/api/v1/packages" -TimeoutSec 10 -UseBasicParsing
		if ($r.StatusCode -eq 200) { $ok = $true; break }
	} catch { }
	Start-Sleep -Seconds 5
}

Write-Host ""
if ($ok) {
	Write-Host "  SIAP -> $url" -ForegroundColor Green
} else {
	Write-Host "  Tunnel hidup tapi API belum menjawab 200." -ForegroundColor Yellow
	Write-Host "  Periksa: docker compose $($composeArgs -join ' ') logs api --tail=30" -ForegroundColor Yellow
}
Write-Host ""
Write-Host "  URL ini berubah setiap cloudflared restart. Jalankan skrip ini lagi"
Write-Host "  untuk mendapatkan yang baru sekaligus menyelaraskan .env."
