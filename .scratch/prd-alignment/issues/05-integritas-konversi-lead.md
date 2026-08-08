# 05 — Integritas konversi lead

**What to build:** Konversi lead menjadi peserta berhasil sepenuhnya atau tidak meninggalkan jejak sama sekali. Peserta baru menerima password portalnya lewat WhatsApp tanpa admin harus mengetikkannya manual.

Cacat inti: identitas portal dan data peserta ditulis dalam dua operasi terpisah tanpa transaksi. Bila penulisan peserta gagal — misalnya karena tipe kamar tidak dikenal skema — identitas portal sudah tersimpan dengan hash password yang tidak pernah sampai ke siapa pun. Percobaan ulang kemudian mengenali baris yatim itu sebagai "pelanggan lama", tidak menerbitkan password baru, dan memberi tahu admin bahwa password lama tetap berlaku. Peserta terkunci permanen dan tidak ada alur reset password untuk akun portal.

Tiket ini juga memperkenalkan dukungan transaksi pada lapisan repository, yang akan dipakai ulang tiket 06.

**Blocked by:** 01 — Seam pengujian HTTP; 03 — Aktivasi validasi masukan §19.3

**Status:** ready-for-agent

- [ ] Lapisan repository memperoleh mekanisme unit-of-work sehingga beberapa operasi dapat dijalankan dalam satu transaksi, tanpa membocorkan tipe basis data ke lapisan aplikasi
- [ ] Mekanisme itu ditambahkan berdampingan dengan antarmuka repository yang ada, tidak menggantinya, sehingga tidak ada pemanggil lama yang rusak
- [ ] Pembuatan identitas portal, data peserta, dan penandaan lead sebagai terkonversi berjalan dalam satu transaksi
- [ ] Kegagalan pada tahap manapun mengembalikan seluruh perubahan; tidak ada identitas portal yatim yang tersisa
- [ ] Percobaan ulang setelah kegagalan berperilaku sama persis dengan percobaan pertama — peserta yang belum pernah punya akun tetap diperlakukan sebagai peserta baru dan menerima password baru
- [ ] Tipe kamar ditolak lebih dulu bila tidak dikenal skema, dengan pesan yang jelas, bukan gagal di tengah penulisan
- [ ] Password sementara dikirim otomatis ke WhatsApp peserta sesuai FR-PORTAL-01; nilai mentahnya tetap dikembalikan ke admin sebagai cadangan
- [ ] Pesan yang menjanjikan "password dikirim via pesan terpisah" kini benar-benar didahului atau diikuti pesan berisi password tersebut
- [ ] Pelanggan lama tetap memakai akun portal yang ada dan tidak menerima password baru, sesuai FR-CRM-08
- [ ] Ada test yang membuktikan kegagalan penulisan peserta tidak meninggalkan identitas portal
