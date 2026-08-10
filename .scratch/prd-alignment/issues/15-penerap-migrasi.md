# 15 — Penerap migrasi untuk basis data berisi data

**What to build:** Sebuah deployment yang sudah berjalan dapat menerima migrasi baru tanpa kehilangan datanya. Saat ini tidak bisa: satu-satunya mekanisme adalah mount `db/migrations` ke `docker-entrypoint-initdb.d`, yang **hanya dijalankan Postgres saat volume pertama kali dibuat**. Tidak ada pustaka migrasi di `go.mod`, tidak ada `cmd/migrate`, dan satu-satunya target Makefile yang menyentuh migrasi adalah `rebuild-fresh` — yang menghapus seluruh basis data.

Ditemukan saat pengujian endpoint tiket 01–14 pada deployment lokal: volume berada di migrasi 007, sementara tiket 06, 07, dan 08 menambahkan migrasi 008 dan 009. Akibatnya konversi lead gagal total:

```
{"error":"CONVERT_FAILED","message":"relation \"lead_status_history\" does not exist (SQLSTATE 42P01)"}
```

Ini bukan cacat lokal. Setiap deployment yang volumenya dibuat sebelum migrasi terbaru menjalankan skema lama secara diam-diam, dan tiga tiket yang menambah skema — jejak audit, order gateway, idempotensi notifikasi — mati bersamanya. Satu-satunya jalan keluar yang tersedia hari ini adalah menghapus data produksi.

**Blocked by:** None — can start immediately.

**Status:** done

- [x] Ada tabel yang mencatat migrasi mana yang sudah diterapkan, beserta waktu penerapannya
- [x] Ada perintah yang menerapkan seluruh migrasi yang belum tercatat, dalam urutan nomornya, dan melewati yang sudah tercatat
- [x] Perintah itu aman dijalankan berulang: menjalankannya dua kali pada basis data yang sama tidak mengubah apa pun pada jalannya yang kedua
- [x] Setiap migrasi diterapkan dalam satu transaksi — migrasi yang gagal di tengah tidak meninggalkan skema separuh jadi, dan tidak tercatat sebagai sudah diterapkan
- [x] Basis data yang sudah ada di migrasi 007 dapat dibawa ke migrasi terbaru tanpa kehilangan baris satu pun
- [x] Basis data kosong menghasilkan skema yang sama persis dengan basis data yang dimigrasikan bertahap
- [x] Migrasi yang sudah terlanjur diterapkan lewat `docker-entrypoint-initdb.d` dikenali sebagai sudah diterapkan, bukan dijalankan ulang — kalau tidak, volume lama akan menabrak objek yang sudah ada
- [x] Ada target Makefile untuk menjalankannya, dan `docker compose up` pada volume lama menghasilkan skema terbaru tanpa langkah manual
- [x] Kegagalan migrasi saat start dilaporkan dengan jelas dan menghentikan proses — server yang berjalan di atas skema yang salah lebih buruk daripada server yang menolak start
- [x] Berkas migrasi tetap berupa SQL polos di `db/migrations` dengan penomoran yang sama; tidak ada berkas yang perlu ditulis ulang
