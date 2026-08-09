# 17 — Dua jalur yang menjawab dengan kode galat yang salah

**What to build:** Dua permintaan berhenti menjawab `500` untuk keadaan yang bukan kesalahan server. Keduanya ditemukan saat pengujian endpoint tiket 01–14, keduanya kecil, dan keduanya menyembunyikan sebab sebenarnya dari pemanggil.

**Pengecualian aturan irisan vertikal** — dua perbaikan kecil yang tidak berbagi kode, dikelompokkan agar tidak jadi dua tiket sepele. Mengikuti pola tiket 12.

### 1. Penandatanganan URL untuk berkas ber-path URL absolut

Deployment tanpa kunci penyimpanan menyimpan URL absolut pada `file_path` alih-alih path relatif-bucket; seluruh data seed berbentuk begitu. Penanda tangan tetap menempelkannya di belakang nama bucket dan mencoba menandatanganinya:

```
Post ".../object/sign/participant-documents/https://example.com/docs/Paspor_Rini.pdf"
```

Hasilnya `500` pada jalur yang seharusnya sukses. Pemeriksaan kepemilikannya sendiri sudah benar — `pathBelongsToParticipant` secara eksplisit meloloskan URL absolut sebagai fallback manual, dan penolakan lintas peserta tetap menjawab `403`/`404` dengan tepat. Yang belum ditangani hanya apa yang terjadi **setelah** akses diizinkan. Sisi frontend sudah menanganinya sejak tiket 04; sisi server belum, padahal tiket 04 justru memindahkan keputusannya ke server.

### 2. Lead dengan paket yang tidak dikenal

`POST /leads` dengan `package_id` yang tidak ada menjawab `500`. Yang terjadi adalah pelanggaran kunci asing — masukan yang salah, bukan server yang rusak. Endpoint ini publik dan hanya dibatasi laju, jadi masukan sembarang wajar terjadi dan tidak seharusnya terbaca sebagai kegagalan sistem di log.

**Blocked by:** None — can start immediately.

**Status:** done

- [x] Berkas privat yang `file_path`-nya sudah berupa URL absolut dikembalikan apa adanya kepada pemanggil yang berhak, tanpa mencoba menandatanganinya
- [x] Pemeriksaan kepemilikan tetap berjalan lebih dulu: pemanggil yang tidak berhak tetap menerima `404` pada jalur portal dan `403` pada jalur staf, apa pun bentuk path-nya
- [x] Perilaku untuk path relatif-bucket tidak berubah
- [x] `POST /leads` dengan `package_id` atau `batch_id` yang tidak ada menjawab `400` atau `404` dengan pesan yang menyebut field bermasalah, bukan `500`
- [x] Pelanggaran kunci asing dikenali sebagai kelas galat, bukan dicocokkan dari teks pesannya
- [x] Galat basis data yang sesungguhnya tetap `500` dan tetap tercatat lengkap di log server
- [x] Ada test untuk kedua jalur di seam HTTP
