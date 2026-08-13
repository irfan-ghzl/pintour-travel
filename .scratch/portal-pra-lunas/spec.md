# Spec: Portal terbuka sejak peserta dibuat, isinya yang dikunci

Status: ready-for-agent

## Problem Statement

Peserta tidak bisa membayar secara online, karena tombol bayarnya berada di
dalam portal dan portal baru terbuka setelah pembayaran dikonfirmasi.

Rantainya tertutup rapat: `create-payment` berada di grup rute portal, sehingga
menuntut token portal; token portal hanya terbit dari login; login menolak
peserta yang belum punya perjalanan aktif; dan status aktif itu hanya menyala di
dalam alur konfirmasi pembayaran. Untuk bisa membayar, peserta harus sudah
membayar.

Dibuktikan pada sistem berjalan: konversi lead menghasilkan peserta dengan
status tidak aktif dan invoice Rp 37.800.000 berstatus diterbitkan, sementara
login portal peserta yang sama dijawab 401.

Akibat yang menyertainya, semuanya berasal dari sumber yang sama:

- **Gerbang pembayaran praktis mati.** Integrasi Midtrans hanya berguna untuk
  invoice kedua dan seterusnya, yaitu setelah peserta membayar invoice pertama
  lewat jalur lain.
- **Pelanggan lama diperlakukan berbeda dari pelanggan baru.** Nomor yang sudah
  punya perjalanan aktif dari tour sebelumnya lolos login, sehingga bisa
  membayar online. Peserta pertama kali tidak pernah bisa. Perbedaan ini tidak
  pernah dimaksudkan sebagai kebijakan.
- **Kredensial dikirim sebelum bisa dipakai.** Saat konversi, peserta menerima
  WhatsApp berisi password dan ajakan "gunakan portal untuk mengunggah dokumen".
  Pada data uji, pesan itu tiba 17 menit sebelum portalnya terbuka. Peserta yang
  menurut akan ditolak dan menyimpulkan passwordnya salah.
- **Bukti transfer tidak bisa diunggah peserta.** Skenario UAT-05 melingkar:
  unggah bukti butuh portal, portal butuh pembayaran dikonfirmasi, konfirmasi
  butuh bukti.
- **Dokumen tertunda.** Paspor dan visa butuh waktu berminggu-minggu, dan masa
  menunggu pembayaran justru saat paling tepat mengurusnya.

## Solution

Gerbangnya dipindahkan dari pintu ke isi.

Portal terbuka sejak peserta dibuat. Sebelum lunas, isinya terbatas pada apa
yang dibutuhkan untuk menyelesaikan pembayaran dan menyiapkan keberangkatan:
invoice, tombol bayar online, unggah bukti transfer, dan dokumen perjalanan.
Yang tetap terkunci adalah isi perjalanannya — itinerary, briefing, dan kontak
tour leader — yang memang belum menjadi hak peserta yang belum membayar.

Pola ini bukan hal baru di sistem ini. Briefing sudah dikunci per-isi: ia
menjawab `403 NOT_YET` dengan alasan "tersedia H-14 sebelum keberangkatan",
sementara sisa portal tetap bisa dibuka. Spec ini menerapkan pola yang sama pada
sumbu pembayaran, bukan sumbu waktu.

## User Stories

1. Sebagai peserta yang baru dikonversi, saya ingin bisa masuk portal memakai
   password yang dikirimkan lewat WhatsApp, sehingga pesan itu tidak menjanjikan
   sesuatu yang belum bisa saya lakukan.
2. Sebagai peserta yang belum membayar, saya ingin melihat invoice saya di
   portal, sehingga saya tahu persis berapa yang harus dibayar dan kapan jatuh
   temponya.
3. Sebagai peserta yang belum membayar, saya ingin mengunduh PDF invoice,
   sehingga saya punya dokumen resmi untuk keperluan transfer atau reimbursement
   kantor.
4. Sebagai peserta yang belum membayar, saya ingin menekan tombol bayar online,
   sehingga saya bisa melunasi lewat Midtrans tanpa menunggu admin.
5. Sebagai peserta yang membayar lewat transfer bank, saya ingin mengunggah
   bukti transfer sendiri dari portal, sehingga saya tidak perlu mengirimkannya
   lewat WhatsApp konsultan.
6. Sebagai peserta yang sudah mengunggah bukti, saya ingin melihat statusnya
   menunggu atau disetujui, sehingga saya tahu apakah perlu menghubungi
   konsultan.
7. Sebagai peserta yang membayar sebagian, saya ingin melihat sisa tagihan saya,
   sehingga saya tahu berapa yang masih harus dilunasi.
8. Sebagai peserta yang belum membayar, saya ingin mengunggah paspor dan
   dokumen lain, sehingga pengurusan visa bisa dimulai selagi pembayaran
   berjalan.
9. Sebagai peserta, saya ingin melihat daftar dokumen yang diwajibkan negara
   tujuan, sehingga saya tahu apa saja yang perlu disiapkan.
