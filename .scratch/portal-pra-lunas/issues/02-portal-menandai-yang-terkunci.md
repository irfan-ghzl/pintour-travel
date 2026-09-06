# 02 — Portal menandai menu yang belum terbuka, bukan menyembunyikannya

**What to build:** Peserta yang belum lunas melihat portal yang jujur: invoice,
tombol bayar, unggah bukti, dan dokumen bisa dipakai; itinerary, briefing, dan
tour leader tampil terkunci beserta alasannya. Setelah pembayaran dikonfirmasi,
ketiganya terbuka tanpa peserta perlu melakukan apa pun.

Menyembunyikan menu membuat peserta menyangka fiturnya tidak ada. Menampilkannya
sebagai terkunci memberi tahu bahwa ia sedang menunggu pembayaran.

Penguncian di antarmuka ini kenyamanan, bukan kontrol akses — yang menegakkan
aturan tetap API dari tiket 01.

**Blocked by:** 01 — Peserta yang belum lunas bisa masuk portal dan membayar online.

**Status:** done

- [x] Menu itinerary, briefing, dan tour leader tampil dalam keadaan terkunci
      bagi peserta yang belum lunas, tidak bisa ditekan, dan menyebutkan
      alasannya
- [x] Membuka halaman terkunci lewat URL langsung menampilkan penjelasan yang
      sama, bukan halaman kosong atau galat mentah
- [x] Halaman invoice menampilkan tombol bayar online dan unggah bukti transfer
      bagi peserta yang belum lunas
- [x] Status bukti transfer yang diunggah peserta terbaca — menunggu, disetujui,
      atau ditolak beserta alasannya
- [x] Sisa tagihan terbaca bila sudah ada bukti yang disetujui sebagian
- [x] Setelah pembayaran dikonfirmasi, ketiga menu terbuka tanpa login ulang
- [x] Halaman dokumen bisa dipakai mengunggah sejak sebelum lunas

## Comments

Terverifikasi di peramban pada sistem berjalan (13 Agu 2026), sebagai peserta
yang belum membayar:

- Menu samping: Itinerary dan Briefing tampil sebagai elemen tak bisa ditekan
  bertanda gembok, bertuliskan "Terbuka setelah pembayaran dikonfirmasi";
  Dashboard, Riwayat, Invoice, Dokumen, Asuransi, dan Profil tetap tautan biasa.
- `/portal/itinerary` dan `/portal/briefing` dibuka langsung lewat URL
  menampilkan penjelasannya beserta tombol "Lihat invoice & bayar", bukan
  halaman kosong atau "gagal memuat".
- Halaman invoice menampilkan "Bayar Online" dan "Upload bukti di sini".
- Dengan satu bukti Rp 5.000.000 disetujui dan satu Rp 2.000.000 ditolak:
  halaman membaca "Sudah Dibayar Rp 5.000.000", "Sisa Tagihan Rp 13.900.000",
  status kedua bukti, dan alasan penolakannya.
- Setelah pembayaran dikonfirmasi, tanpa login ulang, kedua menu kembali menjadi
  tautan biasa.

Dua hal ikut diperbaiki karena baru terlihat begitu peserta belum lunas benar-benar
bisa masuk portal:

1. **Tombol bayar hilang tepat setelah pembayaran sebagian.** Keduanya bergantung
   pada status `diterbitkan`/`menunggu_bayar`; bukti sebagian yang disetujui
   memindahkan invoice ke `dibayar`, sehingga peserta yang baru membayar separuh
   kehilangan satu-satunya cara melunasi sisanya. Sekarang keduanya membaca sisa
   tagihan.
2. **Halaman pembayaran menampilkan nominal yang salah.** Ia menulis total
   tagihan, sementara Midtrans hanya menagih sisanya — dua angka berbeda pada
   satu alur yang sama.
