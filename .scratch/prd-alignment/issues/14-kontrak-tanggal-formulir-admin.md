# 14 — Kontrak tanggal formulir admin

**What to build:** Admin benar-benar dapat menerbitkan invoice dan membuka batch keberangkatan dari antarmuka. Keduanya saat ini mustahil: formulirnya mengirim tanggal polos `YYYY-MM-DD` dari `<input type="date">`, sedangkan field tujuannya bertipe `time.Time` yang hanya menerima RFC3339. Permintaannya gagal didekode sebelum menyentuh handler, dan admin melihat pesan galat generik tanpa petunjuk apa yang salah.

Terbukti secara langsung:

```
{"due_date":"2026-08-15"}             -> err=parsing time "2026-08-15" as "2006-01-02T15:04:05Z07:00"
{"due_date":"2026-08-15T00:00:00Z"}   -> err=<nil>
```

Dua alur yang terkena, keduanya lewat jalur yang sama:

- `POST /admin/invoices` — `web/src/pages/admin/AdminInvoicesPage.tsx:188` mengisi `due_date` pada `CreateInvoiceRequest` (`web/src/types/index.ts:206`, bertipe `string`, tanpa konversi), mendarat di `invoice.Invoice.DueDate`.
- `POST /admin/packages/:package_id/batches` — `web/src/pages/admin/AdminPackagesPage.tsx:343` mengisi `departure_date` dan `return_date`, mendarat di `pkg.PackageBatch`.

Filter tanggal pada halaman leads dan log chatbot **tidak** terkena: keduanya query param yang di-parse `time.Parse("2006-01-02")` di sisi server.

**Kenapa tiket ini ada.** Cacat ini tidak tercakup tiket 01–13 maupun spec — empat pass review yang melahirkan spec tidak menemukannya, karena hanya terlihat bila tombol simpan benar-benar ditekan dan repo belum punya test frontend. FR-INV-01 menuntut sistem dapat menghasilkan invoice; selama ini berlaku, klaim "seluruh 69 FR dapat ditunjukkan berlaku di kode" belum benar. Ditemukan saat mengerjakan tiket 03 dan tercatat di berkasnya; bukan regresi dari tiket manapun.

Backend dan frontend dalam satu tiket karena cacatnya justru terletak di antara keduanya dan tidak dapat didemokan terpisah.

**Blocked by:** 01 — Seam pengujian HTTP

**Status:** done

- [x] Payload tanggal yang benar-benar dikirim formulir invoice admin diterima, dan invoice tersimpan dengan tanggal jatuh tempo yang sama dengan yang dipilih admin
- [x] Payload tanggal yang benar-benar dikirim formulir batch admin diterima, untuk tanggal berangkat maupun tanggal pulang
- [x] Keputusan tentang sisi mana yang mengalah tercatat di berkas ini beserta alasannya, dan diterapkan sama pada ketiga field
- [x] Tidak ada field tanggal lain pada permintaan yang masih menuntut bentuk berbeda dari yang dikirim antarmukanya — seluruh `<input type="date">` yang masuk ke body permintaan ditelusuri
- [x] Tanggal yang dikirim tanpa zona waktu tidak bergeser sehari saat ditulis maupun saat ditampilkan kembali
- [x] Galat dekode tanggal menghasilkan pesan yang menyebut field bermasalah, bukan pesan cadangan generik — mengikuti pola yang dipasang tiket 03
- [x] Ada test di seam HTTP yang menembakkan payload persis seperti yang dikirim antarmuka, sehingga kontrak ini tidak dapat putus lagi tanpa ketahuan

---

## Keputusan: **API yang mengalah** — tanggal kalender lewat tipe tersendiri

Ketiga field memakai tipe baru `calendar.Date` (`internal/domain/calendar/date.go`),
bukan `time.Time`. Antarmuka tidak diubah sama sekali: ia tetap mengirim
`YYYY-MM-DD` seperti sebelumnya.

**Alasannya, berurut dari yang paling menentukan:**

1. **Skemanya sudah menyatakan hal yang sama.** `invoices.due_date`,
   `package_batches.departure_date`, dan `package_batches.return_date` ketiganya
   kolom `DATE` di `db/migrations/003_prd_schema.sql`. Basis data sudah
   memodelkan ketiganya sebagai hari kalender; yang menyimpang justru tipe Go-nya.
   Memaksa antarmuka mengirim RFC3339 berarti menambah satu konversi di sisi
   klien agar cocok dengan tipe Go yang sejak awal tidak cocok dengan kolomnya.

2. **Ini memang tanggal, bukan momen.** "Jatuh tempo 15 Agustus" adalah hari yang
   sama di Jakarta dan di Jeddah. Melekatkan tengah malam pada zona tertentu
   mengarang ketelitian yang tidak dimiliki faktanya.

3. **Ia menutup kriteria "tidak bergeser sehari" pada akarnya, bukan menambalnya.**
   Kalau antarmuka yang mengalah dan mengirim `2026-08-15T00:00:00+07:00`, nilai
   itu adalah 14 Agustus di UTC — dan kolomnya `DATE`, jadi Postgres menyimpan
   hari yang salah. Sebaliknya `2026-08-15T00:00:00Z` yang ditampilkan kembali di
   zona sebelah barat Greenwich terbaca 14 Agustus. Tanggal polos tidak punya
   zona sehingga tidak bisa bergeser sama sekali.

4. **Satu perubahan tipe versus tiga konversi yang harus diingat selamanya.**
   Setiap formulir baru yang memuat `<input type="date">` harus mengingat
   konversinya. Itu persis cara cacat ini lahir.

