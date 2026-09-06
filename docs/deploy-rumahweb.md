# Deploy ke Rumahweb Cloud VPS

Server sungguhan yang selalu hidup, dibayar langsung — tanpa menunggu verifikasi
GitHub Student Pack. Pipeline-nya sama dengan jalur mana pun:
[`.github/workflows/deploy.yml`](../.github/workflows/deploy.yml) — test → build
image → dorong ke GHCR → SSH ke server → tarik & nyalakan → panggil aplikasinya
untuk memastikan benar-benar melayani.

**Perkiraan waktu:** 30 menit kerja. Tidak ada masa tunggu — VPS aktif dalam
hitungan menit setelah dibayar.

---

## Bagian 0 — Biaya, dan apakah Paket S cukup

| Komponen | Spesifikasi | Biaya |
| --- | --- | --- |
| VPS KVM Paket S | 1 vCPU / 1 GB RAM / 20 GB SSD / bandwidth unmetered | Rp30.000/bulan* |
| Domain `.cloud` | 1 tahun, bawaan paket S | Rp0 |
| Firewall | `ufw` di VPS sendiri | Rp0 |

\* Harga promo saat order (diskon 50% dari Rp60.000). **Cek harga
perpanjangan di halaman checkout sebelum membayar** — promo semacam ini di
kebanyakan penyedia biasanya berlaku hanya siklus pertama.

### Apakah 1 GB RAM cukup?

**Ya.** Paket S punya spesifikasi yang **nyaris identik** dengan droplet
DigitalOcean yang sudah dipakai dan diukur langsung
([`deploy-digitalocean.md`](./deploy-digitalocean.md) Bagian 4): sama-sama
1 vCPU / 1 GB RAM. Di sana, stack ini — Postgres, Redis, Tesseract, API, web —
diukur idle hanya **124 MiB**. Yang membuat 1 GB tetap perlu dijaga bukan
beban normal, tapi lonjakan Tesseract saat memproses paspor hasil pindai
(200–400 MB sesaat). Solusinya sama di sini: swap 2 GB, dipasang otomatis oleh
`scripts/server-bootstrap.sh` di Bagian 6.

Paket M (2 GB RAM, Rp120.000/bulan) memberi ruang lebih lega kalau nanti mau
menjalankan beban lebih berat bersamaan (mis. migrasi besar sambil aplikasi
tetap melayani), tapi untuk menjalankan stack ini sendirian, Paket S sudah
terbukti cukup lewat preseden yang sama di repo ini.

---

## Bagian 1 — Pesan VPS

<https://www.rumahweb.com/vps-indonesia/> → geser slider ke **S**.

| Kolom | Nilai |
| --- | --- |
| Availability Zone | Bogor atau Bekasi — keduanya dekat Jakarta, pilih salah satu |
| Sistem Operasi | **Ubuntu 24.04 64bit** |
| Control Panel | **Tanpa Control Panel** (satu-satunya pilihan untuk Paket S — cocok, karena semuanya dikelola lewat Docker/SSH, bukan panel) |

Klik **Deploy Paket S**, lalu selesaikan pembayaran. Tidak ada kolom untuk
menempelkan SSH key di alur ini — beda dari DigitalOcean. Kredensial awal
diambil manual di langkah berikutnya.

---

## Bagian 2 — Ambil kredensial awal

Login **Clientzone** Rumahweb → **VPS** → **Manage** → **Information
Account**. Di situ tertera **IP VPS** dan **password root** awal.

Simpan keduanya sementara — password ini hanya dipakai sekali di Bagian 3
untuk memasang kunci SSH, lalu tidak dipakai lagi.

---

## Bagian 3 — Kunci SSH, lalu matikan login password

Rumahweb memberi akses awal lewat **password**, bukan kunci seperti
DigitalOcean. Login password yang dibiarkan aktif mulai menerima percobaan
masuk otomatis dalam hitungan menit — jadi urutan di bawah ini bukan opsional.

