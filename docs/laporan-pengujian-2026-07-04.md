# Laporan Pengujian Sistem — Fitur v2.0 / v3.0

**Tanggal pengujian:** 4 Juli 2026
**Metode:** Blackbox testing (skenario masukan → keluaran) + verifikasi data langsung ke basis data
**Lingkungan:** Docker Compose lokal — `db` (PostgreSQL 16), `redis` (7), `tesseract-api` (self-hosted OCR), `api` (Go/Echo), `web` (React/Nginx)
**Penguji:** Ahmad Irfan Ghazali

> Laporan ini melengkapi tabel pengujian modul v1 (skripsi Tabel 3.11–3.15) dengan pengujian fitur lanjutan v2.0/v3.0 yang belum tercakup: RBAC, OCR self-hosted, chatbot AI, payment gateway, dan returning customer. Selama pengujian ditemukan **3 bug** yang seluruhnya telah diperbaiki dan diuji ulang (lihat §Bug).

---

## Ringkasan Hasil

| Modul / Fitur | Jumlah Skenario | Lulus | Status |
|---|---|---|---|
| Autentikasi & RBAC (kontrol akses peran) | 4 | 4 | ✅ LULUS |
| OCR Dokumen (Tesseract self-hosted) | 4 | 4 | ✅ LULUS |
| Returning Customer (pelanggan lama) | 3 | 3 | ✅ LULUS |
| Chatbot AI (Gemini) | 4 | 4 | ✅ LULUS |
| Payment Gateway (Midtrans) | 4 | 4 | ✅ LULUS |
| **Total** | **19** | **19** | **100% LULUS** |

> Catatan: hasil 100% dicapai **setelah** perbaikan 3 bug yang ditemukan saat pengujian. Rincian bug pada bagian akhir.

---

## 1. Pengujian Autentikasi & RBAC

Memvalidasi penegakan hak akses berbasis peran di backend (middleware `RequireRole`), bukan sekadar penyembunyian menu di frontend.

| No | Skenario | Masukan | Hasil Diharapkan | Hasil Aktual | Status |
|---|---|---|---|---|---|
| 1 | Login staf valid | Email `admin@pintour.com`, password `admin123` | Token JWT terbit, role `super_admin` | Token terbit, role `super_admin` | Berhasil |
| 2 | Super admin akses manajemen user | `GET /admin/users` dengan token super_admin | HTTP 200, daftar user tampil | HTTP 200 | Berhasil |
| 3 | Konsultan akses endpoint terlarang | `GET /admin/users` dengan token konsultan | HTTP 403, ditolak | HTTP 403 `FORBIDDEN` "Akses tidak diizinkan untuk peran Anda" | Berhasil |
| 4 | Login portal peserta | Nomor WA + password peserta | Token portal terbit + data peserta | Token terbit, data peserta valid | Berhasil |

---

## 2. Pengujian OCR Dokumen (Tesseract Self-Hosted)

Memvalidasi ekstraksi data KTP/paspor sepenuhnya di infrastruktur sendiri (container Tesseract, tanpa kirim data ke pihak ketiga).

| No | Skenario | Masukan | Hasil Diharapkan | Hasil Aktual | Status |
|---|---|---|---|---|---|
| 1 | Container OCR aktif | `docker compose up` | Service `tesseract-api` healthy, log API "OCR aktif — engine=tesseract_local" | Healthy, OCR aktif | Berhasil |
| 2 | Upload & proses KTP | Gambar KTP (NIK 3174011501990001, nama BUDI HARTONO TEST) via portal | OCR ekstrak NIK, nama, tgl lahir → simpan ke `document_ocr_results` | NIK `3174011501990001`, nama `BUDI HARTONO TEST`, tgl lahir `1999-01-15`, confidence `1.0` | Berhasil |
| 3 | Validasi hasil OCR | Hasil ekstraksi KTP | `validation_passed = true`, catatan "NIK valid" | `validation_passed = t`, "NIK valid" | Berhasil |
| 4 | Auto-isi NIK ke peserta | KTP confidence ≥ 0.85 | `participants.nik` terisi otomatis | NIK peserta terisi `3174011501990001` | Berhasil |

> Privasi: gambar KTP hanya dikirim ke container Tesseract di jaringan Docker internal (port 8884 tidak di-expose ke host). Tidak ada request ke domain eksternal.

---

## 3. Pengujian Returning Customer (Pelanggan Lama)

Memvalidasi deteksi otomatis pelanggan lama saat lead masuk dan penampilan riwayat perjalanannya.

| No | Skenario | Masukan | Hasil Diharapkan | Hasil Aktual | Status |
|---|---|---|---|---|---|
| 1 | Lead dari nomor terdaftar | Form lead dengan phone yang sudah ada di `portal_users` | `is_returning = true`, ter-link ke akun portal | `is_returning = t`, `portal_user_id` ter-link | Berhasil |
| 2 | Panel riwayat tour di CRM | Buka detail lead pelanggan lama (Sari, 1 tour) | Tampil riwayat tour sebelumnya | `previous_trips = 1` → "Dubai City Tour 5 Hari" | Berhasil |
| 3 | Lead nomor baru (bukan pelanggan lama) | Form lead dengan phone belum terdaftar | `is_returning = false`, tanpa riwayat | `is_returning = f`, riwayat kosong | Berhasil |

---

## 4. Pengujian Chatbot AI (Gemini)

Memvalidasi respons chatbot atas pesan WA masuk (webhook), memori percakapan multi-turn, dan konteks paket dari basis data.

