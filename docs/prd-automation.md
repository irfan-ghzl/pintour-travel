# PRD — Automation System
**Pintour Travel**
**Versi**: 1.0
**Tanggal**: 1 Mei 2026
**Status**: Draft

---

## Latar Belakang

Saat ini seluruh proses bisnis Pintour Travel dijalankan secara manual oleh admin:
- Admin memantau inquiry satu per satu lalu menghubungi customer via WA
- Admin mengubah status booking secara manual di setiap tahap
- Admin menghitung total pembayaran secara manual untuk menentukan payment_status
- Admin harus ingat jadwal keberangkatan untuk mengirim reminder
- Quotation yang sudah melewati tanggal berlaku tidak otomatis kadaluarsa

Otomasi bertujuan mengurangi pekerjaan repetitif admin dan memastikan tidak ada proses yang terlewat.

---

## Ruang Lingkup

Dokumen ini mencakup **4 area otomasi** yang dibagi berdasarkan domain bisnis:

1. **Auto Notifikasi WA** — Notifikasi otomatis ke customer dan admin
2. **Auto Status Booking** — Perubahan status booking berdasarkan kondisi data
3. **Auto Payment Status** — Update `payment_status` berdasarkan total pembayaran yang masuk
4. **Auto Quotation Expiry** — Kadaluarsa penawaran berdasarkan `valid_until`

Semua berjalan **in-process** menggunakan cron goroutine di dalam container `pintour_api`. Tidak membutuhkan platform eksternal.

---

## Feature 1 — Auto Notifikasi WA

### 1.1 Notifikasi Customer saat Inquiry Masuk

**Trigger**: Customer submit form inquiry dari e-catalog
**Saat ini**: Customer tidak mendapat konfirmasi apapun; admin harus buka CMS dan klik link WA secara manual
**Target**:
- Customer otomatis menerima pesan WA konfirmasi bahwa inquiry sudah diterima
- Admin otomatis menerima notifikasi WA berisi ringkasan inquiry baru

**Template WA ke Customer**:
```
Halo [FullName]! 👋
Terima kasih sudah menghubungi Pintour Travel.

Kami telah menerima permintaan konsultasi Anda:
• Destinasi: [Destination]
• Jumlah orang: [NumPeople]
• Estimasi keberangkatan: [DepartureDate]

Tim kami akan segera menghubungi Anda dalam 1×24 jam.
Atau chat langsung: https://wa.me/6282121952655
```

**Template WA ke Admin**:
```
📋 INQUIRY BARU — [BookingCode]
Nama: [FullName]
WA: [Phone]
Destinasi: [Destination]
Tanggal: [DepartureDate]
Orang: [NumPeople]
Budget: Rp [Budget]

Balas sekarang: https://wa.me/[Phone]
```

**Implementasi**:
- Hook di `POST /api/v1/inquiries` handler setelah inquiry berhasil disimpan
- Panggil `service.WhatsApp.SendToCustomer(phone, template)` dan `SendToAdmin(adminPhone, template)`
- Gunakan `https://wa.me/{phone}?text={encoded}` (tidak butuh WA Business API)
- Non-blocking: jalankan di goroutine, gagal kirim tidak block response

**Catatan**: Karena wa.me adalah link bukan API push, pengiriman otomatis membutuhkan integrasi WA Business API (Fonnte/Wablas/dll) agar benar-benar auto-send. Jika belum ada budget, fallback ke: simpan notif di DB → tampilkan di dashboard admin sebagai "tugas hari ini".

---

### 1.2 Notifikasi Customer saat Status Booking Berubah

**Trigger**: Admin mengubah `booking_status`
**Target**: Customer otomatis mendapat WA sesuai status baru

| Status Baru | Pesan WA ke Customer |
|-------------|----------------------|
| `confirmed` | "Booking Anda BK-xxx dikonfirmasi! Total: Rp xxx" |
| `awaiting_docs` | "Mohon upload dokumen: passport, KTP sebelum [date]" |
| `visa_process` | "Dokumen Anda sedang diproses visa. Estimasi selesai: [date]" |
| `ready_to_depart` | "Selamat! Anda siap berangkat. Briefing: [tanggal] di [lokasi]" |
| `departed` | "Selamat menikmati perjalanan bersama Pintour! 🌏" |
| `completed` | "Terima kasih telah berwisata bersama kami. Bagikan pengalaman Anda!" |
| `cancelled` | "Booking BK-xxx dibatalkan. Hubungi kami untuk info refund." |