10. Sebagai peserta yang dokumennya ditolak, saya ingin membaca alasannya dan
    mengunggah ulang, tanpa harus sudah lunas.
11. Sebagai peserta yang belum membayar, saya ingin memperbarui data profil
    saya, sehingga kesalahan ejaan nama bisa saya perbaiki sebelum tiket
    diterbitkan.
12. Sebagai peserta yang belum membayar, saya ingin melihat pesan yang jelas
    ketika membuka itinerary, sehingga saya tahu ia terbuka setelah pembayaran
    dikonfirmasi dan bukan sedang rusak.
13. Sebagai peserta yang belum membayar, saya ingin menu yang belum tersedia
    ditandai terkunci, sehingga saya tidak menekannya berulang kali.
14. Sebagai peserta yang baru melunasi, saya ingin itinerary dan briefing
    langsung terbuka tanpa login ulang, sehingga tidak ada langkah tambahan
    setelah membayar.
15. Sebagai peserta pertama kali, saya ingin diperlakukan sama dengan pelanggan
    lama, sehingga kemampuan membayar online tidak bergantung pada apakah saya
    pernah ikut tour sebelumnya.
16. Sebagai pelanggan lama, saya ingin riwayat perjalanan saya tetap bisa
    dibuka, sehingga tour yang sudah saya bayar dulu tidak ikut terkunci oleh
    tagihan yang baru.
17. Sebagai peserta, saya ingin hak atas data pribadi saya — melihat salinan
    data dan mengajukan penghapusan — tersedia tanpa syarat pembayaran, sehingga
    hak itu tidak bisa ditahan oleh tagihan.
18. Sebagai admin, saya ingin tetap bisa mengonfirmasi pembayaran secara manual,
    sehingga peserta yang transfer tanpa mengunggah bukti tetap bisa dilayani.
19. Sebagai admin, saya ingin melihat bukti transfer yang diunggah peserta di
    antrean review, sehingga saya bisa menyetujuinya tanpa menunggu kiriman
    WhatsApp.
20. Sebagai konsultan, saya ingin peserta bisa mengurus pembayaran dan dokumen
    sendiri, sehingga saya tidak menjadi perantara untuk setiap berkas.
21. Sebagai tour leader, saya ingin peserta yang belum lunas tidak melihat
    kontak saya, sehingga saya tidak dihubungi oleh orang yang keberangkatannya
    belum pasti.
22. Sebagai pemilik usaha, saya ingin isi perjalanan tetap terkunci sampai
    pembayaran dikonfirmasi, sehingga itinerary yang disusun tim tidak bisa
    diambil tanpa membayar.
23. Sebagai penguji sidang, saya ingin melihat peserta membayar online dari awal
    sampai lunas dalam satu alur, sehingga integrasi payment gateway terbukti
    dipakai dan bukan sekadar ada.

## Implementation Decisions

### Pintu: login berhenti mensyaratkan status aktif

`PortalLogin` tidak lagi menolak peserta yang belum lunas. Ia tetap memverifikasi
nomor dan password seperti sekarang, dan tetap menolak nomor yang tidak dikenal
dengan pesan yang sama — kegagalan autentikasi dan kegagalan otorisasi tidak
boleh terlihat berbeda dari luar.

Token portal yang terbit tetap membawa `participant_id` dan `portal_user_id`
seperti sekarang. Bentuk klaimnya tidak berubah.

### Isi: gerbang per-endpoint mengikuti pola briefing

Endpoint dibagi dua, dan pembagiannya ditentukan oleh satu pertanyaan: apakah
peserta membutuhkannya untuk membayar atau bersiap?

**Terbuka sebelum lunas**

- Profil dan pembaruannya
- Daftar invoice, PDF invoice, sisa tagihan
- Unggah bukti transfer, dan pembuatan pembayaran online
- Dokumen: daftar, unggah, syarat per negara, dan URL bertanda tangan untuk
  berkas miliknya sendiri
- Hak data pribadi §25.5: salinan data dan pengajuan penghapusan
- Riwayat perjalanan lampau, yang memang sudah dibayar

**Terkunci sampai pembayaran dikonfirmasi**

- Itinerary perjalanan berjalan
- Briefing (tetap dengan gerbang H-14 yang sudah ada, kini bertumpuk dengan
  gerbang pembayaran)
- Kontak tour leader batch

Endpoint yang terkunci menjawab `403` dengan kode dan pesan yang menjelaskan
syaratnya, mengikuti persis bentuk yang sudah dipakai briefing hari ini. Peserta
harus bisa membedakan "belum boleh" dari "rusak" tanpa bertanya.

### `is_active` tetap berarti "sudah dikonfirmasi lunas"

Kolomnya tidak diubah maknanya dan tidak diganti nama. Yang berubah hanyalah
siapa yang membacanya: dulu gerbang login, sekarang gerbang isi. Mengganti
namanya menjadi sesuatu seperti `is_paid` akan lebih jujur, tapi ia tersebar di
repository, penjadwal, dan laporan — itu refactor lebar tersendiri, bukan bagian
spec ini.

### Notifikasi WhatsApp menyesuaikan

