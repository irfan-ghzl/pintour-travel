# 03 — Aktivasi validasi masukan §19.3

**What to build:** Masukan pengguna yang tidak sah ditolak dengan pesan jelas sebelum menyentuh basis data. PRD §19.3 menyatakan "Semua input dari user divalidasi menggunakan `go-playground/validator` di layer handler" — validator sudah terpasang di aplikasi tapi tidak pernah dipanggil, dan tidak ada satu pun tag validasi di seluruh repositori. Tiket ini membuat klaim itu berlaku.

Dikerjakan dengan pola expand: pengikat JSON bersama mulai memanggil validator lebih dulu (tanpa tag terpasang, nol perubahan perilaku), lalu tag dipasang per domain sehingga tiap batch tetap hijau.

**Blocked by:** 01 — Seam pengujian HTTP

**Status:** ready-for-agent

- [x] Pengikat JSON bersama menjalankan validator setelah dekode, sehingga seluruh handler memperoleh validasi tanpa perubahan satu per satu
- [x] Galat validasi menghasilkan respons `400` yang menyebut field bermasalah, bukan galat internal
- [x] Peran pengguna divalidasi terhadap empat peran resmi §5.3 saat membuat dan menyunting akun staf
- [x] Tipe kamar divalidasi terhadap nilai yang dikenal skema
- [x] Status invoice, dokumen, dan lead divalidasi terhadap kosakata yang dikenal skema
- [x] Format email dan nomor telepon divalidasi pada endpoint yang menerimanya
- [x] Nilai batas paginasi divalidasi sehingga permintaan tidak dapat meminta jumlah baris tak wajar
- [x] Payload yang sebelumnya diterima dan sah tetap diterima — tidak ada regresi pada alur yang sudah berjalan
- [x] Ada test yang membuktikan setiap kelompok tag di atas menolak nilai tidak sah dan menerima nilai sah

## Comments

### Pelaksanaan (2026-08-08)

**Mekanisme.** `internal/delivery/http/binding.go` baru menampung tiga hal: `bindJSON` (dekode lalu validasi), `validationFailure` (galat yang membawa nama field), dan `invalidPayload(c, err, cadangan)` yang memetakannya ke `400`. `bindJSON` sendiri sudah ada sebelumnya di `package_handler.go` sebagai pembungkus `json.NewDecoder` satu baris — ia dipindah dan diberi langkah validasi, sehingga ke-31 pemanggilnya memperoleh validasi tanpa disentuh. Yang disentuh hanya baris penanganan galatnya: `badRequest(c, "…")` → `invalidPayload(c, err, "…")`, dengan pesan cadangan lama dipertahankan persis. Nama field pada pesan memakai nama JSON, bukan nama field Go (`RegisterTagNameFunc`).

`UserHandler.Login` adalah satu-satunya handler yang memakai `c.Bind` dan kini ikut memakai `bindJSON`. Tidak ada lagi dekode JSON di lapisan delivery di luar `bindJSON`.

**Kosakata satu sumber.** Tag `oneof=` yang mengulang daftar yang sudah punya bentuk kanonik diganti tag khusus yang membaca daftar itu: `lead_status` (dari `lead.Statuses` yang sudah ada), `staff_role` (dari `user.Roles`, baru), dan `document_type` (dari `document.Types`, baru). Tanpa ini status lead hidup di tiga tempat sekaligus. Kosakata yang belum punya bentuk kanonik (status invoice, tipe kamar, kategori paket, status batch) tetap ditulis literal pada tag — membuat konstanta domain untuknya adalah materi tiket 11.

**Tiga keputusan pelaksanaan yang perlu dicatat.**

1. **Batas paginasi memangkas, tidak menolak.** `queryPageSize` menurunkan `per_page`/`limit` di atas 100 menjadi 100, bukan menjawab `400`. Halaman yang terlalu besar adalah pemanggil yang meminta terlalu banyak dari sumber daya yang memang haknya, bukan permintaan cacat; respons melaporkan ukuran yang benar-benar dilayani pada `meta.limit`. Ini menyimpang dari kata "divalidasi" pada kriteria dan dari "galat validasi dipetakan ke `400`" pada spec — hasil yang diminta ("tidak dapat meminta jumlah baris tak wajar") tetap tercapai.
2. **`phone_id` membuang pemisah sebelum menilai.** `normalizePhone` hanya mengupas `+` dan `0` di depan, tidak spasi/tanda hubung. Menilai hasilnya apa adanya akan menolak `0812-3456-7890` yang selama ini diterima formulir publik — dan, lebih buruk, mengunci peserta yang nomor tersimpannya berpemisah dari login portal. Validator karena itu menilai salinan bersih; nilai yang disimpan tidak berubah.
3. **NIK divalidasi `max=20`, bukan 16 digit.** Kolomnya `VARCHAR(20)` tanpa CHECK. Aturan 16-digit itu benar menurut hukum tapi tidak dikenal skema, jadi tempatnya adalah CHECK di migrasi — diangkat sebagai usulan, bukan diputuskan diam-diam di tag.

**Cacat yang ditemukan sambil jalan dan ikut ditutup.** `pax` punya `CHECK (pax BETWEEN 1 AND 50)` dan repository menulisnya apa adanya, jadi payload tanpa `pax` mengirim `0` ke Postgres dan gagal di sana. Sekarang ditolak `400` bila di luar rentang, dan default ke 1 bila tidak diisi.

**Kebijakan password.** `min=8` sudah berlaku di `ResetPassword` sebelum tiket ini; kini berlaku juga di `CreateUser` dan `ResetPasswordAdmin` — dua tempat lain yang menetapkan password. Ini penolakan baru atas payload yang dulu lolos, disengaja agar satu kebijakan berlaku di ketiganya.

**Dua hal yang sengaja tidak dikerjakan.**

- Cabang form-encoded pada webhook Fonnte (`chatbot_handler.go`) tidak melewati `bindJSON` sama sekali dan galat cabang JSON-nya dibuang. Itu jalur masuk dari gateway, bukan masukan pengguna, dan Fonnte adalah materi tiket 08.
- `ReviewProof` dan `ReviewDocument` memakai bentuk payload yang nyaris identik (`status` + alasan penolakan). Menyatukannya jadi satu tipe bersama menahan diri karena nama field JSON-nya berbeda (`notes` vs `rejection_reason`) dan penyatuan akan mengubah kontrak salah satunya.

**Cacat di luar cakupan yang perlu tiket sendiri → sekarang tiket 14.** Formulir invoice admin mengirim `due_date` sebagai `YYYY-MM-DD` dari `<input type="date">`, sedangkan `time.Time` hanya menerima RFC3339 — pembuatan invoice dari UI gagal dekode sebelum dan sesudah tiket ini. Penelusuran lanjutan menemukan formulir batch keberangkatan punya cacat yang sama pada `departure_date`/`return_date`. Bukan regresi dari tiket manapun, dan tidak tercakup tiket 01–13; diangkat sebagai tiket 14.

**Coverage.** 24,4% → **33,2%**.
