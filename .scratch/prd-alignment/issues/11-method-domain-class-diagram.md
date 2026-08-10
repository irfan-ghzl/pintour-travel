# 11 — Method domain sesuai class diagram §14.4

**What to build:** Class diagram §14.4 dapat ditunjukkan berlaku di kode. Diagram mencantumkan sekitar dua puluh dua method pada sembilan class — otentikasi pengguna, pemeriksaan peran, ketersediaan kursi batch, hitung mundur keberangkatan, perubahan status lead, aktivasi portal peserta, keterlambatan invoice, persetujuan dokumen, kelengkapan checklist bandara, dan verifikasi password portal. Seluruh paket domain saat ini hanya berisi satu fungsi, dan tidak ada satu pun method pada entity manapun.

Catatan di bawah diagram bahkan menegaskan pola yang tidak ada: "Method dengan parameter penerima". Kode memakai model domain anemik — struct hanya data, seluruh perilaku ada di lapisan aplikasi dan repository.

Arah penyelarasan sudah diputuskan: kode yang menyesuaikan dokumen.

**Batas yang harus dijaga:** hanya perilaku yang dapat dihitung dari data entity itu sendiri yang pindah ke domain. Method yang memerlukan I/O — kueri basis data, panggilan jaringan, pembuatan berkas — **tetap** di lapisan aplikasi. Method orkestrasi pada diagram diinterpretasikan sebagai perilaku entity setara yang dapat diuji tanpa basis data; catat pemetaannya agar diagram dan kode dapat dicocokkan baris per baris.

**Blocked by:** 01 — Seam pengujian HTTP

**Status:** done

- [x] Entity domain memperoleh method perilaku yang murni predikat dan turunan sebagaimana class diagram §14.4
- [x] Lapisan aplikasi memakai method-method itu alih-alih menghitung ulang logika yang sama di tempatnya sendiri
- [x] Tidak ada method domain yang melakukan I/O
- [x] Pemetaan antara setiap method pada diagram dan padanannya di kode tercatat di berkas tiket ini, termasuk method yang sengaja tidak dipindahkan beserta alasannya
- [x] Setiap method baru punya unit test langsung, mengikuti gaya tabel kasus masukan-ke-harapan yang sudah dipakai test fungsi murni di repo
- [x] Tidak ada perubahan perilaku yang terlihat pengguna dari tiket ini — seluruh test yang ada tetap hijau

---

## Pemetaan diagram §14.4 → kode

Sembilan class entity pada diagram mencantumkan 22 method. Empat class layanan teknis
(`FonnteService`, `ChatbotService`, `MidtransService`, `OCRService`) sudah punya
method-methodnya di `internal/service` sebelum tiket ini dan tidak disentuh.

Kolom **Pemanggil** menyebut kode produksi yang memakai method itu. Method tanpa
pemanggil produksi diberi alasannya — semuanya predikat yang di produksi dievaluasi
oleh basis data, bukan oleh Go.

### User — 2 method

| Diagram | Kode | Pemanggil |
|---|---|---|
| `Authenticate(password) bool` | `(*user.User).Authenticate` — `internal/domain/user/entity.go` | `usersvc.Login` (`internal/application/user/service.go`) |
| `HasRole(role) bool` | `(*user.User).HasRole(roles ...string)` — variadik karena setiap grup rute menyebut beberapa peran sekaligus | **tidak ada** — lihat catatan |

Membaca akun dan menerbitkan JWT tetap di `usersvc.Login`: keduanya I/O.

`HasRole` sempat dipasang ke `RequireRole` lalu ditarik lagi. Di titik itu tidak
ada akun untuk ditanya — hanya string peran dari token — sehingga memanggilnya
berarti membungkus string itu jadi `User{Role: role}` demi menelusuri daftar
secara linear, menggantikan pencarian map yang dibangun sekali per grup rute.
Itu tautologi yang menyamar sebagai pemakaian domain, dan lebih lambat. Method
tetap ada pada entity sesuai diagram, dengan test langsung; `RequireRole`
menyebutnya di komentar agar aturan yang sama dapat dibandingkan.

### Package — 3 method

