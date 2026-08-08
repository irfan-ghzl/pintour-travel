# 09 — Kelengkapan fitur portal peserta

**What to build:** Portal peserta memenuhi apa yang dijanjikan FR-PORTAL-09, FR-PORTAL-10, dan FR-PORTAL-12: riwayat perjalanan yang informatif, arsip perjalanan lama yang lengkap, dan formulir konsultasi yang benar-benar terisi otomatis.

Ketiganya adalah fitur yang kurang dari yang dijanjikan, bukan cacat perilaku — kode yang ada sudah koheren, cakupannya saja yang belum sampai.

**Keputusan yang perlu diambil dan dicatat:** FR-PORTAL-09 menyebut badge "Selesai" dan "Dibatalkan". Skema saat ini tidak punya konsep perjalanan dibatalkan. Tentukan apakah menambah status pada data peserta atau menurunkan penanda dari data yang sudah ada, lalu catat keputusannya di berkas tiket ini.

**Blocked by:** 01 — Seam pengujian HTTP

**Status:** ready-for-agent

- [ ] Kartu riwayat perjalanan menampilkan penanda status penyelesaian sesuai FR-PORTAL-09
- [ ] Keputusan tentang padanan status "dibatalkan" tercatat di berkas tiket ini beserta alasannya
- [ ] Peserta dapat mengunduh itinerary perjalanan lama, bukan hanya invoicenya, sesuai FR-PORTAL-10
- [ ] Unduhan artefak perjalanan lama tetap memeriksa kepemilikan terhadap identitas portal pada token
- [ ] Formulir konsultasi pada halaman detail paket terisi otomatis dengan nama, nomor WhatsApp, email, dan tipe kamar dari perjalanan terakhir, sesuai FR-PORTAL-12
- [ ] Data pra-pengisian diambil dari endpoint portal, bukan dari penyimpanan peramban, agar tidak bergantung pada nilai yang mungkin usang
- [ ] Field yang terisi otomatis tetap dapat disunting peserta
- [ ] Status returning customer tetap diputuskan di sisi server berdasarkan nomor telepon dan tidak pernah dipercaya dari klien
- [ ] Alur status invoice bergerak melalui seluruh tahap yang dijanjikan FR-INV-03, termasuk tahap yang saat ini tidak pernah tercapai
- [ ] Ada test yang membuktikan peserta tidak dapat mengunduh artefak perjalanan milik identitas portal lain