**Kompatibilitas mundur dijaga:** `calendar.Date` tetap menerima RFC3339 dan
mengambil harinya, jadi klien lama dan payload yang sudah tersimpan tidak putus.
Yang berubah hanya bahwa bentuk yang *antarmuka kirim* sekarang juga diterima.

### Yang ditemukan test saat menulisnya

Dua celah yang tidak terlihat dari membaca kode, keduanya sudah ditutup:

1. **`required` diam-diam tidak berlaku pada tipe baru.** `calendar.Date` menyimpan
   harinya di field tak terekspor, dan `go-playground/validator` membaca struct
   semacam itu sebagai selalu terisi — sehingga formulir batch dengan tanggal
   pulang kosong lolos dan tersimpan. Ditutup dengan `RegisterCustomTypeFunc` di
   `internal/delivery/http/binding.go`, yang menyerahkan `time.Time` di baliknya
   agar `required` berarti sama seperti di field lain.

2. **Pesan galatnya masih generik.** Galat dekode dari `UnmarshalJSON` jatuh ke
   pesan cadangan handler ("format tidak valid"). Ditutup dengan
   `calendar.ParseError` — tipe galat tersendiri yang `invalidPayload` kenali dan
   kutip nilainya: `tanggal "15-08-2026" tidak dikenali, gunakan format 2006-01-02
   (contoh: 2026-08-15)`.

### Penelusuran seluruh `<input type="date">`

Tujuh input tanggal di antarmuka, seluruhnya ditelusuri:

| Berkas | Field | Masuk ke | Status |
|---|---|---|---|
| `AdminInvoicesPage.tsx:189` | `due_date` | body `POST /admin/invoices` | **diperbaiki** — `calendar.Date` |
| `AdminPackagesPage.tsx:343` | `departure_date` | body `POST .../batches` | **diperbaiki** — `calendar.Date` |
| `AdminPackagesPage.tsx:348` | `return_date` | body `POST .../batches` | **diperbaiki** — `calendar.Date` |
| `AdminLeadsPage.tsx:155` | `date_from` | query param | sudah benar — `time.Parse("2006-01-02")` di server |
| `AdminLeadsPage.tsx:163` | `date_to` | query param | sudah benar |
| `ChatbotLogsPage.tsx:48` | `date_from` | query param | sudah benar |
| `ChatbotLogsPage.tsx:51` | `date_to` | query param | sudah benar |

Keempat query param diberi test di seam HTTP juga
(`TestAdminDateFilters_TakePlainDates`), supaya "seluruh tanggal di antarmuka"
adalah klaim yang ada testnya, bukan hasil pembacaan.

### Sisi tampilan

`new Date("2026-08-15")` di peramban dibaca sebagai tengah malam UTC, lalu
`toLocaleDateString` merendernya di zona pembaca — di sebelah barat Greenwich
tercetak 14 Agustus. Cacat ini sudah ada sebelumnya (payload lama pun berupa
instant), dan tidak terlihat di Indonesia karena WIB berada di sebelah timur.
Ditutup lewat `web/src/utils/date.ts`: `formatDate` memecah string tanggalnya
sendiri dan menyematkannya ke tengah malam **lokal**, sehingga tidak ada
perenderan yang dapat menggeser harinya.

Dipakai di **sepuluh** tempat — seluruh perenderan `due_date`, `departure_date`,
dan `return_date` di antarmuka:

| Berkas | Field |
|---|---|
| `AdminInvoicesPage.tsx` (2×) | `due_date` |
| `AdminPackagesPage.tsx` (2×) | `departure_date`, `return_date` |
| `PaymentPage.tsx` | `due_date` |
| `PortalInvoicePage.tsx` | `due_date` |
| `PackageDetailPage.tsx` | `departure_date` |
| `PortalItineraryPage.tsx` (3×) | `departure_date`, `return_date` |

Empat di antaranya — halaman detail paket dan itinerary portal — baru ketahuan
saat review menelusuri ulang seluruh perenderan, bukan hanya formulir yang
mengirimnya. Keduanya menampilkan tanggal yang formatnya berubah pada tiket ini,
jadi melewatkannya berarti kriteria "tidak bergeser sehari saat ditampilkan
kembali" hanya berlaku separuh.

### Yang sengaja tidak disentuh

`participant.BatchDepartureDate` masih `*time.Time`. Ia kolom hasil join yang
hanya dibaca — tidak pernah muncul di body permintaan manapun — sehingga di luar
kalimat tiket ini ("`<input type="date">` yang masuk ke body permintaan").
Countdown portal yang menurunkannya sudah dihitung dalam hari kalender penuh
sejak tiket 11, jadi tidak bergeser oleh jam berapa ia ditanyakan.

### Test yang menjaga kontraknya

`internal/delivery/http/admin_date_contract_test.go` — payloadnya disalin
field-per-field dari kedua formulir, termasuk `tour_leader_id: null` yang formulir
batch kirim saat belum ada tour leader:

- payload formulir diterima dan tersimpan dengan tanggal yang dipilih admin;
- tanggalnya dibaca kembali persis sama, diuji pada batas tahun (1 Januari,
  31 Desember) tempat pergeseran sehari paling kentara;
- RFC3339 masih diterima;
- empat bentuk tanggal yang salah ditolak `400` dengan pesan yang menyebut tanggal
  dan mengutip nilainya, tanpa menulis baris apa pun;
- tanggal kosong ditolak dengan pesan yang menyebut nama fieldnya.