**Implementasi**:
- Hook di `PATCH /admin/bookings/:id/status` handler
- Kirim WA ke `booking.CustomerPhone` setelah status berhasil diupdate

---

### 1.3 Reminder Pra-Keberangkatan (Cron)

**Trigger**: Cron job harian jam 08:00
**Target**: Kirim WA reminder ke customer berdasarkan jarak ke `departure_date`

| H- | Pesan |
|----|-------|
| H-7 | "Keberangkatan Anda [tujuan] tinggal 7 hari lagi. Pastikan dokumen sudah lengkap." |
| H-3 | "3 hari lagi berangkat! Cek packing list Anda. Link WA Group: [wa_group_link]" |
| H-1 | "Besok berangkat! Briefing jam [waktu]. Siapkan dokumen asli." |

**Implementasi**:
- `internal/scheduler/jobs/departure_reminder.go`
- Query: `SELECT * FROM bookings WHERE departure_date = NOW() + INTERVAL 'X days' AND booking_status NOT IN ('cancelled', 'completed')`

---

## Feature 2 — Auto Status Booking

### 2.1 Auto-progress ke `departed` dan `completed` (Cron)

**Trigger**: Cron job harian jam 00:05
**Kondisi**:
- Jika `departure_date = today` AND `booking_status = 'ready_to_depart'` → set `departed`
- Jika `departure_date + durasi_paket <= today` AND `booking_status = 'departed'` → set `completed`

**Catatan**: Durasi paket diambil dari `tour_packages.duration_days`. Jika booking tidak terikat paket (custom quotation), gunakan field `duration_days` opsional di tabel bookings (perlu ditambah di migration).

**Implementasi**:
- `internal/scheduler/jobs/booking_auto_status.go`
- Jalankan setiap hari jam 00:05 WIB

---

### 2.2 Auto-suggest ke `awaiting_docs` (Trigger Lunak)

**Trigger**: Saat booking `payment_status` berubah menjadi `dp` atau `lunas`
**Aksi**: Jika `booking_status` masih `confirmed`, sistem otomatis suggest perubahan ke `awaiting_docs`
**Catatan**: Ini adalah "suggest" bukan force-change, karena admin mungkin memiliki alasan lain

**Implementasi alternatif**: Tampilkan banner di AdminBookingsPage "DP sudah masuk — pertimbangkan ubah status ke Awaiting Docs"

---

### 2.3 Auto-check Kelengkapan Dokumen

**Trigger**: Cron job harian jam 09:00
**Logika**:
- Ambil semua booking dengan status `awaiting_docs`
- Cek apakah semua participant sudah memiliki dokumen required yang verified
- Jika ya → kirim notifikasi ke admin: "Dokumen booking BK-xxx lengkap, siap proses visa"

**Dokumen required per participant**: passport + ktp (minimal)

**Implementasi**:
- `internal/scheduler/jobs/doc_completeness_check.go`
- Output: Notifikasi di DB (tabel `admin_notifications`) atau WA ke admin

---

## Feature 3 — Auto Payment Status

### 3.1 Auto-update `payment_status` saat Payment Diverifikasi

**Trigger**: Admin verify payment (`PATCH /admin/payments/:pid/verify`)
**Logika** (setelah payment diverifikasi):
```
total_verified = SUM(amount) WHERE booking_id = X AND verified = true
total_price    = bookings.total_price WHERE id = X

IF total_verified >= total_price    → payment_status = 'lunas'
ELSE IF total_verified > 0          → payment_status = 'dp'
ELSE                                → payment_status = 'belum_bayar'
```

**Implementasi**:
- Tambah function `recalculatePaymentStatus(ctx, bookingID)` di `PaymentService`
- Dipanggil di `VerifyPayment` handler setelah verify berhasil
- Dipanggil juga di `DeletePayment` handler

**Perubahan tabel** (tidak butuh migration baru, cukup update query):
- Tambahkan `belum_bayar` sebagai valid value di `ValidPaymentStatuses`

---

### 3.2 Tampil Sisa Pembayaran di UI

**Lokasi**: Tab Pembayaran di `AdminBookingsPage.tsx`
**Tampilan**:
```
┌─────────────────────────────────────┐
│ Total Tagihan      Rp 15.000.000    │
│ Total Dibayar      Rp  5.000.000    │
│ Sisa               Rp 10.000.000    │
│ ████░░░░░░░░░░░  33% Terbayar       │
└─────────────────────────────────────┘
```

**Implementasi**:
- Frontend only, tidak butuh endpoint baru
- Hitung dari `payments.reduce(sum, p => sum + p.amount, 0)` vs `booking.total_price`

