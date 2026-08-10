# 16 — Pengingat pembayaran benar-benar terkirim (FR-INV-05)

**What to build:** Peserta yang belum membayar menerima pengingat WhatsApp pada H+1, H+3, dan H+6 sebagaimana FR-INV-05. Hari ini tidak satu pun terkirim: kuerinya gagal sebelum mengembalikan baris, setiap hari, tanpa ada yang menyadarinya.

Penyebabnya penulisan parameter interval. Kueri menyusun intervalnya dengan penggabungan string — `(NOW() - ($1 || ' days')::interval)` — sehingga Postgres menuntut parameter bertipe `text`, sementara pemanggilnya mengirim `int`. Driver menolaknya sebelum kueri sampai ke server. Direproduksi terhadap basis data sungguhan dengan driver yang sama:

```
int arg       -> err=failed to encode args[0]: unable to encode 1 into text format for text (OID 25)
make_interval -> err=<nil>
```

Kegagalannya hanya masuk log dan job berlanjut ke pengingat berikutnya, yang gagal dengan cara yang sama — sehingga tiga kegagalan per hari tampak seperti tidak ada peserta yang perlu diingatkan.

Tiket 08 menulis ulang fungsi ini untuk memperbaiki nomor invoice dan referensi notifikasinya, tetapi mempertahankan konstruksi kueri yang sama. Perbaikan tiket 08 karena itu belum pernah benar-benar berjalan.

**Blocked by:** None — can start immediately.

**Status:** done

- [x] Interval hari dibangun dari parameter bertipe angka, bukan dari penggabungan string
- [x] Ketiga pengingat (H+1, H+3, H+6) mengembalikan baris terhadap basis data sungguhan, bukan galat pengkodean parameter
- [x] Peserta dengan invoice belum lunas yang diterbitkan H+1 lalu menerima tepat satu pengingat, dan pesannya menyebut nomor invoicenya
- [x] Pengingat H+6 tercatat dengan jenis pesannya sendiri, terbedakan dari H+1
- [x] Catatan notifikasinya menunjuk ke invoice, sesuai jenis referensi yang ditulisnya
- [x] Dedup per hari tetap berlaku: menjalankan job dua kali dalam satu hari tidak mengirim pengingat kedua
- [x] Kegagalan kueri pada job terjadwal tidak lagi berlalu sebagai "tidak ada yang perlu dikirim" — job yang gagal terbaca sebagai gagal
- [x] Pola parameter yang sama disisir di seluruh kode agar tidak ada kueri lain yang menyimpannya
- [x] Ada test yang menjalankan kueri ini terhadap basis data sungguhan; test dilewati dengan penanda bila basis data tidak tersedia, mengikuti seam kedua pada spec