Pesan kredensial portal yang dikirim saat konversi kini benar apa adanya, karena
portalnya memang sudah bisa dibuka. Kalimatnya diperiksa ulang agar menyebutkan
apa yang bisa dilakukan sekarang — membayar, mengunggah bukti, mengunggah
dokumen — dan apa yang menyusul setelah lunas.

Pesan invoice tidak perlu lagi mengatakan PDF-nya belum bisa diambil, karena
sekarang bisa.

### Frontend menandai yang terkunci, bukan menyembunyikannya

Menu itinerary, briefing, dan tour leader tetap tampil bagi peserta yang belum
lunas, dalam keadaan tidak bisa ditekan dan disertai alasan. Menyembunyikannya
membuat peserta menyangka fiturnya tidak ada; menampilkannya sebagai terkunci
memberi tahu bahwa ia menunggu pembayaran.

Penguncian di frontend adalah kenyamanan, bukan kontrol akses. Yang menegakkan
aturan tetap API — dan itulah yang diuji.

## Testing Decisions

Test yang baik di sini menguji apa yang dilihat peserta dari luar: status kode
dan isi respons untuk setiap endpoint portal, pada dua keadaan pembayaran. Bukan
apakah suatu fungsi dipanggil.

### Seam

Seam HTTP yang sudah mapan: `RegisterRoutes` + `httptest` + fake repository +
service aplikasi asli. Prior art langsung ada di `access_control_test.go`, yang
sudah menguji kepemilikan berkas portal antar peserta, dan pada gerbang briefing
yang sudah diuji lewat seam yang sama. Tidak ada seam baru.

### Yang diuji

1. Peserta yang belum lunas berhasil login dan menerima token portal.
2. Nomor tidak dikenal dan password salah tetap ditolak, dengan pesan yang sama
   seperti sekarang.
3. Untuk peserta yang belum lunas, setiap endpoint dalam daftar "terbuka"
   menjawab 200.
4. Untuk peserta yang belum lunas, setiap endpoint dalam daftar "terkunci"
   menjawab 403 dengan kode yang menjelaskan syaratnya.
5. Setelah pembayaran dikonfirmasi, endpoint yang tadinya terkunci menjawab 200
   tanpa perlu login ulang.
6. Pembuatan pembayaran online berhasil untuk peserta yang belum lunas —
   pembuktian langsung bahwa gerbang pembayaran tidak lagi mustahil dijangkau.
7. Peserta pertama kali dan pelanggan lama menerima perlakuan identik pada
   keenam poin di atas.
8. Pembatasan kepemilikan yang sudah ada tidak melemah: peserta tetap tidak bisa
   membuka invoice, dokumen, atau perjalanan milik peserta lain.
9. Hak data pribadi §25.5 dapat diakses peserta yang belum lunas.

Poin 8 dijaga oleh test yang sudah ada; spec ini menuntut test itu tetap hijau,
bukan menulis ulang.

### Yang tidak diuji otomatis

Keadaan terkunci di antarmuka diverifikasi manual mengikuti `docs/UAT.md`. Repo
ini belum punya perkakas test frontend dan spec ini tidak menambahkannya.

## Out of Scope

- **Mengganti nama `is_active`.** Maknanya tetap "sudah dikonfirmasi lunas".
- **Mengubah aturan konfirmasi pembayaran manual.** Ia masih menandai lunas
  sepenuhnya tanpa menerima nominal; itu dibahas terpisah.
- **Sisa tagihan pada jalur konfirmasi manual.** Sisa tagihan yang diturunkan
  dari bukti yang disetujui sudah bekerja pada jalur online dan tidak diubah di
  sini.
- **Satu peserta per orang.** Lead dengan beberapa pax masih menghasilkan satu
  baris peserta.
- **Mengubah gerbang H-14 pada briefing.** Ia tetap berlaku dan kini bertumpuk
  dengan gerbang pembayaran.

## Further Notes

Keputusan yang paling layak dipertanyakan dalam spec ini adalah membuka unggah
dokumen sebelum pembayaran. Alasan membukanya konkret: paspor dan visa butuh
berminggu-minggu, dan menahannya sampai lunas memindahkan pekerjaan itu ke masa
yang paling sempit. Alasan menahannya juga sah: sistem jadi menyimpan pindaian
paspor milik orang yang mungkin tidak pernah jadi berangkat, dan §25 bicara soal
menyimpan data pribadi seperlunya.

Spec ini memilih membukanya, dengan catatan bahwa peserta yang membatalkan tetap
tercakup alur penghapusan §25.5 yang sudah ada. Bila keputusan itu dibalik,
satu-satunya yang berubah adalah letak tiga endpoint dokumen — sisanya tetap.

Perlu dicatat bahwa perubahan ini membuat integrasi Midtrans benar-benar terpakai
untuk pertama kalinya. Sampai hari ini setiap pengujian pembayaran online
berlangsung pada peserta yang portalnya sudah terbuka karena invoice sebelumnya
sudah lunas — yaitu satu-satunya keadaan ketika fitur itu tidak dibutuhkan.
