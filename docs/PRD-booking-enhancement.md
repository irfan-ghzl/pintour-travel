# PRD — Booking Enhancement
**Pintour Travel — Internal CRM**
**Version:** 1.0
**Tanggal:** 1 Mei 2026
**Author:** Engineering Team

---

## 1. Latar Belakang

Alur bisnis Pintour saat ini berjalan secara manual di luar sistem:
- Konfirmasi pembayaran (DP & pelunasan) dilakukan lewat chat WA, tanpa rekam jejak di sistem
- Pengumpulan dokumen peserta (paspor, KTP, rekening koran, dokumen visa) tidak tertracking per peserta
- Admin tidak bisa melihat booking sedang di fase mana (setelah booking dibuat, statusnya stagnan di `confirmed`)
- Tidak ada penugasan tour leader di dalam sistem

Akibatnya, admin harus mengingat semua progress secara manual dan rawan informasi terlewat menjelang keberangkatan.

---

## 2. Tujuan

1. Admin bisa mencatat dan memverifikasi pembayaran (DP & pelunasan) lengkap dengan bukti transfer
2. Admin bisa tracking kelengkapan dokumen tiap peserta sebelum pengajuan visa
3. Status booking mencerminkan fase perjalanan bisnis yang nyata
4. Admin bisa menugaskan tour leader dan mencatat link grup WA briefing

---

## 3. Scope

### In Scope
- Feature A: Payment Tracking
- Feature B: Participant Document Collection
- Feature C: Booking Lifecycle & Tour Leader Assignment

### Out of Scope
- Payment gateway (Midtrans/Xendit) — tidak relevan dengan model bisnis manual saat ini
- UTM/Meta Ads tracking — dihandle di Meta Ads Manager
- Invoice sebagai dokumen terpisah — quotation yang di-approve sudah cukup
- Airport handling checklist — operasional lapangan, bukan sistem

---

## 4. User Stories

### Feature A — Payment Tracking

> **Sebagai admin**, saya ingin mencatat pembayaran DP dan pelunasan beserta bukti transfernya, sehingga saya tidak perlu mencari-cari screenshot di WA saat ada pertanyaan dari customer.

**Acceptance Criteria:**
- Admin dapat menambah record pembayaran untuk sebuah booking
- Setiap record pembayaran memiliki: tipe (dp / pelunasan), nominal, tanggal bayar, URL bukti transfer
- Admin dapat menandai pembayaran sebagai "terverifikasi" beserta catatan
- Di halaman detail booking, tampil daftar semua pembayaran dan total yang sudah diterima
- `payment_status` di booking terupdate otomatis: `pending` → `dp` → `lunas` berdasarkan record payments

---

### Feature B — Participant Document Collection

> **Sebagai admin**, saya ingin tracking dokumen apa saja yang sudah diterima dari setiap peserta, sehingga saya tahu siapa yang belum lengkap sebelum pengajuan visa.

**Acceptance Criteria:**
- Admin dapat menambah record dokumen untuk tiap peserta dalam booking
- Tipe dokumen: `passport`, `ktp`, `bank_statement`, `visa_support` (+ free text untuk dokumen khusus negara tertentu)
- Setiap dokumen menyimpan: tipe, URL file, status verifikasi (verified / pending)
- Di halaman detail booking, tampil progress kelengkapan dokumen per peserta (misal: "3/4 dokumen lengkap")
- Admin dapat menandai dokumen sebagai verified

---

### Feature C — Booking Lifecycle & Tour Leader

> **Sebagai admin**, saya ingin melihat booking sedang di fase mana dan siapa tour leadernya, sehingga saya tahu booking mana yang perlu tindak lanjut segera.

**Acceptance Criteria:**
- `booking_status` memiliki nilai: `confirmed` → `awaiting_docs` → `visa_process` → `ready_to_depart` → `departed` → `completed` (+ `cancelled`)
- Admin dapat mengubah status booking secara manual
- Admin dapat menugaskan tour leader (dari daftar user dengan role `tour_leader` / `staff`)
- Admin dapat menyimpan link grup WA briefing di booking
- Di list booking, status ditampilkan dengan badge warna yang berbeda per fase

