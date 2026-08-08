# 13 — Capai ambang coverage §21.10

**What to build:** Exit criteria pengujian §21.10 — "Coverage unit test backend ≥ 70%" — benar-benar terpenuhi dan dapat diverifikasi siapa pun dengan satu perintah. Pengukuran saat spec ini ditulis: **12,6%** untuk seluruh backend, dengan dua puluh paket tanpa berkas test sama sekali.

**Pengecualian aturan irisan vertikal** — tiket penutup, bukan irisan fitur. Sebagian besar angka seharusnya sudah naik sendiri karena tiket 02–12 masing-masing membawa testnya. Tiket ini menutup sisanya dan mengunci ambangnya.

**Batasan cara:** menaikkan angka dengan menumpuk test fungsi murni pada kode yang tidak berisiko **tidak diterima** sebagai pemenuhan tiket ini. Celah ditutup dengan menguji perilaku yang dijanjikan PRD kepada penggunanya, lewat seam HTTP, sesuai keputusan pengujian pada spec.

**Blocked by:** 02, 03, 04, 05, 06, 07, 08, 09, 10, 11, 12 — seluruh tiket perbaikan

**Status:** ready-for-agent

- [ ] Coverage backend menyeluruh mencapai sekurang-kurangnya ambang §21.10, diukur atas seluruh paket internal — bukan hanya paket yang kebetulan punya berkas test
- [ ] Perintah pengukurannya didokumentasikan sehingga penguji dapat memverifikasi angkanya sendiri
- [ ] Tidak ada paket yang berisi logika keputusan bisnis tanpa satu pun test
- [ ] Celah yang tersisa ditutup lewat seam HTTP terlebih dahulu; seam fungsi murni hanya untuk logika yang memang murni
- [ ] Dua celah seam yang diketahui — endpoint dashboard/analytics dan laporan yang memakai koneksi basis data langsung — ditangani atau dicatat eksplisit sebagai pengecualian beserta alasannya
- [ ] Berkas ringkasan hasil pengukuran disimpan agar dapat dirujuk sebagai bukti exit criteria
- [ ] Butir exit criteria §21.10 lain yang di luar jangkauan kode — uji performa, UAT, dan skor SUS — dicatat sebagai belum terpenuhi, bukan dibiarkan tampak selesai