### 1. Buat kunci di laptop (bukan di server)

Kalau `~/.ssh/pintour` **sudah ada** dari percobaan sebelumnya, pakai yang itu —
jangan buat ulang. Kunci lama tetap sah; yang hilang saat OS di-install ulang
adalah `authorized_keys` di server, bukan kunci di laptop. Membuat ulang hanya
memaksa `DEPLOY_SSH_KEY` di GitHub ikut diganti tanpa alasan.

**Windows (PowerShell)**
```powershell
ssh-keygen -t ed25519 -C "pintour-deploy" -f "$env:USERPROFILE\.ssh\pintour"
```
Kosongkan passphrase — GitHub Actions tidak bisa mengetikkannya saat deploy.

**macOS / Linux**
```bash
ssh-keygen -t ed25519 -C "pintour-deploy" -f ~/.ssh/pintour
```

### 2. Masuk sekali pakai password, lalu tempel kunci publik

```bash
ssh root@<ip-vps>
```
Masukkan password dari Bagian 2. Setelah masuk:

```bash
mkdir -p ~/.ssh && chmod 700 ~/.ssh
echo "ISI_PINTOUR.PUB_DI_SINI" >> ~/.ssh/authorized_keys
chmod 600 ~/.ssh/authorized_keys
```

Ambil isi kunci publik dari laptop di jendela terpisah:

```powershell
Get-Content "$env:USERPROFILE\.ssh\pintour.pub" | Set-Clipboard
```

### 3. Uji kunci sebelum menutup pintu password

Dari laptop, **jendela baru** (jangan tutup sesi SSH yang lama dulu):

```bash
ssh -i ~/.ssh/pintour root@<ip-vps>
```

Berhasil masuk tanpa diminta password → lanjut ke langkah 4. Kalau gagal,
`authorized_keys` di langkah 2 salah tempel — perbaiki dulu dari sesi lama
yang masih terbuka.

### 4. Matikan login password

Di sesi manapun yang masih terbuka:

```bash
cp /etc/ssh/sshd_config /root/sshd_config.bak
sed -i 's/^#*PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config /etc/ssh/sshd_config.d/*.conf
sshd -t && systemctl reload ssh
```

`reload` (SIGHUP), bukan `restart` — sshd membaca ulang konfigurasi tanpa
memutus koneksi yang sedang berjalan, jadi kalau ada yang salah pintu masuk
yang sekarang masih terbuka. `sshd -t` di depannya menolak menerapkan
konfigurasi yang tidak sah, yang kalau lolos akan membuat sshd gagal naik.

Image Ubuntu Rumahweb menaruh `PasswordAuthentication yes` di
`/etc/ssh/sshd_config.d/50-cloud-init.conf` — file itu **dibaca lebih dulu**
daripada `sshd_config` utama, dan di OpenSSH nilai pertama yang ketemu yang
menang. Tanpa menyertakan `sshd_config.d/*.conf` di perintah `sed` atas,
`PasswordAuthentication no` di `sshd_config` kalah diam-diam. (Berkas itu ada
walaupun `cloud-init status` menjawab `disabled` — ia ditulis saat image
dibuat, bukan saat boot.)

Di Ubuntu 24.04 `ssh.service` dan `ssh.socket` sama-sama aktif, tapi yang
memegang port 22 adalah listener `sshd -D` dari `ssh.service`. Karena itu nama
unit yang dipakai `ssh`, bukan `sshd` — `sshd` itu nama di
RHEL/CentOS/AlmaLinux, dan salah nama membuat perintah gagal dengan
`Unit sshd.service not found` tanpa menerapkan apa pun.

### 5. Buktikan dari luar, bukan dari berkas

```bash
sshd -T | grep -i passwordauthentication
```

Ini **belum cukup**. `sshd -T` membaca berkas konfigurasi, sementara daemon
yang sedang melayani masih memegang konfigurasi lama sampai di-reload — jadi
perintah ini bisa menjawab `no` sementara server sungguhan masih menerima
password. Yang menentukan adalah metode yang **ditawarkan server** ke koneksi
baru. Dari laptop:

```bash
ssh -o PubkeyAuthentication=no -o BatchMode=yes root@<ip-vps> true
```

Harus dijawab `Permission denied (publickey).` — hanya `publickey` di dalam
kurung. Kalau masih tertulis `(publickey,password)`, reload-nya belum berlaku
dan password masih bisa dipakai orang lain.

Password root tidak berlaku lagi setelah ini. Kalau kunci hilang, pemulihan
lewat **Rescue Mode** di Clientzone — bukan lewat password SSH.

---

## Bagian 4 — Firewall

Rumahweb tidak punya produk firewall terpisah seperti DigitalOcean Cloud
Firewall; aturannya dipasang langsung di VPS lewat `ufw`. Berbeda dari image
Ubuntu kebanyakan, image Rumahweb **tidak membawa `ufw`** — pasang dulu:

```bash
apt-get update -qq && apt-get install -y ufw
```

Urutan di bawah ini tidak boleh dibalik. Membuka 22 **sebelum** mengaktifkan
adalah satu-satunya yang memisahkan firewall yang bekerja dari terkunci di luar
server sendiri:

```bash
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw --force enable
```

`--force` dipakai karena `ufw enable` lewat SSH menampilkan konfirmasi
interaktif yang tidak bisa dijawab dari perintah non-interaktif, dan tanpanya
perintah menggantung.

Kalau ragu, pasang jaring pengaman sebelum mengaktifkan — firewall mati sendiri
setelah 3 menit kalau ternyata kamu terkunci:

```bash
nohup sh -c 'sleep 180; ufw --force disable' >/dev/null 2>&1 &
```

Setelah terbukti masih bisa SSH masuk dari jendela baru, bunuh jaring itu
dengan PID-nya (`kill <pid>`) — **bukan** dengan `pkill -f 'sleep 180'`, karena
pola itu ikut cocok dengan baris perintah shell yang sedang menjalankannya dan
akan membunuh sesi SSH-mu sendiri.

Postgres (5432) dan Redis (6379) **tidak dibuka** — Compose sudah menutupnya
lewat `expose`; ini lapisan kedua.
---

## Bagian 5 — Domain

Paket S sudah termasuk **1 domain `.cloud` gratis setahun** — cukup untuk
kebutuhan TLS di Bagian 8, tidak perlu beli domain terpisah.

**Clientzone** → **Domain** → **Manage Domain** → **DNS Management** → buat
**A record** `@` menunjuk ke IP VPS dari Bagian 2.

Domain harus sudah menunjuk ke IP **sebelum** deploy pertama — Let's Encrypt
memverifikasi dengan benar-benar menghubungi domain itu lewat port 80. Periksa
dulu:

```bash
nslookup pintour.cloud
```

Propagasi DNS bisa perlu sampai beberapa jam.

---

## Bagian 6 — Bootstrap server

Memasang Docker, membuat swap 2 GB, meng-clone repo, menyiapkan `.env` kosong
bermode 600. Skrip yang sama persis dengan jalur DigitalOcean — sudah
mendeteksi sendiri bahwa dijalankan sebagai `root` dan tidak butuh perubahan:

```bash
curl -fsSL https://raw.githubusercontent.com/irfan-ghzl/pintour-travel/main/scripts/server-bootstrap.sh | bash
```

Karena sudah root, tidak perlu SSH ulang setelah bootstrap.

---

## Bagian 7 — Isi `.env` di server

```bash
nano /srv/pintour/.env
```

Berkas ini **tidak pernah ada di repo** dan tidak pernah lewat GitHub Actions.
Salin kerangkanya dari `.env.example`. Yang wajib benar — API menolak start
bila tidak:

```
SITE_DOMAIN=pintour.cloud
APP_URL=https://pintour.cloud
PORTAL_BASE_URL=https://pintour.cloud
POSTGRES_PASSWORD=<baru>
REDIS_PASSWORD=<baru>
JWT_SECRET=<hasil openssl rand -hex 64>
MIDTRANS_ENV=production
```

