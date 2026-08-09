# Gap Analysis — Spec Automasi vs. Codebase

**Tanggal:** 2026-06-26
**Sumber spec:** `prompt_implementasi_automasi.md`
**Branch:** `refactor`
**Legend:** ✅ Done · 🟡 Partial · ❌ Missing

---

## 0. Catatan Penting — Mismatch Lintas-Sektor (baca dulu)

Spec ditulis untuk layout & konvensi yang **tidak sama** dengan codebase. Sebelum implementasi apa pun, keputusan-keputusan ini harus diselesaikan karena memengaruhi banyak item sekaligus:

| # | Mismatch | Spec | Codebase aktual | Dampak |
|---|----------|------|-----------------|--------|
| M1 | **Arsitektur** | `internal/service` + `internal/handler` flat | Clean arch: `domain` / `application` / `delivery/http` / `infrastructure/postgres` | Semua path di spec harus dipetakan ulang |
| M2 | **Portal auth** | Token (JWT/UUID 30 hari) + `portal_tokens` | Phone + password (`portal_password`, bcrypt), `PortalLogin` | §1.3, §5.6 token-flow tidak cocok; portal sudah password-based |
| M3 | **Status invoice** | `menunggu_pembayaran` / `partial` / `lunas` / `expired` | `diterbitkan` / `menunggu_bayar` / confirm-flow | §1.3, §1.4, §2.3, §5.2 badge harus diselaraskan |
| M4 | **Status lead** | `baru` / `deal` / `expired` | `baru` / `deal` / `tidak_deal` | §2.1, §2.2 query harus pakai vocab nyata |
| M5 | **Pembayaran parsial** | `paid_amount += amount_claimed`, hitung partial/lunas | Tidak ada akumulasi; `ConfirmPayment` = lunas penuh | §1.4 logika inti belum ada |
| M6 | **country_doc_requirements** | keyed by `package_id` | repo keyed by `country_code` (`List(ctx, countryCode)`) | §1.2 perlu penyesuaian skema/query |
| M7 | **Nomor invoice** | `INV-{YYYYMMDD}-{6 random}` | `INV-{YYYYMM}-{4 seq}` (`NextSequence`) | §1.1 format beda — putuskan mana yang dipakai |
| M8 | **Email** | ~15 template Resend | Hanya `Send` + `SendResetPassword` | §3.2, §3.3 hampir seluruhnya kosong |

---

## §1 — Automasi Proses Bisnis (Event-Driven)

| Item | Status | Temuan |
|------|--------|--------|
| **1.1** Auto-generate invoice saat convert | ❌ | `participant.ConvertFromLead` (`internal/application/participant/service.go:25`) **tidak** membuat invoice. Pembuatan invoice manual lewat `invoice.Create` (`internal/application/invoice/service.go:39`) yang sudah punya gen nomor + kirim WA. Yang hilang: pemanggilan otomatis dari alur convert, dan `amount = batch.price * pax`. Format nomor beda (M7). |
| **1.2** Auto-generate checklist dokumen | ❌ | Tidak ada. Dokumen dibuat manual via `DocumentHandler.UploadDocument` (`internal/delivery/http/document_handler.go:46`). `CountryRequirementRepository` ada tapi keyed `country_code` bukan `package_id` (M6). |
| **1.3** Auto-aktivasi portal | 🟡 | `invoice.ConfirmPayment` (`service.go:166`) meng-`Activate` participant + kirim WA doc-request. **Tapi** dipicu manual (admin confirm), bukan dari transisi status→lunas; portal password-based, **tidak** ada token/`SendPortalActivated` (M2). |
| **1.4** Auto-kalkulasi sisa bayar | ❌ | `ReviewProof` (`service.go:209`) hanya approve/reject bukti. Tidak ada `paid_amount +=`, tidak ada status partial/lunas terhitung, tidak ada rantai ke aktivasi portal, tidak ada `SendPaymentReceived` (M5). |
| **1.5** Auto-update status batch | ❌ | `DocumentHandler.ReviewDocument` (`document_handler.go:66`) approve/reject + WA hanya saat **reject**. Tidak ada cek kesiapan batch, tidak ada `SendDocApproved` saat semua dok lengkap, tidak ada batch→`siap_berangkat`. |
| **1.6** Auto-generate airport checklist | 🟡 | Mekanisme **ada**: `airportRepo.InitForBatch(batchID)` (`internal/infrastructure/postgres/airport_repo.go:15`) bulk-init checklist. **Tapi** dipicu manual lewat airport handler, belum ter-trigger otomatis dari kesiapan batch (§1.5). |