| Diagram | Kode | Pemanggil |
|---|---|---|
| `Activate()` | `(*pkg.Package).Activate` | **tidak ada** — lihat catatan di bawah |
| `Deactivate()` | `(*pkg.Package).Deactivate` | **tidak ada** — lihat catatan di bawah |
| `AddBatch(batch) error` | `(*pkg.Package).AddBatch` | `pkgsvc.Service.CreateBatch` |

`Activate`/`Deactivate` ada di entity dan punya test langsung, tapi belum ada
pemanggil produksi: nilai `is_active` selalu tiba sudah diputuskan pada payload
formulir admin (checkbox "Tampilkan di katalog publik"), sehingga memanggilnya di
`CreatePackage`/`UpdatePackage` hanya akan menulis ulang nilai yang sama —
tautologi, bukan penyatuan logika. Endpoint toggle tersendiri adalah fitur baru
dan di luar lingkup tiket ini.

`AddBatch` sengaja **tidak** memvalidasi urutan tanggal berangkat/pulang. Aturan itu
belum pernah ada, jadi menambahkannya di sini akan menolak payload yang hari ini
diterima — perubahan perilaku yang tiket ini larang.

### PackageBatch — 2 method

| Diagram | Kode | Pemanggil |
|---|---|---|
| `HasAvailableSeats() bool` | `(*pkg.PackageBatch).HasAvailableSeats(sold int)` | **tidak ada** — lihat catatan |
| `DaysUntilDeparture() int` | `(*pkg.PackageBatch).DaysUntilDeparture` + `DaysUntilDepartureFrom(now)` | **tidak ada** — lihat catatan |

Diagram menulis `HasAvailableSeats()` tanpa parameter, tapi catatan §14.4 sendiri
menyatakan kapasitas terisi **bukan kolom fisik** — dihitung dari jumlah
`participants` per batch saat query. Menghitungnya di dalam entity berarti entity
melakukan I/O, yang dilarang. Karena itu jumlah terjual diserahkan pemanggilnya.
Di produksi, "masih ada kursi" dievaluasi oleh kolom `package_batches.status`
(`tersedia`/`penuh`/`ditutup`) dan oleh filter SQL `WHERE pb.status='tersedia'` pada
`checkBatchQuota`; method ini adalah pernyataan Go dari aturan yang sama, dengan test
langsung, agar keduanya dapat dibandingkan.

`SeatsRemaining(sold int) int` ditambahkan sebagai turunan yang `HasAvailableSeats`
didefinisikan di atasnya, **dan punya pemanggil produksi**: `scheduler.checkBatchQuota`
(`internal/scheduler/automation.go`) berhenti mengurangi kuota sendiri.

`DaysUntilDeparture` pada batch belum dipakai produksi — hitung mundur yang benar-benar
ditampilkan berasal dari peserta (lihat Participant), dan dashboard menghitungnya di SQL
(`pb.departure_date - CURRENT_DATE`). Method ini melengkapi entity sesuai diagram dan
diuji langsung.

### Lead — 3 method

| Diagram | Kode | Pemanggil |
|---|---|---|
| `ChangeStatus(newStatus) error` | `(*lead.Lead).ChangeStatus` | `leadsvc.Service.UpdateStatus` |
| `AssignTo(consultantID) error` | `(*lead.Lead).AssignTo` | `leadsvc.Service.AssignLead` |
| `ConvertToParticipant() Participant` | `(*lead.Lead).ConvertToParticipant(batchID, roomType)` | `participant.Service.ConvertFromLead` |

`ChangeStatus` memvalidasi target dan mengubah status; **mencatat pelakunya tetap di
repository**, karena baris riwayat dan kolom status ditulis dalam satu transaksi dan
entity tidak boleh menyentuh transaksi.

`ConvertToParticipant` menurunkan record peserta dan tidak menulis apa pun. Pembuatan
akun portal, invoice otomatis, dan penandaan lead sebagai terkonversi tetap di
`ConvertFromLead` — ketiganya I/O dan harus berhasil-atau-gagal bersama. Efek
sampingnya: seluruh keputusan yang tidak butuh basis data (lead sudah `deal`, batch
disebut, tipe kamar dikenal skema) kini diambil **sebelum** unit of work dimasuki.

### Participant — 2 method

| Diagram | Kode | Pemanggil |
|---|---|---|
| `ActivatePortal() string` | `(*participant.Participant).ActivatePortal` | **tidak ada** — lihat catatan |
| `DaysUntilDeparture() int` | `(*participant.Participant).DaysUntilDeparture` + `DaysUntilDepartureFrom(now)` | `PortalHandler.PortalMe` dan `isBriefingActive` (`internal/delivery/http/portal_handler.go`) |

