# 01 — Seam pengujian HTTP

**What to build:** Kemampuan menulis test yang menembakkan permintaan HTTP sungguhan ke server aplikasi sebagai peran tertentu, lalu memeriksa responsnya — menembus routing, middleware, handler, service aplikasi, dan domain dalam satu jalur. Setelah tiket ini selesai, setiap perbaikan perilaku pada tiket berikutnya dapat dibuktikan dengan test yang gagal lebih dulu.

Seam ini sudah ada di kode produksi dan tidak menuntut refactor: fungsi pendaftaran rute menerima satu struct berisi seluruh dependensi, seluruh repository di dalamnya sudah berupa interface, dan adapter eksternal sudah punya mode nir-operasi saat konfigurasinya kosong.

Satu-satunya perubahan produksi yang diizinkan di tiket ini: alamat dasar klien gateway pembayaran dibuat dapat disetel saat konstruksi, agar sisi pembuatan transaksi dapat diuji terhadap server uji lokal.

**Blocked by:** None — can start immediately.

**Status:** ready-for-agent

- [x] Tersedia pembangun harness yang menghasilkan instance server siap-uji dari repository palsu dan service aplikasi asli, dengan nilai bawaan wajar sehingga test hanya perlu menyebut dependensi yang relevan baginya
- [x] Repository palsu tersedia untuk seluruh interface domain yang dipakai pendaftaran rute, mengikuti pola fake yang sudah ada pada test service invoice dan lead — bukan pustaka mock baru
- [x] Harness dapat menerbitkan sesi staf untuk keempat peran (`super_admin`, `admin`, `konsultan`, `tour_leader`) dan token portal untuk peserta, sehingga test dapat menyatakan "sebagai peran X" dalam satu baris
- [x] Adapter eksternal (WhatsApp, email, penyimpanan, gateway, chatbot, OCR) berjalan nir-operasi dalam test tanpa memerlukan konfigurasi, dan tidak melakukan panggilan jaringan
- [x] Alamat dasar klien gateway pembayaran dapat disetel saat konstruksi; nilai bawaannya tetap sama seperti sekarang untuk sandbox dan produksi
- [x] Ada test bukti-konsep untuk tiap grup rute yang membuktikan matriks hak akses §5.3 berlaku: peran yang berhak menerima respons sukses, peran yang tidak berhak menerima `403`
- [x] Ada test bukti-konsep yang membuktikan token portal ditolak di rute staf
- [x] Seluruh test berjalan tanpa Docker dan tanpa koneksi basis data
- [x] Dua celah yang diketahui didokumentasikan di dalam berkas harness: endpoint dashboard/analytics dan laporan memakai koneksi basis data langsung sehingga di luar jangkauan seam ini

## Comments

### Pelaksanaan (2026-08-08)

Berkas yang ditambahkan, seluruhnya di paket `httpdelivery` agar `portalClaims` yang tidak terekspor bisa dipakai menerbitkan token portal:

- `internal/delivery/http/harness_test.go` — `newHarness(t, opts...)`, penerbit token staf/portal, klien `as(role)` / `asParticipant(id)` / `anonymous()`, dan catatan **Known gaps**.
- `internal/delivery/http/fakes_test.go` — 17 fake, satu untuk tiap interface repository domain yang disentuh `RegisterRoutes`. Tiap fake menyematkan `fakeErr`, jadi `Fail(err)` membuat seluruh methodnya gagal (dipakai membedakan galat basis data dari "tidak ditemukan" di tiket 07), dan tiap fake punya `Seed` untuk mengisi baris tanpa lewat `Create`.
- `internal/delivery/http/rbac_test.go` — matriks §5.3 per grup rute, plus penolakan token portal dan permintaan tanpa kredensial.
- `internal/delivery/http/payment_gateway_seam_test.go`, `public_routes_test.go`, `harness_smoke_test.go`.
- `internal/service/payment_gateway_test.go` — mengunci nilai bawaan alamat dasar sandbox/produksi.

Perubahan produksi (satu-satunya yang diizinkan tiket ini): `NewMidtransService` menerima `...MidtransOption` dengan `WithMidtransBaseURL`. Pemanggil lama tidak berubah dan nilai bawaan sandbox/produksi identik.

**Keputusan pelaksanaan.** `Services.DB` diisi `*sql.DB` yang koneksinya selalu gagal (`sql.OpenDB` dengan connector buatan sendiri), bukan `nil`. Dengan `nil`, handler yang melewati seam repository akan panic dan menutupi status yang mau diperiksa; dengan handle yang gagal, middleware di depannya tetap bisa dibuktikan.

**Celah 1 ternyata lebih tajam dari dugaan spec.** `GetAnalytics` membuang galat kuerinya (`_ = h.db.QueryRowContext(...)`), jadi terhadap basis data yang tak terjangkau ia menjawab `200` dengan seluruh angka nol, bukan `500`. Karena itu testnya hanya menuntut "bukan `403`", bukan status tertentu — agar tiket yang berhenti menelan galat tersebut tidak memecahkan test kontrol akses. Kebiasaan menelan galat ini sejenis dengan user story 45 tentang laporan PDF yang gagal diam-diam; di luar cakupan tiket ini, layak diangkat saat mengerjakan tiket 10.

**Celah ketiga yang tidak diantisipasi spec — perlu diputuskan sebelum tiket 08.** `FonnteService.Send` keluar pada `apiToken == ""` **sebelum** menulis baris notifikasi, dan alamatnya di-hardcode. Akibatnya `h.Notifications` selalu kosong: tidak ada satu pun notifikasi yang bisa diperiksa lewat seam ini, padahal spec mendaftarkan "notifikasi" sebagai salah satu yang seharusnya tertutup seam HTTP. Memberi harness token Fonnte justru membuat test memanggil jaringan. Tiket 08 karena itu perlu memberi Fonnte alamat dasar yang dapat disetel — perlakuan sama seperti gateway pembayaran di tiket ini — atau memindahkan pemeriksaannya ke lapisan lain. Sudah dicatat sebagai gap 3 di `harness_test.go`.

Catatan kecil yang sudah didokumentasikan tapi bukan celah: `EmailService.Send` dengan kunci kosong mengembalikan galat, bukan nir-operasi seperti adapter lain. Seluruh pemanggil saat ini membuang galat itu, jadi tak ada rute yang berperilaku beda — tapi rute yang mulai meneruskannya akan berbeda antara harness dan deployment.

**Coverage.** Naik dari 12,6% ke **24,4%** hanya dari seam ini plus test bukti-konsep, tanpa satu pun test fungsi murni baru.

**Yang sengaja tidak dikerjakan.** Webhook Midtrans (`POST /api/v1/webhooks/midtrans`) belum punya test: ia butuh tanda tangan yang sah dan perilakunya adalah materi tiket 07, sementara grup rute publik sudah terbukti terjangkau lewat webhook Fonnte.
