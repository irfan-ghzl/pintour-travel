# 03 — Menyaring antrean review dokumen berdasarkan nama peserta

**What to build:** Admin mempersempit antrean review dokumen ke satu peserta
dengan mengetikkan sebagian nama atau nomor WhatsApp, bukan menyalin UUID dari
basis data.

Irisan ini memperkenalkan komponen pemilih peserta yang dipakai ulang tiket 04
dan 05. Tidak butuh endpoint baru: pencarian peserta sudah tersedia dan sudah
disaring per konsultan, sehingga pembatasan akses yang ada ikut berlaku tanpa
ditulis ulang.

**Blocked by:** None — can start immediately.

**Status:** done

- [x] Kolom isian `Filter Participant ID...` tidak ada lagi di halaman Review
      Dokumen
- [x] Mengetik sebagian nama atau sebagian nomor WhatsApp menemukan peserta
- [x] Setiap pilihan menampilkan nomor WhatsApp di samping nama, sehingga dua
      peserta bernama sama tetap bisa dibedakan
- [x] Pencarian tanpa hasil memberi tahu bahwa tidak ada yang cocok, bukan
      membiarkan daftar kosong tanpa penjelasan
- [x] Antrean dokumen menyusut ke peserta yang dipilih, dan pilihannya bisa
      dihapus
- [x] Test di seam HTTP membuktikan `search` mencocokkan potongan nama maupun
      potongan nomor
- [x] Test membuktikan konsultan hanya menerima peserta hasil konversi lead
      miliknya — kotak pencarian tidak menjadi jalan pintas melewati pembatasan
      akses

## Catatan implementasi

Tidak ada endpoint baru — benar seperti dugaan spec. Yang ditemukan justru di
sisi pengujian: `fakeParticipantRepo.List` **mengabaikan** `Filter.Search`
sepenuhnya, sehingga test apa pun tentang pencarian akan lulus melawan repository
palsu yang mengembalikan seluruh tabel. Cacat yang sama persis pernah ditemukan
pada `Filter.AssignedTo` di tiket sebelumnya. Fake-nya sekarang mencerminkan
`name ILIKE $1 OR phone ILIKE $1` milik Postgres.
