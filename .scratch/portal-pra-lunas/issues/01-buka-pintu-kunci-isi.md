# 01 — Peserta yang belum lunas bisa masuk portal dan membayar online

**What to build:** Peserta yang baru dikonversi memakai password dari WhatsApp,
masuk portal, melihat invoicenya, dan menyelesaikan pembayaran lewat Midtrans —
tanpa admin dan tanpa harus sudah membayar lebih dulu. Itinerary, briefing, dan
kontak tour leader tetap tertutup sampai pembayarannya dikonfirmasi.

Pintu dan gerbang isi harus mendarat bersama dalam satu tiket. Membuka login
lebih dulu tanpa mengunci isinya akan menyerahkan itinerary kepada setiap
peserta yang belum membayar; mengunci isi lebih dulu tanpa membuka login tidak
mengubah apa pun karena tidak ada yang bisa masuk.

**Blocked by:** None — can start immediately.

**Status:** done

- [x] Login portal tidak lagi menolak peserta yang belum lunas
- [x] Nomor tidak dikenal dan password salah tetap ditolak dengan pesan yang
      sama seperti sekarang, sehingga kegagalan autentikasi tidak bisa dibedakan
      dari kegagalan otorisasi
- [x] Peserta yang belum lunas menerima 200 pada: profil dan pembaruannya,
      daftar invoice, PDF invoice, unggah bukti transfer, pembuatan pembayaran
      online, dokumen beserta syarat dan unggahannya, URL bertanda tangan untuk
      berkas miliknya, hak data §25.5, dan riwayat perjalanan lampau
- [x] Peserta yang belum lunas menerima 403 pada itinerary, briefing, dan kontak
      tour leader, dengan kode dan pesan yang menjelaskan syaratnya — mengikuti
      bentuk yang sudah dipakai gerbang briefing H-14
- [x] Setelah pembayaran dikonfirmasi, ketiga endpoint itu menjawab 200 tanpa
      perlu login ulang
- [x] Gerbang H-14 pada briefing tetap berlaku dan kini bertumpuk dengan gerbang
      pembayaran
- [x] Peserta pertama kali dan pelanggan lama menerima perlakuan identik
- [x] Pembatasan kepemilikan antar peserta tidak melemah — test yang sudah ada
      tetap hijau
- [x] Catatan batasan pada UAT-05 di `docs/UAT.md` diperbarui, karena langkah
      "admin mengunggah bukti atas nama peserta" tidak lagi menjadi satu-satunya
      jalan

## Comments

Terverifikasi pada sistem berjalan (13 Agu 2026), memakai peserta hasil konversi
yang belum membayar sama sekali:

- Login portal dengan password dari WhatsApp: **200**, token terbit.
- Terbuka: `/me`, `/invoices`, `/documents`, `/documents/requirements`,
  `/my-data`, `/my-trips`, `/consultation-prefill` — semuanya **200**.
- Terkunci: `/itinerary`, `/briefing/pdf`, `/batch-leader` — **403
  PAYMENT_REQUIRED** dengan pesan yang menyebut syaratnya.
- `POST /invoices/{id}/create-payment` mengembalikan `snap_token` dari Midtrans
  sandbox — pembuktian langsung gerbang pembayaran tidak lagi mustahil dijangkau.
- Setelah pembayaran dikonfirmasi, dengan **token yang sama**: `/itinerary` dan
  `/batch-leader` menjadi 200, sementara `/briefing/pdf` berubah menjadi
  **403 NOT_YET** — kedua gerbang bertumpuk, keberangkatannya masih H-27.

Keputusan yang diambil saat implementasi, di luar yang tertulis di tiket:

1. **Penolakan login disatukan.** Nomor tak dikenal dulu dijawab "peserta tidak
   ditemukan" sementara password salah dijawab "nomor WA atau password salah" —
   dua jawaban berbeda yang memberi tahu siapa yang punya akun. Keduanya kini
   memakai pesan yang sama, yang dibutuhkan agar kegagalan autentikasi dan
   otorisasi benar-benar tak terbedakan dari luar.
2. **Konteks tour bawaan menjadi keberangkatan terbaru,** bukan tour aktif
   pertama. Aturan lama menjatuhkan pelanggan lama ke perjalanan tahun lalu yang
   sudah lunas, sehingga tagihan barunya tidak tampak di mana pun — perlakuan
   yang berbeda dari peserta pertama kali, yang justru dilarang tiket ini.
3. **Arsip `/my-trips/{id}/itinerary` ikut dijaga.** Tanpa itu, meminta tour
   sendiri lewat id adalah jalan memutar ke itinerary yang baru saja dikunci.
   Perjalanan yang keberangkatannya sudah lewat tetap terbuka — itu memang
   riwayat, dan tagihan baru tidak boleh mengambilnya (cerita 16).