---

## §2 — Scheduler (Cron Jobs)

Scheduler aktif: `internal/scheduler/jobs.go` (gocron v2) — `sendPaymentReminders`, `sendDepartureReminders`, `activateBriefing`, `sendAirportInfo`, `retentionCleanup`. Jadwal didaftarkan di `Scheduler.Start()` (`jobs.go:37`), bukan `InitScheduler()` seperti spec.

| Item | Status | Temuan |
|------|--------|--------|
| **2.1** `checkStaleLeads` (1 jam) | ❌ | Tidak ada. Perlu query `status='baru' AND created_at <= NOW()-24h`, WA ke `assigned_to`, anti-duplikat via `wa_notifications`. |
| **2.2** `expireLeads` (00:01) | ❌ | Tidak ada. Catatan: vocab lead `tidak_deal` bukan `deal/expired` (M4). |
| **2.3** `expireInvoices` (00:01) | ❌ | Tidak ada. Butuh `SendPaymentOverdue` (WA) + `SendEmailAdminPaymentOverdue` (digest) — keduanya belum ada (§3). |
| **2.4** `checkBatchQuota` (08:00) | ❌ | Tidak ada. Butuh `SendEmailAdminQuotaWarning` + anti-duplikat harian. |
| **2.5** Update init scheduler | 🟡 | Pola registrasi job sudah ada & berbeda dari snippet spec; 4 job baru tinggal ditambah ke slice di `Start()`. Tanda tangan `New(...)` perlu dependency tambahan (lead/invoice/package repo + email). |

---

## §3 — Notifikasi (WA + Email)

### 3.1 WA — `internal/service/fonnte.go`
Sudah ada: `SendLeadsWelcome`, `SendLeadsNotifSales`, `SendInvoice`, `SendPaymentReminder`, `SendDocRequest`, `SendDocRejected`, `SendDepartureReminder`, `Send` generik.

| Fungsi spec | Status |
|-------------|--------|
| `SendPortalActivated` | ❌ (portal password-based, M2) |
| `SendPaymentReceived` | ❌ |
| `SendPaymentRejected` | ❌ (ada `SendDocRejected` utk dokumen, bukan bayar) |
| `SendPaymentOverdue` | ❌ |
| `SendDocApproved` | ❌ |
| `SendBriefingActivated` | 🟡 (pesan briefing di-inline di `activateBriefing`, belum jadi method) |

### 3.2 Email peserta — `internal/service/email.go`
Hanya `Send` + `SendResetPassword`. **Semua** template berikut ❌: `SendEmailInvoice`, `SendEmailPaymentReceived`, `SendEmailPaymentRejected`, `SendEmailPaymentOverdue`, `SendEmailDocRequest`, `SendEmailDocRejected`, `SendEmailDocApproved`, `SendEmailPortalActivated`, `SendEmailBriefingActivated`, `SendEmailReminderH14`.

### 3.3 Email admin
**Semua** ❌: `SendEmailAdminNewLeads`, `SendEmailAdminPaymentProof`, `SendEmailAdminDocUploaded`, `SendEmailAdminPaymentOverdue`, `SendEmailAdminQuotaWarning`. Butuh `users.ListByRole('admin')` (cek apakah ada di `user.Repository`).

---

## §4 — Export Laporan & Analytics

