# 04 — Menerbitkan invoice tambahan tanpa UUID

**What to build:** Admin menerbitkan invoice tambahan di luar paket — upgrade
kamar, penalti, layanan ekstra — dengan memilih peserta dari daftar. Batch tidak
lagi ditanyakan: ia diturunkan dari peserta yang dipilih.

Invoice utama sudah terbit otomatis saat konversi lead menjadi peserta dan tidak
diubah oleh tiket ini. Yang diperbaiki hanya formulir invoice tambahan.

Menurunkan batch dari peserta menghapus seluruh kelas kesalahan "peserta A
ditagih pada batch B", yang hari ini tidak dicegah apa pun.

**Blocked by:** 03 — Menyaring antrean review dokumen berdasarkan nama peserta.

**Status:** done

- [x] Kedua kolom UUID pada formulir invoice tidak ada lagi
- [x] Peserta dipilih lewat pencarian nama atau nomor WhatsApp
- [x] Batch tidak bisa diisi manual; ia mengikuti peserta yang dipilih
- [x] Setelah peserta dipilih, nama paket dan tanggal keberangkatannya tampil
      sebagai keterangan sebelum invoice disimpan
- [x] Nominal tetap bebas diisi, sehingga biaya upgrade dan penalti tetap bisa
      ditagih
- [x] Invoice yang terbit tetap memicu WhatsApp dan surel ke peserta seperti
      sebelumnya
- [x] Pembuatan invoice otomatis saat konversi tidak berubah perilakunya

## Catatan implementasi

Formulirnya tidak menyentuh backend sama sekali: `POST /admin/invoices` tetap
menerima `participant_id` dan `batch_id` yang sama, dan `batch_id` sekarang
diturunkan dari peserta yang dipilih alih-alih diketik. Karena itu jalur notifikasi
dan jalur invoice otomatis saat konversi tidak berubah satu barispun — keduanya
tetap dijaga test yang sudah ada (`notification_test.go`, `conversion_test.go`).

Diuji di peramban sampai satu langkah sebelum tombol terbit: peserta dipilih,
keterangan paket dan tanggal muncul, dan tombol "Terbitkan Invoice" menjadi aktif
— yang hanya mungkin bila `batch_id` benar-benar terisi, karena tombol itu mati
selama salah satu dari keduanya kosong. Invoice-nya sendiri tidak diterbitkan:
itu akan mengirim WhatsApp dan surel ke nomor sungguhan pada sistem berjalan.
