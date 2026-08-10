# 12 — Kualitas & kebersihan lintas modul

**What to build:** Sekumpulan perbaikan mekanis yang masing-masing terlalu kecil untuk berdiri sebagai irisan sendiri, tapi bersama-sama menghilangkan sejumlah cara sistem gagal secara diam-diam.

**Pengecualian aturan irisan vertikal** — ini kumpulan perbaikan tersebar, bukan satu jalur lengkap melalui seluruh lapisan. Dikelompokkan agar tidak menghasilkan sepuluh tiket sepele. Tidak diblokir tiket manapun karena seluruhnya dapat diverifikasi lewat seam pengujian yang sudah ada.

**Blocked by:** None — can start immediately.

**Status:** done

- [x] Dokumen PDF menampilkan karakter dan simbol non-ASCII dengan benar — tanda titik tengah, simbol hak cipta, butir daftar, dan nama beraksen. Modul laporan sudah melakukannya dengan benar dan menjadi acuan polanya
- [x] Ekstraksi berlabel pada OCR memakai baris yang sama untuk pencocokan dan pemotongan, sehingga spasi di depan tidak lagi merusak nilai yang diambil
- [x] Nama berkas unggahan disanitasi termasuk bagian ekstensinya, dan komponen path dikodekan saat disusun menjadi URL permintaan penyimpanan
- [x] Pengunduhan berkas di antarmuka memasang elemen pemicu ke dokumen dan menunda pelepasan URL objek, sehingga unduhan berfungsi di seluruh peramban sasaran
- [x] Pemotongan teks pada laporan tidak lagi memotong di tengah karakter multi-bita
- [x] Utilitas format mata uang disatukan menjadi satu implementasi bersama, menggantikan empat salinan yang tersebar
- [x] Utilitas pembacaan variabel lingkungan dengan nilai bawaan disatukan menjadi satu implementasi bersama, menggantikan tiga salinan dengan nilai bawaan yang berbeda-beda
- [x] Utilitas pemotongan teks aman-Unicode disatukan menjadi satu implementasi bersama
- [x] Parsing respons JSON penyimpanan memakai pustaka JSON standar, bukan pemindaian string buatan sendiri
- [x] Pemilihan berkas pada antarmuka melepaskan URL pratinjau saat pilihan dibatalkan dan saat komponen dilepas
- [x] Aturan pengabaian berkas biner pada Git benar-benar cocok; berkas biner hasil kompilasi yang saat ini tidak terabaikan menjadi terabaikan
- [x] Kode mati dihapus: operasi repository dan utilitas yang tidak punya satu pun pemanggil
- [x] Seluruh berkas yang disentuh diformat sesuai formatter bawaan bahasa

---

## Catatan pelaksanaan

### 1. PDF non-ASCII

`internal/service/pdf.go` menulis "·", "•", "©", dan "—" langsung ke font inti
(Helvetica), yang menerima cp1252 — sehingga setiap satu karakter itu mendarat
sebagai dua atau tiga glyph pengganti. Modul laporan sudah memakai
`UnicodeTranslatorFromDescriptor("")` dengan benar dan jadi acuannya.

Yang dipasang **bukan** `tr(...)` di enam puluh tempat, melainkan pembungkus:
tipe `doc` menyematkan `*gofpdf.Fpdf` dan menimpa `Cell`, `CellFormat`, dan
`MultiCell` agar menerjemahkan teksnya. Baris teks berikutnya yang ditulis di
berkas itu ikut diterjemahkan karena ia ditulis, bukan karena penulisnya ingat.

Testnya (`TestDocTranslatesToCoreFontEncoding`, `TestDocWriterTranslatesText`)
memeriksa byte cp1252 pada aliran dokumen yang kompresinya dimatikan — versi
pertama memeriksa keluaran `GenerateBriefing` apa adanya dan **lolos secara
hampa**, karena aliran halaman terkompresi sehingga teks mentah memang tidak
akan pernah ditemukan di sana.

### 2. OCR ekstraksi berlabel

`findLabeled` mencocokkan baris yang sudah di-trim tapi memotong baris yang
belum: `line[len(label):]` pada baris berspasi depan memakan sekian karakter
pertama nilainya. Keluaran OCR penuh spasi depan, yang menjelaskan kenapa hampir
setiap nama hasil pindaian perlu dikoreksi tangan.

### 3. Sanitasi nama berkas & pengodean path

Dua hal terpisah di `internal/service/storage.go`:

- Ekstensi diambil mentah dari nama unggahan lalu ditempel setelah sanitasi
  batangnya, jadi apa pun yang `sanitizeFilename` tolak bisa diselundupkan
  kembali di belakang titik terakhir. Sekarang lewat `sanitizeExtension`.
- Bucket dan path ditempel ke URL apa adanya. Nama objek berasal dari nama berkas
  yang dipilih peserta, jadi "?" atau "#" di dalamnya mengubah *permintaan mana*
  yang dikirim, bukan objek mana yang dinamai. Sekarang lewat `objectURL`, yang
  mengodekan tiap komponen path dan mempertahankan pemisahnya.