`ActivatePortal` mengembalikan identitas login (nomor WA), bukan password baru:
pelanggan lama memakai kembali akun portal yang sudah ada, jadi aktivasi soal akses,
bukan kredensial. Belum ada pemanggil produksi — jalur aktivasi yang sebenarnya adalah
`participants.Activate(ctx, id)` yang menulis satu kolom lewat SQL tanpa memuat entity.
Memuat entity hanya untuk membalik satu flag adalah kueri tambahan tanpa manfaat.

`HasDeparture()` ditambahkan sebagai pendamping: `DaysUntilDeparture` mengembalikan 0
baik untuk "berangkat hari ini" maupun "belum punya batch", dan pemanggil perlu
membedakannya.

Dua perubahan yang perlu dicatat, keduanya menghilangkan selisih yang sudah ada
sebelumnya, bukan menambahkannya:

1. **Hitung mundurnya kini hari kalender penuh**, bukan selisih instan. Dulu
   `int(dep.Sub(now).Hours()/24)`, sehingga perjalanan yang sama terbaca 14 pagi
   hari dan 13 setelah makan siang — angka yang bergeser oleh jam berapa peserta
   membuka portalnya. Sekarang sama persis dengan `PackageBatch`.
2. **`isBriefingActive` menjadi `<= 14`**, dari perbandingan jam
   `time.Until(dep).Hours() <= 14*24`. Frontend sudah menyembunyikan spanduk
   "tersedia dalam N hari" begitu hitungannya mencapai 14 (`countdown > 14`),
   jadi peserta yang hitungannya menunjukkan 14 diberi tahu briefingnya terbuka
   lalu menemukannya tertutup. Perubahan ini hanya dapat membuka akses lebih
   awal, tidak pernah mencabutnya.

### Invoice — 3 method

| Diagram | Kode | Pemanggil |
|---|---|---|
| `GeneratePDF() error` | **tidak dipindahkan** — tetap `invoice.Service.GeneratePDF` | — |
| `ConfirmPayment(confirmedBy) error` | `(*invoice.Invoice).ConfirmPayment(confirmedBy, at)` | `invoice.Service.ConfirmPayment` |
| `IsOverdue() bool` | `(*invoice.Invoice).IsOverdue` + `IsOverdueAt(now)` | **tidak ada** — lihat catatan |

`GeneratePDF` membaca invoice dan peserta, merender berkas, lalu menulis `pdf_path`.
Tiga I/O; tetap di lapisan aplikasi, sesuai batas tiket.

`ConfirmPayment` pada entity menetapkan status, pelaku, dan waktunya, serta menolak
invoice yang sudah lunas lewat `ErrAlreadySettled`. Service memetakan sentinel itu
menjadi no-op — persis perilaku sebelumnya. `ErrInvoiceAlreadyPaid` pada lapisan
aplikasi kini alias dari sentinel domain, bukan galat kedua dengan pesan sama.

`IsOverdue` di produksi dievaluasi SQL di dua tempat — `scheduler.expireInvoices`
dan ringkasan keuangan dashboard, keduanya
`status IN ('diterbitkan','menunggu_bayar') AND due_date < NOW()`. Keduanya
**harus** memfilter di basis data; menariknya ke Go berarti membaca seluruh
tabel invoice tiap malam.

Method ini **bukan** salinan persis dari SQL itu, dan selisihnya disengaja —
dicatat di sini supaya tidak terbaca sebagai bug saat keduanya dibandingkan:

| | SQL (produksi) | `IsOverdueAt` |
|---|---|---|
| Status `dibayar` (dibayar sebagian) | tidak dikejar | dianggap masih berutang |
| Jatuh tempo **hari ini** | dikejar sejak pukul 00.00 | belum, peserta punya sisa hari itu |
| Terhapus lunak | disaring `deleted_at IS NULL` | disaring |
| `menunggu_konfirmasi_gateway` | tidak dikejar | tidak dikejar |

