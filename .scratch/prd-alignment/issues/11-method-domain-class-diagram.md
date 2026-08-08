# 11 — Method domain sesuai class diagram §14.4

**What to build:** Class diagram §14.4 dapat ditunjukkan berlaku di kode. Diagram mencantumkan sekitar dua puluh dua method pada sembilan class — otentikasi pengguna, pemeriksaan peran, ketersediaan kursi batch, hitung mundur keberangkatan, perubahan status lead, aktivasi portal peserta, keterlambatan invoice, persetujuan dokumen, kelengkapan checklist bandara, dan verifikasi password portal. Seluruh paket domain saat ini hanya berisi satu fungsi, dan tidak ada satu pun method pada entity manapun.

Catatan di bawah diagram bahkan menegaskan pola yang tidak ada: "Method dengan parameter penerima". Kode memakai model domain anemik — struct hanya data, seluruh perilaku ada di lapisan aplikasi dan repository.

Arah penyelarasan sudah diputuskan: kode yang menyesuaikan dokumen.

**Batas yang harus dijaga:** hanya perilaku yang dapat dihitung dari data entity itu sendiri yang pindah ke domain. Method yang memerlukan I/O — kueri basis data, panggilan jaringan, pembuatan berkas — **tetap** di lapisan aplikasi. Method orkestrasi pada diagram diinterpretasikan sebagai perilaku entity setara yang dapat diuji tanpa basis data; catat pemetaannya agar diagram dan kode dapat dicocokkan baris per baris.

**Blocked by:** 01 — Seam pengujian HTTP

**Status:** ready-for-agent

- [ ] Entity domain memperoleh method perilaku yang murni predikat dan turunan sebagaimana class diagram §14.4
- [ ] Lapisan aplikasi memakai method-method itu alih-alih menghitung ulang logika yang sama di tempatnya sendiri
- [ ] Tidak ada method domain yang melakukan I/O
- [ ] Pemetaan antara setiap method pada diagram dan padanannya di kode tercatat di berkas tiket ini, termasuk method yang sengaja tidak dipindahkan beserta alasannya
- [ ] Setiap method baru punya unit test langsung, mengikuti gaya tabel kasus masukan-ke-harapan yang sudah dipakai test fungsi murni di repo
- [ ] Tidak ada perubahan perilaku yang terlihat pengguna dari tiket ini — seluruh test yang ada tetap hijau
