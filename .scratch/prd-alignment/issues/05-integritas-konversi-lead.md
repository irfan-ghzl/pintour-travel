# 05 — Integritas konversi lead

**What to build:** Konversi lead menjadi peserta berhasil sepenuhnya atau tidak meninggalkan jejak sama sekali. Peserta baru menerima password portalnya lewat WhatsApp tanpa admin harus mengetikkannya manual.

Cacat inti: identitas portal dan data peserta ditulis dalam dua operasi terpisah tanpa transaksi. Bila penulisan peserta gagal — misalnya karena tipe kamar tidak dikenal skema — identitas portal sudah tersimpan dengan hash password yang tidak pernah sampai ke siapa pun. Percobaan ulang kemudian mengenali baris yatim itu sebagai "pelanggan lama", tidak menerbitkan password baru, dan memberi tahu admin bahwa password lama tetap berlaku. Peserta terkunci permanen dan tidak ada alur reset password untuk akun portal.

Tiket ini juga memperkenalkan dukungan transaksi pada lapisan repository, yang akan dipakai ulang tiket 06.

**Blocked by:** 01 — Seam pengujian HTTP; 03 — Aktivasi validasi masukan §19.3

**Status:** ready-for-agent

- [x] Lapisan repository memperoleh mekanisme unit-of-work sehingga beberapa operasi dapat dijalankan dalam satu transaksi, tanpa membocorkan tipe basis data ke lapisan aplikasi
- [x] Mekanisme itu ditambahkan berdampingan dengan antarmuka repository yang ada, tidak menggantinya, sehingga tidak ada pemanggil lama yang rusak
- [x] Pembuatan identitas portal, data peserta, dan penandaan lead sebagai terkonversi berjalan dalam satu transaksi
- [x] Kegagalan pada tahap manapun mengembalikan seluruh perubahan; tidak ada identitas portal yatim yang tersisa
- [x] Percobaan ulang setelah kegagalan berperilaku sama persis dengan percobaan pertama — peserta yang belum pernah punya akun tetap diperlakukan sebagai peserta baru dan menerima password baru
- [x] Tipe kamar ditolak lebih dulu bila tidak dikenal skema, dengan pesan yang jelas, bukan gagal di tengah penulisan
- [x] Password sementara dikirim otomatis ke WhatsApp peserta sesuai FR-PORTAL-01; nilai mentahnya tetap dikembalikan ke admin sebagai cadangan
- [x] Pesan yang menjanjikan "password dikirim via pesan terpisah" kini benar-benar didahului atau diikuti pesan berisi password tersebut
- [x] Pelanggan lama tetap memakai akun portal yang ada dan tidak menerima password baru, sesuai FR-CRM-08
- [x] Ada test yang membuktikan kegagalan penulisan peserta tidak meninggalkan identitas portal

## Comments

### Pelaksanaan (2026-08-08)

**Bentuk unit-of-work.** `internal/domain/uow` mendeklarasikan `Runner.Do(ctx, fn)`
dan `Repos{Participants, Leads, PortalUsers}` — seluruhnya antarmuka domain yang
sudah ada, jadi kode di dalam unit terbaca sama dengan kode di luarnya, hanya
memakai repository yang diberikan padanya. `internal/infrastructure/postgres`
mengimplementasikannya dengan satu transaksi. Lapisan aplikasi tidak pernah
melihat `*sql.Tx`.

Yang membuatnya murah: field `db` pada tiga repository (`participantRepo`,
`leadRepo`, `portalUserRepo`) berganti dari `*sql.DB` ke antarmuka `dbtx` yang
berisi tiga method yang sama-sama dimiliki `*sql.DB` dan `*sql.Tx`. Tidak ada
satu pun kueri yang ditulis ulang, dan konstruktor `NewXxxRepo(db *sql.DB)` tidak
berubah — 14 repository lain tidak tersentuh.

**Pemulihan pada panic.** `Do` memasang `defer tx.Rollback()`, bukan rollback di
tiap jalur galat: transaksi yang menggantung memegang lock sampai koneksi mati,
lebih buruk daripada tulisan separuh yang mau dicegah. Setelah `Commit` berhasil,
rollback itu no-op yang mengembalikan `sql.ErrTxDone`, karena itu galatnya
dibuang.

