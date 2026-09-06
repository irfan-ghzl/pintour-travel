# Spec: Menghapus UUID dari antarmuka admin

Status: done

## Problem Statement

Empat halaman admin meminta operator mengetikkan UUID yang tidak pernah
ditampilkan aplikasi di mana pun. Tidak ada satu layar pun yang mencetak
`participant_id` atau `batch_id`, dan tidak ada tombol salin. Untuk melanjutkan
pekerjaannya, admin harus keluar dari aplikasi — membuka basis data atau Swagger
— lalu kembali membawa nilai yang disalin dengan tangan.

Dampaknya berbeda-beda per halaman:

- **Peserta** — filter batch memakai `Filter Batch ID...`. Admin tidak bisa
  menjawab pertanyaan sesederhana "siapa saja yang berangkat 24 September".
- **Review Dokumen** — filter memakai `Filter Participant ID...`. Antrean review
  tidak bisa dipersempit ke satu peserta tanpa UUID-nya.
- **Invoice** — pembuatan invoice meminta `UUID peserta...` dan `UUID batch...`.
- **Airport Handling** — paling parah: seluruh halaman kosong sampai Batch ID
  diketik. UUID itu bukan sekadar penyaring, melainkan **satu-satunya pintu
  masuk** ke fitur tersebut. Tanpa mengetahui UUID, fitur penanganan bandara
  tidak bisa dipakai sama sekali, dan tidak ada petunjuk cara memperolehnya.

Kesalahan yang mudah terjadi dan tidak dicegah apa pun: menempelkan UUID batch
dari paket yang berbeda, atau UUID peserta yang salah — keduanya diterima tanpa
peringatan.

## Solution

Setiap tempat yang hari ini meminta UUID diganti pemilih yang berbicara dalam
istilah domain: batch disebut sebagai **tanggal keberangkatan beserta nama
paket**, peserta disebut sebagai **nama dan nomor WhatsApp**.

Setelah perubahan ini, tidak ada satu pun layar admin yang menuntut operator
mengetahui UUID. Pola yang sama sudah diterapkan pada konversi lead menjadi
peserta dan terbukti bekerja; spec ini merampungkan sisanya.

Khusus Airport Handling, halaman tidak lagi dibuka dalam keadaan kosong.
Keberangkatan terdekat dipilih lebih dulu, sehingga tour leader yang membukanya
di bandara langsung melihat checklist yang relevan.

### Catatan atas pertanyaan "invoice tidak bisa otomatis seperti peserta?"

Invoice **sudah** dibuat otomatis. Saat lead berstatus `deal` dikonversi menjadi
peserta, sistem menerbitkan invoice sendiri dengan nominal `harga batch sesuai
tipe kamar × jumlah pax`, jatuh tempo tujuh hari, lalu mengirimkan WhatsApp dan
surel kepada peserta. Terverifikasi pada sistem berjalan: konversi lead 2 pax
paket Umroh Plus Istanbul menghasilkan `INV-202608-0006` senilai Rp 59.000.000
tanpa ada yang menekan tombol apa pun.

Formulir invoice manual itu **bukan** duplikat dari proses tersebut, melainkan
untuk invoice **tambahan** di luar paket — upgrade kamar, biaya penalti,
layanan ekstra — yang nominalnya tidak bisa diturunkan dari batch. Karena itu
formulirnya dipertahankan, dan yang diperbaiki hanya cara memilih peserta dan
batch-nya.

## User Stories

1. Sebagai admin, saya ingin menyaring daftar peserta berdasarkan tanggal
   keberangkatan, sehingga saya bisa melihat siapa saja yang berangkat pada
   batch tertentu tanpa mengetahui UUID-nya.
2. Sebagai admin, saya ingin pemilih batch menampilkan nama paket beserta
   tanggalnya, sehingga saya bisa membedakan dua keberangkatan yang tanggalnya
   berdekatan dari paket berbeda.
