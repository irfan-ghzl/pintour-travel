# Deploy ke AWS EC2

Dari akun AWS kosong sampai `git merge` yang menyalakan aplikasi sendiri.

Pipeline-nya ada di [`.github/workflows/deploy.yml`](../.github/workflows/deploy.yml):
test → build image → dorong ke GHCR → SSH ke server → tarik & nyalakan →
panggil aplikasinya untuk memastikan benar-benar melayani. Server tidak pernah
membangun apa pun; ia hanya menarik image jadi.

## Kenapa EC2 polos, bukan ECS

Compose di repo ini sudah terbukti jalan. ECS/Fargate berarti membongkarnya
menjadi task definition, memindahkan Postgres ke RDS dan Redis ke ElastiCache —
ElastiCache tidak punya free tier, RDS free tier habis dalam 12 bulan, dan
hasil akhirnya sama saja. Menukar sesuatu yang sudah bekerja dengan sesuatu yang
harus dibuktikan ulang adalah biaya tanpa imbalan.

## Ukuran instance

Stack ini diukur saat idle: **124 MiB** untuk kelima kontainer.

| Kontainer | Memori idle |
| --------- | ----------- |
| tesseract | 71 MiB      |
| postgres  | 28 MiB      |
| redis     | 10 MiB      |
| api (Go)  | 10 MiB      |
| nginx     | 5 MiB       |

`t3.micro` (1 GB, free-tier eligible) cukup — sebagian besar karena image
dibangun di GitHub Actions. Kalau bundling Vite dijalankan di server, 1 GB
langsung habis.

Yang perlu diantisipasi cuma OCR: Tesseract melonjak 200–400 MB sesaat saat
memproses paspor 2–5 MB. Skrip bootstrap membuat swap 2 GB untuk itu.

## Biaya sebenarnya

Dua hal yang sering mengagetkan:

- **IPv4 publik berbayar sejak 2024**, sekitar $0,005/jam (~$3,6/bulan),
  termasuk saat instance-nya free tier. Jadi "gratis" tetap menghasilkan
  tagihan kecil.
- **Model free tier berubah di 2025.** Akun baru cenderung mendapat kredit
  berjangka alih-alih pola lama "12 bulan t2.micro". Periksa halaman billing
  akun sendiri; jangan merencanakan dari asumsi.

Setelah free tier habis: t3.micro ~$7,5 + IPv4 ~$3,6 ≈ **$11/bulan**.

**Pasang billing alarm sebelum menyalakan apa pun** — Billing → Budgets, set
$5, notifikasi ke email. Ini nasihat yang paling sering diabaikan dan paling
mahal akibatnya.

---

## 1. Instance

EC2 → Launch instance:

| Kolom       | Nilai                              |
| ----------- | ---------------------------------- |
| AMI         | Ubuntu Server 24.04 LTS (64-bit x86) |
| Tipe        | `t3.micro`                         |
| Key pair    | buat baru, unduh `.pem`, simpan    |
| Storage     | 20 GB gp3                          |

Berkas `.pem` itu satu-satunya kunci masuk dan AWS tidak menyimpan salinannya.

## 2. Security group

| Port | Sumber       | Untuk               |
| ---- | ------------ | ------------------- |
| 22   | IP Anda saja | SSH                 |
| 80   | 0.0.0.0/0    | HTTP → redirect TLS |
| 443  | 0.0.0.0/0    | HTTPS               |

Postgres (5432) dan Redis (6379) **tidak dibuka**. Compose sudah menutupnya;
security group adalah lapisan kedua, dan dua lapisan yang keliru dibuka
bersamaan jauh lebih jarang daripada satu.

## 3. Elastic IP

Alokasikan Elastic IP dan kaitkan ke instance. Tanpa ini, IP publik berubah
setiap kali instance di-stop/start — dan setiap perubahan mematahkan DNS,
sertifikat, serta `DEPLOY_HOST` sekaligus.

## 4. Domain

Arahkan A record ke Elastic IP. Harus sudah menunjuk **sebelum** deploy
pertama: Let's Encrypt memverifikasi dengan benar-benar menghubungi domain itu.