---

## Feature 4 — Auto Quotation Expiry

### 4.1 Auto-set status `expired` (Cron)

**Trigger**: Cron job harian jam 00:01
**Kondisi**: `valid_until < today` AND `status IN ('draft', 'sent')`
**Aksi**: Update `status = 'expired'`

**Implementasi**:
- `internal/scheduler/jobs/quotation_expiry.go`
- Query: `UPDATE quotations SET status = 'expired' WHERE valid_until < NOW() AND status IN ('draft', 'sent')`

---

### 4.2 Badge "Akan Expired" di UI

**Lokasi**: `AdminQuotationsPage.tsx` — kolom Status
**Kondisi**: `valid_until` dalam 3 hari ke depan AND status masih `sent`
**Tampilan**: Badge kuning "⚠ Exp 2 hari" di sebelah status

**Implementasi**:
- Frontend only
- Hitung `daysUntilExpiry = Math.ceil((new Date(q.valid_until) - Date.now()) / 86400000)`

---

## Arsitektur Implementasi

```
pintour_api container
├── main.go
│   ├── Echo HTTP server (existing)
│   └── go scheduler.Start(db)        ← tambah baris ini
│
└── internal/scheduler/
    ├── scheduler.go                   ← cron runner
    └── jobs/
        ├── booking_auto_status.go     ← Feature 2.1
        ├── departure_reminder.go      ← Feature 1.3
        ├── doc_completeness_check.go  ← Feature 2.3
        └── quotation_expiry.go        ← Feature 4.1
```

**Library**: `github.com/robfig/cron/v3` (tambah di `go.mod`)

**Jadwal Cron**:
```
00:01  → quotation_expiry
00:05  → booking_auto_status
08:00  → departure_reminder (H-7, H-3, H-1)
09:00  → doc_completeness_check
```

---

## Prioritas Implementasi

| # | Feature | Effort | Impact | Prioritas |
|---|---------|--------|--------|-----------|
| 3.1 | Auto payment_status | Kecil (1-2 jam) | Tinggi | 🔴 P1 |
| 3.2 | Sisa pembayaran di UI | Kecil (30 menit) | Tinggi | 🔴 P1 |
| 4.1 | Auto quotation expiry | Kecil (1 jam) | Sedang | 🟡 P2 |
| 4.2 | Badge akan expired | Kecil (30 menit) | Sedang | 🟡 P2 |
| 2.1 | Auto departed/completed | Sedang (2 jam) | Tinggi | 🟡 P2 |
| 1.2 | WA notif per status | Sedang (3 jam) | Tinggi | 🟡 P2 |
| 1.3 | Reminder H-7/3/1 | Sedang (3 jam) | Sedang | 🟠 P3 |
| 2.3 | Check kelengkapan dokumen | Sedang (2 jam) | Sedang | 🟠 P3 |
| 1.1 | WA notif inquiry baru | Besar* | Sedang | 🟠 P3 |
| 2.2 | Auto-suggest status | Kecil | Rendah | ⚪ P4 |

*Butuh WA Business API (Fonnte/Wablas) untuk auto-send. Tanpa itu hanya bisa buat link.

---

## Catatan WA Business API

Fitur 1.1, 1.2, 1.3 membutuhkan kemampuan **push message** ke nomor WA customer.
`wa.me` link hanya bisa digunakan oleh user yang klik sendiri — tidak bisa auto-send dari server.

**Opsi**:
| Provider | Harga | Cara |
|----------|-------|------|
| **Fonnte** | ~Rp 75rb/bulan | REST API, WA non-official |
| **Wablas** | ~Rp 150rb/bulan | REST API, WA non-official |
| **WA Business API (Meta)** | Gratis s/d 1000 conversation/bulan | Official, butuh approval |
| **Fallback: in-app notif** | Gratis | Simpan di DB, tampilkan di CMS |

Rekomendasi: mulai dengan **fallback in-app notification** dulu (tidak butuh biaya), bisa upgrade ke Fonnte/WA API kapan saja.

---

## Definition of Done

- [ ] Cron scheduler berjalan tanpa mengganggu HTTP server
- [ ] Setiap job bisa di-disable via environment variable
- [ ] Error di cron job tidak crash server (recover with log)
- [ ] `payment_status` terupdate otomatis saat payment diverifikasi
- [ ] Quotation kadaluarsa terupdate otomatis tiap malam
- [ ] Booking otomatis `departed` saat hari keberangkatan tiba
- [ ] UI menampilkan sisa pembayaran dan badge quotation akan expired
