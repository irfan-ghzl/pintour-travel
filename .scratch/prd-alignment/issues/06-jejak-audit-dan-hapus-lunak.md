# 06 — Jejak audit status lead & hapus lunak menyeluruh

**What to build:** Setiap perubahan status lead dapat ditelusuri: apa yang berubah, kapan, dan **siapa** yang melakukannya. Baris yang dihapus lunak benar-benar hilang dari seluruh tampilan, termasuk invoice.

Cacat inti: FR-CRM-02 mensyaratkan "Setiap perubahan status harus tercatat dengan timestamp dan pengguna yang melakukan perubahan", tetapi operasi pengubahan status pada repository membuang parameter pelakunya, dan tidak ada tabel riwayat sama sekali. Pekerjaan penjadwal yang meng-expire lead melakukan pembaruan massal tanpa atribusi apa pun.

Cacat kedua: ERD §14.1 menyatakan hapus lunak berlaku "pada seluruh tabel transaksional", padahal beberapa tabel tidak memilikinya — termasuk bukti bayar. Dan tabel invoice memiliki kolomnya tetapi repository-nya satu-satunya yang tidak pernah menyaringnya.

**Blocked by:** 01 — Seam pengujian HTTP; 05 — Integritas konversi lead

**Status:** ready-for-agent

- [x] Tersedia tabel riwayat status lead yang mencatat status lama, status baru, waktu, dan pelaku
- [x] Operasi pengubahan status berhenti membuang parameter pelaku, dan menulis baris riwayat dalam transaksi yang sama dengan perubahan statusnya
- [x] Pekerjaan penjadwal yang mengubah status secara massal menulis riwayat dengan atribusi sistem, sehingga tidak ada lubang di jejak audit
- [x] Konversi lead juga tercatat sebagai perubahan status di riwayat
- [x] Log aktivitas lead pada antarmuka admin menampilkan riwayat status berdampingan dengan catatan konsultan
- [x] Kolom hapus lunak dilengkapi pada tabel transaksional yang belum memilikinya, sehingga klaim §13.1 dan ERD §14.1 berlaku menyeluruh
- [x] Repository invoice menyaring baris terhapus lunak, menyamai perilaku repository lain
- [x] Invoice terhapus lunak tidak muncul di daftar, PDF, laporan, maupun pekerjaan penjadwal
- [x] Penomoran invoice tetap tidak menggunakan ulang nomor milik baris terhapus lunak
- [x] Ada test yang membuktikan perubahan status oleh dua pengguna berbeda menghasilkan dua baris riwayat dengan pelaku yang berbeda

## Comments

### Pelaksanaan (2026-08-09)

**Satu pernyataan, bukan dua.** `UpdateStatus` dan `MarkConverted` memakai rantai
CTE: `prev` membaca status lama dari snapshot pra-update, `updated` menulis status
baru, lalu `INSERT` ke `lead_status_history`. Satu pernyataan berarti jejaknya
tidak bisa hilang karena pernyataan kedua gagal — dan berarti repository tidak
perlu transaksi sendiri, yang penting karena ia bisa sedang berjalan di dalam
unit-of-work tiket 05. Keduanya berbagi satu konstanta SQL (`recordStatusChange`)
dengan dua sisipan: syarat tambahan pada `prev` dan kolom tambahan pada `UPDATE`.

**Atribusi sistem = `changed_by NULL`.** Penjadwal bukan pengguna. NULL tidak
rancu dengan "pengguna terhapus" karena pengguna dihapus lunak, tidak pernah
fisik, jadi `ON DELETE SET NULL` tak pernah terpicu. Pembacaan menerjemahkannya
ke `lead.SystemActor` ("Sistem") supaya jejaknya terbaca sama di manapun.

**`expireLeads` menulis riwayat.** `RETURNING` memberi nilai *baru*, bukan lama,
jadi kandidatnya dikumpulkan lebih dulu di CTE `candidates` — dari snapshot yang
sama — lalu dipakai dua kali: sebagai target `UPDATE` dan sebagai sumber status
lama untuk `INSERT`. Tetap satu pernyataan.

**Catatan sintetis dihapus.** Dulu setiap perubahan status juga menulis catatan
`[SISTEM] Status diubah menjadi 'x'`. Itu tidak bisa memuat status lama, ditulis
oleh panggilan kedua yang bisa gagal diam-diam, dan mengubur catatan konsultan di
antara entri yang tidak ditulis siapa pun. Riwayat menggantikannya; panel admin
menggabungkan keduanya jadi satu lini masa terurut waktu, dengan penanda warna
berbeda — jadi tetap "berdampingan" bagi pembaca, tapi dua catatan bagi sistem.

### Keputusan

**Tabel mana yang dapat hapus lunak.** Migrasi 004 sudah menangani
packages/leads/participants/invoices/documents. Yang ditambahkan 008:
`payment_proofs` dan `airport_checklists` — keduanya disebut user story 28 dan
keduanya transaksional (catatan keuangan dan catatan operasional keberangkatan).
Tabel sisanya sengaja dilewati: `wa_notifications` adalah log kirim, bukan entitas
bisnis; `lead_notes` mengikuti nasib leadnya lewat `ON DELETE CASCADE`; sisanya
adalah tabel master atau tabel lama yang tidak dipakai alur PRD.

**Penomoran invoice sengaja tidak menyaring `deleted_at`.** Ini satu-satunya
pembacaan tabel invoices yang begitu, dan itu memang tujuannya (§13.7): kalau
penomoran hanya menghitung baris hidup, nomor milik invoice yang dihapus akan
diberikan lagi ke invoice berikutnya. Ada testnya, karena dua kriteria tiket ini
saling tarik — "sembunyikan yang terhapus" dan "jangan pakai ulang nomornya".
Fake `NextSequence` diubah agar meniru kueri MAX-nya, bukan penghitung naik.

**Pembaca SQL langsung ikut disaring.** Dashboard, laporan, dan penjadwal membaca
`invoices`/`payment_proofs` tanpa lewat repository, jadi filternya ditambahkan di
sana juga — kalau tidak, "invoice terhapus tidak muncul di laporan" tidak berlaku.

### Yang belum terbukti test

Kueri CTE-nya sendiri hanya berjalan terhadap Postgres sungguhan. Fake meniru
perilakunya (menulis riwayat pada operasi yang sama, menolak konversi kedua),
jadi kontraknya terkunci di seam HTTP, tapi SQL-nya belum. Sama seperti
`postgres/uow.go` pada tiket 05 — keduanya menunggu seam kontrak Postgres.
