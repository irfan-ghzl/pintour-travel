# Deploy ke AWS — dari akun kosong sampai aplikasi hidup

Runbook lengkap. Diikuti berurutan, dari belum punya akun AWS sampai `git merge`
yang menyalakan aplikasi sendiri.

Pipeline-nya ada di [`.github/workflows/deploy.yml`](../.github/workflows/deploy.yml):
test → build image → dorong ke GHCR → SSH ke server → tarik & nyalakan →
panggil aplikasinya untuk memastikan benar-benar melayani. Server tidak pernah
membangun apa pun; ia hanya menarik image jadi.

**Perkiraan waktu:** 60–90 menit, sebagian besar menunggu verifikasi akun dan
propagasi DNS.

---

## Bagian 0 — Sebelum mulai

### Yang perlu disiapkan

- **Kartu kredit atau debit berlogo Visa/Mastercard.** AWS mewajibkannya meski
  Anda hanya memakai free tier, dan akan menahan sekitar $1 untuk verifikasi —
  dikembalikan otomatis. Kartu debit bank Indonesia umumnya diterima selama
  mendukung transaksi internasional.
- **Nomor HP aktif** untuk verifikasi SMS/telepon.
- **Akses ke repo GitHub** ini dengan hak mengatur Settings.

### Biaya sebenarnya — baca ini dulu

Tiga hal yang sering mengagetkan:

- **IPv4 publik berbayar sejak Februari 2024**, sekitar $0,005/jam (~$3,6/bulan),
  **termasuk saat instance-nya free tier**. Jadi "gratis" tetap menghasilkan
  tagihan kecil. Ini bukan kesalahan konfigurasi Anda.
- **Model free tier berubah di 2025.** Akun baru cenderung mendapat kredit
  berjangka (sekitar $100, plus tambahan dari aktivitas onboarding) alih-alih
  pola lama "12 bulan t2.micro gratis". Akun lama umumnya masih di pola lama.
  Periksa halaman billing akun Anda sendiri — ini berubah cukup sering.
- **Setelah free tier habis**: t3.micro ~$7,5 + IPv4 ~$3,6 ≈ **$11/bulan**.

Kalau yang Anda inginkan adalah tagihan yang pasti dan tidak bisa meledak,
**Lightsail** lebih tenang: $5–12/bulan tetap, sudah termasuk IP publik dan kuota
transfer. Tetap hanya sebuah VM dengan Docker, jadi seluruh runbook ini berlaku
mulai Bagian 5 — hanya cara membuat mesinnya yang berbeda.

### Kenapa EC2 polos, bukan ECS/Fargate

Compose di repo ini sudah terbukti jalan. ECS berarti membongkarnya menjadi task
definition, memindahkan Postgres ke RDS dan Redis ke ElastiCache — ElastiCache
tidak punya free tier, RDS free tier habis dalam 12 bulan, dan hasil akhirnya
sama saja. Menukar sesuatu yang sudah bekerja dengan sesuatu yang harus
dibuktikan ulang adalah biaya tanpa imbalan.

### Kenapa `t3.micro` cukup

Stack ini diukur saat idle: **124 MiB** untuk kelima kontainer.

| Kontainer | Memori idle |
| --------- | ----------- |
| tesseract | 71 MiB      |
| postgres  | 28 MiB      |
| redis     | 10 MiB      |
| api (Go)  | 10 MiB      |
| nginx     | 5 MiB       |

1 GB cukup — sebagian besar karena image dibangun di GitHub Actions. Kalau
bundling Vite dijalankan di server, 1 GB langsung habis dan gagalnya muncul saat
deploy, bukan saat build.

Yang perlu diantisipasi hanya OCR: Tesseract melonjak 200–400 MB sesaat saat
memproses paspor 2–5 MB. Skrip bootstrap membuat swap 2 GB untuk itu.

---

## Bagian 1 — Buat akun AWS

1. Buka <https://portal.aws.amazon.com/billing/signup>.
2. Isi email, nama akun (mis. `pintour-skripsi`), dan password. **Email ini
   menjadi akun root** — pakai email yang tidak akan hilang aksesnya.
