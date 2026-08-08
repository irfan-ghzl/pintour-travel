# 08 — Kelengkapan notifikasi

**What to build:** Peserta menerima pesan yang dijanjikan dokumen pada momen yang tepat, riwayat notifikasi dapat ditelusuri dengan benar, dan tidak ada penerima yang terlewat karena perannya.

Cacat inti — **pesan permintaan dokumen tidak terkirim**. FR-AUTO-04 dan diagram §14.5.3 menyatakan peserta menerima pesan WhatsApp berisi daftar dokumen setelah pembayaran dikonfirmasi. Pesan itu hanya dikirim oleh endpoint konfirmasi manual yang lama; jalur penyelesaian yang kini dipakai baik oleh persetujuan bukti bayar maupun oleh gateway tidak pernah mengirimnya.

Cacat kedua — **log pengingat pembayaran tidak akurat**. FR-INV-05 mensyaratkan pengingat pada hari ke-1, ke-3, dan ke-6. Ketiganya memang dikirim, tetapi pengingat hari ke-6 tercatat sebagai pengingat hari ke-1 karena jenis pesannya tidak ada. Pesannya juga dikirim tanpa nomor invoice, dan referensinya menunjuk ke peserta padahal jenis referensinya tertulis invoice.

Penghalang — **tidak ada satu pun notifikasi yang dapat diperiksa dari test**. Ditemukan saat membangun seam tiket 01, tidak diantisipasi spec. `FonnteService.Send` keluar pada `apiToken == ""` **sebelum** menulis baris notifikasinya, dan alamat `https://api.fonnte.com/send` di-hardcode. Akibatnya `h.Notifications` pada harness selalu kosong, sementara memberi harness token sungguhan justru membuat test memanggil jaringan. Lima kriteria di bawah menuntut pemeriksaan baris `WANotification`, jadi kriteria pertama harus dikerjakan lebih dulu.

Perbaikannya menyamai perlakuan gateway pembayaran di tiket 01, dan **tidak** mengubah perilaku deployment: alamat dasar dibuat dapat disetel saat konstruksi, lalu harness diberi token palsu plus server uji lokal. Dengan begitu `Send` berjalan penuh — menulis baris notifikasi, mengirim ke server uji, memperbarui statusnya — tanpa menyentuh jaringan. Jalur nir-operasi saat token kosong dibiarkan apa adanya.

**Blocked by:** 01 — Seam pengujian HTTP; 07 — Ketepatan pembayaran gateway

**Status:** ready-for-agent

- [ ] Alamat dasar klien WhatsApp dapat disetel saat konstruksi; nilai bawaannya tetap `https://api.fonnte.com`, dan perilaku nir-operasi saat token kosong tidak berubah. Harness tiket 01 memperoleh opsi yang menyalakannya terhadap server uji lokal, sejajar dengan `withMidtransServer`
- [ ] Jalur penyelesaian pembayaran yang dipakai bersama oleh konfirmasi manual dan gateway mengirim pesan permintaan dokumen via WhatsApp, memenuhi FR-AUTO-04 dan diagram §14.5.3
- [ ] Endpoint konfirmasi lama tidak lagi menjadi satu-satunya jalur yang mengirimnya, dan tidak ada pesan ganda saat kedua jalur dipakai berurutan
- [ ] Pengingat pembayaran hari ke-6 memiliki jenis pesan tersendiri dan tercatat berbeda dari hari ke-1
- [ ] Pengingat pembayaran menyebutkan nomor invoice peserta
- [ ] Catatan notifikasi menunjuk ke entitas yang benar sesuai jenis referensinya — referensi bertipe invoice menunjuk ke invoice, bukan ke peserta
- [ ] Pencarian penerima email admin mencakup peran super admin, disatukan menjadi satu utilitas bersama yang menggantikan tiga pemanggilan dengan cakupan berbeda-beda
- [ ] Dedup notifikasi jatuh tempo dicatat terlepas dari ada tidaknya nomor WhatsApp penerima, mengikuti pola penanda yang sudah dipakai peringatan kuota
- [ ] Peserta tanpa nomor WhatsApp tidak lagi menerima email jatuh tempo berulang setiap hari
- [ ] Email pemberitahuan lead baru memuat nama paket yang diminati — data diambil dari lead yang sudah tersimpan, bukan dari struktur sebelum penyimpanan
- [ ] Jumlah template direkonsiliasi terhadap tabel §17.1–17.3; template yang kurang pada baris WhatsApp→Peserta ditambahkan. **Baca §17.1 untuk menentukan identitasnya** — analisis sebelumnya baru sampai pada selisih agregat, belum pada template mana yang hilang
- [ ] Ada test yang membuktikan penyelesaian lewat kedua jalur menghasilkan pesan permintaan dokumen
