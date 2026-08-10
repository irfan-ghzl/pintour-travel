# Debug Report — Perbaikan Temuan Code Review

**Tanggal:** 8 Juni 2026
**Konteks:** Tindak lanjut `docs/code-review-2026-06-08.md`. Semua isu Critical/High + beberapa Medium/Low diperbaiki.

## 1. IDOR — Portal Invoice PDF & Upload Bukti Bayar (Critical)

**Root cause:** `PortalInvoicePDF` dan `PortalUploadProof` memakai `c.Param("id")` langsung tanpa memverifikasi bahwa invoice itu milik peserta yang login. ID dari token (`pid`) tidak pernah dibandingkan dengan `invoice.ParticipantID` → Broken Object Level Authorization.

**Fix:**
- `internal/application/invoice/service.go`: tambah `var ErrNotOwned` + dua method ber-scope peserta: `GeneratePDFForParticipant(ctx, invoiceID, participantID)` dan `UploadProofForParticipant(ctx, pp, participantID)`. Keduanya mengambil invoice, lalu menolak (`ErrNotOwned`) jika `inv.ParticipantID != participantID`.
- `internal/delivery/http/portal_handler.go`: kedua handler kini mengambil `pid := portalParticipantID(c)` dan memanggil method ber-scope. `ErrNotOwned` dipetakan ke **404** (bukan 403) agar tidak membocorkan eksistensi invoice milik orang lain.

**Prevention:** endpoint portal baru wajib lewat method `*ForParticipant`. Tambahkan test yang mencoba akses invoice peserta lain dan meng-assert 404.

## 2. Token Reset Password Dapat Ditebak (High)

**Root cause:** `token := fmt.Sprintf("%x", time.Now().UnixNano())` — berbasis waktu, bukan acak; penyerang yang tahu kira-kira waktu request bisa mempersempit ruang tebakan.

**Fix:** `internal/delivery/http/user_handler.go` — helper `secureToken()` menghasilkan 256-bit dari `crypto/rand` (hex). `ForgotPassword` memakainya.

**Prevention:** jangan pakai timestamp/PRNG non-kripto untuk token rahasia. Catatan lanjutan: pindahkan `resetTokenStore` dari map in-memory ke DB/Redis agar tahan restart & multi-instance.

## 3. Enumerasi User via Timing Login (Medium)

**Root cause:** saat email tidak ada, `bcrypt.CompareHashAndPassword` dilewati → respons jauh lebih cepat daripada saat email ada, membocorkan keberadaan akun.

**Fix:** `internal/application/user/service.go` — saat `u == nil`, jalankan dummy bcrypt compare terhadap konstanta `dummyBcryptHash` (cost 12) agar waktu respons setara.

**Prevention:** uji timing untuk email ada vs tidak ada harus setara.

## 4. JWT_SECRET Default di Production (High)

**Root cause:** fallback `JWT_SECRET = "supersecretkey_change_in_production"`. Jika env tak diset di production, token bisa dipalsukan dengan secret yang diketahui publik.

**Fix:** `internal/config/config.go` — konstanta `defaultJWTSecret` + method `Validate()` yang mengembalikan error bila `APP_ENV=production` dan secret kosong/masih default. `cmd/server/main.go` memanggil `cfg.Validate()` dan `log.Fatalf` bila gagal (fail-fast saat boot).

**Prevention:** `Validate()` bisa diperluas untuk mengecek secret/credential wajib lain di production.

## 5. Kebocoran Detail Error Internal (Medium)

**Root cause:** `serverErr` mengirim `err.Error()` mentah ke klien.

**Fix:** `internal/delivery/http/helpers.go` — `serverErr` kini `c.Logger().Error(err)` di server dan mengirim pesan generik "Terjadi kesalahan internal" ke klien. Berlaku untuk semua handler sekaligus.

## 6. JWT Admin Disimpan di localStorage (High)

**Root cause:** backend pakai cookie httpOnly (anti-XSS), tapi frontend menyimpan JWT di `localStorage` + meng-attach `Bearer`, sehingga proteksi XSS jadi sia-sia.

**Fix:**
- `web/src/utils/auth.ts`: token **tidak lagi** disimpan; hanya info user non-sensitif untuk UI. `isLoggedIn()` mengecek keberadaan user, bukan token.
- `web/src/utils/api.ts`: interceptor Bearer dihapus; autentikasi murni lewat cookie httpOnly (`withCredentials: true`).
- Diverifikasi: tidak ada lagi referensi `getToken`/`pintour_token` di `web/src`, dan `tsc --noEmit` lulus.

## 7. Rate Limiter Memory Leak (Medium)

**Root cause:** map `buckets` per-IP tak pernah dibersihkan → tumbuh tanpa batas.

**Fix:** `internal/delivery/http/router.go` — goroutine ticker (per window) yang menghapus bucket kedaluwarsa.

## 8. Binari Ter-commit (Low)

**Root cause:** `seed.exe` & `seed-demo.exe` ter-track walau ada di `.gitignore`.

**Fix:** `git rm --cached seed.exe seed-demo.exe` (staged untuk dihapus dari tracking; file fisik tetap ada di lokal).

## Catatan Verifikasi

- **Frontend:** `npx tsc --noEmit` → **lulus (exit 0)**.
- **Backend (Go):** tidak bisa di-compile di lingkungan sandbox ini (Go tidak terpasang & sumber download/apt diblokir). Verifikasi dilakukan manual: semua simbol baru terdefinisi tepat satu kali, import yang diperlukan (`errors`, `crypto/rand`, `encoding/hex`, `fmt`) sudah ditambahkan, signature lama tidak diubah (test `TestUploadProof_TransitionsStatus` tetap valid). **Mohon jalankan di mesin Anda:** `go build ./... && go test ./...`.

## Belum Ditangani (opsional, dari review)

- #8 review: validasi MIME upload berbasis isi byte (bukan ekstensi).
- #11 review: mismatch deploy SPA ke GitHub Pages dengan `baseURL: '/api/v1'` relatif.
- #12 review: sentralisasi RBAC via middleware `RequireRole(...)`.
- Pindahkan reset-token store ke DB/Redis (lihat #2).
