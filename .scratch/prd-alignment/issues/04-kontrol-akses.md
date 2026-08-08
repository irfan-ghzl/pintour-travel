# 04 — Kontrol akses

**What to build:** Peserta hanya bisa membuka berkasnya sendiri, staf hanya bisa menandatangani akses ke berkas yang memang berhak dilihatnya, token portal dan token staf tidak bisa saling dipakai, dan organisasi tidak pernah kehilangan super admin terakhirnya.

Cacat inti: endpoint penandatanganan URL berkas privat menerima nama bucket dan path bebas dari klien tanpa pemeriksaan kepemilikan sama sekali — tersedia baik di jalur portal maupun jalur staf. Seorang peserta dapat menyusun path peserta lain dan memperoleh URL bertanda tangan ke paspor, KTP, atau bukti transfer milik orang itu.

**Catatan penting:** endpoint ini tidak muncul di matriks hak akses §5.3 maupun di FR manapun. Tingkat aksesnya karena itu **diputuskan**, bukan dibaca dari dokumen. Catat keputusan yang diambil, dan setelah selesai ajukan sebagai usulan baris tambahan pada §5.3 agar dokumen dan kode kembali sejajar.

Pembagian grup rute yang sudah ada **tidak** boleh diubah — keempat belas baris matriks §5.3 sudah diverifikasi cocok dengan kode, termasuk analytics yang memang terbuka untuk semua peran staf per FR-RPT-02.

**Blocked by:** 01 — Seam pengujian HTTP

**Status:** ready-for-agent

- [x] Penandatanganan URL berhenti menerima nama bucket dan path bebas dari klien; penanda tangan menerima pengenal sumber daya domain lalu me-resolve lokasi berkasnya dari basis data
- [x] Pada jalur portal, kepemilikan berkas diverifikasi terhadap identitas portal pada token; permintaan atas berkas milik peserta lain menghasilkan `404`, bukan `403`, agar keberadaannya tidak terungkap
- [x] Pada jalur staf, akses diverifikasi terhadap peran; peran yang tidak berhak melihat dokumen peserta tidak dapat menandatanganinya
- [x] Middleware token portal menolak token tanpa pengenal peserta, simetris dengan penjagaan yang sudah ada di middleware staf
- [x] Token staf yang dikirim sebagai token portal ditolak, dan sebaliknya
- [x] Operasi yang akan menurunkan peran atau menonaktifkan super admin terakhir ditolak dengan pesan yang menjelaskan alasannya
- [x] Keputusan tingkat akses untuk endpoint penandatanganan URL tercatat di berkas tiket ini, beserta usulan baris tambahan untuk §5.3
- [x] Ada test yang membuktikan peserta A tidak dapat memperoleh URL bertanda tangan atas berkas peserta B
- [x] Ada test yang membuktikan pembagian grup rute yang sudah ada tetap berperilaku sesuai matriks §5.3

## Comments

### Pelaksanaan (2026-08-08)

**Keputusan tingkat akses untuk endpoint penandatanganan URL.** Endpoint ini tidak
diatur PRD manapun, jadi tingkatnya diputuskan di sini:

| Jalur | Siapa yang boleh | Alasan |
|---|---|---|
| `GET /api/v1/admin/signed-url` | grup **ops** — `super_admin`, `admin` | Berkas yang dibuka endpoint ini persis berkas yang didaftarkan rute review dokumen dan rute invoice, dan keduanya sudah ops-only di §5.3. Penanda tangan tidak boleh memberi lebih banyak daripada daftarnya: `konsultan` dan `tour_leader` yang tidak bisa melihat daftar dokumen peserta karena itu juga tidak bisa menandatanganinya. |
| `GET /api/v1/portal/signed-url` | peserta, **hanya berkasnya sendiri** | Kepemilikan diperiksa terhadap identitas portal pada token, bukan terhadap peserta pada token saja — pelanggan lama punya satu akun untuk banyak perjalanan (v3.0), jadi "miliknya" mencakup seluruh tur milik portal user itu. Aturan yang sama sudah dipakai `PortalTripInvoicePDF`, dan sekarang keduanya memanggil satu helper. |

Berkas milik peserta lain dijawab `404`, bukan `403`: `403` mengonfirmasi bahwa
berkas itu ada, dan itulah sebagian besar yang dicari penyerang yang menebak
pengenal.

**Usulan baris tambahan untuk §5.3** (agar dokumen menutup endpoint ini):

| Fitur / Endpoint | super_admin | admin | konsultan | tour_leader |
|---|---|---|---|---|
| Akses berkas privat peserta — signed URL (`/admin/signed-url`) | ✓ | ✓ | ✗ | ✗ |

Baris portal tidak diusulkan karena §5.3 adalah matriks peran staf; kepemilikan
berkas peserta adalah aturan portal, bukan baris matriks.

**Pembagian grup rute yang sudah ada tidak berubah.** Keempat belas baris §5.3
tetap seperti semula — endpoint penandatanganan URL memang tidak termasuk di
dalamnya, jadi memindahkannya dari grup `admin` dasar ke `ops` tidak menyentuh
satu baris pun. `rbac_test.go` dari tiket 01 membuktikannya dan tetap hijau.