### Nilai yang tidak boleh dipakai lagi

Sama seperti jalur DigitalOcean — keduanya pernah tertulis di berkas terlacak
dan masih terbaca di riwayat repo publik:

- **`pintour_pass`** sebagai `POSTGRES_PASSWORD`.
- **Token Fonnte lama.** Rotasi dulu di dashboard Fonnte, dan beri
  `FONNTE_WEBHOOK_TOKEN` nilai yang **berbeda** dari `FONNTE_API_TOKEN`.

---

## Bagian 8 — Secret di GitHub

Repo → **Settings** → **Secrets and variables** → **Actions**:

| Secret | Nilai |
| --- | --- |
| `DEPLOY_HOST` | IP VPS |
| `DEPLOY_USER` | `root` |
| `DEPLOY_SSH_KEY` | **isi** berkas `pintour` (yang tanpa `.pub`) |
| `DEPLOY_PATH` | `/srv/pintour` |
| `HEALTHCHECK_URL` | `https://pintour.cloud` |

```powershell
Get-Content "$env:USERPROFILE\.ssh\pintour" | Set-Clipboard
```

---

## Bagian 9 — Merge dan verifikasi

Merge PR ke `main`. Pantau di tab **Actions**: test → build → deploy →
healthcheck. Deploy pertama paling lama karena belum ada cache build.

Setelah hijau, buka `https://pintour.cloud`. Yang seharusnya terlihat: gembok
TLS terpasang, katalog paket termuat, login admin berfungsi.

---

## Bagian 10 — Operasi sehari-hari

Sama persis dengan jalur DigitalOcean — provider tidak memengaruhi bagian ini.
Dari `/srv/pintour`:

```bash
echo "alias dc='docker compose -f docker-compose.yml -f docker-compose.prod.yml'" >> ~/.bashrc
```

Keadaan — `dc ps`. Log API — `dc logs api --tail=50`.

**Cadangkan sebelum apa pun yang berisiko:**
```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml exec -T db pg_dump -U pintour pintour_db > ~/backup-$(date +%F).sql
```

---

## Bagian 11 — Kalau ada yang tidak beres

**`REMOTE HOST IDENTIFICATION HAS CHANGED!` setelah install ulang OS.** Bukan
serangan — install ulang membuat server menghasilkan host key baru, sementara
laptop masih menyimpan yang lama untuk IP yang sama. SSH menolak menyambung
sampai catatan lamanya dibuang:
```bash
ssh-keygen -R <ip-vps>
```

**`Permission denied (publickey)` setelah Bagian 3.** Kunci belum tertempel
benar di `authorized_keys`, atau izin berkas kunci privat di laptop terlalu
longgar:
```powershell
icacls "$env:USERPROFILE\.ssh\pintour" /inheritance:r /grant:r "$($env:USERNAME):R"
```

**Password root masih diminta setelah Bagian 3 langkah 4.** `sshd_config`
belum ter-restart, atau ada baris `PasswordAuthentication yes` kedua yang
menimpa (cek file `Include` di `/etc/ssh/sshd_config.d/`).

**Kehilangan akses total (kunci hilang, tidak ada password lagi).** Clientzone
→ VPS → **Rescue Mode** — bukan minta reset password lewat SSH yang sudah
tertutup.

**Sertifikat TLS tidak terbit.** Domain `.cloud` belum menunjuk ke IP saat
Caddy mencoba, atau port 80 tertutup `ufw`. Let's Encrypt memverifikasi lewat
HTTP di port 80. Periksa `dc logs caddy`.

**API restart terus.** Hampir selalu `Validate()` menolak konfigurasi. `dc
logs api` menyebutkan persis variabel mana dan kenapa.

**Halaman termuat tapi katalog kosong.** Frontend hidup, API tidak. `dc ps`
lalu `dc logs api`.
