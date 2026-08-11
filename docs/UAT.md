# UAT — Skenario Pengujian Penerimaan

Alur untuk mendemokan seluruh sistem dan menyatakannya diterima. Disusun
mengikuti perjalanan bisnis yang sebenarnya — pengunjung → lead → peserta →
invoice → portal → keberangkatan — bukan mengikuti daftar menu, supaya yang
diuji adalah proses yang dijalani pengguna, bukan tombol yang kebetulan ada.

Setiap skenario punya **hasil yang diharapkan**. Sebuah langkah dinyatakan lulus
hanya bila hasilnya persis seperti tertulis; "tidak error" saja tidak cukup.

> Untuk sidang, ada [ringkasan 15 menit](#lampiran-a--demo-15-menit) di lampiran.
> Dokumen ini yang lengkap.

---

## Persiapan

### Lingkungan

Jalankan salah satu — lihat [RUNNING.md](RUNNING.md):

| Kebutuhan | Mode |
| --------- | ---- |
| UAT lokal, tanpa webhook | `make docker-up` → <http://localhost> |
| UAT penuh termasuk WhatsApp & Midtrans | `powershell -File scripts/quick-tunnel.ps1` → URL publik |

**Skenario 6 (Midtrans) dan 11 (chatbot WhatsApp) hanya bisa dijalankan lewat URL
publik.** Keduanya bergantung pada webhook yang dikirim dari luar, dan `localhost`
tidak bisa dihubungi Midtrans maupun Fonnte.

### Akun

Login staf bawaan: **admin@pintour.com** / **admin123** (peran `super_admin`).

Empat peran perlu disiapkan lebih dulu lewat **Admin → Users**, karena Skenario
13 menguji batas masing-masing:

| Peran | Boleh |
| ----- | ----- |
| `super_admin` | segalanya, termasuk kelola pengguna |
| `admin` | operasional: paket, invoice, dokumen, laporan |
| `konsultan` | CRM lead & peserta miliknya sendiri |
| `tour_leader` | penanganan bandara |

### Data awal

Database kosong cukup — migrasi diterapkan sendiri saat API start. Untuk
mempercepat, isi data contoh:

```bash
go run ./cmd/seed-demo
```

### Mengulang dari nol

```bash
make rebuild-fresh
```

Menghapus database, membangun ulang, memigrasi, dan menyemai admin. Pakai ini
bila UAT perlu diulang bersih.

---

## Skenario

### UAT-01 — Katalog publik

**Aktor:** pengunjung (tanpa login) **Halaman:** `/`

| # | Langkah | Hasil yang diharapkan |
|---|---------|------------------------|
| 1 | Buka halaman utama | Daftar paket tampil beserta harga, durasi, dan destinasi |
| 2 | Ketik nama destinasi di kolom cari | Daftar menyusut sesuai kata kunci |
| 3 | Pilih kategori (Umroh / Halal / Honeymoon) | Hanya paket kategori itu yang tampil |
| 4 | Saring berdasarkan durasi dan bulan keberangkatan | Hasil sesuai kedua saringan sekaligus |
| 5 | Klik satu paket | Halaman detail terbuka: itinerary per hari, fasilitas, syarat, dan jadwal batch |

---

### UAT-02 — Lead masuk dari website

**Aktor:** pengunjung → admin **Halaman:** detail paket → `/admin/leads`

| # | Langkah | Hasil yang diharapkan |
|---|---------|------------------------|
| 1 | Di halaman paket, isi formulir konsultasi (nama, WA, jumlah peserta, catatan) | Muncul konfirmasi pengiriman |
| 2 | Login sebagai admin, buka **Leads** | Lead baru tampil paling atas, berstatus awal |
| 3 | Buka detail lead | Data yang diisi pengunjung tampil utuh, termasuk paket yang diminati |

> Endpoint publik ini dibatasi 60 permintaan/menit per IP. Mengirim formulir
> berkali-kali dengan cepat akan ditolak — itu perilaku yang benar.

---

### UAT-03 — Tindak lanjut CRM

**Aktor:** admin, konsultan **Halaman:** `/admin/leads`

| # | Langkah | Hasil yang diharapkan |
|---|---------|------------------------|
| 1 | Tugaskan lead ke seorang konsultan | Nama konsultan tampil di baris lead |
| 2 | Ubah status lead (mis. *baru* → *dihubungi*) | Status berubah dan tercatat waktunya |
| 3 | Tambahkan catatan tindak lanjut | Catatan tampil berurutan dengan penulis dan waktu |
| 4 | Login sebagai konsultan lain | Lead yang bukan miliknya **tidak** tampil di daftarnya |

Langkah 4 adalah inti kontrol akses CRM: pembatasan terjadi di API, bukan sekadar
menyembunyikan menu.

---

### UAT-04 — Konversi lead menjadi peserta

**Aktor:** admin / konsultan **Halaman:** `/admin/leads` → `/admin/participants`

| # | Langkah | Hasil yang diharapkan |
|---|---------|------------------------|
| 1 | Dari lead yang sudah deal, jalankan **Konversi** | Peserta baru terbentuk, tertaut ke paket dan batch |
| 2 | Buka **Participants** | Peserta tampil dengan nama, kontak, dan batch keberangkatan |
| 3 | Kembali ke lead asal | Statusnya kini terkonversi, dan riwayat perubahannya tercatat |

---

### UAT-05 — Invoice dan pembayaran manual

**Aktor:** admin **Halaman:** `/admin/invoices`

| # | Langkah | Hasil yang diharapkan |
|---|---------|------------------------|
| 1 | Buat invoice untuk peserta tadi | Invoice terbit dengan nomor, rincian, dan total |
| 2 | Unduh PDF invoice | Berkas terunduh dan terbaca, memuat identitas peserta dan rincian biaya |
| 3 | Sebagai peserta di portal, unggah bukti transfer | Bukti terkirim, status menunggu peninjauan |
| 4 | Sebagai admin, tinjau bukti → **setujui** | Status bukti berubah disetujui |
| 5 | Konfirmasi pembayaran | Status invoice menjadi lunas; sisa tagihan berkurang sesuai nominal |

---

### UAT-06 — Pembayaran online (Midtrans sandbox)

**Aktor:** peserta **Prasyarat:** URL publik aktif

| # | Langkah | Hasil yang diharapkan |
|---|---------|------------------------|
| 1 | Di portal, buka invoice → **Bayar online** | Halaman pembayaran Midtrans terbuka |
| 2 | Selesaikan pembayaran memakai kanal simulasi sandbox | Midtrans menyatakan transaksi berhasil |
| 3 | Kembali ke portal, muat ulang | Status invoice berubah lunas **tanpa** admin mengonfirmasi manual |
| 4 | Kirim ulang notifikasi yang sama dari dasbor Midtrans | Status tidak berubah dua kali dan tidak ada pembayaran ganda tercatat |

Langkah 4 menguji kunci idempotensi: satu notifikasi yang datang dua kali harus
diperlakukan sebagai satu kejadian.

---

### UAT-07 — Portal peserta

**Aktor:** peserta **Halaman:** `/portal/login`

| # | Langkah | Hasil yang diharapkan |
|---|---------|------------------------|
| 1 | Login memakai kredensial portal peserta | Masuk ke beranda portal |
| 2 | Buka **Perjalanan Saya** | Seluruh perjalanan milik akun ini tampil, termasuk yang sudah lampau |
| 3 | Buka perjalanan lampau | Invoice dan itinerary lama masih bisa diunduh |
| 4 | Ubah data profil | Perubahan tersimpan dan tampil setelah dimuat ulang |

Langkah 2 membuktikan satu akun melayani banyak perjalanan — pelanggan yang
kembali tidak perlu akun baru.

---

### UAT-08 — Dokumen dan OCR

**Aktor:** peserta, admin **Halaman:** `/portal/documents`, `/admin/documents`

| # | Langkah | Hasil yang diharapkan |
|---|---------|------------------------|
| 1 | Di portal, lihat **Syarat Dokumen** untuk negara tujuan | Daftar dokumen wajib dan opsional tampil sesuai negara |
| 2 | Unggah foto paspor (2–5 MB) | Unggahan diterima, berstatus menunggu |
| 3 | Sebagai admin, buka dokumen itu dan lihat hasil OCR | Teks terbaca; NIK terisi otomatis bila terdeteksi |
| 4 | Tolak dokumen dengan alasan | Peserta melihat status ditolak beserta alasannya |
| 5 | Unggah ulang, lalu setujui | Status berubah disetujui; ringkasan kelengkapan peserta bertambah |
| 6 | Coba unggah berkas > 13 MB | Ditolak dengan jelas, bukan menggantung |

OCR berjalan di sidecar Tesseract di dalam jaringan sendiri — tidak ada citra
dokumen yang dikirim ke pihak ketiga.

---

### UAT-09 — Itinerary dan briefing

**Aktor:** peserta **Halaman:** `/portal/itinerary`, `/portal/briefing`

| # | Langkah | Hasil yang diharapkan |
|---|---------|------------------------|
| 1 | Buka itinerary | Rencana perjalanan per hari tampil sesuai paket |
| 2 | Unduh PDF briefing | Berkas terunduh, memuat jadwal, titik kumpul, dan hal yang perlu dibawa |
| 3 | Lihat informasi tour leader | Nama dan kontak tour leader batch ini tampil |

---

### UAT-10 — Penanganan bandara

**Aktor:** tour_leader **Halaman:** `/admin/airport`

| # | Langkah | Hasil yang diharapkan |
|---|---------|------------------------|
| 1 | Login sebagai `tour_leader`, buka **Airport** | Hanya menu bandara yang tersedia |
| 2 | Inisialisasi checklist untuk satu batch | Seluruh peserta batch tampil sebagai baris checklist |
| 3 | Tandai bagasi, tiket, dan paspor beberapa peserta | Setiap tanda tersimpan dan bertahan setelah muat ulang |
| 4 | Unduh PDF laporan bandara | Berkas memuat rekap kesiapan seluruh peserta |
| 5 | Konfirmasi keberangkatan | Batch tercatat berangkat |

---

### UAT-11 — Chatbot WhatsApp

**Aktor:** calon pelanggan **Prasyarat:** URL publik aktif + webhook Fonnte diarahkan ke sana

| # | Langkah | Hasil yang diharapkan |
|---|---------|------------------------|
| 1 | Kirim pesan ke nomor WhatsApp bisnis | Bot membalas relevan dengan pertanyaannya |
| 2 | Tanyakan paket tertentu | Jawaban memuat informasi paket yang benar, bukan karangan |
| 3 | Buka **Admin → Chatbot Logs** | Percakapan tercatat lengkap |
| 4 | Jalankan **Buat lead dari percakapan** | Lead baru terbentuk berisi data dari obrolan |

---

### UAT-12 — Laporan dan analitik

**Aktor:** admin **Halaman:** `/admin`, `/admin/reports`

| # | Langkah | Hasil yang diharapkan |
|---|---------|------------------------|
| 1 | Buka dasbor | Ringkasan angka tampil dan konsisten dengan data yang baru dibuat |
| 2 | Buka analitik | Grafik tren terisi |
| 3 | Ekspor CSV untuk **leads**, **participants**, **invoices**, **batches** | Keempat berkas terunduh dan terbuka rapi di spreadsheet |

---

### UAT-13 — Kontrol akses (skenario negatif)

**Aktor:** semua peran. Yang diuji di sini adalah apa yang **tidak boleh** terjadi.

| # | Langkah | Hasil yang diharapkan |
|---|---------|------------------------|
| 1 | Sebagai `konsultan`, buka halaman **Users** | Ditolak |
| 2 | Sebagai `tour_leader`, buka **Invoices** | Ditolak |
| 3 | Sebagai `konsultan`, panggil `GET /api/v1/admin/users` langsung lewat URL | **403**, bukan data |
| 4 | Panggil endpoint admin mana pun tanpa login | **401** |
| 5 | Salah password 10 kali berturut-turut | Ditolak oleh pembatas laju, bukan dilayani terus |

Langkah 3 yang paling penting: menyembunyikan menu bukan kontrol akses.
Pembatasan harus berlaku di API meski antarmukanya dilewati.

---

### UAT-14 — Hak atas data pribadi (§25.5)

**Aktor:** peserta, admin **Halaman:** `/portal/profile`

| # | Langkah | Hasil yang diharapkan |
|---|---------|------------------------|
| 1 | Di portal, minta **salinan data saya** | Seluruh data pribadi yang tersimpan dikembalikan |
| 2 | Ajukan penghapusan akun | Permintaan tercatat berstatus menunggu |
| 3 | Sebagai admin, lihat antrean permintaan | Permintaan tampil beserta identitas pemohon |
| 4 | Proses permintaan | Data pribadi teranonimkan; invoice tetap ada demi kewajiban pembukuan |
| 5 | Proses permintaan yang sama sekali lagi | Ditolak — satu permintaan hanya bisa diproses sekali |

> **Batasan yang diketahui:** langkah 3–5 belum punya halaman admin. Untuk saat
> ini dijalankan lewat API (`GET` dan `POST /api/v1/admin/privacy/deletion-requests`).
> Fungsinya lengkap dan teruji; yang belum ada hanya antarmukanya.

---

### UAT-15 — Data induk

**Aktor:** admin / super_admin

| # | Langkah | Hasil yang diharapkan |
|---|---------|------------------------|
| 1 | Buat paket baru lengkap dengan itinerary, fasilitas, dan gambar | Paket muncul di katalog publik |
| 2 | Tambahkan batch keberangkatan | Batch tampil di halaman paket |
| 3 | Atur syarat dokumen untuk satu negara | Syarat itu tampil di portal peserta tujuan negara tersebut |
| 4 | Tambah profil tour leader | Bisa ditugaskan ke batch |
| 5 | Nonaktifkan seorang pengguna | Pengguna itu tidak bisa login lagi |
| 6 | Hapus sebuah paket | Hilang dari katalog, tetapi invoice dan peserta lama tetap utuh |

Langkah 6 menguji penghapusan lunak: data yang sudah dipakai transaksi tidak
boleh ikut hilang.

---

## Batasan yang diketahui

Disebutkan lebih dulu supaya tidak terlihat sebagai kegagalan saat demo:

1. **Antrean penghapusan §25.5 belum punya halaman admin** — API lengkap, UI
   belum. Lihat UAT-14.
2. **Midtrans berjalan di sandbox.** Tidak ada uang sungguhan berpindah. Ini
   disengaja.
3. **Webhook butuh URL publik.** Lewat `localhost`, Skenario 6 dan 11 tidak bisa
   dijalankan sama sekali.
4. **Sebagian gambar paket contoh menunjuk URL luar** yang bisa mati; itu data
   contoh, bukan kerusakan sistem.

---

## Lampiran A — Demo 15 menit

Untuk sidang, jalur ini menyentuh seluruh tulang punggung sistem tanpa
mengulang-ulang:

1. **Katalog** — buka halaman utama, saring, buka satu paket. *(1 menit)*
2. **Lead** — isi formulir konsultasi. *(1 menit)*
3. **CRM** — login admin, tugaskan lead, ubah status, tambah catatan. *(2 menit)*
4. **Konversi** — lead menjadi peserta. *(1 menit)*
5. **Invoice** — terbitkan, unduh PDF, konfirmasi lunas. *(2 menit)*
6. **Portal** — login peserta, tunjukkan perjalanan dan invoice. *(2 menit)*
7. **Dokumen + OCR** — unggah paspor, tunjukkan NIK terisi otomatis, setujui. *(3 menit)*
8. **Bandara** — checklist dan PDF laporan. *(2 menit)*
9. **Laporan** — dasbor dan ekspor CSV. *(1 menit)*

**Siapkan sebelum masuk ruangan:** stack sudah menyala, sudah login di dua
peramban berbeda (staf dan peserta) supaya tidak perlu logout-login di depan
penguji, dan satu berkas foto paspor sudah siap di desktop untuk langkah 7.

---

## Lembar hasil

| Skenario | Judul | Status | Catatan |
| -------- | ----- | ------ | ------- |
| UAT-01 | Katalog publik | ☐ Lulus ☐ Gagal | |
| UAT-02 | Lead masuk | ☐ Lulus ☐ Gagal | |
| UAT-03 | Tindak lanjut CRM | ☐ Lulus ☐ Gagal | |
| UAT-04 | Konversi peserta | ☐ Lulus ☐ Gagal | |
| UAT-05 | Invoice manual | ☐ Lulus ☐ Gagal | |
| UAT-06 | Pembayaran online | ☐ Lulus ☐ Gagal | |
| UAT-07 | Portal peserta | ☐ Lulus ☐ Gagal | |
| UAT-08 | Dokumen & OCR | ☐ Lulus ☐ Gagal | |
| UAT-09 | Itinerary & briefing | ☐ Lulus ☐ Gagal | |
| UAT-10 | Penanganan bandara | ☐ Lulus ☐ Gagal | |
| UAT-11 | Chatbot WhatsApp | ☐ Lulus ☐ Gagal | |
| UAT-12 | Laporan & analitik | ☐ Lulus ☐ Gagal | |
| UAT-13 | Kontrol akses | ☐ Lulus ☐ Gagal | |
| UAT-14 | Hak atas data pribadi | ☐ Lulus ☐ Gagal | |
| UAT-15 | Data induk | ☐ Lulus ☐ Gagal | |

Penguji: ______________________  Tanggal: ____________  Tanda tangan: ____________
