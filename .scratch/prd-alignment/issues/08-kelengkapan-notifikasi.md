# 08 — Kelengkapan notifikasi

**What to build:** Peserta menerima pesan yang dijanjikan dokumen pada momen yang tepat, riwayat notifikasi dapat ditelusuri dengan benar, dan tidak ada penerima yang terlewat karena perannya.

Cacat inti — **pesan permintaan dokumen tidak terkirim**. FR-AUTO-04 dan diagram §14.5.3 menyatakan peserta menerima pesan WhatsApp berisi daftar dokumen setelah pembayaran dikonfirmasi. Pesan itu hanya dikirim oleh endpoint konfirmasi manual yang lama; jalur penyelesaian yang kini dipakai baik oleh persetujuan bukti bayar maupun oleh gateway tidak pernah mengirimnya.

Cacat kedua — **log pengingat pembayaran tidak akurat**. FR-INV-05 mensyaratkan pengingat pada hari ke-1, ke-3, dan ke-6. Ketiganya memang dikirim, tetapi pengingat hari ke-6 tercatat sebagai pengingat hari ke-1 karena jenis pesannya tidak ada. Pesannya juga dikirim tanpa nomor invoice, dan referensinya menunjuk ke peserta padahal jenis referensinya tertulis invoice.

Penghalang — **tidak ada satu pun notifikasi yang dapat diperiksa dari test**. Ditemukan saat membangun seam tiket 01, tidak diantisipasi spec. `FonnteService.Send` keluar pada `apiToken == ""` **sebelum** menulis baris notifikasinya, dan alamat `https://api.fonnte.com/send` di-hardcode. Akibatnya `h.Notifications` pada harness selalu kosong, sementara memberi harness token sungguhan justru membuat test memanggil jaringan. Lima kriteria di bawah menuntut pemeriksaan baris `WANotification`, jadi kriteria pertama harus dikerjakan lebih dulu.

Perbaikannya menyamai perlakuan gateway pembayaran di tiket 01, dan **tidak** mengubah perilaku deployment: alamat dasar dibuat dapat disetel saat konstruksi, lalu harness diberi token palsu plus server uji lokal. Dengan begitu `Send` berjalan penuh — menulis baris notifikasi, mengirim ke server uji, memperbarui statusnya — tanpa menyentuh jaringan. Jalur nir-operasi saat token kosong dibiarkan apa adanya.

**Blocked by:** 01 — Seam pengujian HTTP; 07 — Ketepatan pembayaran gateway

**Status:** ready-for-agent

- [x] Alamat dasar klien WhatsApp dapat disetel saat konstruksi; nilai bawaannya tetap `https://api.fonnte.com`, dan perilaku nir-operasi saat token kosong tidak berubah. Harness tiket 01 memperoleh opsi yang menyalakannya terhadap server uji lokal, sejajar dengan `withMidtransServer`
- [x] Jalur penyelesaian pembayaran yang dipakai bersama oleh konfirmasi manual dan gateway mengirim pesan permintaan dokumen via WhatsApp, memenuhi FR-AUTO-04 dan diagram §14.5.3
- [x] Endpoint konfirmasi lama tidak lagi menjadi satu-satunya jalur yang mengirimnya, dan tidak ada pesan ganda saat kedua jalur dipakai berurutan
- [x] Pengingat pembayaran hari ke-6 memiliki jenis pesan tersendiri dan tercatat berbeda dari hari ke-1
- [x] Pengingat pembayaran menyebutkan nomor invoice peserta
- [x] Catatan notifikasi menunjuk ke entitas yang benar sesuai jenis referensinya — referensi bertipe invoice menunjuk ke invoice, bukan ke peserta
- [x] Pencarian penerima email admin mencakup peran super admin, disatukan menjadi satu utilitas bersama yang menggantikan tiga pemanggilan dengan cakupan berbeda-beda
- [x] Dedup notifikasi jatuh tempo dicatat terlepas dari ada tidaknya nomor WhatsApp penerima, mengikuti pola penanda yang sudah dipakai peringatan kuota
- [x] Peserta tanpa nomor WhatsApp tidak lagi menerima email jatuh tempo berulang setiap hari
- [x] Email pemberitahuan lead baru memuat nama paket yang diminati — data diambil dari lead yang sudah tersimpan, bukan dari struktur sebelum penyimpanan
- [x] Jumlah template direkonsiliasi terhadap tabel §17.1–17.3; template yang kurang pada baris WhatsApp→Peserta ditambahkan. **Baca §17.1 untuk menentukan identitasnya** — analisis sebelumnya baru sampai pada selisih agregat, belum pada template mana yang hilang
- [x] Ada test yang membuktikan penyelesaian lewat kedua jalur menghasilkan pesan permintaan dokumen

## Comments

### Pelaksanaan (2026-08-09)

