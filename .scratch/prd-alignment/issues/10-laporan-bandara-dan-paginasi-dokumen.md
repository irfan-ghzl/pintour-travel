# 10 — Laporan bandara & paginasi dokumen

**What to build:** Laporan pasca-handling layak diserahkan ke manajemen, dashboard checklist tidak membebani server saat dibuka, dan halaman review dokumen tetap responsif setelah data menumpuk.

Cacat inti — **laporan tidak lengkap**. FR-AIR-06 mensyaratkan laporan mencatat jam selesai, jumlah peserta, dan tour leader yang bertugas. Laporan yang dihasilkan menampilkan potongan pengenal internal sebagai nama batch, tanggal keberangkatan kosong karena nilainya tidak pernah diisi, dan nama tour leader diambil dari siapa pun yang terakhir menyentuh checklist alih-alih yang ditugaskan pada batch.

Cacat kedua — **endpoint baca menulis data**. Dashboard checklist menginisialisasi baris checklist setiap kali dibuka, dan antarmuka memuat ulang tiap sepuluh detik — sehingga satu tab terbuka menjalankan penulisan atas seluruh peserta batch enam kali per menit, dengan galatnya dibuang.

Cacat ketiga — **daftar dokumen tanpa batas**. Berbeda dari seluruh daftar lain di sistem, daftar dokumen tidak memiliki paginasi; dashboard admin bahkan mengambil seluruh baris hanya untuk menghitung jumlahnya.

**Blocked by:** 01 — Seam pengujian HTTP

**Status:** ready-for-agent

- [ ] Laporan pasca-handling memuat nama paket dan tanggal keberangkatan yang sebenarnya, diambil dari batch
- [ ] Laporan mencantumkan tour leader yang ditugaskan pada batch, bukan penyentuh checklist terakhir
- [ ] Unduhan laporan tidak gagal diam-diam saat data ringkasan tidak tersedia; kondisi galat ditangani dan disampaikan
- [ ] Inisialisasi checklist dipindahkan keluar dari endpoint baca sehingga membuka dashboard tidak lagi menulis data
- [ ] Inisialisasi tetap terjadi pada momen yang tepat dalam alur, dan galatnya tidak lagi dibuang diam-diam
- [ ] Daftar dokumen memperoleh paginasi pada filter, repository, dan antarmuka, mengikuti pola paginasi yang sudah dipakai daftar lain
- [ ] Dashboard admin memperoleh jumlah dokumen menunggu review tanpa mengambil seluruh barisnya
- [ ] Ringkasan "sudah disetujui dari total" pada halaman review dihitung dari seluruh dokumen peserta, bukan dari hasil filter yang sedang aktif
- [ ] Ada test yang membuktikan membuka daftar checklist tidak mengubah data
