# Code Review — Pintour Travel

**Tanggal:** 8 Juni 2026
**Scope:** Backend Go (Echo + pgx, hexagonal/DDD) + Frontend React/TS, fokus keamanan, correctness, dan maintainability.

## Ringkasan

Arsitektur bersih dan rapi (pemisahan domain / application / infrastructure / delivery konsisten), SQL seluruhnya parameterized, password pakai bcrypt cost 12, dan ada rate limiting di endpoint auth. Tapi ada **2 isu IDOR kritis di portal peserta** dan beberapa kelemahan auth/secret yang perlu diperbaiki sebelum dianggap production-ready. Untuk konteks skripsi, fondasinya sudah kuat — perbaikan di bawah sifatnya terarah, bukan rombak besar.

## Critical Issues

| # | File | Lokasi | Isu | Severity |
|---|------|--------|-----|----------|
| 1 | `internal/delivery/http/portal_handler.go` | `PortalInvoicePDF` (~129) | **IDOR / Broken Object Level Authorization.** Mengambil `c.Param("id")` lalu `invoices.GeneratePDF(ctx, id)` tanpa cek bahwa invoice itu milik peserta yang login (`pid`). Peserta mana pun bisa mengunduh invoice PDF orang lain dengan menebak/iterasi ID. | 🔴 Critical |
| 2 | `internal/delivery/http/portal_handler.go` | `PortalUploadProof` (~140) | **IDOR.** `pp.InvoiceID = c.Param("id")` tanpa validasi kepemilikan — peserta bisa menempelkan bukti bayar ke invoice peserta lain. | 🔴 Critical |

**Perbaikan #1 & #2:** scope-kan operasi ke participant. Mis. tambahkan `GeneratePDFForParticipant(ctx, invoiceID, pid)` / verifikasi `invoice.ParticipantID == pid` sebelum generate atau update; kalau tidak cocok kembalikan `404` (bukan `403`, agar tidak membocorkan eksistensi). Pola ini sudah benar di `PortalInvoices`/`PortalDocuments` yang memakai `pid` dari token — tinggal samakan.

## High

| # | File | Lokasi | Isu |
|---|------|--------|-----|
| 3 | `internal/delivery/http/user_handler.go` | `ForgotPassword` | Token reset = `fmt.Sprintf("%x", time.Now().UnixNano())` — **dapat ditebak** (berbasis waktu, bukan acak). Ganti dengan `crypto/rand` (mis. 32 byte → hex). Token juga hanya disimpan di map in-memory: hilang saat restart & tidak jalan multi-instance. Pindahkan ke DB/Redis dengan TTL. |
| 4 | `web/src/utils/auth.ts` + `web/src/utils/api.ts` | `authStorage`, interceptor | Backend sengaja pakai cookie **httpOnly** untuk cegah XSS (§19.1), tapi frontend menyimpan JWT admin di `localStorage` dan meng-attach sebagai `Bearer`. Ini **meniadakan proteksi XSS** tersebut — token bisa dicuri JS mana pun. Untuk admin, andalkan cookie httpOnly saja dan hapus penyimpanan token di localStorage. |
| 5 | `internal/config/config.go` | `Load()` JWT | Default fallback `JWT_SECRET = "supersecretkey_change_in_production"`. Jika env tak diset di production, server tetap boot dengan secret yang diketahui publik → **token bisa dipalsukan**. Tambahkan fail-fast: kalau `APP_ENV=production` dan secret kosong/masih default, `log.Fatal`. |

## Medium

| # | File | Isu | Kategori |
|---|------|-----|----------|
| 6 | `internal/delivery/http/helpers.go` | `serverErr` mengembalikan `err.Error()` mentah ke klien → bocor detail internal (query, path, dll). Log detail di server, kirim pesan generik ke klien. | Security |
| 7 | `internal/application/user/service.go` | `Login`: saat user tak ada, `bcrypt.CompareHashAndPassword` dilewati → **timing side-channel** untuk enumerasi email. Lakukan dummy bcrypt compare agar waktu respons konsisten. | Security |
| 8 | `internal/service/storage.go` | Content-Type ditentukan dari **ekstensi file**, bukan sniffing byte asli → bisa dispoof. Validasi MIME dari isi + whitelist tipe. | Security |
| 9 | `internal/delivery/http/router.go` | Rate limiter in-memory tidak pernah meng-evict IP lama → map tumbuh tanpa batas (memory leak perlahan) & tidak konsisten antar instance. OK untuk single-instance, tapi catat untuk scaling. | Performance |

## Low / Maintainability

| # | Lokasi | Isu |
|---|--------|-----|
| 10 | `seed.exe`, `seed-demo.exe` | **Ter-commit ke git** (terkonfirmasi via `git ls-files`) walau `.gitignore` punya `*.exe`. Hapus dari tracking: `git rm --cached seed.exe seed-demo.exe`. Binari tak boleh masuk repo. |
| 11 | `.github/workflows/deploy.yml` + `web/src/utils/api.ts` | SPA dideploy ke **GitHub Pages** (static host) sementara `baseURL: '/api/v1'` relatif — API tak akan reachable dari host Pages. Perlu base URL absolut ke backend atau host frontend bersama backend. |
| 12 | `internal/delivery/http/user_handler.go` | RBAC dicek manual per-handler (`claimRole != "super_admin"`). Aman, tapi rawan lupa di endpoint baru. Pertimbangkan middleware `RequireRole(...)` agar terpusat. |

## What Looks Good

- **Arsitektur bersih**: pemisahan layer hexagonal/DDD konsisten dan mudah diuji.
- **Tidak ada SQL injection**: WHERE dinamis dibangun dengan placeholder `$N`, nilai selalu lewat args (lead/package/invoice/participant repo). Bagus.
- **Hardening auth yang sudah benar**: bcrypt cost 12, cek signing-method HMAC di validasi JWT (cegah `alg=none`), cookie `HttpOnly`+`Secure`(prod)+`SameSite`, rate limit khusus auth (10/min), respons `forgot-password` generik (anti-enumerasi di titik itu).
- **RBAC** untuk user management benar-benar dicek (`super_admin`).
- **Soft delete**, migrasi terstruktur, dan **ada unit test** (invoice, lead, portal, storage, pdf, fonnte).

## Verdict

**Request Changes** — perbaiki #1 & #2 (IDOR) sebelum apa pun, lalu #3–#5. Sisanya bisa menyusul. Untuk skripsi, dua IDOR itu adalah temuan paling penting untuk ditutup dan paling enak dibahas di bab keamanan.