3. Sebagai admin, saya ingin pemilih batch mengurutkan keberangkatan terdekat di
   atas, sehingga batch yang sedang saya urus tidak tenggelam di antara puluhan
   batch lampau.
4. Sebagai admin, saya ingin bisa mengosongkan filter batch dengan satu klik,
   sehingga saya kembali melihat seluruh peserta tanpa memuat ulang halaman.
5. Sebagai admin, saya ingin filter batch yang sedang aktif terbaca jelas di
   layar, sehingga saya tidak salah menyimpulkan bahwa daftar peserta hanya
   berisi sedikit orang.
6. Sebagai admin, saya ingin menyaring antrean review dokumen berdasarkan nama
   peserta, sehingga saya bisa memeriksa kelengkapan satu orang tanpa menyalin
   UUID dari basis data.
7. Sebagai admin, saya ingin mengetikkan sebagian nama atau nomor WhatsApp untuk
   menemukan peserta, sehingga saya tidak perlu mengingat ejaan lengkapnya.
8. Sebagai admin, saya ingin pemilih peserta menampilkan nomor WhatsApp di
   samping nama, sehingga dua peserta bernama sama tetap bisa dibedakan.
9. Sebagai admin, saya ingin pemilih peserta memberi tahu bila pencarian tidak
   menemukan siapa pun, sehingga saya tahu kata kuncinya yang salah dan bukan
   sistemnya yang rusak.
10. Sebagai admin, saya ingin memilih peserta saat membuat invoice tambahan,
    sehingga saya tidak salah menagih orang lain.
11. Sebagai admin, saya ingin batch pada formulir invoice terisi mengikuti
    peserta yang saya pilih, sehingga saya tidak perlu memilihnya dua kali dan
    tidak mungkin memasangkan peserta ke batch yang bukan miliknya.
12. Sebagai admin, saya ingin melihat nama paket dan tanggal keberangkatan
    peserta yang saya pilih sebelum menyimpan invoice, sehingga saya bisa
    memastikan orangnya benar.
13. Sebagai admin, saya ingin tetap bisa menerbitkan invoice tambahan dengan
    nominal bebas, sehingga biaya upgrade kamar dan penalti tetap bisa ditagih.
14. Sebagai tour leader, saya ingin halaman Airport Handling langsung
    menampilkan keberangkatan terdekat saat dibuka, sehingga saya bisa mulai
    bekerja di bandara tanpa mencari apa pun.
15. Sebagai tour leader, saya ingin memilih batch lain dari daftar, sehingga saya
    bisa berpindah antar keberangkatan pada hari yang sama.
16. Sebagai tour leader, saya ingin daftar batch pada halaman bandara
    menampilkan tanggal, nama paket, dan jumlah peserta, sehingga saya tahu
    berapa orang yang akan saya tangani sebelum membuka checklist-nya.
17. Sebagai tour leader, saya ingin halaman bandara tidak pernah tampil kosong
    tanpa penjelasan, sehingga saya tidak menyangka fiturnya rusak.
18. Sebagai tour leader yang hanya menangani batch tertentu, saya ingin daftar
    batch mendahulukan yang berangkat dalam waktu dekat, sehingga saya tidak
    menggulir melewati keberangkatan tahun lalu.
19. Sebagai admin, saya ingin batch yang sudah penuh atau ditutup tetap terlihat
    di pemilih namun tidak bisa dipilih untuk penugasan baru, sehingga alasan
    ketidaktersediaannya jelas dan bukan sekadar hilang dari daftar.
20. Sebagai admin, saya ingin pemilih batch bisa disaring ke keberangkatan
    mendatang saja, sehingga daftar tetap pendek seiring bertambahnya riwayat.
21. Sebagai konsultan, saya ingin pemilih peserta hanya menampilkan peserta yang
    memang boleh saya lihat, sehingga pembatasan akses tidak bocor lewat kotak
    pencarian.