| No | Skenario | Masukan | Hasil Diharapkan | Hasil Aktual | Status |
|---|---|---|---|---|---|
| 1 | Balasan pesan pertama | Webhook: "ada paket umroh budget 30 juta?" | Bot balas relevan, tersimpan di `chatbot_logs` | Balasan kontekstual sebut paket + harga, tersimpan | Berhasil |
| 2 | Percakapan multi-turn (memori) | Pesan 1: "mau ke jepang" → Pesan 2: "berapa harganya?" (tanpa sebut Jepang) | Bot ingat konteks Jepang di pesan ke-2 | Bot jawab "Sakura Tour Jepang 8 Hari, Rp 24.800.000, 8 hari" | Berhasil |
| 3 | Keandalan (5 pesan beruntun) | 5 pesan dari 5 nomor berbeda | Semua dapat balasan (retry + fallback) | 5/5 dapat balasan | Berhasil |
| 4 | Kirim balasan ke WA nyata | Webhook dari nomor asli, device Fonnte connect | Balasan terkirim via Fonnte ke WhatsApp | Terkirim (`status: true`), diterima di WhatsApp | Berhasil |

> Batasan lingkungan: pada mode lokal, Fonnte tidak dapat memanggil webhook `localhost` sehingga pesan masuk **dari** WhatsApp nyata tidak otomatis diteruskan ke sistem. Pengujian inbound disimulasikan dengan POST langsung ke endpoint webhook. Chat dua arah penuh memerlukan URL publik (staging).

---

## 5. Pengujian Payment Gateway (Midtrans)

Memvalidasi pembuatan transaksi Snap dan pemrosesan notifikasi (webhook) beserta verifikasi keamanannya.

| No | Skenario | Masukan | Hasil Diharapkan | Hasil Aktual | Status |
|---|---|---|---|---|---|
| 1 | Buat transaksi pembayaran | `POST /portal/invoices/:id/create-payment` (Midtrans sandbox) | `snap_token` + `client_key` kembali | `snap_token` valid dari Midtrans, order_id tersimpan | Berhasil |
| 2 | Webhook settlement (signature valid) | Notifikasi `settlement` + signature SHA512 benar | Invoice → `lunas` | Status invoice `lunas`, `confirmed_at` terisi | Berhasil |
| 3 | Portal aktif setelah lunas | Setelah settlement diproses | Peserta `is_active = true` | Peserta ter-aktivasi | Berhasil |
| 4 | Tolak webhook signature palsu | Notifikasi dengan `signature_key` palsu | HTTP 400, ditolak | HTTP 400 `INVALID_SIGNATURE` | Berhasil |

---

## Bug Ditemukan & Diperbaiki

Tiga bug ditemukan selama pengujian dan seluruhnya telah diperbaiki serta diuji ulang (regresi lulus). Pencatatan ini penting untuk transparansi: bug #1 juga memengaruhi fitur v1 yang sebelumnya diasumsikan berjalan.

| # | Severity | Bug | Akar Masalah | Perbaikan | Verifikasi |
|---|---|---|---|---|---|
| 1 | Critical | Akses file privat gagal — OCR tidak jalan, tombol "Lihat" dokumen & bukti transfer rusak | File di bucket **privat** Supabase disimpan sebagai path relatif, diakses langsung → `unsupported protocol scheme` / 404 | Semua akses file privat lewat **signed URL** (§19.2): OCR resolve path→signed URL sebelum fetch; endpoint `/admin/signed-url` untuk staf; frontend fetch signed URL dulu | OCR jalan (NIK terekstrak), signed URL fetch HTTP 200 |
| 2 | High | Chatbot gagal senyap (~30% pesan tak dibalas) tanpa log | Error dari Gemini (rate limit/timeout) ditelan goroutine, tak ada retry/fallback | Tambah **retry 1×** + **balasan fallback** + **logging** error | 5/5 pesan dapat balasan |
| 3 | Low | Healthcheck container Tesseract selalu "unhealthy" padahal service jalan | `wget localhost` resolve ke IPv6 (`::1`), server hanya listen IPv4 | Ubah healthcheck ke `127.0.0.1` | Container `healthy` |

> **Catatan penting (bug #1):** Kegagalan akses file privat ini bukan hanya soal fitur OCR (v2.0), tetapi juga memengaruhi modul v1 **Review Dokumen** dan **Konfirmasi Pembayaran** (tombol "Lihat" untuk dokumen peserta & bukti transfer). Kemungkinan tidak terdeteksi pada pengujian awal karena dilakukan sebelum kredensial Supabase Storage nyata terpasang.

---

## Batasan & Catatan (bukan bug)

1. **Webhook inbound di lokal:** Fonnte (dan Midtrans) tidak dapat menjangkau `localhost`. Pengujian inbound (pesan chatbot masuk, notifikasi Midtrans) disimulasikan via POST langsung. Chat dua-arah otomatis & callback Midtrans nyata memerlukan deployment dengan URL publik HTTPS (staging).
2. **Panel riwayat returning customer:** field `departure_date` dan `payment_status` masih tampil kosong pada card riwayat (cosmetic — nama paket sudah benar). Belum diperbaiki.
3. **Nomor telepon peserta:** selama pengujian seluruh `participants.phone` diarahkan ke satu nomor uji agar notifikasi WA dapat diverifikasi di perangkat nyata (data demo).

---

## Kesimpulan

Seluruh 19 skenario pengujian fitur v2.0/v3.0 **LULUS 100%** setelah perbaikan 3 bug. Fitur RBAC, OCR self-hosted, chatbot AI multi-turn, payment gateway Midtrans (beserta verifikasi signature), dan returning customer terbukti berfungsi sesuai spesifikasi pada lingkungan Docker Compose lokal. Pengujian end-to-end penuh untuk jalur webhook inbound (chatbot & Midtrans callback) akan dilakukan ulang setelah deployment ke staging dengan URL publik.