Belum punya domain? Pakai `sslip.io` — hostname yang menyelesaikan dirinya ke IP
yang tertulis di dalamnya, dan Let's Encrypt mau menerbitkan sertifikat
untuknya. Untuk Elastic IP `203.0.113.45`:

```
SITE_DOMAIN=203-0-113-45.sslip.io
```

Cukup untuk demo dan sidang. Menukarnya ke domain sungguhan nanti hanya
mengubah satu baris di `.env` dan satu deploy ulang.

## 5. Bootstrap

```bash
ssh -i pintour.pem ubuntu@<elastic-ip>
```

Lalu di server — memasang Docker, membuat swap, meng-clone repo, menyiapkan
`.env` kosong dengan mode 600. Aman diulang:

```bash
curl -fsSL https://raw.githubusercontent.com/irfan-ghzl/pintour-travel/main/scripts/server-bootstrap.sh | sudo bash
```

Keluar dan SSH lagi setelahnya — keanggotaan grup `docker` baru berlaku di sesi
baru.

## 6. Isi `.env` di server

```bash
nano /srv/pintour/.env
```

Berkas ini **tidak pernah ada di repo** dan tidak pernah lewat GitHub Actions.
Ketik langsung di server. Salin kerangkanya dari `.env.example`, lalu pastikan
yang berikut benar — `Validate()` menolak start bila tidak:

```
SITE_DOMAIN=<domain dari langkah 4>
APP_URL=https://<domain>
PORTAL_BASE_URL=https://<domain>
POSTGRES_PASSWORD=<baru>
REDIS_PASSWORD=<baru>
JWT_SECRET=<openssl rand -hex 64>
MIDTRANS_ENV=production
```

Nilai yang **tidak boleh dipakai lagi**, karena ada di riwayat repo publik:

- `pintour_pass` sebagai `POSTGRES_PASSWORD`
- token Fonnte lama — rotasi di dashboard Fonnte lebih dulu, dan beri
  `FONNTE_WEBHOOK_TOKEN` nilai yang berbeda dari `FONNTE_API_TOKEN`

## 7. Secret di GitHub

Settings → Secrets and variables → Actions:

| Secret            | Nilai                                       |
| ----------------- | ------------------------------------------- |
| `DEPLOY_HOST`     | Elastic IP                                  |
| `DEPLOY_USER`     | `ubuntu`                                    |
| `DEPLOY_SSH_KEY`  | isi `.pem` utuh, termasuk baris BEGIN/END   |
| `DEPLOY_PATH`     | `/srv/pintour`                              |
| `HEALTHCHECK_URL` | `https://<domain>`                          |

Tidak ada rahasia aplikasi di sini, dan itu disengaja: Actions tidak perlu tahu
password database untuk bisa menyalakan ulang stack, dan setiap rahasia yang
lewat CI adalah rahasia yang bisa tercetak di log karena satu langkah salah
tulis.

## 8. Merge ke main

Sisanya otomatis. Workflow berhenti merah bila aplikasinya tidak menjawab 200 —
deploy yang hijau padahal situsnya mati adalah kegagalan yang paling mahal,
karena tidak ada yang tahu sampai ada yang membukanya.

---

## Memeriksa dan memperbaiki

Melihat keadaan:

```bash
cd /srv/pintour && docker compose -f docker-compose.yml -f docker-compose.prod.yml ps
```

Log API — di sinilah kegagalan konfigurasi muncul, lengkap dengan sebabnya:

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml logs api --tail=50
```

Kembali ke versi sebelumnya. Setiap image ditandai SHA commit, jadi rollback
adalah memilih nama, bukan membangun ulang:

```bash
TAG=<sha-lama> docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

Migrasi tidak perlu dijalankan manual: API menerapkannya sendiri saat start dan
berhenti bila gagal. Perlu diingat migrasi tidak ikut mundur saat rollback —
skema yang sudah maju tetap di depan, dan itulah sebabnya migrasi di proyek ini
ditulis agar versi kode sebelumnya masih bisa berjalan di atasnya.
