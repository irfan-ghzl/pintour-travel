# 02 — Ketahanan runtime

**What to build:** Server tetap hidup dan responsif saat menghadapi kondisi tidak wajar. Galat tak terduga pada pekerjaan latar tidak lagi menjatuhkan layanan bagi seluruh pengguna, permintaan berukuran tidak wajar ditolak sebelum menghabiskan memori, dan token reset password yang tidak pernah dipakai tidak menumpuk selamanya.

Latar belakang: runtime HTTP hanya memulihkan panic pada goroutine yang dibuatnya sendiri. Aplikasi ini menjalankan sekitar enam belas goroutine latar untuk notifikasi, OCR, chatbot, dan blast WhatsApp — panic di salah satunya mematikan seluruh proses.

**Blocked by:** 01 — Seam pengujian HTTP

**Status:** ready-for-agent

- [x] Permintaan yang memicu panic di handler menghasilkan respons galat terstruktur, bukan koneksi terputus, dan panic-nya tercatat di log
- [x] Panic pada pekerjaan latar tidak menjatuhkan proses; kejadiannya tercatat di log dengan jejak tumpukan
- [x] Pembungkus pemulih panic untuk pekerjaan latar disediakan satu kali sebagai utilitas bersama dan dipakai seluruh pemanggil — tidak disalin per tempat
- [x] Permintaan dengan body melebihi batas wajar ditolak dengan status yang sesuai, sebelum body dibaca habis ke memori
- [x] Batas ukuran body berlaku pada seluruh endpoint publik: form leads, login staf, login portal, dan kedua webhook
- [x] Token reset password yang telah kedaluwarsa dibersihkan otomatis secara berkala, mengikuti pola penyapu yang sudah dipakai pembatas laju
- [x] Permintaan lupa-password berulang tidak menyebabkan pertumbuhan memori tanpa batas

## Comments

### Pelaksanaan (2026-08-08)

**Utilitas bersama: `internal/safe`.** `safe.Go(label, fn)` menjalankan pekerjaan latar pada goroutine baru dengan pemulih panic; `safe.Recovered(label, fn)` membungkus tanpa membuat goroutine (dipakai job gocron yang goroutine-nya milik penjadwal); `safe.LogPanic` dipakai pemanggil yang harus memulihkan sendiri karena masih perlu bertindak — yaitu middleware HTTP yang wajib menjawab `500`. Label ikut ke log, jadi laporan panic menyebut pekerjaan apa yang mati tanpa harus membaca jejak tumpukan.

Seluruh 16 goroutine latar dipindahkan ke `safe.Go`. Satu `go` yang sengaja **tidak** dibungkus: goroutine listener di `cmd/server/main.go` — itu bukan pekerjaan latar melainkan layanannya sendiri; memulihkan panic di sana menyisakan proses hidup tanpa yang mendengarkan, menyembunyikan kegagalan alih-alih bertahan darinya. Alasannya ditulis sebagai komentar di tempatnya.

**Job penjadwal.** gocron v2 sebenarnya sudah memulihkan panic pada job (`executor.callJobWithRecover`), tapi nilai panic-nya dibuang tanpa jejak tumpukan. Job dibungkus `safe.Recovered` agar sebabnya sampai ke log dalam bentuk yang sama dengan pekerjaan latar lainnya.

**Middleware dipasang di `RegisterRoutes`, bukan di `main.go`.** Konsekuensinya harness tiket 01 ikut mendapatkannya tanpa perubahan apa pun, jadi perilaku yang diuji adalah perilaku yang dideploy — dan entry point baru tidak bisa lupa memasangnya.

**Keputusan batas body: satu middleware, plafon mengikuti jenis permintaan.** Rancangan awal dua tingkat (batas global longgar + batas ketat per rute publik) ternyata salah dan ketahuan oleh test: middleware global berjalan lebih dulu, jadi untuk permintaan tanpa `Content-Length` ia sudah menyangga 12 MB sebelum batas ketat sempat melihatnya. Gantinya satu middleware global: `multipart/*` mendapat 12 MB (unggah dokumen/bukti bayar §16.2 maksimal 5 MB plus bingkai multipart), selainnya 1 MB. Efek sampingnya seluruh rute — termasuk rute staf — ikut terbatasi, bukan hanya lima endpoint publik yang diminta tiket.

Konsekuensi yang perlu diketahui: klien bisa mengirim `Content-Type: multipart/form-data` ke endpoint JSON untuk mendapat plafon 12 MB. Tetap terbatas, dan dekode di handler tetap gagal — memori terjaga, hanya plafonnya lebih longgar untuk permintaan yang berbohong soal jenisnya.

