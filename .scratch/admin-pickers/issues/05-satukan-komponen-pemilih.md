# 05 — Konversi lead memakai pemilih yang sama dengan halaman lain

**What to build:** Dialog konversi lead memakai komponen pemilih batch dan
peserta yang sama dengan keempat halaman lainnya, bukan salinan tersendiri.
Perilakunya bagi admin tidak berubah — yang hilang adalah kemungkinan kedua
pemilih menyimpang seiring waktu.

Dialog konversi sudah memakai dropdown batch sejak sebelumnya, tapi dibangun
lebih dulu daripada komponen bersama. Membiarkan dua implementasi berarti
memelihara dua perilaku, dan yang jarang dipakai akan tertinggal diam-diam.

**Blocked by:** 01 — Airport Handling bisa dipakai tanpa mengetahui UUID;
03 — Menyaring antrean review dokumen berdasarkan nama peserta.

**Status:** done

- [x] Dialog konversi memakai komponen pemilih batch bersama
- [x] Batch yang ditawarkan tetap terbatas pada paket yang diminati lead itu
- [x] Batch penuh atau ditutup tetap terlihat namun tidak bisa dipilih, dan
      alasannya terbaca
- [x] Peringatan pelanggan lama tetap muncul seketika saat dialog dibuka
- [x] Konversi tetap menghasilkan peserta, akun portal, dan invoice otomatis
      seperti sebelumnya
- [x] Tidak ada lagi dua implementasi pemilih batch di dalam kode

## Catatan implementasi

`BatchPicker` menerima `packageId`; bila diberikan ia membaca
`GET /admin/packages/{package_id}/batches` alih-alih daftar lintas paket, dan
tidak mengulang nama paket pada labelnya karena dialog sudah menyebutnya di atas.
Pembatasan ke paket lead karena itu bukan aturan yang dititipkan ke pemanggil,
melainkan konsekuensi dari argumen yang ia terima.

`disableUnavailable` dinyalakan hanya di sini. Pada keempat halaman lain
pemilihnya menyaring, dan menyaring peserta pada batch yang sudah penuh tetap
masuk akal; yang tidak boleh adalah **menugaskan** orang baru ke batch penuh.

Peringatan pelanggan lama dan jalur konversinya sendiri tidak disentuh — keduanya
membaca `convertLead.is_returning` dan memanggil `POST /admin/participants/convert`
persis seperti sebelumnya.

Diverifikasi di peramban pada sistem berjalan: dialog untuk lead paket "Korea
Selatan Honeymoon" hanya menawarkan batch paket itu, dan sebuah batch berstatus
`penuh` yang dibuat sementara tampil dalam keadaan tidak bisa dipilih beserta
alasannya. Batch sementara itu dihapus lagi setelahnya.

**Cacat yang ditemukan menjalankan tiket ini:** dialog konversi tidak pernah
berfungsi bagi `konsultan`. `GET /admin/packages/{package_id}/batches` ada di
grup ops, sedangkan konversi ada di grup sales yang memuat konsultan — jadi peran
yang justru paling sering mengkonversi lead menerima 403 dan melihat "Gagal
memuat daftar keberangkatan", bukan daftar batch. Cacat ini mendahului tiket ini
(dropdown lamanya memanggil endpoint yang sama), tapi ia membuat checkbox "batch
yang ditawarkan tetap terbatas pada paket yang diminati lead" tidak benar bagi
peran tersebut, jadi diperbaiki di sini. Keputusan aksesnya dicatat di tiket 01.
