# 03 — Aktivasi validasi masukan §19.3

**What to build:** Masukan pengguna yang tidak sah ditolak dengan pesan jelas sebelum menyentuh basis data. PRD §19.3 menyatakan "Semua input dari user divalidasi menggunakan `go-playground/validator` di layer handler" — validator sudah terpasang di aplikasi tapi tidak pernah dipanggil, dan tidak ada satu pun tag validasi di seluruh repositori. Tiket ini membuat klaim itu berlaku.

Dikerjakan dengan pola expand: pengikat JSON bersama mulai memanggil validator lebih dulu (tanpa tag terpasang, nol perubahan perilaku), lalu tag dipasang per domain sehingga tiap batch tetap hijau.

**Blocked by:** 01 — Seam pengujian HTTP

**Status:** ready-for-agent

- [ ] Pengikat JSON bersama menjalankan validator setelah dekode, sehingga seluruh handler memperoleh validasi tanpa perubahan satu per satu
- [ ] Galat validasi menghasilkan respons `400` yang menyebut field bermasalah, bukan galat internal
- [ ] Peran pengguna divalidasi terhadap empat peran resmi §5.3 saat membuat dan menyunting akun staf
- [ ] Tipe kamar divalidasi terhadap nilai yang dikenal skema
- [ ] Status invoice, dokumen, dan lead divalidasi terhadap kosakata yang dikenal skema
- [ ] Format email dan nomor telepon divalidasi pada endpoint yang menerimanya
- [ ] Nilai batas paginasi divalidasi sehingga permintaan tidak dapat meminta jumlah baris tak wajar
- [ ] Payload yang sebelumnya diterima dan sah tetap diterima — tidak ada regresi pada alur yang sudah berjalan
- [ ] Ada test yang membuktikan setiap kelompok tag di atas menolak nilai tidak sah dan menerima nilai sah