22. Sebagai admin, saya ingin pilihan filter bertahan saat saya berpindah
    halaman daftar, sehingga menelusuri halaman kedua tidak mengembalikan saya
    ke seluruh data.
23. Sebagai admin, saya ingin memuat pemilih tidak memperlambat halaman,
    sehingga daftar utama tetap tampil lebih dulu.
24. Sebagai admin baru, saya ingin memakai keempat halaman itu tanpa pernah
    membuka basis data, sehingga saya bisa dilatih tanpa akses langsung ke data
    produksi.
25. Sebagai penguji sidang, saya ingin melihat seluruh alur operasional
    dijalankan dari antarmuka, sehingga sistem terbukti utuh dan bukan sekadar
    kumpulan endpoint.

## Implementation Decisions

### Endpoint baru: daftar batch lintas paket

Hari ini batch hanya bisa didaftar per paket (`GET /admin/packages/{package_id}/batches`).
Tiga dari empat halaman dalam spec ini membutuhkan batch **tanpa** mengetahui
paketnya lebih dulu, jadi ditambahkan satu endpoint baru:

`GET /admin/batches` mengembalikan batch dari seluruh paket, membawa nama paket,
tanggal keberangkatan, status, kuota, dan jumlah peserta yang sudah terdaftar.
Mendukung parameter penyaring:

- `upcoming=true` — hanya keberangkatan yang belum lewat
- `status` — `tersedia` / `penuh` / `ditutup`
- `search` — cocokkan nama paket
- `page`, `per_page`

Diurutkan tanggal keberangkatan terdekat lebih dulu. Ini satu-satunya seam baru
dalam spec ini, dan ia melayani tiga halaman sekaligus.

Kontrol akses: masuk ke grup `ops` (`super_admin` + `admin`) mengikuti endpoint
paket lainnya, ditambah `tour_leader` karena halaman bandara membutuhkannya.
Keputusan ini memperluas §5.3 dan harus dicatat sebagai baris baru di matriks
RBAC.

### Peserta tidak butuh endpoint baru

`GET /admin/participants` sudah menerima `search` (nama atau nomor WA) dan sudah
disaring per konsultan. Pemilih peserta memakai endpoint itu apa adanya,
sehingga pembatasan akses yang sudah ada ikut berlaku tanpa ditulis ulang.

### Halaman Peserta

Kolom teks `Filter Batch ID...` diganti pemilih batch yang memuat dari
`GET /admin/batches?upcoming=true`. Nilai yang dikirim ke `GET /admin/participants`
tetap `batch_id`; hanya cara memilihnya yang berubah. Filter yang aktif
ditampilkan sebagai label yang bisa dihapus.

### Halaman Review Dokumen

Kolom teks `Filter Participant ID...` diganti pemilih peserta dengan pencarian.
Parameter yang dikirim tetap `participant_id`.

### Halaman Invoice

Dua kolom UUID diganti:

- **Peserta** — pemilih dengan pencarian nama/nomor.
- **Batch** — tidak lagi diisi manual. Batch diturunkan dari peserta yang
  dipilih, karena setiap peserta sudah terikat pada satu batch. Kolomnya menjadi
  keterangan yang tampil setelah peserta dipilih, bukan masukan.

Menurunkan batch dari peserta menghapus seluruh kelas kesalahan "peserta A
ditagih pada batch B" yang hari ini tidak dicegah apa pun.

Pembuatan invoice otomatis saat konversi **tidak diubah**. Formulir manual tetap
ada untuk invoice tambahan.

### Halaman Airport Handling

Kolom `Masukkan Batch ID...` diganti pemilih batch yang memuat dari
`GET /admin/batches?upcoming=true`. Saat halaman dibuka tanpa pilihan, batch
dengan keberangkatan terdekat dipilih otomatis sehingga checklist langsung
tampil. Keadaan kosong hanya muncul bila memang tidak ada batch mendatang, dan
teksnya menjelaskan hal itu alih-alih menyuruh mengetik UUID.