3. Verifikasi email dengan kode yang dikirim.
4. Isi data kontak. Pilih **Personal** kecuali ini memang atas nama badan usaha.
5. Isi data kartu. Penahanan ~$1 akan muncul dan hilang sendiri.
6. Verifikasi identitas lewat SMS atau telepon.
7. Pilih support plan: **Basic — Free**. Yang berbayar tidak dibutuhkan.

Aktivasi kadang butuh beberapa menit sampai beberapa jam. Anda akan menerima
email saat akun siap.

---

## Bagian 2 — Amankan akun root

Kerjakan sebelum menyalakan apa pun. Akun root bisa melakukan segalanya,
termasuk yang tidak bisa dibatalkan, dan satu-satunya hal yang berdiri di
depannya hanyalah password.

### 2.1 Nyalakan MFA di akun root

1. Login sebagai root → klik nama akun kanan atas → **Security credentials**.
2. **Multi-factor authentication (MFA)** → **Assign MFA device**.
3. Pilih **Authenticator app**, pindai QR dengan Google Authenticator atau
   Authy, masukkan dua kode berurutan.

Akun AWS tanpa MFA yang kredensialnya bocor adalah cerita yang berulang, dan
tagihannya jatuh ke pemilik akun.

### 2.2 Buat user IAM untuk pemakaian sehari-hari

Jangan pakai root untuk pekerjaan biasa.

1. **IAM** → **Users** → **Create user**.
2. Nama `pintour-admin`, centang **Provide user access to the AWS Management
   Console**.
3. **Attach policies directly** → `AdministratorAccess`.
4. Simpan URL login khusus akun ini (`https://<account-id>.signin.aws.amazon.com/console`).
5. Nyalakan MFA untuk user ini juga.

Setelah ini, login pakai `pintour-admin`. Root disimpan untuk keadaan yang
benar-benar membutuhkannya — mengubah data billing, menutup akun.

---

## Bagian 3 — Billing alarm

**Kerjakan sekarang, sebelum instance pertama.** Ini nasihat AWS yang paling
sering diabaikan dan paling mahal akibatnya: tanpa alarm, kesalahan konfigurasi
baru ketahuan saat tagihan datang.

1. **Billing and Cost Management** → **Budgets** → **Create budget**.
2. Pilih **Zero spend budget** (memberi tahu begitu tagihan lewat $0,01) atau
   **Cost budget** dengan batas $5.
3. Masukkan email Anda sebagai penerima notifikasi.

Dua budget pertama gratis.

Sekalian nyalakan **Billing and Cost Management → Billing preferences →
Free tier usage alerts**, supaya Anda diberi tahu saat mendekati batas free
tier, bukan setelah melewatinya.

---

## Bagian 4 — Pilih region

Pilih **sekali** dan jangan berpindah — resource di region berbeda tidak saling
melihat, dan setengah dari kebingungan pengguna baru AWS berasal dari melihat
konsol di region yang salah.

| Region                   | Kode             | Catatan                        |
| ------------------------ | ---------------- | ------------------------------ |
| Asia Pacific (Jakarta)   | `ap-southeast-3` | Latensi terbaik dari Indonesia |
| Asia Pacific (Singapore) | `ap-southeast-1` | Alternatif, pilihan layanan lebih lengkap |

Untuk sidang di Indonesia, Jakarta. Pastikan region di kanan atas konsol sudah
benar sebelum melangkah.

---

## Bagian 5 — Key pair

1. **EC2** → **Key pairs** → **Create key pair**.
2. Nama `pintour`, tipe **RSA**, format **.pem**.
3. Berkas terunduh otomatis. **AWS tidak menyimpan salinannya** — hilang berarti
   Anda kehilangan satu-satunya jalan masuk ke server.

Simpan di tempat yang tidak ikut ter-commit. Jangan pernah taruh di dalam folder
repo — meski `.gitignore` sudah memblok `*.pem`, berkas kunci tidak punya urusan
di sana.

---

## Bagian 6 — Luncurkan instance

