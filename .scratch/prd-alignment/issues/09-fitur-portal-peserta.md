# 09 — Kelengkapan fitur portal peserta

**What to build:** Portal peserta memenuhi apa yang dijanjikan FR-PORTAL-09, FR-PORTAL-10, dan FR-PORTAL-12: riwayat perjalanan yang informatif, arsip perjalanan lama yang lengkap, dan formulir konsultasi yang benar-benar terisi otomatis.

Ketiganya adalah fitur yang kurang dari yang dijanjikan, bukan cacat perilaku — kode yang ada sudah koheren, cakupannya saja yang belum sampai.

**Keputusan yang perlu diambil dan dicatat:** FR-PORTAL-09 menyebut badge "Selesai" dan "Dibatalkan". Skema saat ini tidak punya konsep perjalanan dibatalkan. Tentukan apakah menambah status pada data peserta atau menurunkan penanda dari data yang sudah ada, lalu catat keputusannya di berkas tiket ini.

**Blocked by:** 01 — Seam pengujian HTTP

**Status:** ready-for-agent

- [x] Kartu riwayat perjalanan menampilkan penanda status penyelesaian sesuai FR-PORTAL-09
- [x] Keputusan tentang padanan status "dibatalkan" tercatat di berkas tiket ini beserta alasannya
- [x] Peserta dapat mengunduh itinerary perjalanan lama, bukan hanya invoicenya, sesuai FR-PORTAL-10
- [x] Unduhan artefak perjalanan lama tetap memeriksa kepemilikan terhadap identitas portal pada token
- [x] Formulir konsultasi pada halaman detail paket terisi otomatis dengan nama, nomor WhatsApp, email, dan tipe kamar dari perjalanan terakhir, sesuai FR-PORTAL-12
- [x] Data pra-pengisian diambil dari endpoint portal, bukan dari penyimpanan peramban, agar tidak bergantung pada nilai yang mungkin usang
- [x] Field yang terisi otomatis tetap dapat disunting peserta
- [x] Status returning customer tetap diputuskan di sisi server berdasarkan nomor telepon dan tidak pernah dipercaya dari klien
- [x] Alur status invoice bergerak melalui seluruh tahap yang dijanjikan FR-INV-03, termasuk tahap yang saat ini tidak pernah tercapai
- [x] Ada test yang membuktikan peserta tidak dapat mengunduh artefak perjalanan milik identitas portal lain

## Comments

### Keputusan: padanan status "dibatalkan"

**Diturunkan dari data yang ada, bukan kolom status baru.**

FR-PORTAL-09 meminta badge "Selesai" / "Dibatalkan" pada kartu riwayat. Skema
tidak punya konsep perjalanan dibatalkan, jadi keputusannya antara menambah
kolom status pada `participants` atau menurunkan penandanya.

Dipilih **menurunkan**: perjalanan yang tanggal keberangkatannya sudah lewat dan
invoicenya tidak pernah lunas adalah perjalanan yang tidak jadi. Aturannya satu
baris (`tripCompletionStatus`).

Alasan menolak kolom baru: tidak ada satu pun alur di PRD yang membatalkan
peserta — tidak ada endpoint, tidak ada layar admin, tidak ada FR yang
menyebutnya. Kolom itu akan lahir tanpa penulis, jadi nilainya selamanya default
dan badge-nya selamanya "Selesai" untuk semua orang, termasuk perjalanan yang
jelas-jelas batal. Penurunan dari data setidaknya menjawab dengan benar hari ini.

**Konsekuensi yang perlu diketahui pemilik produk:** perjalanan lampau yang lunas
sebagian akan berlabel "Dibatalkan". Bila kelak ada alur pembatalan yang
sesungguhnya (mis. refund), kolom status peserta menjadi pilihan yang benar dan
`tripCompletionStatus` tinggal membacanya.

### Pelaksanaan (2026-08-09)

**Arsip perjalanan lama diperluas ke itinerary.** `GET
/portal/my-trips/:participant_id/itinerary`, memakai pemeriksaan kepemilikan yang
sama dengan unduhan invoice (`portalOwnsParticipant` dari tiket 04). Badan
renderingnya dibagi dengan `/portal/itinerary` lewat `itineraryOf`, jadi
itinerary perjalanan lampau adalah dokumen yang sama dengan yang peserta lihat
saat perjalanan itu masih akan datang. Ada test untuk kedua artefak (itinerary
dan invoice) terhadap identitas portal lain — keduanya `404`, bukan `403`, dengan
alasan yang sama seperti endpoint signed URL.

**Pra-pengisian pindah dari localStorage ke endpoint.** `GET
/portal/consultation-prefill` mengembalikan nama, nomor, email, dan tipe kamar
dari perjalanan terakhir. Sebelumnya `PackageDetailPage` membaca
`localStorage.portal_participant` — hanya berisi nama dan nomor, dan berisi
nilai dari kapan pun terakhir ditulis, sehingga koreksi admin minggu lalu akan
dikirim ulang dalam bentuk usang. Pra-pengisian hanya menimpa field yang masih
kosong, jadi apa pun yang sudah disunting peserta tidak tertimpa.

**Status pelanggan lama tetap keputusan server.** Endpoint prefill sengaja
**tidak** mengembalikannya, dan `CreateLead` tetap membuang `is_returning` serta
`portal_user_id` yang datang dari klien lalu menentukannya sendiri dari nomor
telepon. Ada testnya: klien yang mengaku pelanggan lama diabaikan.

**Tahap FR-INV-03 yang tidak pernah tercapai adalah `dibayar`.** Alurnya
`diterbitkan → menunggu_bayar → dibayar → lunas`, tapi pembayaran sebagian yang
disetujui meninggalkan invoice di `menunggu_bayar` — tidak bisa dibedakan dari
invoice yang belum dibayar sepeser pun. Sekarang bukti bayar disetujui yang
belum menutup tagihan memindahkannya ke `dibayar`, yang artinya persis itu: uang
diterima, sisa masih ada.