### 4 & 10. Unduhan dan URL objek di antarmuka

Tiga tempat mengunduh blob, masing-masing benar sebagiannya saja. Disatukan jadi
`web/src/utils/download.ts`:

- elemen `<a>` harus terpasang di dokumen sebelum diklik — Firefox mengabaikan
  klik pada elemen lepas, sehingga ekspor laporan dan unduh invoice tidak
  melakukan apa pun di sana;
- URL objeknya harus hidup melewati klik — melepasnya di baris berikutnya
  berlomba dengan pengambilan peramban sendiri.

`useFileUpload` melepaskan URL pratinjau di keempat jalan keluarnya: diganti
berkas lain, pemilih ditutup tanpa memilih, pilihan ditolak validasi, dan
komponen dilepas.

### 5, 6, 7, 8. Utilitas yang disatukan

| Utilitas | Sebelumnya | Sekarang |
|---|---|---|
| Format rupiah | 4 salinan (`application/invoice`, `delivery/http/report_handler`, `scheduler/automation`, `service/pdf`) | `format.Rupiah` |
| Pemotongan teks | 3 salinan (`report_handler`, `payment_gateway`, dan satu yang memotong per-byte) | `format.TruncateRunes` + `format.Ellipsis` |
| Kapitalisasi awal | `titelize` dengan aritmetika byte `s[0]-32` | `format.Title` (aman-rune) |
| Env dengan nilai bawaan | 3 pembaca + 5 pembacaan inline | `config.Env`, `config.EnvInt`, `config.EnvFloat` |

Ketiga pembaca env berbeda pada satu titik: `config.getEnv` memakai `LookupEnv`
sehingga variabel yang di-set kosong dikembalikan kosong, dua lainnya
memperlakukan kosong sebagai belum di-set. Disatukan ke perilaku kedua —
`PORTAL_BASE_URL` kosong bukan alamat dasar, dan memperlakukannya sebagai alamat
menghasilkan tautan yang diawali "/portal" begitu saja.

Nilai bawaannya juga tersebar: `PORTAL_BASE_URL` dan `APP_URL` masing-masing
dibaca di lima tempat dengan nilai bawaannya sendiri-sendiri. Sekarang
`config.PortalBaseURL()` dan `config.AppURL()`.

`format.Ellipsis` diuji dengan properti, bukan hanya contoh: hasilnya tidak
pernah melebihi `max` karakter untuk kombinasi masukan-dan-lebar manapun — sifat
yang lebar kolom laporan PDF bergantung padanya.

### 9. Parsing JSON penyimpanan

`extractJSONField` — pemindai string buatan sendiri — diganti `encoding/json`.
Testnya sekarang menembak server uji lokal dan mencakup kasus yang membuat
pemindai lama salah: nilai yang memuat tanda kutip ter-escape.

### 11. Pengabaian berkas biner

Pola `*.exe` sudah benar (komentar di ujung barisnya dipisah pada pekerjaan
sebelumnya). Yang belum tercakup: `go build ./cmd/x` di Linux/macOS menghasilkan
binari **tanpa ekstensi**, dan pustaka bersama. Ditambahkan `/server`, `/seed`,
`/seed-demo`, `/run-jobs`, `/pintour-server` (keluaran Makefile dan Dockerfile),
`*.dll`, `*.so`, `*.dylib`, serta berkas coverage berpola. Diverifikasi dengan
`git check-ignore -v` atas berkas nyata, bukan dengan membaca polanya.

### 12. Kode mati

Delapan operasi repository tanpa satu pun pemanggil produksi — masing-masing ada
di antarmuka, di implementasi Postgres, dan di dua sampai tiga repository palsu:

`user.CountActiveleadsByConsultant`, `document.CountByStatus`,
`invoice.GetByNumber`, `invoice.GetByOrderID`, `invoice.ListUnpaidOlderThan`,
`airport.GetByParticipant`, `participant.ListWithUnpaidInvoiceDaysOld`,
`portaluser.UpdatePassword`.

Ditambah lima utilitas: `StorageService.ComputeETag`, `queryStringPtr`,
`queryFloat64Ptr`, `queryIntPtr`, `scanUser`. Sisanya diverifikasi dengan
`staticcheck -checks=U1000` atas seluruh `internal/` dan `cmd/`, yang sekarang
bersih.

Catatan: `portaluser.UpdatePassword` dihapus karena tidak ada jalur reset
password portal di kode manapun. Kalau alur itu nanti dibuat, methodnya ditulis
ulang bersama endpointnya — bukan dibiarkan menunggu sebagai janji yang tidak
pernah ditepati.

### 13. Format

`gofmt -l internal/ cmd/` bersih; `go vet ./internal/... ./cmd/...` bersih;
`tsc --noEmit` pada `web/` bersih.
