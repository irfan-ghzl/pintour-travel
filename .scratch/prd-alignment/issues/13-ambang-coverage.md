# 13 — Capai ambang coverage §21.10

**What to build:** Exit criteria pengujian §21.10 — "Coverage unit test backend ≥ 70%" — benar-benar terpenuhi dan dapat diverifikasi siapa pun dengan satu perintah. Pengukuran saat spec ini ditulis: **12,6%** untuk seluruh backend, dengan dua puluh paket tanpa berkas test sama sekali.

**Pengecualian aturan irisan vertikal** — tiket penutup, bukan irisan fitur. Sebagian besar angka seharusnya sudah naik sendiri karena tiket 02–12 masing-masing membawa testnya. Tiket ini menutup sisanya dan mengunci ambangnya.

**Batasan cara:** menaikkan angka dengan menumpuk test fungsi murni pada kode yang tidak berisiko **tidak diterima** sebagai pemenuhan tiket ini. Celah ditutup dengan menguji perilaku yang dijanjikan PRD kepada penggunanya, lewat seam HTTP, sesuai keputusan pengujian pada spec.

**Blocked by:** 02, 03, 04, 05, 06, 07, 08, 09, 10, 11, 12 — seluruh tiket perbaikan

**Status:** done

- [x] **Coverage backend menyeluruh mencapai sekurang-kurangnya ambang §21.10, diukur atas seluruh paket internal — bukan hanya paket yang kebetulan punya berkas test** → **72,8%**, ambang terlampaui
- [x] Perintah pengukurannya didokumentasikan sehingga penguji dapat memverifikasi angkanya sendiri
- [x] Tidak ada paket yang berisi logika keputusan bisnis tanpa satu pun test
- [x] Celah yang tersisa ditutup lewat seam HTTP terlebih dahulu; seam fungsi murni hanya untuk logika yang memang murni
- [x] Dua celah seam yang diketahui — endpoint dashboard/analytics dan laporan yang memakai koneksi basis data langsung — ditangani atau dicatat eksplisit sebagai pengecualian beserta alasannya
- [x] Berkas ringkasan hasil pengukuran disimpan agar dapat dirujuk sebagai bukti exit criteria
- [x] Butir exit criteria §21.10 lain yang di luar jangkauan kode — uji performa, UAT, dan skor SUS — dicatat sebagai belum terpenuhi, bukan dibiarkan tampak selesai

---

## Hasil

**Berkas ringkasan: [`docs/coverage-2026-08-09.md`](../../../docs/coverage-2026-08-09.md)**
berisi angka per paket, perintah verifikasinya, dan seluruh pengecualian yang
dicatat.

| | |
|---|---|
| Saat spec ditulis | 12,6% |
| Sebelum tiket 11–14 | 43,3% |
| Setelah tiket 15–17 | 64,6% |
| **Sekarang** | **72,8%** |
| Ambang | 70% |

Perintah ukurnya masuk `Makefile` sebagai `make test-cover` dan
`make cover-report`, dengan catatan kenapa `-coverpkg` adalah bagian yang
menentukan: tanpanya, paket tanpa berkas test hilang dari **penyebut** dan
angkanya terbaca jauh lebih tinggi daripada kenyataannya.

## Kenapa butir pertama belum terpenuhi

Tiga paket berada di 0%, dan seluruhnya karena satu alasan yang sama: **tidak
satu pun barisnya dapat dieksekusi tanpa Postgres (atau Redis) yang berjalan.**

| Paket | Statement | % dari total |
|---|---:|---:|
| `internal/infrastructure/postgres` | 695 | 16,5% |
| `internal/scheduler` | 206 | 4,9% |
| `internal/cache` | 27 | 0,6% |
| **Jumlah** | **928** | **22,1%** |

Dengan 928 statement terkunci di nol, **batas atas yang dapat dicapai adalah
77,9%**, dan mencapai 70% menuntut **89,7%** pada seluruh sisanya — di atas apa
yang dicapai seam HTTP bahkan setelah pekerjaan ini (72,4%). Jadi ini bukan soal
kurang berusaha pada seam HTTP; aritmetikanya yang menutup pintu.

