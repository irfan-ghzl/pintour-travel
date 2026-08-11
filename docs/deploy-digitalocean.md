# Deploy ke DigitalOcean lewat GitHub Student Pack

Server sungguhan yang selalu hidup, gratis setahun penuh, memakai hak yang sudah
Anda miliki sebagai mahasiswa.

Runbook lengkap dari belum punya apa-apa sampai `git merge` yang menyalakan
aplikasi sendiri. Pipeline-nya sama dengan jalur mana pun:
[`.github/workflows/deploy.yml`](../.github/workflows/deploy.yml) — test → build
image → dorong ke GHCR → SSH ke server → tarik & nyalakan → panggil aplikasinya
untuk memastikan benar-benar melayani.

**Perkiraan waktu:** 40 menit kerja, ditambah menunggu verifikasi Student Pack —
bisa beberapa jam, bisa seminggu. **Ajukan verifikasinya lebih dulu**, kerjakan
yang lain sambil menunggu.

---

## Bagian 0 — Biaya

| Komponen                                | Biaya                        |
| --------------------------------------- | ---------------------------- |
| Droplet 1 GB / 1 vCPU / 25 GB / 1 TB transfer | $6/bulan                |
| Reserved IP (selama menempel di droplet) | $0                          |
| Cloud Firewall                           | $0                          |
| DNS di DigitalOcean                      | $0                          |
| **Kredit Student Pack**                  | **$200, berlaku 1 tahun**    |

$200 dibagi $6 adalah 33 bulan, tapi kreditnya hangus setelah setahun — jadi
yang Anda dapat adalah **satu tahun penuh gratis**, bukan 33 bulan. Setelah itu
$6/bulan, atau hapus droplet-nya kalau sudah tidak dipakai.

DigitalOcean tetap meminta kartu atau PayPal untuk verifikasi meski kredit sudah
masuk. Yang ditagih tetap kreditnya lebih dulu.

Dibanding AWS free tier, jalur ini lebih tenang: satu harga tetap, tidak ada
komponen tersembunyi yang menagih terpisah, dan tidak ada tenggat 12 bulan yang
diam-diam berubah menjadi tagihan.

---

## Bagian 1 — GitHub Student Developer Pack

### Syarat

- Berstatus mahasiswa aktif.
- **Email kampus** (`@...ac.id`) — jalur tercepat. Kalau tidak punya, siapkan
  foto **KTM** atau surat keterangan aktif kuliah yang mencantumkan nama,
  institusi, dan tanggal.

### Langkah

1. Buka <https://education.github.com/pack> → **Sign up for Student Developer
   Pack**.
2. Login dengan akun GitHub yang **sama** dengan yang dipakai repo ini.
3. Tambahkan email kampus di GitHub → Settings → Emails, lalu pilih email itu
   saat verifikasi.
4. Kalau tidak ada email kampus, unggah foto KTM. Pastikan nama, institusi, dan
   tahun akademik terbaca jelas.

### Yang memperbesar peluang lolos sekali jalan

- **Nama di profil GitHub cocok dengan nama di KTM.** Ini penyebab penolakan
  paling umum — pemeriksanya membandingkan keduanya secara harfiah.
- Ajukan **saat berada di kampus atau di kota kampus**; GitHub memakai perkiraan
  lokasi sebagai pertimbangan.
- Foto terang, seluruh kartu masuk bingkai, tidak buram.

Hasilnya bisa datang dalam hitungan jam, bisa juga seminggu. Kalau ditolak,
ajukan ulang dengan dokumen yang lebih jelas — tidak ada batas percobaan.

---

## Bagian 2 — Klaim kredit DigitalOcean

1. Setelah pack aktif, buka <https://education.github.com/pack> → cari
   **DigitalOcean** → **Get access**.
2. Ikuti tautannya, buat akun DigitalOcean baru (penawaran ini untuk pengguna
   baru).
3. Verifikasi dengan kartu atau PayPal.
4. Pastikan kredit $200 sudah muncul di **Billing** sebelum membuat droplet.

Sekalian klaim **Namecheap** di halaman pack yang sama: satu domain `.me` gratis
setahun. Itu menyelesaikan kebutuhan domain di Bagian 6 tanpa biaya.

---

## Bagian 3 — Kunci SSH

Buat di laptop Anda, bukan di server.

### Windows (PowerShell)

```powershell
ssh-keygen -t ed25519 -C "pintour-deploy" -f "$env:USERPROFILE\.ssh\pintour"
```

Kosongkan passphrase — GitHub Actions tidak bisa mengetikkannya saat deploy.

Menyalin kunci **publik** untuk ditempel ke DigitalOcean:

```powershell
Get-Content "$env:USERPROFILE\.ssh\pintour.pub" | Set-Clipboard
```

Berkas `pintour` (tanpa `.pub`) adalah kunci **privat**. Isinya nanti masuk ke
GitHub Secrets, dan tidak boleh ke tempat lain mana pun.