**Satu celah tambahan yang ditemukan saat pelaksanaan, dan ditutup di sini.**
Memeriksa kepemilikan terhadap baris di basis data belum cukup, karena peserta
yang menulis baris itu sendiri: `POST /portal/documents` dan
`POST /portal/invoices/:id/proofs` sama-sama menerima `file_path` dari klien.
Peserta A bisa mencatat objek milik peserta B sebagai dokumennya sendiri, lalu
meminta URL untuk dokumennya sendiri — IDOR yang sama, hanya satu langkah lebih
panjang. Sekarang kedua endpoint itu menolak `file_path` yang tidak berada di
bawah prefix penyimpanan peserta itu sendiri (`<participant_id>/…`, prefix yang
memang selalu dipakai `StorageService.Upload`), termasuk yang mencoba keluar
lewat `..`. URL absolut tetap diterima karena itulah bentuk fallback manual saat
storage tidak terkonfigurasi, dan URL absolut tidak membuka objek privat manapun.

**Perubahan kontrak API.** `GET /{admin,portal}/signed-url` sekarang menerima
`?type=document|payment_proof&id=<id sumber daya>`, bukan `?bucket=&path=`.
`web/src/utils/api.ts#openSignedFile` ikut berubah menjadi `(kind, id, path?)`;
argumen `path` hanya dipakai untuk fallback URL absolut, tidak pernah dikirim ke
server. Swagger diregenerasi.

**Perubahan yang merambat ke luar lapisan HTTP.**
`invoice.PaymentProofRepository` mendapat `GetByID` (bukti bayar tidak mencatat
pemilik, jadi pemiliknya harus diambil lewat invoice), dan service invoice
mendapat `GetProofWithOwner` yang menyatukan rantai bukti→invoice→peserta.
Ticket 07 kemungkinan besar memakai `GetByID` yang sama.

**Satu perubahan fake yang perlu dicatat.** `errNotFound` pada `fakes_test.go`
sekarang `sql.ErrNoRows` — yaitu galat yang benar-benar dikembalikan repository
Postgres, karena mereka meneruskannya tanpa dibungkus. Tanpa itu handler yang
membedakan "baris tidak ada" dari "kueri gagal" berperilaku beda di harness
dibanding di deployment. `ListByRole` pada fake user juga mulai menyaring akun
nonaktif, sama seperti kuerinya (`AND is_active=true`) — penting karena
penjagaan super admin terakhir menghitung lewat method itu.

**Harness.** Ditambah opsi `withStorageServer(url)`. Tidak perlu perubahan
produksi seperti gateway pembayaran di tiket 01: `NewStorageService` memang sudah
menerima alamat dasarnya dari konfigurasi.

### Temuan review yang diperbaiki (2026-08-08)

**Penjagaan `..` bisa dilewati dengan `%2f`.** Versi pertama penjagaan path
memecah string pada `/` literal, jadi `own-id/..%2fother-id/passport.jpg` lolos —
dan `net/http` meneruskan `%2f` apa adanya ke wire (diverifikasi), sehingga
penyimpanan tetap menerimanya sebagai penelusuran direktori. Sekarang path yang
mengandung `%` ditolak seluruhnya: `StorageService.Upload` menyusun path dari id,
timestamp, dan `sanitizeFilename` (hanya alfanumerik, `-`, `_`, `.`), jadi path
yang sah tidak pernah memuat `%`. Kasus tersandi ditambahkan ke test.

**Arti "identitas portal" dicatat eksplisit.** `ListByPortalUser` mencocokkan
`portal_user_id` **atau** `phone`, jadi dua peserta yang dibooking dengan satu
nomor WhatsApp terhitung satu identitas. Ini bukan pelebaran yang dibuat tiket
ini — `PortalMyTrips` dan `PortalTripInvoicePDF` sudah memakai aturan yang sama,
dan `PortalLogin` sendiri mengautentikasi lewat nomor telepon, jadi nomor yang
sama memang berarti akun yang sama. Sekarang konsekuensi itu tertulis di komentar
`portalOwnsParticipant` agar menjadi keputusan, bukan kebetulan.

**Rapikan sesuai konvensi sekitar:** prosa dipindah ke atas blok godoc (swag
menyerap baris tanpa tag setelah `@Router` ke dalam deskripsi operasi);
`roleSuperAdmin` dipakai konsisten di seluruh `user_handler.go`; konstanta bucket
dipakai juga oleh dua endpoint upload di berkas yang sama; `GetByID`/`GetByInvoice`
pada repo bukti bayar berbagi satu daftar kolom dan satu `scanProof`, mengikuti
pola `invoiceRepo.scan` yang sudah ada; sentinel error memakai bahasa Inggris
karena tidak pernah sampai ke klien; `portalOwnsParticipant` tidak lagi menerima
`ctx` dan `c` sekaligus; `openSignedFile` melempar galat alih-alih diam saat
respons tidak memuat url, supaya `.catch` di pemanggil benar-benar jalan.

### Diangkat sebagai pekerjaan terpisah

**Fallback URL manual bisa mengarahkan reviewer ke situs pilihan peserta.** Saat
storage tidak terkonfigurasi, `FileUpload` meminta peserta menempel tautan Google
Drive/Cloudinary, dan `file_path` berupa URL absolut memang diterima — perilaku
yang sudah ada sebelum tiket ini dan sengaja dipertahankan. Konsekuensinya:
tombol "Lihat" di halaman review admin membuka URL yang ditentukan peserta.
Itu **tidak** membuka objek privat manapun (objek di bucket privat tak terbaca
tanpa tanda tangan), jadi bukan kebocoran berkas — tapi tetap vektor phishing
terhadap admin. Menyempitkannya berarti mengubah alur fallback itu sendiri
(mis. allowlist host, atau menghapus fallback dan mewajibkan storage), yaitu
keputusan produk di luar sembilan kriteria tiket ini.

**`GET /portal/signed-url` belum punya pemanggil.** Portal frontend tidak pernah
memanggilnya (juga sebelum tiket ini). Endpointnya sekarang benar dan teruji;
memakainya di UI portal adalah pekerjaan tiket 09 (fitur portal peserta).
