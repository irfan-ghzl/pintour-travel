# Setup External Services

Panduan setup 5 dependency eksternal yang dipakai Pintour Travel:
**Redis**, **JWT_SECRET**, **Fonnte WA Gateway**, **Resend Email**, dan **Supabase**.

Semua opsional — sistem akan _degrade gracefully_ kalau salah satu kosong.

---

## 1. Redis (opsional — caching)

Redis dipakai untuk cache session, rate-limit bucket, dan optimasi query.
Sistem tetap jalan tanpa Redis, hanya saja tidak ada caching.

### 1.1 Pakai Docker (paling cepat)

```powershell
docker run -d --name pintour_redis -p 6379:6379 redis:7-alpine
```

Atau pakai `docker compose up -d redis` (sudah ada di `docker-compose.yml`).

### 1.2 Install lokal (Windows)

1. Download Redis dari [github.com/microsoftarchive/redis/releases](https://github.com/microsoftarchive/redis/releases)
2. Extract & jalankan `redis-server.exe`
3. Verifikasi: `redis-cli ping` → harus return `PONG`

### 1.3 Install lokal (macOS/Linux)

```bash
# macOS
brew install redis && brew services start redis

# Ubuntu/Debian
sudo apt install redis-server && sudo systemctl start redis
```

### 1.4 Set di `.env`

```env
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
```

Kalau pakai password di Redis (production), isi `REDIS_PASSWORD`.

---

## 2. JWT_SECRET

JWT_SECRET = string random yang dipakai untuk signing JWT token (admin login + portal peserta).
**Wajib** unik & rahasia. Jangan commit ke git.

### 2.1 Generate Secret (3 cara)

**Cara A — PowerShell (Windows):**
```powershell
$bytes = New-Object byte[] 64
[Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
($bytes | ForEach-Object { '{0:x2}' -f $_ }) -join ''
```

**Cara B — Bash (Linux/macOS/Git Bash):**
```bash
openssl rand -hex 64
```

**Cara C — Online generator:**
[https://generate-secret.vercel.app/64](https://generate-secret.vercel.app/64)

### 2.2 Aturan Penting

| Aturan | Penjelasan |
|---|---|
| **Minimal 32 karakter** | Lebih panjang = lebih aman |
| **Random, bukan kata sembarang** | Hindari "secret123", "myapp_jwt", dll |
| **Beda per environment** | Dev secret ≠ staging ≠ production |
| **Jangan share di chat/git** | Treat seperti password root |

### 2.3 Set di `.env`

```env
JWT_SECRET=hasil_generate_64_chars_hex
JWT_EXPIRATION_HOURS=8
```

`JWT_EXPIRATION_HOURS=8` sesuai PRD §10.3 (token expire 8 jam).

---

## 3. Fonnte (WhatsApp Gateway)

Dipakai untuk semua notifikasi WhatsApp ke leads/peserta/konsultan
(12 template: welcome, invoice, reminder, dll — lihat PRD §17.1).

### 3.1 Daftar Akun

1. Buka [https://md.fonnte.com/](https://md.fonnte.com/)
2. Klik **"Sign Up"** → daftar dengan email
3. Verifikasi email (cek inbox/spam)

### 3.2 Hubungkan Nomor WhatsApp

1. Di dashboard, klik **"Add Device"** atau **"Connect Device"**
2. Beri nama device (cth: "Pintour Sales")
3. Scan QR code dengan WhatsApp di HP:
   - Buka WhatsApp → ⋮ (3-dot menu) → **Linked Devices** → **Link a Device**
   - Scan QR di layar Fonnte
4. Tunggu status berubah jadi **"Connected"** ✅

> ⚠️ Gunakan nomor WA **bisnis** (bukan personal), karena WA bisa memblokir
> nomor yang kirim pesan masif secara mendadak. Idealnya nomor yang sudah
> diverifikasi sebagai WhatsApp Business.

### 3.3 Copy Token

1. Di dashboard Fonnte, klik device yang sudah connected
2. Tab **"Token"** → copy string panjang itu
3. Format token: alphanumeric ~30 chars (cth: `abc123XyZ456_DEF789`)

### 3.4 Set di `.env`

```env
FONNTE_API_TOKEN=token_yang_dicopy_dari_dashboard
```

### 3.5 Test Pengiriman

Restart server (`go run ./cmd/server`), lalu submit form leads dari frontend.
Cek di:
- Dashboard Fonnte → tab **"Log"** (lihat pesan terkirim)
- Database tabel `wa_notifications` → status harus `sent`

### 3.6 Free Tier Limit

| Item | Limit |
|---|---|
| Pesan per hari | ~1000 (cek dashboard) |
| Rate limit | 1 pesan/detik direkomendasikan |
| Nomor pengirim | 1 device per akun |

PRD §17.2 sudah handle rate limit dengan `time.Sleep(time.Second)` antar
pesan blast — jadi tidak akan kena throttle.

---

## 4. Resend (Email Transaksional)

Dipakai untuk **reset password admin** (PRD §16.3 + FR-USER-04).
Free tier: 3.000 email/bulan, 100 email/hari.

### 4.1 Daftar Akun

1. Buka [https://resend.com](https://resend.com)
2. Klik **"Sign Up"** → daftar dengan email/GitHub
3. Verifikasi email

### 4.2 Verifikasi Domain (penting untuk production)

Untuk **dev**, bisa skip — pakai email default Resend
(`onboarding@resend.dev`) atau email yang sama dengan akun.

Untuk **production** (kirim atas nama `noreply@pintour.app`):

1. Di Resend dashboard → **Domains** → **Add Domain**
2. Masukkan domain (cth: `pintour.app`)
3. Tambah DNS records yang ditampilkan (SPF, DKIM, MX) di registrar domain
4. Tunggu verifikasi (~ beberapa menit)

### 4.3 Generate API Key

1. Resend dashboard → **API Keys** → **Create API Key**
2. Beri nama (cth: "Pintour Dev")
3. Permission: **Full Access** (atau **Sending Access** saja untuk production)
4. Copy key (format: `re_abc123XYZ456...`)
5. ⚠️ Hanya ditampilkan sekali — simpan sekarang juga

### 4.4 Set di `.env`

```env
RESEND_API_KEY=re_xxxxxxxxxxxxxxxxxxxxxx
MAIL_FROM=noreply@pintour.app
```

Kalau belum verify domain, pakai `MAIL_FROM=onboarding@resend.dev`.

### 4.5 Test Pengiriman

1. Restart backend
2. Buka `http://localhost:3000/forgot-password`
3. Submit dengan email yang **terdaftar di tabel users**
4. Cek inbox email tersebut
5. Klik link reset → set password baru → login dengan password baru

Cek juga di **Resend dashboard → Emails** untuk lihat log.

---

## 5. Supabase (Storage + opsional DB)

Dipakai untuk file storage (PRD §16.2):
- `package-images` (public) — foto paket
- `participant-documents` (private) — paspor/KTP/rekening peserta
- `invoices-pdf` (private) — PDF invoice
- `tour-leader-photos` (public) — foto profil tour leader

Kita pakai Postgres lokal/Docker, jadi Supabase di sini **khusus untuk
Storage saja**. Tapi kamu juga bisa pakai Supabase Postgres kalau mau.

### 5.1 Daftar & Buat Project

1. Buka [https://supabase.com](https://supabase.com)
2. **Sign Up** dengan GitHub
3. **New Project**:
   - Name: `pintour-travel`
   - Database Password: generate strong password
   - Region: pilih terdekat (Singapore untuk Indonesia)
   - Pricing Plan: **Free** (500MB DB + 1GB Storage)
4. Tunggu provisioning (~2 menit)

### 5.2 Ambil API Credentials

1. Project dashboard → **Settings** (⚙️) → **API**
2. Copy:
   - **Project URL** (cth: `https://xxxxx.supabase.co`)
   - **service_role** key (di section "Project API keys" — yang **secret**, BUKAN `anon` public)

> ⚠️ `service_role` key punya akses **bypass RLS** — treat seperti password.
> Jangan pernah expose ke frontend.

### 5.3 Buat 4 Bucket Storage

1. Sidebar → **Storage** → **New bucket**
2. Buat 4 bucket berikut:

| Bucket Name | Public? | Use case |
|---|---|---|
| `package-images` | ✅ Public | Foto paket wisata (terlihat di katalog publik) |
| `tour-leader-photos` | ✅ Public | Foto profil tour leader (tampil di briefing) |
| `participant-documents` | ❌ Private | Paspor, KTP, rekening koran peserta |
| `invoices-pdf` | ❌ Private | PDF invoice peserta |

**Cara buat:** klik **New bucket** → masukkan nama → centang "Public bucket"
untuk yang public, biarkan uncheck untuk private.

### 5.4 Set RLS (opsional untuk dev)

Untuk private bucket, default policy menolak semua akses kecuali via
`service_role` key. Backend Go kita pakai `service_role` jadi akses penuh.

Kalau mau strict, tambah policy custom di **Storage → Policies**.

### 5.5 Set di `.env`

```env
SUPABASE_URL=https://xxxxx.supabase.co
SUPABASE_SERVICE_KEY=eyJhbGciOiJIUzI1NiIs.....servicekey.....
```

### 5.6 Test Upload

1. Restart backend
2. Login peserta di `http://localhost:3000/portal/login`
3. Buka **Dokumen Saya** → klik **Upload**
4. Pilih file PDF/JPG (max 5MB)
5. Cek di Supabase dashboard → **Storage → participant-documents** → file harus ada

Kalau gagal, cek log backend untuk error dari API Supabase.

### 5.7 Kalau Supabase Belum Setup

Tidak masalah! Backend akan return error `STORAGE_UNAVAILABLE`, dan
frontend `FileUpload` component otomatis **fallback ke input URL manual**
(peserta upload ke Google Drive lalu paste link).

---

## 6. Checklist Setup Lengkap

Setelah semua di atas, isi `.env`:

```env
SERVER_PORT=8080
APP_ENV=development
APP_URL=http://localhost:3000
PORTAL_BASE_URL=http://localhost:3000

DATABASE_URL=postgres://pintour:pintour_pass@localhost:5432/pintour_db?sslmode=disable

REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

JWT_SECRET=<128 chars hex random>
JWT_EXPIRATION_HOURS=8

FONNTE_API_TOKEN=<token dari md.fonnte.com>

RESEND_API_KEY=re_<key dari resend.com>
MAIL_FROM=noreply@pintour.app

SUPABASE_URL=https://<id>.supabase.co
SUPABASE_SERVICE_KEY=eyJ...<service_role key>
```

Verifikasi semua aktif:

```powershell
go run ./cmd/server
```

Lihat log startup:
```
🚀 Pintour API v2.0 — listening on :8080 (env=development, driver=pgx/v5)
📚 Swagger: http://localhost:8080/swagger/index.html
Scheduler started (gocron): WA jobs + retention cleanup
```

Kalau muncul **warning** seperti:
- `Supabase Storage not configured` → set `SUPABASE_URL` + `SUPABASE_SERVICE_KEY`
- `FONNTE_API_TOKEN not set — WA jobs will skip sends` → set `FONNTE_API_TOKEN`
- `redis not reachable` → start Redis atau biarkan kosong (opsional)

---

## 7. Production Tips

| Item | Dev | Production |
|---|---|---|
| `JWT_SECRET` | Hex random | **Wajib** generate ulang, simpan di secret manager |
| `APP_ENV` | `development` | `production` (cookie jadi Secure-only) |
| `DATABASE_URL` | Lokal docker | Managed Postgres (Supabase/Railway) dengan SSL |
| `FONNTE_API_TOKEN` | Nomor test | Nomor bisnis terverifikasi |
| `RESEND_API_KEY` | Default sender | API key terbatas + domain verified |
| `SUPABASE_*` | Free tier | Pro tier untuk storage > 1GB |
| `JWT_EXPIRATION_HOURS` | 8 (PRD spec) | Pertimbangkan turun jadi 4 untuk security |

---

## 8. Troubleshooting

### Backend tidak start
- `connect ECONNREFUSED 127.0.0.1:5432` → Postgres belum start, jalankan `docker compose up -d db`
- `pq: SSL is not enabled on the server` → tambah `?sslmode=disable` di `DATABASE_URL`
- `invalid signing method` → JWT_SECRET berubah, hapus cookie `pintour_session` di browser

### Frontend tidak bisa login
- Cookie tidak ke-set → cek `withCredentials: true` di axios + CORS `AllowCredentials: true`
- 401 Unauthorized → cek `JWT_SECRET` di backend match dengan saat token dibuat

### WA tidak terkirim
- Cek di `wa_notifications` table: kalau `status='pending'` selamanya → `FONNTE_API_TOKEN` kosong
- Kalau `status='failed'` + reason "token tidak valid" → token salah/kadaluarsa
- Kalau `status='sent'` tapi WA tidak muncul → cek device di Fonnte dashboard masih "Connected"

### Upload file gagal
- 503 `STORAGE_UNAVAILABLE` → set `SUPABASE_URL` + `SUPABASE_SERVICE_KEY`
- 400 `ukuran file melebihi 5MB` → file terlalu besar (PRD §19.3 limit)
- Berhasil upload tapi tidak terlihat → cek bucket policy & nama bucket di kode

### Reset password email tidak masuk
- Cek folder spam
- Cek Resend dashboard → Emails: kalau status `delivered` tapi tidak masuk inbox → DNS SPF/DKIM
- Kalau `bounced` → email tidak valid atau domain Resend belum verified

---

_Last updated: Mei 2026_