**EC2** → **Instances** → **Launch instances**.

| Kolom             | Nilai                                  |
| ----------------- | -------------------------------------- |
| Name              | `pintour-prod`                          |
| AMI               | Ubuntu Server 24.04 LTS (64-bit x86)   |
| Instance type     | `t3.micro`                              |
| Key pair          | `pintour` (dari Bagian 5)               |
| Storage           | 20 GB gp3                              |

20 GB dipilih agar tetap di bawah batas free tier EBS 30 GB, sambil menyisakan
ruang untuk data Postgres dan image Docker.

Pastikan tipe instance-nya bertanda **Free tier eligible**. Di sebagian region
yang gratis adalah `t2.micro`, bukan `t3.micro` — ikuti label yang ditampilkan
konsol, bukan tulisan di sini.

---

## Bagian 7 — Security group

Saat launch, pilih **Create security group** dengan aturan berikut:

| Type       | Port | Source        | Untuk                  |
| ---------- | ---- | ------------- | ---------------------- |
| SSH        | 22   | My IP         | akses Anda             |
| HTTP       | 80   | Anywhere      | redirect ke HTTPS      |
| HTTPS      | 443  | Anywhere      | aplikasi               |

Postgres (5432) dan Redis (6379) **tidak dibuka sama sekali**. Compose sudah
menutupnya lewat `expose`; security group adalah lapisan kedua. Dua lapisan yang
keliru dibuka bersamaan jauh lebih jarang terjadi daripada satu.

SSH dibatasi ke IP Anda. Kalau IP rumah Anda berubah-ubah dan akses jadi
terputus, ubah aturannya lewat konsol — itu jauh lebih baik daripada membuka
port 22 ke seluruh dunia, yang akan mulai menerima percobaan login otomatis
dalam hitungan menit.

---

## Bagian 8 — Elastic IP

**EC2** → **Elastic IPs** → **Allocate Elastic IP address** → pilih alamatnya →
**Actions** → **Associate** → pilih instance `pintour-prod`.

Tanpa ini, IP publik berubah setiap kali instance di-stop/start — dan satu
perubahan mematahkan DNS, sertifikat TLS, serta `DEPLOY_HOST` sekaligus.

Catat alamatnya. Runbook ini memakai `203.0.113.45` sebagai contoh.

> Elastic IP yang **tidak terpasang** ke instance mana pun tetap ditagih. Kalau
> nanti instance dihapus, lepaskan juga Elastic IP-nya.

---

## Bagian 9 — Domain

Aplikasi ini butuh nama, bukan sekadar IP: sertifikat TLS diterbitkan untuk
nama, dan tautan portal yang dikirim ke peserta memakai nama itu.

### Pilihan A — tanpa beli domain (cukup untuk sidang)

`sslip.io` menyelesaikan hostname menjadi IP yang tertulis di dalamnya, dan
Let's Encrypt mau menerbitkan sertifikat untuknya. Untuk Elastic IP
`203.0.113.45`:

```
203-0-113-45.sslip.io
```

Tanpa daftar, tanpa DNS, langsung bisa dipakai.

### Pilihan B — domain sendiri

`.my.id` sekitar Rp 15rb/tahun di registrar Indonesia. Buat **A record** yang
menunjuk ke Elastic IP.

Apa pun pilihannya, domain **harus sudah menunjuk ke IP sebelum deploy pertama**
— Let's Encrypt memverifikasi dengan benar-benar menghubungi domain itu. Periksa
dulu:

```bash
nslookup 203-0-113-45.sslip.io
```

---

## Bagian 10 — Masuk lewat SSH

### Windows

Berkas `.pem` yang baru diunduh mewarisi izin dari folder Downloads, dan OpenSSH
menolak kunci yang bisa dibaca user lain. Perbaiki dulu, sekali saja:

```powershell
icacls "$env:USERPROFILE\.ssh\pintour.pem" /inheritance:r /grant:r "$($env:USERNAME):R"
```

Lalu:

```powershell
ssh -i "$env:USERPROFILE\.ssh\pintour.pem" ubuntu@203.0.113.45
```

