# 02 — Ketahanan runtime

**What to build:** Server tetap hidup dan responsif saat menghadapi kondisi tidak wajar. Galat tak terduga pada pekerjaan latar tidak lagi menjatuhkan layanan bagi seluruh pengguna, permintaan berukuran tidak wajar ditolak sebelum menghabiskan memori, dan token reset password yang tidak pernah dipakai tidak menumpuk selamanya.

Latar belakang: runtime HTTP hanya memulihkan panic pada goroutine yang dibuatnya sendiri. Aplikasi ini menjalankan sekitar enam belas goroutine latar untuk notifikasi, OCR, chatbot, dan blast WhatsApp — panic di salah satunya mematikan seluruh proses.

**Blocked by:** 01 — Seam pengujian HTTP

**Status:** ready-for-agent

- [ ] Permintaan yang memicu panic di handler menghasilkan respons galat terstruktur, bukan koneksi terputus, dan panic-nya tercatat di log
- [ ] Panic pada pekerjaan latar tidak menjatuhkan proses; kejadiannya tercatat di log dengan jejak tumpukan
- [ ] Pembungkus pemulih panic untuk pekerjaan latar disediakan satu kali sebagai utilitas bersama dan dipakai seluruh pemanggil — tidak disalin per tempat
- [ ] Permintaan dengan body melebihi batas wajar ditolak dengan status yang sesuai, sebelum body dibaca habis ke memori
- [ ] Batas ukuran body berlaku pada seluruh endpoint publik: form leads, login staf, login portal, dan kedua webhook
- [ ] Token reset password yang telah kedaluwarsa dibersihkan otomatis secara berkala, mengikuti pola penyapu yang sudah dipakai pembatas laju
- [ ] Permintaan lupa-password berulang tidak menyebabkan pertumbuhan memori tanpa batas