| Item | Status | Temuan |
|------|--------|--------|
| **4.1** Export Excel | ❌ | `github.com/xuri/excelize/v2` **tidak** di `go.mod`. Tidak ada `report_service`/`report_handler`. |
| **4.2** Export PDF laporan | ❌ | `gofpdf` ada & dipakai utk invoice + airport report (`internal/service/pdf.go`), tapi report leads/participants/invoices/batches belum ada. |
| **4.3** Dashboard analytics | ❌ | `DashboardHandler.GetStats` (`dashboard_handler.go:26`) **stub** ("Connect to a real database"). Endpoint `/dashboard/analytics` + agregasi belum ada. |

---

## §5 — Frontend (React + TS + Tailwind)

Struktur aktual: `web/src/pages/{admin,portal,catalog}`, `web/src/components`, `web/src/utils/{api,auth}.ts`. **Tidak ada** folder `services/`, `hooks/`, atau `components/ui/`. **recharts tidak terpasang**; React Query terpasang.

| Item | Status | Temuan |
|------|--------|--------|
| **5.1** Dashboard analytics + charts | ❌ | `AdminDashboardPage.tsx` ada tapi tanpa analytics; recharts belum ada; backend stub (§4.3). |
| **5.2** Verifikasi pembayaran | 🟡 | `AdminInvoicesPage.tsx` ada; flow approve/reject proof + auto-calc backend belum lengkap (§1.4). Badge status pakai vocab beda (M3). |
| **5.3** Review dokumen | 🟡 | `AdminDocumentsPage.tsx` ada (approve/reject jalan). Filter/progres per-peserta perlu dicek. |
| **5.4** Export laporan | ❌ | `ReportPage.tsx` belum ada; tidak ada menu sidebar. |
| **5.5** CRM stale indicator | ❌ | `AdminLeadsPage.tsx` ada; badge stale + pulse belum ada (backend §2.1 juga belum). |
| **5.6** Portal dashboard | 🟡 | `PortalDashboardPage.tsx` ada, **tapi** auth password (`PortalProtectedRoute`, `PortalLoginPage`), bukan token (M2). |
| **5.7** Portal upload dokumen | 🟡 | `PortalDocumentsPage.tsx` + `FileUpload.tsx` ada. |
| **5.8** Komponen UI | 🟡 | `FileUpload.tsx` ✅. ❌: `StatusBadge`, `CountdownTimer`, `ProgressBar`, `ConfirmModal`, `RejectModal`; tidak ada `components/ui/`. |
| **5.9** Services & hooks | ❌ | Pola beda — semua lewat `utils/api.ts`; tidak ada `services/*`, `hooks/useCountdown`, `hooks/useFileUpload`. |
| **5.10** Routing & sidebar | 🟡 | Routing ada; perlu tambah `/admin/reports`; route portal token tidak relevan (M2). |

---

## Ringkasan Status

- **Sepenuhnya ada (✅):** sebagian §1.6 (mekanisme), sebagian WA §3.1, infra scheduler/PDF/Resend dasar.
- **Partial (🟡):** §1.3, §1.6, §2.5, §3.1 briefing, mayoritas §5 (kerangka halaman ada).
- **Hilang total (❌):** §1.1, §1.2, §1.4, §1.5, semua §2.1–2.4, hampir semua email §3.2/§3.3, semua §4, beberapa komponen/halaman §5.

## Rekomendasi Urutan (setelah keputusan M1–M8)

1. **Putuskan M2–M7 dulu** (auth portal, vocab status, format invoice, skema country-req, pembayaran parsial). Ini fondasi; tanpa ini implementasi akan rework.
2. **§3 notifikasi** (WA + email) — dependency untuk hampir semua automasi & scheduler.
3. **§1.1 + §1.2** (auto-invoice & checklist saat convert) — paling kritikal alur bisnis.
4. **§1.4 + §1.3** (kalkulasi bayar parsial → aktivasi portal).
5. **§1.5 + §1.6** (kesiapan batch → airport checklist auto-trigger; mekanisme `InitForBatch` sudah ada).
6. **§2.1–2.4** scheduler baru.
7. **§4** export + analytics.
8. **§5** frontend per modul (mengikuti backend yang sudah jadi).