**Fake unit-of-work benar-benar melakukan rollback.** `fakeUnitOfWork` menyalin
isi fake sebelum `fn` dan mengembalikannya bila `fn` gagal. Ini bukan hiasan:
dengan passthrough, ketiga test "tidak ada yang tertinggal" tetap hijau terhadap
service yang belum transaksional sama sekali. Sudah diverifikasi — passthrough
membuat ketiganya merah, dan yang ketiga mereproduksi persis cacat aslinya
("percobaan ulang mengenali peserta sebagai pelanggan lama / tidak menerbitkan
password baru").

**Tipe kamar ditolak di dua tempat, sengaja.** Tag `room_type` pada payload
handler (400, menyebut field) menangkap pemanggil HTTP; penjagaan
`IsValidRoomType` di service (422) menjaga invariannya di lapisan yang melakukan
penulisan, untuk pemanggil non-HTTP. Kosakatanya satu: `participant.RoomTypes`,
dibaca oleh `registerVocabulary` dan oleh penjaga service — mengikuti pola
`lead.Statuses`/`user.Roles` dari tiket 03, bukan `oneof=` yang mengulang daftar.

### Keputusan & penyimpangan

**Template notifikasi baru: `PORTAL_CREDENTIALS`.** FR-PORTAL-01 menuntut
password sampai ke peserta, dan tidak ada template yang mengirimkannya. Ini
menambah satu baris pada rekapitulasi §17.3 — **tiket 08 perlu menghitungnya**
saat merekonsiliasi jumlah template.

**Klaim ke admin dibuat bersyarat, bukan selalu.** Draf pertama selalu menjawab
"password sudah dikirim otomatis". Itu tidak benar pada deployment tanpa gateway
WA: `FonnteService.Send` no-op tanpa token. `ConvertResult` sekarang membawa
`CredentialsSent`, respons memuat `credentials_sent`, dan pesannya mengikuti
kenyataan — bila tidak terkirim, admin diminta meneruskan sendiri. `FonnteService`
mendapat `Enabled()`, meniru `StorageService.Enabled()`.

**`SendDocRequest` tidak lagi menjanjikan pesan yang tak pernah dikirim.**
Barisnya berubah dari "Password: dikirim via pesan terpisah" menjadi rujukan ke
pesan kredensial yang memang dikirim lebih dulu saat konversi.

**Satu pengerasan di luar sepuluh kriteria, dicatat karena diputuskan sendiri.**
`MarkConverted` sekarang `WHERE id=$2 AND status='deal'` dan mengembalikan
`lead.ErrNotConvertible` bila tak ada baris terpengaruh. Pemeriksaan status oleh
pemanggil berjalan **sebelum** transaksi dibuka, jadi dua permintaan convert bisa
sama-sama lolos dan sama-sama menulis peserta untuk satu lead — unit melindungi
dari tulisan separuh, bukan dari dijalankan dua kali. Ditemukan saat review,
diperbaiki karena judul tiket ini adalah integritas konversi dan perbaikannya
lima baris. Ada testnya.

**`lookupPortalUser` berhenti menelan seluruh galat.** Sebelumnya galat apa pun
dari `GetByPhone` berarti "tidak ada" → pelanggan lama dianggap baru → `INSERT`
menabrak indeks unik nomor telepon. Sekarang hanya `sql.ErrNoRows` yang berarti
"tidak ada". Konsekuensinya lapisan aplikasi mengimpor `database/sql` untuk satu
sentinel; sentinel not-found milik domain akan lebih bersih, tapi itu perubahan
lintas seluruh repository dan di luar tiket ini.

**Tidak menambah salinan `PORTAL_BASE_URL`.** Nilai itu sudah dibaca ad-hoc di
tiga tempat (dua handler invoice + service invoice), persis yang dikeluhkan user
story 58. `ConvertFromLead` menerimanya sebagai parameter — mengikuti bentuk
`ConfirmPayment` — dan `ParticipantHandler` mendapatkannya dari `Services.PortalURL`
yang diisi dari `cfg.Server.PortalBaseURL`. Penyatuan tiga salinan lama tetap
pekerjaan tiket 12.

### Yang belum terbukti test

**`internal/infrastructure/postgres/uow.go` nol test.** Semantik rollback dijamin
`database/sql` + Postgres; yang menjadi tanggung jawab kode ini hanya urutan
Begin/Commit/Rollback, sepuluh baris. Membuktikannya butuh basis data sungguhan —
itu "seam kedua" pada spec, yang baru dibangun tiket 06.

**Pengiriman WA-nya sendiri.** Cabang "gateway aktif → pesan terkirim" tidak bisa
dijangkau harness: mengaktifkan gateway membuat test memanggil jaringan. Yang
diuji sekarang: template merender password, nomor, dan tautan portal
(`TestPortalCredentialsMessageCarriesTheCredential` — fungsi murni), dan respons
tidak mengklaim pengiriman yang tidak terjadi. Cabang satunya menjadi dapat
diuji setelah **kriteria pertama tiket 08** (alamat dasar Fonnte dapat disetel).

**Coverage.** 33,3% → 33,5%. Kecil karena jalur konversi sudah tersentuh test
lain; nilai tiket ini ada pada perilaku yang dikunci, bukan pada angkanya.