**Permintaan tanpa `Content-Length` (chunked).** Yang mendeklarasikan ukuran ditolak dari deklarasinya, sebelum satu byte dibaca. Yang tidak, dibaca sampai batas lalu berhenti — sanggahan itu memakan paling banyak sebesar batasnya sendiri, dan jawabannya tetap `413` alih-alih muncul di handler sebagai galat dekode yang tak menjelaskan apa-apa. Tanpa ini batas ukuran hanya formalitas yang bisa dilewati dengan menghilangkan satu header.

**Token reset password.** `map` global + `sync.Mutex` telanjang diganti tipe `resetTokenStore` dengan `issue`/`consume`/`sweep`. Pemeriksaan kedaluwarsa pindah ke dalam `consume`, jadi tidak ada lagi pemanggil yang bisa lupa melakukannya. Penyapunya mengikuti pola pembatas laju (ticker + eviksi), dijalankan `RegisterRoutes` lewat penjaga "hanya sekali per store" — tanpa penjaga itu tiap harness test akan meninggalkan satu goroutine.

**Bukti merah-hijau.** `TestPanicInBackgroundWorkDoesNotKillTheProcess` diverifikasi dengan melepas pembungkusnya: binary test-nya benar-benar mati, bukan sekadar gagal. Itu memang bentuk kegagalan yang dijanjikan tiket ini.

**Detektor race tidak bisa dijalankan di mesin ini** — `-race` menuntut cgo dan gcc tidak tersedia. Perlu dijalankan di CI/Docker sebelum digabung.

**Coverage.** 24,4% → **34,8%**.

### Hasil code review (2026-08-08)

Enam temuan ditindaklanjuti:

1. **Penyapu dijadikan satu bentuk bersama.** `safe.Every(label, interval, fn)` menggantikan dua loop ticker identik (bucket rate limiter dan token reset) yang sebelumnya disalin. Sekarang klaim "mengikuti pola penyapu pembatas laju" bukan lagi kemiripan, melainkan fungsi yang sama.
2. **Loop penyapunya kini benar-benar diuji.** Sebelumnya badan ticker bisa dihapus tanpa ada test yang gagal — testnya hanya memanggil `sweep()` langsung. `TestSweeperEvictsWithoutBeingAsked` (interval 1 ms) dan `TestEveryKeepsTickingThroughAPanic` menutup celah itu.
3. **`sync.Once` menggantikan penghitung sweeper.** `TestSweeperStartsOnlyOnce` yang dulu menguji penghitung buatan sendiri ikut dihapus — sekarang jaminannya datang dari tipe, bukan dari assertion.
4. **CORS masuk ke dalam pemulih panic.** `e.Use(echoCORSAdapter(...))` di `main.go` dipindah ke *setelah* `RegisterRoutes`. Echo menjalankan middleware global sesuai urutan pendaftaran, jadi sebelumnya adapter CORS berada di luar jangkauan pemulih.
5. **Sanggahan chunked untuk multipart tidak lagi menyangga 12 MB.** Permintaan berkas tanpa `Content-Length` kini dibatasi mengalir lewat `http.MaxBytesReader` — jawabannya turun jadi galat handler alih-alih `413`, tapi biayanya satu megabyte, bukan dua belas. Jalur JSON tetap menyangga 1 MB demi jawaban `413` yang jelas.
6. **Helper respons pindah ke `helpers.go`** menyusul `badRequest`/`notFound`/`serverErr`, dan amplop `500` diekstrak jadi `serverErrEnvelope()` sehingga middleware dan `serverErr` tidak menyalin pesannya masing-masing.

Yang sengaja **tidak** dikerjakan, semuanya di luar cakupan tiket:

- **`rateLimiter()` masih menjalankan satu goroutine penyapu per pemanggilan** (3 per `RegisterRoutes`), tanpa cara menghentikannya. Di produksi `RegisterRoutes` sekali jalan; di test tiap harness meninggalkan tiga goroutine ticker yang tertidur. Memberi mereka siklus hidup menuntut pemilik yang saat ini tidak ada.
- **Fake di harness tidak punya mutex**, padahal tiket ini mulai menggerakkannya dari goroutine latar. Aman selama satu test membuat satu lead; perlu dibereskan sebelum ada test yang membuat dua secara bersamaan — dan `-race` di CI yang akan membuktikannya.
- **Pemicu OCR terduplikasi** antara `document_handler.triggerOCR` dan blok sebaris di `portal_handler`. Duplikasinya sudah ada sebelum tiket ini; keduanya hanya ikut dibungkus.
- **Jalan pintas plafon multipart**: klien bisa mengaku `Content-Type: multipart/form-data` untuk mendapat 12 MB di rute JSON mana pun. Tetap terbatas, hanya lebih longgar.

Dua temuan review ditolak karena salah alamat: `secureToken()` dan penyapu bucket rate limiter dianggap pekerjaan tiket ini, padahal keduanya sudah ada di working tree sebagai bagian refactor branch — tiket ini hanya mengubah `go func` penyapunya jadi `safe.Every`.
