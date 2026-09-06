# 01 — Airport Handling bisa dipakai tanpa mengetahui UUID

**What to build:** Tour leader membuka halaman Airport Handling dan langsung
melihat checklist keberangkatan terdekat, tanpa mengetik apa pun. Ia bisa
berpindah ke keberangkatan lain lewat daftar yang menyebut tanggal, nama paket,
dan jumlah peserta.

Ini irisan paling mendesak: tiga halaman lain masih bisa dipakai sebagian tanpa
UUID, sedangkan halaman ini tidak menampilkan apa pun sampai UUID diketik —
sehingga fiturnya praktis tidak ada bagi siapa pun yang tidak membuka basis data.

Irisan ini juga memperkenalkan dua hal yang dipakai ulang tiket berikutnya:
endpoint daftar batch lintas paket, dan komponen pemilih batch.

**Blocked by:** None — can start immediately.

**Status:** done

- [x] `GET /admin/batches` mengembalikan batch dari seluruh paket dalam satu
      panggilan, masing-masing membawa nama paket, tanggal keberangkatan, status,
      kuota, dan jumlah peserta terdaftar
- [x] Endpoint menerima `upcoming`, `status`, `search`, `page`, `per_page`, dan
      mengurutkan keberangkatan terdekat lebih dulu
- [x] Endpoint ditolak 401 tanpa token; `super_admin`, `admin`, dan
      `tour_leader` diizinkan; peran lain ditolak 403
- [x] Perluasan akses `tour_leader` dicatat sebagai baris baru pada matriks RBAC
      §5.3, bukan diterapkan diam-diam
- [x] Halaman Airport Handling tidak lagi punya kolom isian Batch ID
- [x] Saat dibuka tanpa pilihan, halaman memilih keberangkatan terdekat sendiri
      dan checklist langsung tampil
- [x] Keadaan kosong hanya muncul bila memang tidak ada batch mendatang, dengan
      teks yang menjelaskan hal itu
- [x] Berpindah batch lewat pemilih memuat ulang checklist dan ringkasan
      progresnya
- [x] Test di seam HTTP menutupi: gabungan lintas paket, `upcoming` menyingkirkan
      keberangkatan lampau, urutan terdekat lebih dulu, penyaring `status`, dan
      ketiga keputusan peran

## Keputusan kontrol akses

`GET /api/v1/admin/batches` tidak tercakup baris mana pun pada matriks §5.3,
karena endpoint-nya baru. Tingkat aksesnya karena itu **diputuskan**, bukan
dibaca dari dokumen — mengikuti cara yang sama yang dipakai untuk
`GET /admin/signed-url` di `.scratch/prd-alignment/issues/04-kontrol-akses.md`.

| Endpoint | Diberikan kepada | Alasan |
| --- | --- | --- |
| `GET /api/v1/admin/batches` | grup **pickers** — `super_admin`, `admin`, `konsultan`, `tour_leader` | Setiap peran staf memilih keberangkatan di suatu tempat, jadi setiap peran staf boleh membacanya. Isinya tidak melebihi yang sudah beredar: tanggal keberangkatan, nama paket, status, dan kuota sudah dikirim ke pengunjung anonim lewat `GET /packages/{slug}`; tambahannya hanya jumlah peserta terdaftar, dan staf penjualan sudah melihat daftar pesertanya sendiri. Ditulis lengkap alih-alih memakai grup dasar `admin`, supaya peran yang ditambahkan nanti tidak ikut diberi akses tanpa ada yang memutuskannya. |
| `GET /api/v1/admin/packages/{package_id}/batches` | **dipindahkan** dari grup ops ke grup **sales** — `super_admin`, `admin`, `konsultan` | Ditemukan saat review, lalu dibuktikan: rute ini menjawab **403 kepada `konsultan`**, padahal `POST /admin/participants/convert` ada di grup sales dan memang boleh dipakai konsultan. Artinya dialog konversi — inti pekerjaan harian konsultan — tidak pernah bisa menampilkan satu batch pun baginya sejak dropdown-nya dibuat; yang tampil adalah "Gagal memuat daftar keberangkatan". Membukanya untuk sales tidak menyingkap apa pun yang baru: baris yang sama sudah dikirim ke pengunjung anonim lewat `GET /packages/{slug}` di katalog publik. Membuat dan mengubah batch tetap ops-only, karena di situlah kewenangannya. |

