# 03 — Pesan WhatsApp menjanjikan yang memang sudah bisa dilakukan

**What to build:** Pesan kredensial portal yang diterima peserta saat konversi
menyebutkan apa yang bisa dilakukan sekarang — membayar, mengunggah bukti
transfer, mengunggah dokumen — dan apa yang menyusul setelah pembayaran
dikonfirmasi.

Sebelum tiket 01, pesan itu mengajak peserta "gunakan portal untuk mengunggah
dokumen" pada saat portalnya masih tertutup; pada data uji ia tiba 17 menit
sebelum portal terbuka. Setelah 01, ajakan itu benar — dan pesannya perlu
menyebut kemampuan terpentingnya, yaitu membayar.

**Blocked by:** 01 — Peserta yang belum lunas bisa masuk portal dan membayar online.

**Status:** done

- [x] Pesan kredensial portal menyebutkan peserta dapat membayar dan mengunggah
      bukti transfer lewat portal
- [x] Pesan itu menyebutkan itinerary dan briefing terbuka setelah pembayaran
      dikonfirmasi, sehingga tidak ada yang menyangka fiturnya rusak
- [x] Pesan invoice tidak lagi mengatakan PDF-nya baru tersedia setelah
      pembayaran, karena sekarang bisa diunduh sejak invoice terbit
- [x] Pesan portal aktif tetap dikirim saat pembayaran dikonfirmasi, dan isinya
      menyesuaikan: yang bertambah adalah isi perjalanan, bukan akses portalnya
- [x] Isi pesan diuji lewat perenderan templatnya, mengikuti cara pesan
      kredensial sudah diuji hari ini — tanpa perlu mengirim lewat gateway

## Comments

Isi pesan diuji lewat perenderan templatnya (`internal/service/portal_messages_test.go`),
dan pesan yang benar-benar terkirim diperiksa pada sistem berjalan (13 Agu 2026)
lewat baris `wa_notifications` milik peserta uji:

- `PORTAL_CREDENTIALS` — "akun portal peserta Anda sudah aktif dan bisa dipakai
  sekarang", diikuti daftar yang bisa dilakukan (invoice, bayar online, bukti
  transfer, dokumen) dan kalimat "Itinerary, briefing, dan kontak tour leader
  terbuka setelah pembayaran dikonfirmasi".
- `INVOICE_SENT` — "📄 Unduh Invoice: <portal>/portal/invoices", tanpa lagi
  menunda PDF sampai pembayaran.
- `PORTAL_ACTIVATED` — dijadikan "Pembayaran Dikonfirmasi!" dan menyebut yang
  bertambah adalah isi perjalanan, bukan akses portalnya. Diuji lewat
  perenderan templat saja: mengirimkannya sungguhan berarti mengirim WhatsApp ke
  nomor uji, dan itu ditahan dengan sengaja.