### macOS / Linux

```bash
chmod 400 ~/.ssh/pintour.pem && ssh -i ~/.ssh/pintour.pem ubuntu@203.0.113.45
```

Username-nya `ubuntu` untuk AMI Ubuntu. Amazon Linux memakai `ec2-user` — salah
username memberi pesan `Permission denied (publickey)` yang terlihat seperti
masalah kunci padahal bukan.

---

## Bagian 11 — Bootstrap server

Di dalam SSH tadi. Memasang Docker, membuat swap 2 GB, meng-clone repo,
menyiapkan `.env` kosong bermode 600:

```bash
curl -fsSL https://raw.githubusercontent.com/irfan-ghzl/pintour-travel/main/scripts/server-bootstrap.sh | sudo bash
```

Aman dijalankan berulang — setiap langkah memeriksa keadaan lebih dulu.

Setelah selesai, **keluar dan SSH lagi**. Keanggotaan grup `docker` baru berlaku
di sesi baru, dan tanpa itu langkah deploy gagal dengan `permission denied` pada
`docker.sock` — kegagalan yang membingungkan karena perintahnya sendiri benar.

---

## Bagian 12 — Isi `.env` di server

```bash
nano /srv/pintour/.env
```

Berkas ini **tidak pernah ada di repo** dan tidak pernah lewat GitHub Actions.
Diketik langsung di server. Salin kerangkanya dari `.env.example`.

Yang wajib benar — API menolak start bila tidak:

```
SITE_DOMAIN=203-0-113-45.sslip.io
APP_URL=https://203-0-113-45.sslip.io
PORTAL_BASE_URL=https://203-0-113-45.sslip.io
POSTGRES_PASSWORD=<baru>
REDIS_PASSWORD=<baru>
JWT_SECRET=<hasil openssl rand -hex 64>
MIDTRANS_ENV=production
```

Membuat `JWT_SECRET`:

```bash
openssl rand -hex 64
```

### Nilai yang tidak boleh dipakai lagi

Keduanya pernah tertulis di berkas terlacak dan masih terbaca di riwayat repo
publik:

- **`pintour_pass`** sebagai `POSTGRES_PASSWORD` — dari compose lama.
- **Token Fonnte lama.** Rotasi dulu di dashboard Fonnte. Sekalian beri
  `FONNTE_WEBHOOK_TOKEN` nilai yang **berbeda** dari `FONNTE_API_TOKEN` — saat
  ini keduanya bernilai sama, padahal perannya berbeda.

Menghapus baris dari berkas tidak menghapusnya dari riwayat. Rotasi adalah satu-
satunya yang benar-benar menutup lubangnya.

---

## Bagian 13 — Secret di GitHub

Repo → **Settings** → **Secrets and variables** → **Actions** → **New repository
secret**:

| Secret            | Nilai                                          |
| ----------------- | ---------------------------------------------- |
| `DEPLOY_HOST`     | `203.0.113.45`                                 |
| `DEPLOY_USER`     | `ubuntu`                                       |
| `DEPLOY_SSH_KEY`  | **isi** berkas `.pem`, utuh termasuk baris BEGIN/END |
| `DEPLOY_PATH`     | `/srv/pintour`                                 |
| `HEALTHCHECK_URL` | `https://203-0-113-45.sslip.io`                |

`DEPLOY_SSH_KEY` diisi isi berkasnya, bukan nama berkasnya. Buka `.pem` dengan
editor teks dan salin seluruhnya, termasuk `-----BEGIN` dan `-----END`.

Perhatikan tidak ada satu pun rahasia aplikasi di sini, dan itu disengaja:
Actions tidak perlu tahu password database untuk bisa menyalakan ulang stack,
dan setiap rahasia yang lewat CI adalah rahasia yang bisa tercetak di log karena
satu langkah salah tulis.

---

## Bagian 14 — Merge dan verifikasi

Merge PR ke `main`. Workflow jalan otomatis: test → build → deploy → healthcheck.

Pantau di tab **Actions**. Deploy pertama paling lama karena belum ada cache
build.

