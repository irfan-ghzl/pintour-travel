# 02 — Menyaring peserta berdasarkan tanggal keberangkatan

**What to build:** Admin membuka halaman Peserta dan menyaring daftarnya lewat
pemilih keberangkatan, bukan kolom UUID. Ia bisa menjawab pertanyaan sesederhana
"siapa saja yang berangkat 24 September" tanpa keluar dari aplikasi.

Memakai ulang endpoint daftar batch dan komponen pemilih dari tiket 01.

**Blocked by:** 01 — Airport Handling bisa dipakai tanpa mengetahui UUID.

**Status:** done

- [x] Kolom isian `Filter Batch ID...` tidak ada lagi di halaman Peserta
- [x] Pemilih menampilkan tanggal keberangkatan beserta nama paketnya, sehingga
      dua keberangkatan berdekatan dari paket berbeda tetap bisa dibedakan
- [x] Daftar peserta menyusut sesuai batch yang dipilih
- [x] Filter yang sedang aktif terbaca jelas dan bisa dihapus dengan satu klik,
      mengembalikan seluruh peserta tanpa memuat ulang halaman
- [x] Pilihan filter bertahan saat berpindah halaman daftar
- [x] Parameter yang dikirim ke endpoint peserta tetap `batch_id` — hanya cara
      memilihnya yang berubah

## Catatan implementasi

Pemilih memuat `upcoming=true` seperti disebut spec, tapi halaman menambahkan satu
centang "termasuk keberangkatan yang sudah lewat". Alasannya bukan user story 20
— itu sudah ditutup oleh `upcoming=true` sebagai bawaan — melainkan agar
perubahan ini tidak **mengurangi** kemampuan: kolom UUID lama, betapapun tidak
terpakainya, secara teori bisa menyaring peserta perjalanan tahun lalu, dan
pemilih yang hanya memuat keberangkatan mendatang tidak bisa.

Daftar dibatasi 100 baris karena `queryPageSize` meng-clamp di `maxPerPage`
(§19.3). Urutannya terdekat lebih dulu, jadi yang terpotong adalah yang paling
tidak relevan — tapi pemotongannya tetap diberitahukan ("… N lainnya tidak muat
di daftar ini") alih-alih terjadi diam-diam. Ditemukan saat review.

Isian batch pada label pemilih disebut "kursi terisi", bukan "peserta".
Alasannya ditemukan saat pengujian di peramban: halaman bandara menampilkan
"Total Peserta" di sebelahnya, dan angkanya berbeda karena checklist bandara
hanya dibuat untuk peserta yang portalnya sudah aktif. Kedua angka benar; menyebut
keduanya "peserta" membuat perbedaannya terbaca sebagai salah hitung.