---

## 5. Database Schema

### 5.1 Tabel Baru: `payments`

```sql
CREATE TABLE payments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id      UUID            NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    payment_type    VARCHAR(50)     NOT NULL DEFAULT 'dp',  -- 'dp', 'pelunasan', 'full'
    amount          NUMERIC(15, 2)  NOT NULL,
    paid_at         TIMESTAMPTZ     NOT NULL,
    proof_url       TEXT,                                    -- URL bukti transfer
    notes           TEXT,
    verified_by     UUID            REFERENCES users(id) ON DELETE SET NULL,
    verified_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payments_booking ON payments(booking_id);
```

### 5.2 Tabel Baru: `participant_documents`

```sql
CREATE TABLE participant_documents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    participant_id  UUID            NOT NULL REFERENCES booking_participants(id) ON DELETE CASCADE,
    doc_type        VARCHAR(100)    NOT NULL,  -- 'passport', 'ktp', 'bank_statement', 'visa_support', dll
    file_url        TEXT            NOT NULL,
    notes           TEXT,
    verified        BOOLEAN         NOT NULL DEFAULT FALSE,
    verified_by     UUID            REFERENCES users(id) ON DELETE SET NULL,
    verified_at     TIMESTAMPTZ,
    uploaded_at     TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_participant_docs_participant ON participant_documents(participant_id);
```

### 5.3 Perubahan Tabel: `bookings`

```sql
ALTER TABLE bookings
    ADD COLUMN tour_leader_id  UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN wa_group_link   TEXT,
    ADD COLUMN briefing_done   BOOLEAN NOT NULL DEFAULT FALSE;
```

**Perubahan `booking_status` enum** (constraint atau dokumentasi):

| Nilai | Deskripsi |
|---|---|
| `confirmed` | Booking baru dibuat, belum ada tindak lanjut |
| `awaiting_docs` | Menunggu pengumpulan dokumen peserta |
| `visa_process` | Dokumen lengkap, sedang proses visa |
| `ready_to_depart` | Visa approved, siap berangkat |
| `departed` | Sudah berangkat |
| `completed` | Perjalanan selesai |
| `cancelled` | Booking dibatalkan |

---

## 6. API Endpoints Baru

### Feature A — Payments

| Method | Path | Deskripsi |
|---|---|---|
| `GET` | `/api/v1/admin/bookings/:id/payments` | List semua pembayaran sebuah booking |
| `POST` | `/api/v1/admin/bookings/:id/payments` | Tambah record pembayaran |
| `PATCH` | `/api/v1/admin/payments/:payment_id/verify` | Verifikasi pembayaran |
| `DELETE` | `/api/v1/admin/payments/:payment_id` | Hapus record pembayaran |

**POST `/payments` Request Body:**
```json
{
  "payment_type": "dp",
  "amount": 5000000,
  "paid_at": "2026-05-01T10:00:00Z",
  "proof_url": "https://storage.example.com/proof/abc.jpg",
  "notes": "Transfer BCA"
}
```

---

### Feature B — Participant Documents

| Method | Path | Deskripsi |
|---|---|---|
| `GET` | `/api/v1/admin/bookings/:id/participants/:pid/documents` | List dokumen peserta |
| `POST` | `/api/v1/admin/bookings/:id/participants/:pid/documents` | Upload dokumen |
| `PATCH` | `/api/v1/admin/documents/:doc_id/verify` | Verifikasi dokumen |
| `DELETE` | `/api/v1/admin/documents/:doc_id` | Hapus dokumen |

**POST `/documents` Request Body:**
```json
{
  "doc_type": "passport",
  "file_url": "https://storage.example.com/docs/passport-john.jpg",
  "notes": "Paspor berlaku s.d. 2030"
}
```

---

### Feature C — Tour Leader & WA Group