Setelah hijau, buka `https://<domain>` di peramban. Yang seharusnya terlihat:

- gembok TLS terpasang (Caddy sudah menerbitkan sertifikat sendiri),
- katalog paket termuat — artinya frontend berhasil memanggil API,
- login admin berfungsi.

Workflow berhenti merah bila aplikasinya tidak menjawab 200. Deploy yang hijau
padahal situsnya mati adalah kegagalan yang paling mahal, karena tidak ada yang
tahu sampai ada yang membukanya.

---

## Bagian 15 — Operasi sehari-hari

Semua dijalankan dari `/srv/pintour` di server. Karena selalu memakai dua berkas
compose, buat pintasannya sekali:

```bash
echo "alias dc='docker compose -f docker-compose.yml -f docker-compose.prod.yml'" >> ~/.bashrc
```

Melihat keadaan — `dc ps`. Log API, tempat kegagalan konfigurasi muncul lengkap
dengan sebabnya — `dc logs api --tail=50`.

**Rollback.** Setiap image ditandai SHA commit, jadi kembali ke versi sebelumnya
adalah memilih nama, bukan membangun ulang:

```bash
TAG=<sha-commit-lama> docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

Migrasi tidak ikut mundur. Skema yang sudah maju tetap di depan — itulah sebabnya
migrasi di proyek ini ditulis agar versi kode sebelumnya masih bisa berjalan di
atasnya.

**Cadangkan database** sebelum apa pun yang berisiko:

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml exec -T db pg_dump -U pintour pintour_db > ~/backup-$(date +%F).sql
```

**Menghemat biaya di sela-sela pemakaian.** Instance yang di-stop tidak ditagih
jam komputasinya; EBS dan Elastic IP tetap ditagih. Data aman karena tersimpan di
volume Docker di atas EBS. Nyalakan lagi lewat konsol, dan stack naik sendiri
karena `restart: unless-stopped`.

---

## Bagian 16 — Kalau ada yang tidak beres

**`Permission denied (publickey)` saat SSH.** Username salah (`ubuntu` untuk
Ubuntu, `ec2-user` untuk Amazon Linux), atau izin `.pem` terlalu longgar — lihat
Bagian 10.

**SSH menggantung tanpa pesan.** Security group tidak mengizinkan port 22 dari IP
Anda sekarang. IP rumah berubah; perbarui aturannya.

**Deploy gagal di langkah SSH, `permission denied` pada `docker.sock`.** Belum
SSH ulang setelah bootstrap. Keanggotaan grup `docker` hanya berlaku di sesi
baru.

**Deploy gagal saat `docker compose pull`.** Package GHCR privat dan token tidak
berhak membacanya. Pastikan job `deploy` punya `permissions: packages: read` —
sudah ada di workflow — atau ubah visibilitas package ke publik lewat halaman
Packages di GitHub.

**Sertifikat TLS tidak terbit.** Domain belum menunjuk ke IP saat Caddy mencoba,
atau port 80 tertutup. Let's Encrypt memverifikasi lewat HTTP di port 80 —
menutupnya menghalangi penerbitan sertifikat, bukan mengamankannya. Periksa
`dc logs caddy`.

**API restart terus.** Hampir selalu `Validate()` menolak konfigurasi. `dc logs
api` menyebutkan persis variabel mana dan kenapa — misalnya `PORTAL_BASE_URL`
masih `localhost`, atau `MIDTRANS_SERVER_KEY` terisi sementara `MIDTRANS_ENV`
masih `sandbox`.

**Halaman termuat tapi katalog kosong.** Frontend hidup, API tidak. `dc ps` untuk
melihat kontainer mana yang mati, lalu `dc logs api`.

**Unggahan dokumen ditolak.** Berkas di atas 13 MB memang ditolak nginx. Di bawah
itu tapi tetap gagal — periksa `dc logs api`.

**Tagihan naik tak terduga.** Cek Elastic IP yang tidak terpakai, dan snapshot
EBS lama. Keduanya ditagih meski tidak dipakai.