### macOS / Linux

```bash
ssh-keygen -t ed25519 -C "pintour-deploy" -f ~/.ssh/pintour
```

---

## Bagian 4 — Buat droplet

**Create** → **Droplets**.

| Kolom          | Nilai                                        |
| -------------- | -------------------------------------------- |
| Region         | **Singapore (SGP1)** — terdekat dari Indonesia |
| Image          | Ubuntu 24.04 (LTS) x64                       |
| Size           | Basic → Regular → **$6/mo** (1 GB / 1 vCPU / 25 GB) |
| Authentication | **SSH Key** → New SSH Key → tempel isi `pintour.pub` |
| Hostname       | `pintour-prod`                                |

Jangan pilih autentikasi password. Droplet dengan login password akan mulai
menerima percobaan masuk otomatis dalam hitungan menit setelah menyala.

1 GB cukup: stack ini diukur idle di **124 MiB** untuk kelima kontainer, dan
image dibangun di GitHub Actions sehingga server tidak pernah menjalankan
bundling Vite yang berat. Skrip bootstrap menambahkan swap 2 GB untuk lonjakan
Tesseract saat memproses paspor hasil pindai.

---

## Bagian 5 — Firewall dan Reserved IP

### Firewall

**Networking** → **Firewalls** → **Create Firewall**.

| Arah     | Type  | Port | Sumber        |
| -------- | ----- | ---- | ------------- |
| Inbound  | SSH   | 22   | IP Anda       |
| Inbound  | HTTP  | 80   | All IPv4/IPv6 |
| Inbound  | HTTPS | 443  | All IPv4/IPv6 |

Outbound biarkan bawaannya. Terapkan ke droplet `pintour-prod`.

Postgres (5432) dan Redis (6379) **tidak dibuka**. Compose sudah menutupnya
lewat `expose`; firewall adalah lapisan kedua.

### Reserved IP

**Networking** → **Reserved IPs** → tetapkan ke `pintour-prod`.

Gratis selama menempel pada droplet yang aktif, dan membuat alamatnya tidak
berubah saat droplet di-rebuild. Kalau droplet dihapus, lepaskan juga Reserved
IP-nya — yang menganggur ditagih.

---

## Bagian 6 — Domain

Pakai `.me` gratis dari Namecheap (Bagian 2), atau `.my.id` seharga ~Rp 15rb.

Cara termudah: arahkan nameserver domain ke DigitalOcean
(`ns1.digitalocean.com`, `ns2`, `ns3`), lalu **Networking** → **Domains** →
tambahkan domain → buat **A record** `@` yang menunjuk ke Reserved IP.

Domain harus sudah menunjuk ke IP **sebelum** deploy pertama — Let's Encrypt
memverifikasi dengan benar-benar menghubungi domain itu. Periksa dulu:

```bash
nslookup pintour.me
```

Propagasi DNS bisa perlu sampai beberapa jam. Ini alasan lain untuk mengerjakan
bagian ini lebih awal.

---

## Bagian 7 — Masuk lewat SSH

DigitalOcean memberi akses sebagai **`root`**, bukan `ubuntu` seperti AWS.

### Windows

```powershell
ssh -i "$env:USERPROFILE\.ssh\pintour" root@<reserved-ip>
```

Kalau muncul penolakan soal izin berkas yang terlalu longgar:

```powershell
icacls "$env:USERPROFILE\.ssh\pintour" /inheritance:r /grant:r "$($env:USERNAME):R"
```

### macOS / Linux

```bash
chmod 400 ~/.ssh/pintour && ssh -i ~/.ssh/pintour root@<reserved-ip>
```

---

## Bagian 8 — Bootstrap server

Memasang Docker, membuat swap 2 GB, meng-clone repo, menyiapkan `.env` kosong
bermode 600. Aman dijalankan berulang:

```bash
curl -fsSL https://raw.githubusercontent.com/irfan-ghzl/pintour-travel/main/scripts/server-bootstrap.sh | bash
```

Skrip mengenali sendiri bahwa ia dijalankan sebagai `root` dan melewati langkah
grup `docker` — root memang sudah bisa. Karena itu, tidak seperti di AWS, **tidak
perlu SSH ulang** setelah bootstrap di sini.

---

## Bagian 9 — Isi `.env` di server

```bash
nano /srv/pintour/.env
```

Berkas ini **tidak pernah ada di repo** dan tidak pernah lewat GitHub Actions.
Salin kerangkanya dari `.env.example`. Yang wajib benar — API menolak start bila
tidak:

```
SITE_DOMAIN=pintour.me
APP_URL=https://pintour.me
PORTAL_BASE_URL=https://pintour.me
POSTGRES_PASSWORD=<baru>
REDIS_PASSWORD=<baru>
JWT_SECRET=<hasil openssl rand -hex 64>
MIDTRANS_ENV=production
```

### Nilai yang tidak boleh dipakai lagi

