# 07 — Ketepatan pembayaran gateway

**What to build:** Uang yang masuk lewat gateway pembayaran dihitung tepat satu kali, pembayaran tidak hilang saat peserta memuat ulang halaman, transaksi yang ditandai mencurigakan ditahan, dan halaman invoice tetap terbuka apa pun status pembayarannya.

Cacat inti — **penghitungan ganda**: pemrosesan notifikasi gateway hanya berhenti lebih awal bila invoice sudah berstatus lunas. Untuk pembayaran sebagian, invoice masih berstatus menunggu bayar, sehingga setiap pengiriman ulang notifikasi dari gateway membuat bukti bayar baru dengan nominal yang sama. Empat kali kirim ulang atas satu uang muka bisa membuat invoice dinyatakan lunas.

Cacat kedua — **order tertimpa**: pengenal order gateway ditimpa setiap kali transaksi baru dibuat. Peserta yang membuka halaman pembayaran, memuat ulang, lalu menyelesaikan pembayaran di tab pertama akan mengirim notifikasi dengan pengenal order lama yang sudah tidak dikenali sistem — pembayarannya diterima gateway tapi tidak pernah tercatat. Diperparah oleh antarmuka yang membuat transaksi lewat kueri yang dapat diambil ulang otomatis saat jendela kembali fokus.

Backend dan frontend dalam satu tiket karena keduanya satu alur dan tidak dapat didemokan terpisah.

**Blocked by:** 01 — Seam pengujian HTTP

**Status:** ready-for-agent

- [x] Pemrosesan notifikasi gateway menjadi idempoten terhadap identitas transaksi, bukan terhadap status invoice
- [x] Notifikasi berulang untuk transaksi yang sama tidak pernah menghasilkan bukti bayar kedua, termasuk pada invoice yang baru terbayar sebagian
- [x] Relasi invoice ke order gateway menjadi satu-ke-banyak, sehingga pembayaran atas sesi yang dibuka lebih awal tetap dapat dicocokkan
- [x] Penjagaan status fraud menahan transaksi apa pun metode pembayarannya — tidak ada pengecualian berdasarkan jenis pembayaran
- [x] Galat basis data saat memproses webhook dibedakan dari kondisi "tidak ditemukan" dan dikembalikan sebagai galat, sehingga gateway mengirim ulang alih-alih menganggapnya selesai
- [x] Penyelesaian bukti bayar memverifikasi bahwa bukti tersebut milik invoice yang dirujuk
- [x] Antarmuka membuat transaksi pembayaran sebagai aksi eksplisit sekali jalan, bukan kueri yang dapat diambil ulang otomatis
- [x] Kosakata status invoice di frontend disamakan dengan skema, termasuk status menunggu konfirmasi gateway
- [x] Halaman invoice peserta tidak lagi gagal dirender saat invoice berstatus menunggu konfirmasi gateway; komponen status menangani status yang tidak dikenal dengan anggun
- [x] Ada test yang membuktikan lima kali pengiriman notifikasi settlement atas satu uang muka hanya menghasilkan satu bukti bayar dan tidak menandai invoice lunas
- [x] Ada test yang membuktikan notifikasi atas order lama tetap dicocokkan ke invoice yang benar

## Comments

### Pelaksanaan (2026-08-09)

**Kunci idempotensi = identitas transaksi.** Tabel `gateway_notifications`
menyimpan satu baris per notifikasi yang sudah diterapkan, dengan
`idempotency_key` UNIQUE berisi `order_id:transaction_id` (jatuh ke
`order_id:transaction_status` bila gateway tidak mengirim transaction_id).
Klaimnya `INSERT … ON CONFLICT DO NOTHING`, bukan baca-lalu-tulis: dua
pengiriman ulang yang tiba bersamaan akan sama-sama melihat kunci belum ada dan
sama-sama mencatat pembayaran. Indeks uniknya yang jadi wasit, dan ia hanya bisa
memilih satu.

**Klaim dan bukti bayarnya satu transaksi.** Dipakai unit-of-work tiket 05,
yang `Repos`-nya diperluas dengan `Invoices`, `Proofs`, dan `GatewayOrders`.
Alasannya spesifik: klaim yang berhasil tapi buktinya gagal ditulis akan
menghilangkan pembayaran **selamanya**, karena gateway tidak akan mengirim
notifikasi itu lagi. Kompensasi (klaim → proses → lepas klaim bila gagal)
menyisakan celah pada crash; transaksi tidak.

**Order jadi satu-ke-banyak.** Tabel `invoice_gateway_orders`, satu baris per
sesi Snap. `invoices.midtrans_order_id` tetap diisi untuk pembacaan "sesi saat
ini", tapi pencocokan notifikasi lewat tabel baru — dan `FindInvoiceIDByOrder`
punya `UNION ALL` ke kolom lama supaya sesi yang dibuat sebelum migrasi 009 dan
dibayar sesudahnya tetap ketemu.

**Webhook menjawab 500, bukan 200, saat sisi kita gagal.** Sebelumnya galat
pemrosesan dijawab `200 received_with_error` "supaya Midtrans tidak retry-storm"
— justru itu satu-satunya jawaban yang menghilangkan pembayaran, karena 200
berarti "sudah diterima" dan retry berhenti. Order tak dikenal tetap `404`
(final, retry tak akan mengubah apa pun).

**Penjagaan fraud tidak lagi mengecualikan bank transfer.** Syarat lamanya
`fraudStatus != "accept" && paymentType != "bank_transfer"` — artinya transaksi
bank transfer yang ditandai `challenge` lolos utuh. Sekarang status fraud saja
yang menentukan; ada test untuk tiga metode pembayaran.

**Satu jalur penyelesaian, dua pengikatan.** `settleApprovedProof` dipakai
persetujuan admin dan persetujuan otomatis gateway, dengan repository yang
berbeda: yang pertama memakai milik service (`ownRepos()`), yang kedua memakai
yang diberikan unit. Pola yang sama dengan `resolvePortalIdentity` di tiket 05.
Notifikasi dikumpulkan sebagai `settlement` di dalam unit dan dikirim setelah
commit — tidak ada yang diberi tahu tentang pembayaran yang di-rollback.

**Bukti bayar diverifikasi milik invoicenya.** `ReviewProofAndSettle` memuat
buktinya lebih dulu (`GetByID` dari tiket 04) dan menolak dengan
`ErrProofNotForInvoice` → `422` bila `InvoiceID`-nya beda. Tanpa itu, salah
tempel id menyelesaikan invoice yang keliru sementara kedua catatan tetap
terlihat wajar.

**Frontend.** `PaymentPage` membuat transaksi lewat `useMutation` sekali jalan,
bukan `useQuery` — sebagai query, react-query mengambilnya ulang setiap jendela
kembali fokus, jadi tiap pindah tab membuka order baru. Penjaganya `useRef`,
bukan state mutation, karena React menjalankan effect dua kali di development.
Kosakata status ditambah `menunggu_konfirmasi_gateway`, dan seluruh pembacaan
peta status lewat helper yang jatuh ke nilai netral untuk status tak dikenal —
supaya status baru di skema tidak lagi mengosongkan halaman invoice peserta.

### Yang belum terbukti test

Kueri `ON CONFLICT DO NOTHING` dan `UNION ALL` hanya berjalan terhadap Postgres
sungguhan; fake meniru perilakunya. Sama seperti tiket 05 dan 06 — menunggu seam
kontrak Postgres. Sisi frontend diverifikasi lewat typecheck dan pembacaan kode,
karena repo belum punya infrastruktur test frontend (dinyatakan di luar cakupan
oleh spec).
