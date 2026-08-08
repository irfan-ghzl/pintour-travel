# 12 — Kualitas & kebersihan lintas modul

**What to build:** Sekumpulan perbaikan mekanis yang masing-masing terlalu kecil untuk berdiri sebagai irisan sendiri, tapi bersama-sama menghilangkan sejumlah cara sistem gagal secara diam-diam.

**Pengecualian aturan irisan vertikal** — ini kumpulan perbaikan tersebar, bukan satu jalur lengkap melalui seluruh lapisan. Dikelompokkan agar tidak menghasilkan sepuluh tiket sepele. Tidak diblokir tiket manapun karena seluruhnya dapat diverifikasi lewat seam pengujian yang sudah ada.

**Blocked by:** None — can start immediately.

**Status:** ready-for-agent

- [ ] Dokumen PDF menampilkan karakter dan simbol non-ASCII dengan benar — tanda titik tengah, simbol hak cipta, butir daftar, dan nama beraksen. Modul laporan sudah melakukannya dengan benar dan menjadi acuan polanya
- [ ] Ekstraksi berlabel pada OCR memakai baris yang sama untuk pencocokan dan pemotongan, sehingga spasi di depan tidak lagi merusak nilai yang diambil
- [ ] Nama berkas unggahan disanitasi termasuk bagian ekstensinya, dan komponen path dikodekan saat disusun menjadi URL permintaan penyimpanan
- [ ] Pengunduhan berkas di antarmuka memasang elemen pemicu ke dokumen dan menunda pelepasan URL objek, sehingga unduhan berfungsi di seluruh peramban sasaran
- [ ] Pemotongan teks pada laporan tidak lagi memotong di tengah karakter multi-bita
- [ ] Utilitas format mata uang disatukan menjadi satu implementasi bersama, menggantikan empat salinan yang tersebar
- [ ] Utilitas pembacaan variabel lingkungan dengan nilai bawaan disatukan menjadi satu implementasi bersama, menggantikan tiga salinan dengan nilai bawaan yang berbeda-beda
- [ ] Utilitas pemotongan teks aman-Unicode disatukan menjadi satu implementasi bersama
- [ ] Parsing respons JSON penyimpanan memakai pustaka JSON standar, bukan pemindaian string buatan sendiri
- [ ] Pemilihan berkas pada antarmuka melepaskan URL pratinjau saat pilihan dibatalkan dan saat komponen dilepas
- [ ] Aturan pengabaian berkas biner pada Git benar-benar cocok; berkas biner hasil kompilasi yang saat ini tidak terabaikan menjadi terabaikan
- [ ] Kode mati dihapus: operasi repository dan utilitas yang tidak punya satu pun pemanggil
- [ ] Seluruh berkas yang disentuh diformat sesuai formatter bawaan bahasa