Keduanya pernah tertulis di berkas terlacak dan masih terbaca di riwayat repo
publik:

- **`pintour_pass`** sebagai `POSTGRES_PASSWORD` — dari compose lama.
- **Token Fonnte lama.** Rotasi dulu di dashboard Fonnte, dan beri
  `FONNTE_WEBHOOK_TOKEN` nilai yang **berbeda** dari `FONNTE_API_TOKEN`.

Menghapus baris dari berkas tidak menghapusnya dari riwayat. Rotasi adalah
satu-satunya yang benar-benar menutupnya.

---

## Bagian 10 — Secret di GitHub

Repo → **Settings** → **Secrets and variables** → **Actions**:

| Secret            | Nilai                                      |
| ----------------- | ------------------------------------------ |
| `DEPLOY_HOST`     | Reserved IP                                |
| `DEPLOY_USER`     | `root`                                     |
| `DEPLOY_SSH_KEY`  | **isi** berkas `pintour` (yang tanpa `.pub`) |
| `DEPLOY_PATH`     | `/srv/pintour`                             |
| `HEALTHCHECK_URL` | `https://pintour.me`                       |

`DEPLOY_SSH_KEY` diisi isi berkasnya, bukan namanya:

```powershell
Get-Content "$env:USERPROFILE\.ssh\pintour" | Set-Clipboard
```

Tidak ada rahasia aplikasi di daftar ini, dan itu disengaja: Actions tidak perlu
tahu password database untuk menyalakan ulang stack, dan setiap rahasia yang
lewat CI adalah rahasia yang bisa tercetak di log karena satu langkah salah
tulis.

---

## Bagian 11 — Merge dan verifikasi

Merge PR ke `main`. Pantau di tab **Actions**: test → build → deploy →
healthcheck. Deploy pertama paling lama karena belum ada cache build.

Setelah hijau, buka `https://<domain>`. Yang seharusnya terlihat: gembok TLS
terpasang, katalog paket termuat (artinya frontend berhasil memanggil API), dan
login admin berfungsi.

Workflow berhenti merah bila aplikasinya tidak menjawab 200.

---

## Bagian 12 — Operasi sehari-hari

Dari `/srv/pintour`. Buat pintasannya sekali:

```bash
echo "alias dc='docker compose -f docker-compose.yml -f docker-compose.prod.yml'" >> ~/.bashrc
```

Keadaan — `dc ps`. Log API, tempat kegagalan konfigurasi muncul lengkap dengan
sebabnya — `dc logs api --tail=50`.

**Rollback.** Setiap image ditandai SHA commit, jadi kembali ke versi sebelumnya
adalah memilih nama, bukan membangun ulang:

```bash
TAG=<sha-commit-lama> docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

Migrasi tidak ikut mundur — skema yang sudah maju tetap di depan.

**Cadangkan sebelum apa pun yang berisiko**, terutama sebelum sidang:

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml exec -T db pg_dump -U pintour pintour_db > ~/backup-$(date +%F).sql
```

DigitalOcean juga menawarkan **Snapshots** ($0,06/GB/bulan) dan **Backups**
otomatis (20% harga droplet). Untuk menjelang sidang, snapshot manual sekali
sebelum hari H adalah asuransi yang murah.

---

## Bagian 13 — Kalau ada yang tidak beres

**`Permission denied (publickey)`.** Di DigitalOcean user-nya `root`, bukan
`ubuntu`. Atau izin kunci privat terlalu longgar — lihat Bagian 7.

**SSH menggantung tanpa pesan.** Cloud Firewall tidak mengizinkan port 22 dari IP
Anda sekarang. IP rumah berubah; perbarui aturannya.

**Deploy gagal saat `docker compose pull`.** Package GHCR privat dan token tidak
berhak membacanya. Job `deploy` sudah punya `permissions: packages: read`; kalau
masih gagal, ubah visibilitas package ke publik lewat halaman Packages di GitHub.

**Sertifikat TLS tidak terbit.** Domain belum menunjuk ke IP saat Caddy mencoba,
atau port 80 tertutup. Let's Encrypt memverifikasi lewat HTTP di port 80 —
menutupnya menghalangi penerbitan sertifikat, bukan mengamankannya. Periksa
`dc logs caddy`.

**API restart terus.** Hampir selalu `Validate()` menolak konfigurasi. `dc logs
api` menyebutkan persis variabel mana dan kenapa — misalnya `PORTAL_BASE_URL`
masih `localhost`, atau `MIDTRANS_SERVER_KEY` terisi sementara `MIDTRANS_ENV`
masih `sandbox`.

**Halaman termuat tapi katalog kosong.** Frontend hidup, API tidak. `dc ps` lalu
`dc logs api`.

**Kredit habis lebih cepat dari perkiraan.** Cek Reserved IP yang tidak menempel
di droplet mana pun, snapshot yang menumpuk, dan droplet kedua yang lupa dihapus.
