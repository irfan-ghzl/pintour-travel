# Cara Menjalankan

Ada empat cara menjalankan sistem ini, dan yang membedakan bukan aplikasinya —
melainkan **di mana ia berjalan** dan **siapa yang bisa menjangkaunya**.

| Mode | Jalan di | Bisa dijangkau | Untuk |
| ---- | -------- | -------------- | ----- |
| [1. Docker penuh](#1-docker-penuh) | laptop | laptop ini saja | pemakaian sehari-hari |
| [2. Hybrid](#2-hybrid--untuk-mengubah-kode) | laptop | laptop ini saja | mengubah kode Go/React |
| [3. Tunnel](#3-url-publik-sementara) | laptop | internet | demo, sidang, uji webhook |
| [4. Server](#4-produksi) | server | internet | produksi |

nginx ada di keempatnya — dialah yang menyajikan React dan mem-proxy `/api/` ke
Go API. Yang berganti hanya siapa yang membukakan pintu dari luar.

## Prasyarat

- **Docker Desktop** — satu-satunya yang wajib untuk mode 1 dan 3.
- **Go 1.25** dan **Node 22** — hanya untuk mode 2.
- **`.env`** di root. Salin kerangkanya:

```bash
cp .env.example .env
```

---

## 1. Docker penuh

Cara yang dipakai untuk hampir semua hal.

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d --build
```

Atau lewat Makefile, yang sudah memuat kedua berkas itu:

```bash
make docker-up
```

**Kedua berkas compose itu wajib, dan bukan sekadar kebiasaan.** Berkas dasar
memaksa `APP_ENV=production` supaya `Validate()` benar-benar berjalan di server
sungguhan. Menjalankannya sendirian di laptop membuat API **menolak start** —
karena `PORTAL_BASE_URL` memang `localhost` dan Midtrans memang sandbox, dan
keduanya benar untuk laptop. Overlay dev yang menurunkannya kembali ke
`development` sekaligus membuka port basis data.

Setelah naik:

| Alamat                                          | Isi                    |
| ----------------------------------------------- | ---------------------- |
| <http://localhost>                               | aplikasi (nginx)       |
| <http://localhost:8080>                          | API langsung, tanpa nginx |
| <http://localhost:8080/swagger/index.html>       | Swagger UI             |
| `localhost:5432`                                 | Postgres (psql, test kontrak) |
| `localhost:6379`                                 | Redis                  |

Migrasi diterapkan sendiri oleh API saat start, setiap kali, dan API berhenti
bila gagal. Tidak ada langkah manual.

Login admin bawaan: **admin@pintour.com** / **admin123**.

Kalau database masih kosong:

```bash
go run ./cmd/seed
```

Mematikan — `make docker-down`. Data tetap aman; volume tidak ikut terhapus.

---

## 2. Hybrid — untuk mengubah kode

Postgres dan Redis di Docker, Go dan React langsung di laptop. Dipakai saat
sedang mengubah kode, karena tidak perlu membangun ulang image setiap kali.

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d db redis tesseract-api
```

Lalu di dua terminal terpisah:

```bash
make run
```

```bash
make web-dev
```

API di `:8080`, Vite dev server di <http://localhost:3000> dengan hot reload.
Vite mem-proxy `/api` ke `:8080`, jadi frontend tetap melihat satu origin persis
seperti di belakang nginx.

Binary Go membaca `.env` sendiri, jadi tidak perlu meng-export apa pun.

---

## 3. URL publik sementara

Untuk demo, sidang, dan menguji webhook. Fonnte dan Midtrans mengirim ke alamat
publik — selama stack hanya hidup di `127.0.0.1`, keduanya tidak punya jalan
masuk sama sekali.

```bash
powershell -File scripts/quick-tunnel.ps1
```

Skrip itu menyalakan stack beserta Cloudflare quick tunnel, menunggu URL-nya
muncul, menulisnya ke `.env` (`APP_URL`, `PORTAL_BASE_URL`, `SITE_DOMAIN`),
membuat ulang API supaya membaca nilai baru, lalu memanggil aplikasinya untuk
memastikan benar-benar melayani.

Tidak butuh akun Cloudflare dan tidak butuh domain. Imbalannya: **URL berubah
setiap cloudflared restart**. Jalankan skripnya lagi untuk mendapat yang baru
sekaligus menyelaraskan `.env`.

Mematikan:

```bash
docker compose -f docker-compose.yml -f docker-compose.quicktunnel.yml down
```

> Mode ini menurunkan `APP_ENV` ke `development` dengan sengaja, supaya Midtrans
> tetap sandbox selama demo. Konsekuensinya cookie tidak diberi flag `Secure`.
> Itu sebabnya ini jembatan sementara, bukan tempat aplikasi ini tinggal.

Sudah punya domain di Cloudflare? Pakai named tunnel yang URL-nya tetap —
`docker-compose.tunnel.yml`, isi `TUNNEL_TOKEN` di `.env`.

---

## 4. Produksi

Jalan di server, bukan di laptop, dan dinyalakan oleh
[`.github/workflows/deploy.yml`](../.github/workflows/deploy.yml) setiap merge ke
`main`: test → build image → dorong ke GHCR → SSH → tarik & nyalakan → panggil
aplikasinya untuk memastikan benar-benar melayani.

Runbook lengkap dari nol:

- **[DigitalOcean lewat GitHub Student Pack](deploy-digitalocean.md)** — gratis
  setahun, ini yang disarankan.
- **[AWS EC2](deploy-aws.md)** — kalau memang ingin AWS.

Di server, perintahnya memakai overlay produksi, bukan dev:

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml ps
```

---

## Perintah harian

```bash
make help
```

| Perintah             | Kegunaan                                            |
| -------------------- | --------------------------------------------------- |
| `make docker-up`     | nyalakan semua (overlay dev)                        |
| `make docker-down`   | matikan, data tetap                                 |
| `make docker-logs`   | ikuti log semua service                             |
| `make docker-clean`  | matikan **dan hapus volume** — data hilang          |
| `make rebuild`       | bangun ulang image, data tetap                      |
| `make rebuild-fresh` | **hapus database**, bangun ulang, migrasi, seed admin |
| `make rebuild-api`   | bangun ulang API saja, lebih cepat                  |

---

## Mendemokan sistem

[UAT.md](UAT.md) memuat 15 skenario penerimaan yang mengikuti perjalanan bisnis —
pengunjung, lead, peserta, invoice, portal, keberangkatan — beserta jalur demo 15
menit untuk sidang.

## Test

```bash
make test
```

Coverage sesuai kriteria §21.10:

```bash
make test-cover
```

Dua flag di target itu wajib dan tidak boleh dihapus. Tanpa `-coverpkg`, paket
tanpa berkas test hilang dari penyebut dan angkanya terbaca jauh lebih tinggi
dari yang sebenarnya. Tanpa `-count=1`, hasil test dari cache membawa profil
coverage lama dan angkanya melenceng.

Test kontrak repository **dilewati** tanpa database. Untuk ikut menjalankannya,
nyalakan Postgres (mode 1 atau 2) lalu:

```powershell
$env:TEST_DATABASE_URL = "postgres://pintour:<password>@localhost:5432/pintour_db?sslmode=disable"; go test ./internal/... -count=1
```

Awalan variabel di depan perintah (`VAR=x go test`) adalah sintaks bash dan
tidak dikenali PowerShell. Di bash:

```bash
TEST_DATABASE_URL="postgres://pintour:<password>@localhost:5432/pintour_db?sslmode=disable" go test ./internal/... -count=1
```

Setiap test membuat database sendiri dan menghapusnya lagi, jadi data
pengembangan tidak tersentuh.

---

## Migrasi

Diterapkan otomatis saat API start. Untuk menjalankannya terpisah:

```bash
make migrate
```

Mundur satu langkah — **membuang data**, baca berkas `.down.sql`-nya dulu:

```bash
make migrate-down
```

---

## Kalau ada yang tidak beres

**API menolak start, log berisi "konfigurasi produksi tidak aman".** Compose
dijalankan tanpa overlay dev. Pakai `make docker-up`, atau sertakan
`-f docker-compose.dev.yml`.

**`psql` atau test kontrak tidak bisa connect ke `localhost:5432`.** Sama —
berkas dasar sengaja menutup semua port kecuali web. Overlay dev yang
membukanya.

**Perubahan kode tidak muncul.** Kontainer masih memakai image lama.
`make rebuild-api` untuk backend, `make rebuild` untuk keduanya.

**Unggah dokumen ditolak.** Di atas 13 MB memang ditolak nginx. Di bawah itu tapi
tetap gagal — lihat log API.

**Port 80 sudah dipakai.** Ganti lewat `.env`: `WEB_PORT=8081`.

**Halaman termuat tapi katalog kosong.** Frontend hidup, API tidak. Lihat status
kontainer dan log API.