| Method | Path | Deskripsi |
|---|---|---|
| `PATCH` | `/api/v1/admin/bookings/:id/leader` | Set tour leader |
| `PATCH` | `/api/v1/admin/bookings/:id/wa-group` | Set link grup WA |
| `PATCH` | `/api/v1/admin/bookings/:id/booking-status` | (sudah ada, perlu update enum) |

**PATCH `/leader` Request Body:**
```json
{
  "tour_leader_id": "uuid-of-staff-user"
}
```

---

## 7. UI/UX Changes (Admin Panel)

### Halaman Detail Booking (baru)
Saat ini halaman detail booking sudah ada expandable row di list. Perlu ditambahkan:

**Tab/Section baru:**
1. **Pembayaran** — tabel daftar payments + tombol "Tambah Pembayaran" + badge total terbayar vs total harga
2. **Dokumen Peserta** — per peserta, tampil checklist dokumen dengan status (✓ verified / ⏳ pending / ✗ belum ada)
3. **Info Keberangkatan** — field tour leader, link grup WA, checkbox briefing done

### Halaman List Booking
- Tambah badge status dengan warna per fase lifecycle
- Tambah kolom "Tour Leader"
- Filter by `booking_status` yang diperluas

---

## 8. Domain & Service Layer

### Modul Baru: `payment`
```
internal/
  domain/
    payment/
      entity.go       -- Payment, CreateParams, Filter
      repository.go   -- Repository interface
  application/
    payment/
      service.go      -- CreatePayment, VerifyPayment, ListByBooking
  infrastructure/
    postgres/
      payment_repo.go
  delivery/
    http/
      payment_handler.go
```

### Modul Baru: `document`
```
internal/
  domain/
    document/
      entity.go       -- ParticipantDocument, CreateParams
      repository.go
  application/
    document/
      service.go
  infrastructure/
    postgres/
      document_repo.go
  delivery/
    http/
      document_handler.go
```

### Perubahan Modul `booking`
- Tambah field `TourLeaderID`, `WAGroupLink`, `BriefingDone` di entity `Booking`
- Update `BookingStatus` valid values di service layer
- Tambah method `SetTourLeader`, `SetWAGroup` di service

---

## 9. Migration Plan

1. Buat migration file `002_payments_and_documents.sql`
2. Tambah tabel `payments` dan `participant_documents`
3. `ALTER TABLE bookings` untuk kolom baru
4. Update seed data jika diperlukan

---

## 10. File Upload Strategy

Untuk `proof_url` (bukti transfer) dan `file_url` (dokumen peserta), sistem **tidak menyimpan file langsung** di server app. Strategi yang direkomendasikan:

**Opsi 1 (Sekarang — Simple):** Admin upload file ke storage manual (Google Drive/S3), lalu paste URL-nya ke form. Sistem hanya menyimpan URL string.

**Opsi 2 (Future):** Tambah endpoint `POST /api/v1/admin/upload` yang menerima multipart form dan mengembalikan URL dari object storage (S3 / Supabase Storage / Cloudflare R2).

Untuk MVP, gunakan Opsi 1.

---

## 11. Prioritas & Estimasi

| Feature | Prioritas | Kompleksitas | Notes |
|---|---|---|---|
| A — Payment Tracking | P0 | Medium | Paling kritikal untuk operasional |
| C — Booking Lifecycle | P1 | Low | Mostly schema + enum change |
| B — Participant Documents | P2 | Medium | Bergantung file upload strategy |

**Urutan implementasi yang disarankan:** C → A → B

---

## 12. Definisi Done

- [ ] Migration SQL dibuat dan dapat dijalankan
- [ ] Domain entity dan repository interface didefinisikan
- [ ] Service layer dengan business logic
- [ ] HTTP handler dengan validasi input
- [ ] Route terdaftar di router
- [ ] Frontend admin panel menampilkan dan bisa mengelola data
- [ ] Tidak ada breaking change pada endpoint yang sudah ada