**Usulan baris tambahan / perubahan untuk §5.3:**

| Fitur | super_admin | admin | konsultan | tour_leader |
| --- | --- | --- | --- | --- |
| Daftar keberangkatan lintas paket (baca) — **baris baru** | ✓ | ✓ | ✓ | ✓ |
| Daftar keberangkatan satu paket (baca) — **baris berubah** | ✓ | ✓ | ✓ (dulu —) | — |

Kedua baris ini adalah **usulan**: §5.3 hidup di PRD yang gitignored, sehingga
tidak ada berkas ter-track di repo ini yang bisa diubah. Yang dijamin adalah
keputusannya tidak diambil diam-diam — ia tertulis di sini, dan dijaga test.
Preseden bentuk yang sama ada di
`.scratch/prd-alignment/issues/04-kontrol-akses.md`.

### Koreksi setelah tiket ini ditutup

`konsultan` mula-mula **tidak** diberi akses, dengan alasan yang tertulis di
atas: satu-satunya tempat ia memilih batch dianggap dialog konversi, yang memakai
daftar per-paket. Alasan itu melewatkan **filter keberangkatan di halaman
Peserta** — halaman yang memang boleh dibuka konsultan — sehingga baginya filter
itu hanya menampilkan "Gagal memuat daftar keberangkatan".

Bentuknya sama persis dengan cacat yang ditemukan tiket ini pada rute batches
per-paket: sebuah rute ditempatkan mengikuti satu pemakaian yang terpikir, lalu
pemakaian kedua ditemukan belakangan oleh orang yang memakainya. Karena itu
grupnya kini menyebut keempat peran, bukan diperluas satu per satu setiap kali
ada yang mengeluh.

Memberi akses ke daftar keberangkatan **tidak** membuka penanganan bandara:
seluruh rute `/admin/airport/*` tetap di grup airport, dan menu bandara tetap
tidak muncul di sidebar konsultan. Dijaga
`TestAdminBatches_DoesNotOpenAirportHandling`.

Dibuktikan oleh `TestAdminBatches_RoleMatrix`,
`TestAdminBatches_RejectsRequestWithoutToken`, dan
`TestPackageBatchListing_KonsultanCanOfferDeparturesWhenConverting` di
`internal/delivery/http/admin_pickers_test.go`.

## Catatan implementasi

`participant_count` dihitung oleh kueri (subquery berkorelasi atas
`participants`), bukan kolom — §14.4 memang menyebut cacah terisi bukan kolom.

Ia mula-mula dipasang pada **setiap** pembacaan batch, dengan alasan
`participant_count: 0` tidak boleh ambigu antara "belum ada yang mendaftar" dan
"tidak dihitung". Review membatalkan itu: `ListByPackage` juga melayani
`GET /packages/{slug}`, rute **publik tanpa autentikasi**, sehingga perubahan itu
mulai menyiarkan jumlah kursi terjual tiap keberangkatan kepada setiap pengunjung
katalog — persis "mengubah kontrak endpoint yang ada" yang dilarang bagian Out of
Scope. Sekarang hanya `ListAll` yang menghitung.

Keambiguan yang jadi alasan semula diselesaikan dengan tipenya, bukan dengan
menyebarkan kuerinya: `ParticipantCount *int` dengan `omitempty`, sehingga tidak
ada berarti "tidak dihitung" dan `0` berarti "belum ada yang mendaftar". Sisi
TypeScript memakai `participant_count?: number`, dan `batchLabel` menghilangkan
bagian "kursi terisi" saat cacahnya tidak ada. Dijaga
`TestBatchListings_OnlyTheAdminListingCountsParticipants`.

Urutan "terdekat lebih dulu" adalah dua kunci, bukan satu: keberangkatan yang
belum lewat mendahului yang sudah lewat, lalu di dalam masing-masing paruh yang
paling dekat dengan hari ini didahulukan. Urutan menaik polos akan menenggelamkan
keberangkatan minggu depan di bawah seluruh batch yang pernah dijalankan.

Kueri SQL-nya diuji terhadap Postgres sungguhan di
`TestPackageBatchRepo_ListAllOrdersByNearestAndCountsParticipants`
(`internal/infrastructure/postgres/contract_test.go`), karena urutan dan cacahnya
adalah klaim tentang SQL yang tidak bisa dibuktikan oleh repository palsu.
