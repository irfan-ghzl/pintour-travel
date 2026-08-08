# 06 — Jejak audit status lead & hapus lunak menyeluruh

**What to build:** Setiap perubahan status lead dapat ditelusuri: apa yang berubah, kapan, dan **siapa** yang melakukannya. Baris yang dihapus lunak benar-benar hilang dari seluruh tampilan, termasuk invoice.

Cacat inti: FR-CRM-02 mensyaratkan "Setiap perubahan status harus tercatat dengan timestamp dan pengguna yang melakukan perubahan", tetapi operasi pengubahan status pada repository membuang parameter pelakunya, dan tidak ada tabel riwayat sama sekali. Pekerjaan penjadwal yang meng-expire lead melakukan pembaruan massal tanpa atribusi apa pun.

Cacat kedua: ERD §14.1 menyatakan hapus lunak berlaku "pada seluruh tabel transaksional", padahal beberapa tabel tidak memilikinya — termasuk bukti bayar. Dan tabel invoice memiliki kolomnya tetapi repository-nya satu-satunya yang tidak pernah menyaringnya.

**Blocked by:** 01 — Seam pengujian HTTP; 05 — Integritas konversi lead

**Status:** ready-for-agent

- [ ] Tersedia tabel riwayat status lead yang mencatat status lama, status baru, waktu, dan pelaku
- [ ] Operasi pengubahan status berhenti membuang parameter pelaku, dan menulis baris riwayat dalam transaksi yang sama dengan perubahan statusnya
- [ ] Pekerjaan penjadwal yang mengubah status secara massal menulis riwayat dengan atribusi sistem, sehingga tidak ada lubang di jejak audit
- [ ] Konversi lead juga tercatat sebagai perubahan status di riwayat
- [ ] Log aktivitas lead pada antarmuka admin menampilkan riwayat status berdampingan dengan catatan konsultan
- [ ] Kolom hapus lunak dilengkapi pada tabel transaksional yang belum memilikinya, sehingga klaim §13.1 dan ERD §14.1 berlaku menyeluruh
- [ ] Repository invoice menyaring baris terhapus lunak, menyamai perilaku repository lain
- [ ] Invoice terhapus lunak tidak muncul di daftar, PDF, laporan, maupun pekerjaan penjadwal
- [ ] Penomoran invoice tetap tidak menggunakan ulang nomor milik baris terhapus lunak
- [ ] Ada test yang membuktikan perubahan status oleh dua pengguna berbeda menghasilkan dua baris riwayat dengan pelaku yang berbeda