### Konsistensi

Ketiga pemilih (batch, peserta) dibuat sebagai komponen yang dipakai bersama
oleh keempat halaman, bukan disalin per halaman. Pola konversi lead yang sudah
ada disesuaikan agar memakai komponen yang sama.

## Testing Decisions

Test yang baik di sini menguji **perilaku yang teramati dari luar** — status
kode, isi respons, dan penyaringan — bukan cara komponen React menyimpan state.

### Seam yang dipakai

Seam HTTP yang sudah mapan: `RegisterRoutes` + `httptest` + fake repository +
service aplikasi yang asli. Ada 24 berkas test di seam ini sebagai prior art,
antara lain `access_control_test.go` untuk pembatasan peran dan
`lead_scope_test.go` untuk penyaringan per konsultan. Tidak ada seam baru untuk
pengujian.

### Yang diuji

Endpoint `GET /admin/batches`:

1. Mengembalikan batch dari lebih dari satu paket dalam satu panggilan.
2. `upcoming=true` menyingkirkan keberangkatan yang sudah lewat.
3. Urutan menempatkan keberangkatan terdekat lebih dulu.
4. Setiap baris membawa nama paket dan jumlah peserta.
5. Penyaring `status` bekerja untuk ketiga nilainya.
6. Ditolak 401 tanpa token.
7. `tour_leader` diizinkan; peran di luar daftar ditolak 403.

Pencarian peserta pada `GET /admin/participants`:

8. `search` mencocokkan potongan nama maupun potongan nomor WhatsApp.
9. Konsultan hanya menerima peserta hasil konversi lead miliknya — menegaskan
   bahwa kotak pencarian tidak menjadi jalan pintas melewati pembatasan yang
   baru diperbaiki.

### Yang tidak diuji otomatis

Repo ini belum punya perkakas test frontend, dan spec ini tidak menambahkannya.
Perilaku antarmuka diverifikasi manual mengikuti skenario di `docs/UAT.md`, dan
bila perlu ditambahkan ke suite Playwright di proyek sibling `pintour-e2e`.

## Out of Scope

- **Pembayaran bertahap lewat tombol konfirmasi admin.** Konfirmasi manual masih
  bersifat penuh; sisa tagihan sudah bekerja pada jalur bukti transfer dan
  pembayaran online. Perubahan itu menyentuh skema dan dibahas terpisah.
- **Satu peserta per orang.** Lead `pax: 2` masih menghasilkan satu baris
  peserta. Memecahnya menjadi satu baris per orang mengubah dokumen, checklist
  bandara, dan invoice sekaligus — bukan bagian dari spec ini.
- **Halaman admin untuk antrean penghapusan §25.5.** Masih dijalankan lewat API.
- **Membuat batch dari halaman-halaman ini.** Pemilih hanya membaca; batch tetap
  dibuat di menu Paket.
- **Mengubah kontrak endpoint yang ada.** `batch_id` dan `participant_id` tetap
  menjadi parameter yang dikirim; hanya cara memperolehnya yang berubah.

## Further Notes

Pola ini sudah terbukti sekali. Konversi lead menjadi peserta dulu memakai dua
kolom UUID yang sama persis, dan sekarang berupa tombol pada lead berstatus
`deal` dengan pemilih batch yang disaring ke paket lead tersebut. Perubahan itu
diuji di peramban pada sistem berjalan dan menghasilkan peserta yang benar.
Spec ini menerapkan pola yang sama ke empat tempat tersisa.

Airport Handling perlu diperlakukan sebagai yang paling mendesak. Ketiga halaman
lain masih bisa dipakai sebagian tanpa UUID — daftarnya tetap tampil, hanya
tidak bisa disaring. Halaman bandara tidak menampilkan apa pun sampai UUID
diketik, sehingga secara praktis fiturnya tidak ada bagi siapa pun yang tidak
membuka basis data.
