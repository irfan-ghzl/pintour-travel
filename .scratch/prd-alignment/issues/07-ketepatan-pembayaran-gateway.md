# 07 — Ketepatan pembayaran gateway

**What to build:** Uang yang masuk lewat gateway pembayaran dihitung tepat satu kali, pembayaran tidak hilang saat peserta memuat ulang halaman, transaksi yang ditandai mencurigakan ditahan, dan halaman invoice tetap terbuka apa pun status pembayarannya.

Cacat inti — **penghitungan ganda**: pemrosesan notifikasi gateway hanya berhenti lebih awal bila invoice sudah berstatus lunas. Untuk pembayaran sebagian, invoice masih berstatus menunggu bayar, sehingga setiap pengiriman ulang notifikasi dari gateway membuat bukti bayar baru dengan nominal yang sama. Empat kali kirim ulang atas satu uang muka bisa membuat invoice dinyatakan lunas.

Cacat kedua — **order tertimpa**: pengenal order gateway ditimpa setiap kali transaksi baru dibuat. Peserta yang membuka halaman pembayaran, memuat ulang, lalu menyelesaikan pembayaran di tab pertama akan mengirim notifikasi dengan pengenal order lama yang sudah tidak dikenali sistem — pembayarannya diterima gateway tapi tidak pernah tercatat. Diperparah oleh antarmuka yang membuat transaksi lewat kueri yang dapat diambil ulang otomatis saat jendela kembali fokus.

Backend dan frontend dalam satu tiket karena keduanya satu alur dan tidak dapat didemokan terpisah.

**Blocked by:** 01 — Seam pengujian HTTP

**Status:** ready-for-agent

- [ ] Pemrosesan notifikasi gateway menjadi idempoten terhadap identitas transaksi, bukan terhadap status invoice
- [ ] Notifikasi berulang untuk transaksi yang sama tidak pernah menghasilkan bukti bayar kedua, termasuk pada invoice yang baru terbayar sebagian
- [ ] Relasi invoice ke order gateway menjadi satu-ke-banyak, sehingga pembayaran atas sesi yang dibuka lebih awal tetap dapat dicocokkan
- [ ] Penjagaan status fraud menahan transaksi apa pun metode pembayarannya — tidak ada pengecualian berdasarkan jenis pembayaran
- [ ] Galat basis data saat memproses webhook dibedakan dari kondisi "tidak ditemukan" dan dikembalikan sebagai galat, sehingga gateway mengirim ulang alih-alih menganggapnya selesai
- [ ] Penyelesaian bukti bayar memverifikasi bahwa bukti tersebut milik invoice yang dirujuk
- [ ] Antarmuka membuat transaksi pembayaran sebagai aksi eksplisit sekali jalan, bukan kueri yang dapat diambil ulang otomatis
- [ ] Kosakata status invoice di frontend disamakan dengan skema, termasuk status menunggu konfirmasi gateway
- [ ] Halaman invoice peserta tidak lagi gagal dirender saat invoice berstatus menunggu konfirmasi gateway; komponen status menangani status yang tidak dikenal dengan anggun
- [ ] Ada test yang membuktikan lima kali pengiriman notifikasi settlement atas satu uang muka hanya menghasilkan satu bukti bayar dan tidak menandai invoice lunas
- [ ] Ada test yang membuktikan notifikasi atas order lama tetap dicocokkan ke invoice yang benar
