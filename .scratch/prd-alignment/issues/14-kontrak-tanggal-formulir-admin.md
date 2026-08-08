# 14 — Kontrak tanggal formulir admin

**What to build:** Admin benar-benar dapat menerbitkan invoice dan membuka batch keberangkatan dari antarmuka. Keduanya saat ini mustahil: formulirnya mengirim tanggal polos `YYYY-MM-DD` dari `<input type="date">`, sedangkan field tujuannya bertipe `time.Time` yang hanya menerima RFC3339. Permintaannya gagal didekode sebelum menyentuh handler, dan admin melihat pesan galat generik tanpa petunjuk apa yang salah.

Terbukti secara langsung:

```
{"due_date":"2026-08-15"}             -> err=parsing time "2026-08-15" as "2006-01-02T15:04:05Z07:00"
{"due_date":"2026-08-15T00:00:00Z"}   -> err=<nil>
```

Dua alur yang terkena, keduanya lewat jalur yang sama:

- `POST /admin/invoices` — `web/src/pages/admin/AdminInvoicesPage.tsx:188` mengisi `due_date` pada `CreateInvoiceRequest` (`web/src/types/index.ts:206`, bertipe `string`, tanpa konversi), mendarat di `invoice.Invoice.DueDate`.
- `POST /admin/packages/:package_id/batches` — `web/src/pages/admin/AdminPackagesPage.tsx:343` mengisi `departure_date` dan `return_date`, mendarat di `pkg.PackageBatch`.

Filter tanggal pada halaman leads dan log chatbot **tidak** terkena: keduanya query param yang di-parse `time.Parse("2006-01-02")` di sisi server.

**Kenapa tiket ini ada.** Cacat ini tidak tercakup tiket 01–13 maupun spec — empat pass review yang melahirkan spec tidak menemukannya, karena hanya terlihat bila tombol simpan benar-benar ditekan dan repo belum punya test frontend. FR-INV-01 menuntut sistem dapat menghasilkan invoice; selama ini berlaku, klaim "seluruh 69 FR dapat ditunjukkan berlaku di kode" belum benar. Ditemukan saat mengerjakan tiket 03 dan tercatat di berkasnya; bukan regresi dari tiket manapun.

Backend dan frontend dalam satu tiket karena cacatnya justru terletak di antara keduanya dan tidak dapat didemokan terpisah.

**Keputusan yang perlu diambil dan dicatat:** sisi mana yang mengalah — antarmuka mengirim RFC3339, atau API menerima tanggal polos lewat tipe tanggal tersendiri. Pilihan kedua lebih jujur terhadap arti field-nya (jatuh tempo dan tanggal berangkat adalah tanggal kalender, bukan momen waktu) tapi menyentuh tipe domain. Putuskan satu, terapkan konsisten pada ketiga field, dan catat alasannya di berkas ini.

**Blocked by:** 01 — Seam pengujian HTTP

**Status:** ready-for-agent

- [ ] Payload tanggal yang benar-benar dikirim formulir invoice admin diterima, dan invoice tersimpan dengan tanggal jatuh tempo yang sama dengan yang dipilih admin
- [ ] Payload tanggal yang benar-benar dikirim formulir batch admin diterima, untuk tanggal berangkat maupun tanggal pulang
- [ ] Keputusan tentang sisi mana yang mengalah tercatat di berkas tiket ini beserta alasannya, dan diterapkan sama pada ketiga field
- [ ] Tidak ada field tanggal lain pada permintaan yang masih menuntut bentuk berbeda dari yang dikirim antarmukanya — seluruh `<input type="date">` yang masuk ke body permintaan ditelusuri
- [ ] Tanggal yang dikirim tanpa zona waktu tidak bergeser sehari saat ditulis maupun saat ditampilkan kembali
- [ ] Galat dekode tanggal menghasilkan pesan yang menyebut field bermasalah, bukan pesan cadangan generik — mengikuti pola yang dipasang tiket 03
- [ ] Ada test di seam HTTP yang menembakkan payload persis seperti yang dikirim antarmuka, sehingga kontrak ini tidak dapat putus lagi tanpa ketahuan
