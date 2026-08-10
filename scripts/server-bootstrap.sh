#!/usr/bin/env bash
#
# Menyiapkan EC2 kosong sampai siap menerima deploy dari GitHub Actions.
#
#   curl -fsSL https://raw.githubusercontent.com/irfan-ghzl/pintour-travel/main/scripts/server-bootstrap.sh | sudo bash
#
# Aman dijalankan berulang: setiap langkah memeriksa keadaan lebih dulu, jadi
# menjalankannya dua kali tidak menggandakan swap atau menimpa .env yang sudah
# terisi.
#
# Yang TIDAK dilakukan skrip ini, dan memang tidak boleh: mengisi .env. Rahasia
# diketik langsung di server oleh yang berhak, bukan diturunkan dari skrip yang
# tersimpan di repo publik.

set -euo pipefail

REPO="${REPO:-https://github.com/irfan-ghzl/pintour-travel.git}"
APP_DIR="${APP_DIR:-/srv/pintour}"
DEPLOY_USER="${DEPLOY_USER:-${SUDO_USER:-ubuntu}}"

log() { printf '\n\033[1;32m==>\033[0m %s\n' "$1"; }

if [ "$(id -u)" -ne 0 ]; then
	echo "Jalankan dengan sudo." >&2
	exit 1
fi

# ── Docker ───────────────────────────────────────────────────────────────────
if command -v docker >/dev/null 2>&1; then
	log "Docker sudah ada — dilewati"
else
	log "Memasang Docker"
	curl -fsSL https://get.docker.com | sh
fi

# Supaya user deploy bisa menjalankan docker tanpa sudo. Tanpa ini langkah SSH
# di workflow gagal dengan permission denied pada docker.sock — kegagalan yang
# membingungkan karena perintahnya sendiri benar.
if ! id -nG "$DEPLOY_USER" | grep -qw docker; then
	log "Menambahkan $DEPLOY_USER ke grup docker"
	usermod -aG docker "$DEPLOY_USER"
	echo "    (perlu login SSH baru sebelum berlaku)"
fi

systemctl enable --now docker

# ── Swap ─────────────────────────────────────────────────────────────────────
# t3.micro punya 1 GB. Stack idle hanya ~125 MB, tapi Tesseract melonjak
# 200–400 MB saat memproses paspor hasil pindai. Tanpa swap, lonjakan itu
# membunuh proses yang kebetulan paling besar — biasanya Postgres.
if swapon --show | grep -q '/swapfile'; then
	log "Swap sudah aktif — dilewati"
else
	log "Membuat swap 2 GB"
	fallocate -l 2G /swapfile
	chmod 600 /swapfile
	mkswap /swapfile
	swapon /swapfile
	grep -q '^/swapfile' /etc/fstab || echo '/swapfile none swap sw 0 0' >>/etc/fstab
fi

# ── Repo ─────────────────────────────────────────────────────────────────────
if [ -d "$APP_DIR/.git" ]; then
	log "Repo sudah ada di $APP_DIR — menarik yang terbaru"
	git -C "$APP_DIR" fetch --depth 1 origin main
	git -C "$APP_DIR" reset --hard origin/main
else
	log "Meng-clone repo ke $APP_DIR"
	mkdir -p "$(dirname "$APP_DIR")"
	git clone --depth 1 "$REPO" "$APP_DIR"
fi
chown -R "$DEPLOY_USER":"$DEPLOY_USER" "$APP_DIR"

# ── .env ─────────────────────────────────────────────────────────────────────
# Dibuat kosong dan hanya bisa dibaca pemiliknya. Bawaan umask membuat berkas
# terbaca setiap user di mesin ini; di VPS itu berarti terbaca proses lain.
if [ -s "$APP_DIR/.env" ]; then
	log ".env sudah terisi — tidak disentuh"
else
	log "Membuat .env kosong (600) — isi manual setelah ini"
	install -o "$DEPLOY_USER" -g "$DEPLOY_USER" -m 600 /dev/null "$APP_DIR/.env"
fi

log "Selesai. Yang tersisa, dikerjakan manual:"
cat <<'SISA'

  1. Isi rahasianya:  nano /srv/pintour/.env
     Wajib, dan API menolak start bila keliru:
       SITE_DOMAIN=<domain yang sudah menunjuk ke IP ini>
       APP_URL=https://<domain>          (bukan localhost)
       PORTAL_BASE_URL=https://<domain>  (bukan localhost)
       POSTGRES_PASSWORD=<baru — JANGAN pintour_pass, nilai itu ada di riwayat repo>
       REDIS_PASSWORD=<baru>
       JWT_SECRET=<acak panjang: openssl rand -hex 64>
       FONNTE_API_TOKEN=<token hasil rotasi, yang lama sudah bocor>
       MIDTRANS_ENV=production  (atau kosongkan MIDTRANS_SERVER_KEY)

  2. Isi lima secret di GitHub → Settings → Secrets and variables → Actions:
       DEPLOY_HOST DEPLOY_USER DEPLOY_SSH_KEY DEPLOY_PATH HEALTHCHECK_URL

  3. Merge ke main. Sisanya otomatis.

SISA