**Penghalangnya dibuka lebih dulu.** `NewFonnteService` menerima
`...FonnteOption` dengan `WithFonnteBaseURL`; nilai bawaannya tetap
`https://api.fonnte.com` dan jalur nir-operasi saat token kosong tidak
tersentuh. Harness dapat `withFonnteServer(url)`, sejajar dengan
`withMidtransServer`. Gap 3 pada `harness_test.go` sekarang ditandai **CLOSED**.
Efeknya: `Send` berjalan penuh — menulis baris notifikasi, mengirim, memperbarui
statusnya — tanpa menyentuh jaringan, jadi lima kriteria di bawahnya bisa
benar-benar diperiksa.

**Pesan permintaan dokumen pindah ke momen yang benar.** Bukan ke endpoint,
tapi ke tempat "invoice ini kini lunas" diputuskan — `notifyPaymentReceived`
saat `fullyPaid`. Dengan begitu ketiga jalur (persetujuan bukti bayar oleh
admin, penyelesaian otomatis gateway, konfirmasi manual) mengirimnya, karena
ketiganya melewati titik itu. `ConfirmPayment` berhenti mengirim sendiri, dan
kini keluar lebih awal bila invoice sudah lunas — jadi konfirmasi setelah bukti
bayar disetujui tidak mengulang instruksi yang sama. Ada testnya untuk kedua
jalur dan untuk urutan keduanya.

**Pengingat pembayaran diperbaiki tiga-tiganya sekaligus**, karena ketiganya ada
di satu baris pemanggilan: nomor invoice dulu dikirim string kosong, id yang
dipakai sebagai referensi adalah id **peserta** padahal `reference_type`-nya
"invoice", dan hari ke-6 tidak punya jenis pesan sehingga tercatat sebagai hari
ke-1. Sekarang penjadwal membaca invoicenya (bukan hanya pesertanya) lewat kueri
`s.db` — sejalan dengan gaya `automation.go` — dan meneruskan nomor serta id
invoice yang benar. `TypePaymentReminder6` ditambahkan.

**Penerima notifikasi admin disatukan.** `domainUser.AdminRoles` +
`domainUser.ListAdmins` menggantikan tiga pemanggilan yang cakupannya
berbeda-beda; dua di antaranya (`portal_handler.notifyAdmins` dan email lead
baru) hanya mencari peran `admin`, sehingga pemegang `super_admin` buta terhadap
lead baru, bukti bayar, dan dokumen masuk.

**Email lead baru memuat nama paket.** Dulu memakai struct `l` yang masuk ke
`Create`; `PackageName` diisi oleh join saat pembacaan, jadi nilainya selalu
kosong. Sekarang memakai `full` — lead yang sudah dibaca ulang — yang memang
sudah diambil beberapa baris di atasnya untuk keperluan lain.

**Dedup jatuh tempo tidak lagi bergantung pada nomor WA.** Penanda dedupnya
selama ini adalah efek samping dari pengiriman WA; peserta tanpa nomor tidak
pernah menghasilkan penanda, jadi emailnya terkirim ulang setiap hari selamanya.
`markNotified` menulis penanda itu terlepas dari ada tidaknya nomor, mengikuti
pola `QUOTA_WARNING` yang sudah dipakai `checkBatchQuota`.

### Rekonsiliasi §17.1–17.3 — temuannya terbalik dari dugaan

§17.1.1 mendaftar **17** template WhatsApp→Peserta. Ketujuh belasnya **sudah ada
di kode dan punya pengirim** — diperiksa satu per satu terhadap konstanta di
`internal/domain/notification/entity.go`. Jadi tidak ada template §17.1.1 yang
hilang; selisih agregat yang dicatat analisis sebelumnya tidak menunjuk ke
template yang kurang.

Yang justru terjadi: kode kini punya **dua template lebih banyak** dari §17.1.1,
dan keduanya dituntut FR-nya sendiri:

| Template | Dituntut oleh | Ada di §17.1.1? |
|---|---|---|
| `PORTAL_CREDENTIALS` | FR-PORTAL-01 (password sampai ke peserta) | tidak |
| `PAYMENT_REMINDER_6` | FR-INV-05 (pengingat H+1, H+3, **H+6**) | tidak |

Ini satu-satunya tempat dalam penyelarasan ini di mana kode tidak bisa
"menyesuaikan ke dokumen": menghapus keduanya berarti melanggar FR yang lain.
Karena itu **diusulkan sebagai amandemen dokumen**, bukan diperbaiki di kode —
perlakuan yang sama dengan baris §5.3 pada tiket 04:

**Usulan tambahan §17.1.1 (Notifikasi WA — Peserta):**

| Kode | Trigger | Isi Pesan |
|---|---|---|
| `PORTAL_CREDENTIALS` | Lead dikonversi menjadi peserta baru | Link portal + username + password sementara |
| `PAYMENT_REMINDER_6` | H+6 invoice belum dibayar | Reminder terakhir sebelum jatuh tempo + nomor invoice |

**Usulan §17.3 (Rekapitulasi):** baris WhatsApp→Peserta **17 → 19**, total
**35 → 37 template**.