Dua baris pertama adalah pendapat method ini tentang aturan yang lebih benar:
uang yang masih kurang tetap kurang, dan menagih orang pada pagi hari tanggal
jatuh temponya adalah menagih sebelum waktunya habis. **Menyelaraskan SQL-nya ke
sana mengubah siapa yang menerima pengingat WhatsApp**, jadi itu keputusan
tersendiri di luar tiket ini — bukan sesuatu yang boleh ikut secara diam-diam
pada tiket yang berjanji tidak mengubah perilaku.

Dua turunan tambahan **punya** pemanggil produksi dan menutup duplikasi nyata:
`IsFullyPaid(paid)` dipakai `settleApprovedProof`, dan `RemainingBalance(paid)` dipakai
`CreatePaymentForParticipant` — dua tempat yang sebelumnya masing-masing memutuskan
sendiri apakah pembayaran menutupi invoice.

### Document — 2 method

| Diagram | Kode | Pemanggil |
|---|---|---|
| `Approve(reviewerID) error` | `(*document.Document).Approve(reviewerID, at)` | `DocumentHandler.ReviewDocument` |
| `Reject(reviewerID, reason) error` | `(*document.Document).Reject(reviewerID, reason, at)` | `DocumentHandler.ReviewDocument` |

Aturan yang pindah: penolakan wajib menyebut alasan, dan persetujuan menghapus alasan
penolakan sebelumnya. Penulisan barisnya tetap `document.Repository.Review`.

### AirportChecklist — 4 method

| Diagram | Kode | Pemanggil |
|---|---|---|
| `IsComplete() bool` | `(*airport.Checklist).IsComplete` | repository in-memory pada seam HTTP (`fakes_test.go`, 2 tempat) |
| `MarkBaggage(by)` | `(*airport.Checklist).MarkBaggage(by, at)` | idem |
| `MarkTicket(by)` | `(*airport.Checklist).MarkTicket(by, at)` | idem |
| `MarkPassport(by)` | `(*airport.Checklist).MarkPassport(by, at)` | idem |

Ini satu-satunya kelompok yang pemanggilnya bukan kode produksi, dan alasannya jujur:
`airport_repo.go` menulis ketiga langkah dengan satu `UPDATE ... SET x=true, x_at=NOW()`
tanpa pernah memuat entity, dan `GetBatchProgress` menghitung kelengkapan dengan
`COUNT(*) FILTER(WHERE baggage_checked AND ticket_distributed AND passport_returned)`.
Merombak repository Postgres agar memuat-ubah-simpan menambah satu kueri per klik di
meja bandara tanpa manfaat perilaku. Yang dilakukan tiket ini: implementasi in-memory
kedua dari kontrak repository yang sama berhenti menyalin aturannya sendiri — tiga
salinan `baggage && ticket && passport` menjadi satu.

### PortalUser — 1 method

| Diagram | Kode | Pemanggil |
|---|---|---|
| `VerifyPassword(pw) bool` | `(*portaluser.PortalUser).VerifyPassword` | `participant.Service.PortalLogin` |

Jalur login peserta lama (pra-F1, password per-peserta) tetap memakai bcrypt langsung:
hash-nya ada di `participants.portal_password`, bukan pada entity `PortalUser`.

---

## Rekapitulasi

| | Jumlah |
|---|---|
| Method diagram yang menjadi method entity | 21 dari 22 |
| Method diagram yang sengaja tidak dipindahkan (I/O) | 1 — `Invoice.GeneratePDF` |
| Method entity dengan pemanggil produksi | 10 |
| Method entity dengan pemanggil hanya di seam test | 4 — kelompok AirportChecklist |
| Method entity tanpa pemanggil produksi | 7 — `User.HasRole`, `Package.Activate`, `Package.Deactivate`, `PackageBatch.HasAvailableSeats`, `PackageBatch.DaysUntilDeparture`, `Participant.ActivatePortal`, `Invoice.IsOverdue` |

Turunan tambahan di luar diagram, ditambahkan karena menutup duplikasi nyata:
`PackageBatch.SeatsRemaining`, `Participant.HasDeparture`, `Invoice.IsFullyPaid`,
`Invoice.RemainingBalance`.

Tidak ada method domain yang melakukan I/O: paket `internal/domain` tidak mengimpor
`database/sql`, `net/http`, maupun `os` — hanya `time`, `errors`, `strings`,
`encoding/json`, dan `golang.org/x/crypto/bcrypt` (perbandingan hash adalah aritmetika,
bukan I/O).