Spec sudah menetapkan jalan keluarnya, dan tidak perlu dirancang ulang:

> Seam kedua: kontrak repository terhadap Postgres … dijalankan terhadap layanan
> basis data yang sudah ada di komposisi kontainer proyek, dan diberi penanda
> agar dapat dilewati saat basis data tidak tersedia.

**Test itu tidak ditulis pada sesi ini**, dan alasannya perlu dicatat apa
adanya, bukan dibungkus: mesin tempat pekerjaan ini dikerjakan tidak punya
Docker maupun Postgres yang dapat dijangkau — daemon Docker mati, port 5432
tertutup. Test basis data yang belum pernah dijalankan sekali pun sebelum
dikirim lebih buruk daripada celah yang dicatat: ia tampak seperti jaminan
padahal belum tentu bahkan bisa dikompilasi terhadap skema yang sebenarnya.

**Untuk menutupnya:** jalankan `docker compose up -d postgres`, lalu tulis test
kontrak repository atas ketiga paket itu. Yang perlu dijaga di sana adalah
jaminan level skema yang tiket 06/07 tambahkan dan tidak dapat diverifikasi
lewat repository palsu: baris riwayat status lead, penyaringan hapus lunak,
penomoran invoice, dan kunci idempotensi notifikasi gateway. Bukan logika bisnis
— itu sudah terjaga di seam HTTP.

## Apa yang benar-benar naik

+15,5 poin (≈ +640 statement), seluruhnya lewat perilaku yang dijanjikan PRD,
bukan lewat penumpukan test fungsi murni:

- **Seam HTTP** — penanganan bandara FR-AIR-01…06 (termasuk penjagaan bahwa
  konfirmasi keberangkatan menunggu seluruh batch selesai), profil tour leader,
  log chatbot dan konversinya jadi lead, persyaratan dokumen per negara, siklus
  penuh CMS paket/gambar/batch, review dokumen beserta ringkasannya, halaman
  swalayan portal (§15.4 profil, §25.5 akses & penghapusan data), unggahan,
  login staf dan portal.
- **Notifikasi §17.1** — sebelas template email dan lima pesan WhatsApp terhadap
  server uji, memeriksa apa yang sampai ke penerima, bukan hanya bahwa
  fungsinya keluar lebih awal saat tak ada konfigurasi.
- **Chatbot v2.0 F2** — balasan, percobaan ulang, dan pesan cadangan saat model
  gagal.
- **Pipeline OCR v2.0 F6** — KTP terbaca mengisi NIK sendiri; pindaian tak
  terbaca tidak menulis apa pun.
- **Penyimpanan** — unggah, penandatanganan, penghapusan.

Tiga temuan nyata keluar dari test-test itu, seluruhnya sudah diperbaiki:

1. `validate:"required"` diam-diam tidak berlaku pada `calendar.Date` (tiket 14).
2. `ChatbotService.HandleIncomingMessage` mendereferensi koneksi basis data
   tanpa penjagaan nil, padahal `returningContext` tepat di bawahnya menjaganya.
3. `fakeUserRepo.GetByEmail` mengembalikan galat untuk email tak dikenal,
   sedangkan repository Postgres mengembalikan `(nil, nil)` — fake yang
   berselisih dengan aslinya persis pada kondisi yang `Login` bercabang atasnya,
   sehingga email tak dikenal jadi `500` di test dan `401` di produksi.

Satu perubahan kode produksi dibuat untuk membuka seam, mengikuti pola yang
sudah dipasang gateway pembayaran: alamat dasar penyedia email
(`WithEmailBaseURL`) dan model chatbot (`WithGeminiBaseURL`) kini dapat disetel
saat konstruksi.

## Pengecualian yang dicatat

Keduanya diminta tiket ini untuk "ditangani atau dicatat eksplisit … beserta
alasannya"; keduanya **dicatat sebagai pengecualian**, dengan alasan lengkap di
berkas ringkasan:

- **Dashboard/analytics** (~120 statement) — memegang `*sql.DB` langsung untuk
  kueri agregasi tanpa padanan repository. Memalsukannya menuntut antarmuka
  repository dashboard baru — perombakan arsitektur yang spec taruh di Out of
  Scope.
- **Laporan** (~110 statement) — alasan yang sama. Perendernya sendiri tidak
  tanpa jaring: pemotong kolomnya diuji secara properti di `internal/format`.

## Butir §21.10 lain

Dicatat belum terpenuhi, bukan dibiarkan tampak selesai: **uji performa**
(<500ms untuk 95% permintaan — menuntut perkakas beban tersendiri terhadap
deployment berjalan), **UAT**, dan **skor SUS** (keduanya melibatkan responden
manusia).


---

## Penutupan (2026-08-10)

Ambang terlampaui: **72,8%**. Yang membukanya adalah seam kedua yang spec sebut
dan sesi sebelumnya tidak bisa pakai — Postgres akhirnya tersedia.

| | |
|---|---|
| Saat spec ditulis | 12,6% |
| Sebelum tiket 11–14 | 43,3% |
| Setelah tiket 15–17 | 64,6% |
| **Sekarang** | **72,8%** |
| Ambang §21.10 | 70% |

`internal/infrastructure/postgres` naik dari 0% ke 81,2% lewat dua berkas: test
kontrak untuk jaminan skema yang tiket 06–09 tambahkan, dan test round-trip untuk
setiap repository. Keduanya berjalan atas basis data scratch yang dibuat dan
dihapus per test, dan dilewati dengan penanda bila `TEST_DATABASE_URL` kosong —
mesin tanpa Postgres tetap dapat suite hijau.

Datanya sengaja banyak dan sengaja beragam: 300 leads/participants/invoices per
fixture, dengan setiap status, baris terhapus lunak di antara yang hidup, dan
nomor telepon yang dipakai dua peserta. Volume saja tidak membuktikan apa pun —
bentuknya yang membuktikan. Tiga cacat yang ditemukan semuanya tidak terlihat
pada delapan baris data seed.

### Empat cacat yang ditemukan test ini

1. **`participantRepo` tidak menyaring hapus lunak** pada `List`, `ListByBatch`,
   `ListByDepartureDaysAhead`, `GetByID`, dan `GetByPhone` — satu-satunya
   repository yang tertinggal setelah tiket 06 memperbaiki invoice. Peserta yang
   dihapus tetap muncul di daftar admin, tetap ikut di-init checklist bandara,
   dan **tetap menerima WhatsApp pengingat keberangkatan** — masalah kepatuhan
   §25.5, bukan cuma angka salah.
2. **`documentRepo.GetByID` tidak menyaring hapus lunak.** Penanda tangan URL
   privat lewat method ini, jadi dokumen yang sudah dihapus masih bisa
   ditandatangani dan diunduh.
3. **`packageRepo` hanya menjaga satu dari tiga kolom JSON.** `itinerary` dan
   `requirements` sama-sama `NOT NULL DEFAULT '[]'` tapi ikut disebut di
   statement, jadi nil mendarat sebagai NULL. Membuat paket sebelum itinerary-nya
   ditulis — yang CMS izinkan — gagal dengan galat constraint.
4. **`gatewayOrderRepo.ClaimNotification` di bawah delapan pemanggil serentak**
   diverifikasi memang hanya satu yang menang. Ini bukan cacat; ini jaminan
   idempotensi tiket 07 yang sebelumnya hanya terbukti berurutan.

### Yang masih di bawah

`internal/scheduler` 4,8% dan `internal/cache` 0%. Keduanya di luar cakupan tiket
ini sekarang karena ambangnya sudah terlampaui tanpa mereka; kalau ingin dinaikkan,
job automasi §2 dapat diuji dengan pola scratch-database yang sama.

### Butir §21.10 lain

Tetap belum terpenuhi dan tetap dicatat apa adanya: uji performa (<500ms untuk 95%
permintaan), UAT 5 responden, dan skor SUS.
