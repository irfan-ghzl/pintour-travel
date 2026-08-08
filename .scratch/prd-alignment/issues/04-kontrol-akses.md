# 04 — Kontrol akses

**What to build:** Peserta hanya bisa membuka berkasnya sendiri, staf hanya bisa menandatangani akses ke berkas yang memang berhak dilihatnya, token portal dan token staf tidak bisa saling dipakai, dan organisasi tidak pernah kehilangan super admin terakhirnya.

Cacat inti: endpoint penandatanganan URL berkas privat menerima nama bucket dan path bebas dari klien tanpa pemeriksaan kepemilikan sama sekali — tersedia baik di jalur portal maupun jalur staf. Seorang peserta dapat menyusun path peserta lain dan memperoleh URL bertanda tangan ke paspor, KTP, atau bukti transfer milik orang itu.

**Catatan penting:** endpoint ini tidak muncul di matriks hak akses §5.3 maupun di FR manapun. Tingkat aksesnya karena itu **diputuskan**, bukan dibaca dari dokumen. Catat keputusan yang diambil, dan setelah selesai ajukan sebagai usulan baris tambahan pada §5.3 agar dokumen dan kode kembali sejajar.

Pembagian grup rute yang sudah ada **tidak** boleh diubah — keempat belas baris matriks §5.3 sudah diverifikasi cocok dengan kode, termasuk analytics yang memang terbuka untuk semua peran staf per FR-RPT-02.

**Blocked by:** 01 — Seam pengujian HTTP

**Status:** ready-for-agent

- [ ] Penandatanganan URL berhenti menerima nama bucket dan path bebas dari klien; penanda tangan menerima pengenal sumber daya domain lalu me-resolve lokasi berkasnya dari basis data
- [ ] Pada jalur portal, kepemilikan berkas diverifikasi terhadap identitas portal pada token; permintaan atas berkas milik peserta lain menghasilkan `404`, bukan `403`, agar keberadaannya tidak terungkap
- [ ] Pada jalur staf, akses diverifikasi terhadap peran; peran yang tidak berhak melihat dokumen peserta tidak dapat menandatanganinya
- [ ] Middleware token portal menolak token tanpa pengenal peserta, simetris dengan penjagaan yang sudah ada di middleware staf
- [ ] Token staf yang dikirim sebagai token portal ditolak, dan sebaliknya
- [ ] Operasi yang akan menurunkan peran atau menonaktifkan super admin terakhir ditolak dengan pesan yang menjelaskan alasannya
- [ ] Keputusan tingkat akses untuk endpoint penandatanganan URL tercatat di berkas tiket ini, beserta usulan baris tambahan untuk §5.3
- [ ] Ada test yang membuktikan peserta A tidak dapat memperoleh URL bertanda tangan atas berkas peserta B
- [ ] Ada test yang membuktikan pembagian grup rute yang sudah ada tetap berperilaku sesuai matriks §5.3
