# 10 — Laporan bandara & paginasi dokumen

**What to build:** Laporan pasca-handling layak diserahkan ke manajemen, dashboard checklist tidak membebani server saat dibuka, dan halaman review dokumen tetap responsif setelah data menumpuk.

Cacat inti — **laporan tidak lengkap**. FR-AIR-06 mensyaratkan laporan mencatat jam selesai, jumlah peserta, dan tour leader yang bertugas. Laporan yang dihasilkan menampilkan potongan pengenal internal sebagai nama batch, tanggal keberangkatan kosong karena nilainya tidak pernah diisi, dan nama tour leader diambil dari siapa pun yang terakhir menyentuh checklist alih-alih yang ditugaskan pada batch.

Cacat kedua — **endpoint baca menulis data**. Dashboard checklist menginisialisasi baris checklist setiap kali dibuka, dan antarmuka memuat ulang tiap sepuluh detik — sehingga satu tab terbuka menjalankan penulisan atas seluruh peserta batch enam kali per menit, dengan galatnya dibuang.

Cacat ketiga — **daftar dokumen tanpa batas**. Berbeda dari seluruh daftar lain di sistem, daftar dokumen tidak memiliki paginasi; dashboard admin bahkan mengambil seluruh baris hanya untuk menghitung jumlahnya.

**Blocked by:** 01 — Seam pengujian HTTP

**Status:** ready-for-agent

- [x] Laporan pasca-handling memuat nama paket dan tanggal keberangkatan yang sebenarnya, diambil dari batch
- [x] Laporan mencantumkan tour leader yang ditugaskan pada batch, bukan penyentuh checklist terakhir
- [x] Unduhan laporan tidak gagal diam-diam saat data ringkasan tidak tersedia; kondisi galat ditangani dan disampaikan
- [x] Inisialisasi checklist dipindahkan keluar dari endpoint baca sehingga membuka dashboard tidak lagi menulis data
- [x] Inisialisasi tetap terjadi pada momen yang tepat dalam alur, dan galatnya tidak lagi dibuang diam-diam
- [x] Daftar dokumen memperoleh paginasi pada filter, repository, dan antarmuka, mengikuti pola paginasi yang sudah dipakai daftar lain
- [x] Dashboard admin memperoleh jumlah dokumen menunggu review tanpa mengambil seluruh barisnya
- [x] Ringkasan "sudah disetujui dari total" pada halaman review dihitung dari seluruh dokumen peserta, bukan dari hasil filter yang sedang aktif
- [x] Ada test yang membuktikan membuka daftar checklist tidak mengubah data

## Comments

### Pelaksanaan (2026-08-09)

**Laporan membaca batch, bukan checklist.** `resolveBatchHeading` mengambil nama
paket, tanggal keberangkatan, dan tour leader **yang ditugaskan** dari batch.
Sebelumnya: nama batch adalah delapan karakter pertama UUID, tanggal
keberangkatan tidak pernah diisi sama sekali, dan nama tour leader diambil dari
`HandledByName` — siapa pun yang terakhir mencentang kotak. Laporan yang
diserahkan ke manajemen karena itu menyebut orang yang salah sebagai penanggung
jawab batch.

**Galat ringkasan tidak lagi ditelan.** `GetBatchProgress` dulu dipanggil dengan
`progress, _ :=`. Laporan yang diam-diam melaporkan nol lebih buruk daripada
laporan yang gagal — tour leader akan menyerahkannya dan mempercayainya. Sekarang
`500`, ada testnya.

**Inisialisasi checklist keluar dari endpoint baca.** Dashboard memanggil
`GET /admin/airport/checklist` setiap sepuluh detik dan endpoint itu memanggil
`InitForBatch` — satu tab terbuka menjalankan penulisan atas seluruh peserta
batch enam kali per menit, galatnya dibuang. Sekarang `POST
/admin/airport/checklist/init`, aksi tersendiri, galat disampaikan; frontend
memanggilnya sekali saat batch dipilih (dijaga `useRef` karena effect React
berjalan dua kali di development). Operasinya tetap idempoten
(`ON CONFLICT DO NOTHING`), jadi memanggilnya berulang tetap aman.

**Paginasi dokumen.** `document.Filter` mendapat `Page`/`PerPage`, `List`
mengembalikan `(rows, total, error)`, handler memakai `pageResponse` seperti
daftar lain, dan halaman review mendapat kontrol halaman. Klausa `WHERE`-nya
disusun sekali dan dipakai baik oleh `COUNT` maupun oleh kuerinya, jadi totalnya
tidak mungkin menggambarkan himpunan yang berbeda dari barisnya.

**Dashboard menghitung tanpa mengambil.** Tile "Review Dokumen" dulu mengambil
seluruh dokumen berstatus menunggu lalu memanggil `.length` atasnya. Sekarang
`per_page=1` dan membaca `meta.total`. `CountByStatus` juga ditambahkan ke
repository untuk pemanggil yang benar-benar hanya butuh angkanya.

**Ringkasan "sudah disetujui dari total" dihitung server.** Saat satu peserta
sedang direview, respons memuat `summary` atas **seluruh** dokumen peserta itu.
Menghitung baris di layar membuat angkanya sepakat dengan filter yang sedang
aktif dan dengan tidak ada yang lain — memfilter ke "disetujui" menampilkan
"2 dari 2 disetujui".

**Satu pembersihan kecil di berkas yang sama.** Notifikasi dokumen ditolak
memanggil `ListByParticipant(ctx, "")` lalu membuang hasilnya — kueri yang tidak
mengembalikan apa pun dan tidak dipakai siapa pun, tepat di tengah tiket tentang
kueri yang terbuang.
