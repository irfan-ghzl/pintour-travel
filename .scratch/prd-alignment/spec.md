# Spec: Menyelaraskan Implementasi ke PRD v3.1.0

Status: ready-for-agent

## Problem Statement

PRD v3.1.0 menyatakan dirinya **"Selaras dengan implementasi (terverifikasi terhadap kode)"**, dan changelog v3.0.0/v3.1.0 mengklaim dua kali audit menyeluruh terhadap kode. Empat pass review terhadap repo menunjukkan klaim itu belum sepenuhnya benar.

Dari sisi **pemilik produk**, sistem tampak lengkap tapi ada perilaku yang diam-diam tidak terjadi: peserta yang membayar lewat Midtrans bisa dinyatakan lunas tanpa membayar penuh; peserta yang gagal dikonversi sekali bisa terkunci permanen dari portalnya sendiri; dan satu peserta bisa membuka dokumen peserta lain.

Dari sisi **penguji skripsi**, dokumen menjanjikan hal-hal yang tidak bisa ditunjukkan di kode: coverage unit test ≥70% (aktual 12,6%), validasi input via `go-playground/validator` (terpasang tapi tidak pernah dipanggil), audit trail perubahan status lead (parameter pelakunya dibuang), dan sekitar 22 method pada class diagram §14.4 yang tidak ada satu pun di `internal/domain`.

Dari sisi **developer**, tidak ada satu pun test yang menembus lapisan HTTP, sehingga seluruh RBAC, middleware, dan alur antar-lapisan tidak terlindungi regresi — padahal di situlah mayoritas cacat ditemukan.

## Solution

PRD diperlakukan sebagai **sumber kebenaran**. Kode disesuaikan ke dokumen, bukan sebaliknya. Setiap penyimpangan yang ditemukan pada empat pass review ditutup di sisi implementasi, dan setiap klaim yang belum benar dibuat menjadi benar.

Pekerjaan ini juga menambahkan satu seam pengujian di titik tertinggi yang mungkin (HTTP) supaya klaim coverage §21.10 tercapai secara bermakna — bukan dengan menumpuk unit test fungsi murni, melainkan dengan menguji perilaku yang benar-benar dijanjikan PRD kepada penggunanya.

Hasil akhir: seluruh 69 FR, NFR §10, skema §12, rekapitulasi notifikasi §17.3, diagram §14, dan exit criteria §21.10 dapat ditunjukkan berlaku di kode.

## User Stories

### Keamanan & Kontrol Akses

1. Sebagai **peserta**, saya ingin dokumen paspor dan KTP saya hanya bisa dibuka oleh diri saya dan staf yang berwenang, agar data pribadi saya tidak bocor ke peserta lain.
2. Sebagai **peserta**, saya ingin bukti transfer saya tidak bisa diunduh peserta lain, agar informasi keuangan saya terlindungi.
3. Sebagai **super admin**, saya ingin peran `konsultan` dan `tour_leader` tidak bisa menandatangani akses ke file privat milik peserta manapun, agar prinsip least-privilege pada §5.3 benar-benar ditegakkan.
4. Sebagai **super admin**, saya ingin token portal peserta ditolak di seluruh endpoint staf, dan token staf ditolak di seluruh endpoint portal, agar pemisahan token pada §19.1 berlaku dua arah.
5. Sebagai **super admin**, saya ingin sistem mencegah saya menurunkan peran atau menonaktifkan super admin terakhir, agar organisasi tidak pernah kehilangan akses ke manajemen pengguna.
6. Sebagai **super admin**, saya ingin peran yang saya isikan saat membuat atau menyunting pengguna divalidasi terhadap empat peran resmi, agar tidak tercipta akun yang lolos autentikasi tapi ditolak semua grup RBAC.
7. Sebagai **pemilik sistem**, saya ingin endpoint publik menolak body permintaan berukuran tidak wajar, agar satu permintaan besar tidak menghabiskan memori server.
8. Sebagai **pemilik sistem**, saya ingin token reset password kedaluwarsa dibersihkan otomatis, agar memori proses tidak tumbuh tanpa batas.

### Ketepatan Pembayaran

9. Sebagai **peserta**, saya ingin pembayaran cicilan saya lewat Midtrans dihitung tepat satu kali, agar invoice tidak dinyatakan lunas padahal saya baru membayar sebagian.
10. Sebagai **admin keuangan**, saya ingin notifikasi webhook yang dikirim ulang oleh Midtrans tidak membuat bukti bayar ganda, agar rekonsiliasi keuangan tetap akurat.
11. Sebagai **peserta**, saya ingin pembayaran yang saya selesaikan di sesi Snap yang dibuka lebih awal tetap tercatat, agar uang saya tidak hilang saat saya sempat memuat ulang halaman.
12. Sebagai **admin keuangan**, saya ingin transaksi yang ditandai mencurigakan oleh Midtrans ditahan apa pun metode pembayarannya, agar tidak ada portal yang aktif atas transaksi yang belum dilepas.
13. Sebagai **peserta**, saya ingin halaman invoice tetap terbuka saat pembayaran saya berstatus menunggu konfirmasi gateway, agar saya bisa memantau statusnya.
14. Sebagai **peserta**, saya ingin satu klik "Bayar Online" menghasilkan tepat satu transaksi Midtrans, agar tidak ada order yang menggantung.
15. Sebagai **admin keuangan**, saya ingin galat basis data saat memproses webhook tidak dilaporkan sebagai "invoice tidak ditemukan", agar Midtrans tetap mengirim ulang dan pembayaran tidak hilang diam-diam.
16. Sebagai **admin keuangan**, saya ingin bukti bayar yang saya review dipastikan milik invoice yang sedang saya buka, agar salah tempel ID tidak menyelesaikan invoice yang keliru.
17. Sebagai **peserta**, saya ingin status invoice bergerak melalui seluruh tahap yang dijanjikan §FR-INV-03, agar tampilan status di portal konsisten dengan dokumen.

### Konversi Lead & Akun Portal

18. Sebagai **peserta baru**, saya ingin akun portal saya dan data keberangkatan saya dibuat sebagai satu kesatuan, agar kegagalan di tengah proses tidak meninggalkan akun yatim yang membuat saya tidak bisa login selamanya.
19. Sebagai **peserta baru**, saya ingin password sementara portal dikirimkan otomatis ke WhatsApp saya, agar saya tidak bergantung pada admin yang harus mengetiknya manual.
20. Sebagai **admin**, saya ingin tipe kamar yang saya pilih saat konversi divalidasi sebelum data ditulis, agar proses tidak gagal di tengah jalan karena nilai yang tidak dikenal skema.
21. Sebagai **pelanggan lama**, saya ingin sistem mengenali nomor saya dan memakai kembali akun portal yang sudah ada, agar saya tidak perlu mengelola dua akun untuk dua perjalanan.
22. Sebagai **admin**, saya ingin diberi tahu dengan jelas saat konversi memakai akun lama, agar saya tidak mengirim password baru yang tidak berlaku.

### CRM & Jejak Audit

23. Sebagai **admin**, saya ingin setiap perubahan status lead tercatat lengkap dengan waktu **dan pengguna yang melakukannya**, agar saya bisa menelusuri siapa memindahkan lead ke tahap mana.
24. Sebagai **admin**, saya ingin perubahan status yang dilakukan otomatis oleh penjadwal juga tercatat dengan atribusi sistem, agar riwayat tidak berlubang.
25. Sebagai **konsultan**, saya ingin log aktivitas lead menampilkan riwayat status di samping catatan saya, agar konteks percakapan dengan pelanggan utuh.
26. Sebagai **admin**, saya ingin lead yang sudah dihapus lunak tidak muncul di daftar dan laporan manapun, agar angka yang saya lihat konsisten.
27. Sebagai **admin keuangan**, saya ingin invoice yang dihapus lunak tidak muncul di daftar, PDF, laporan, maupun penjadwal, agar tidak ada tagihan hantu yang mengejar peserta.
28. Sebagai **pemilik data**, saya ingin bukti bayar dan checklist bandara juga tunduk pada mekanisme hapus lunak, agar klaim §13.1 berlaku untuk seluruh tabel transaksional.

### Notifikasi & Automasi

29. Sebagai **peserta**, saya ingin menerima pesan WhatsApp berisi daftar dokumen setelah pembayaran saya dikonfirmasi lewat jalur manapun, agar saya tahu langkah berikutnya tanpa harus membuka portal lebih dulu.
30. Sebagai **peserta**, saya ingin pengingat pembayaran hari ke-6 dibedakan dari hari ke-1 pada log notifikasi, agar riwayat komunikasi saya bisa ditelusuri dengan benar.
31. Sebagai **peserta**, saya ingin pengingat pembayaran menyebutkan nomor invoice saya, agar saya tahu tagihan mana yang dimaksud.
32. Sebagai **admin**, saya ingin setiap catatan notifikasi menunjuk ke entitas yang benar sesuai jenis referensinya, agar penelusuran dari invoice ke riwayat WhatsApp tidak putus.
33. Sebagai **super admin**, saya ingin menerima seluruh email notifikasi admin, agar saya tidak buta terhadap lead baru, bukti bayar, dan dokumen masuk hanya karena peran saya bukan `admin`.
34. Sebagai **admin**, saya ingin email invoice jatuh tempo tidak dikirim berulang setiap hari kepada peserta yang tidak punya nomor WhatsApp, agar tidak dianggap spam.
35. Sebagai **konsultan**, saya ingin email pemberitahuan lead baru memuat nama paket yang diminati, agar saya bisa menyiapkan penawaran sebelum menelepon.
36. Sebagai **pemilik sistem**, saya ingin jumlah template notifikasi di kode cocok dengan rekapitulasi §17.3, agar dokumen dan implementasi bisa diaudit silang.

### Portal Peserta

37. Sebagai **peserta**, saya ingin melihat penanda status penyelesaian pada setiap kartu riwayat perjalanan, agar saya bisa membedakan perjalanan yang tuntas dari yang dibatalkan.
38. Sebagai **peserta**, saya ingin mengunduh itinerary perjalanan lama saya, bukan hanya invoicenya, agar arsip perjalanan saya lengkap.
39. Sebagai **pelanggan lama**, saya ingin formulir konsultasi terisi otomatis dengan nama, nomor WhatsApp, email, dan tipe kamar dari perjalanan terakhir saya, agar saya tidak mengetik ulang data yang sudah dimiliki sistem.
40. Sebagai **peserta**, saya ingin tombol unduh invoice dan laporan benar-benar menghasilkan berkas di peramban apa pun yang saya pakai, agar saya tidak perlu berganti peramban.
41. Sebagai **peserta**, saya ingin dokumen PDF yang saya terima menampilkan huruf dan simbol Indonesia dengan benar, agar materi briefing terbaca.

### Operasional Bandara & Laporan

42. Sebagai **tour leader**, saya ingin laporan pasca-handling memuat tanggal keberangkatan dan nama paket yang sebenarnya, bukan potongan pengenal internal, agar laporan bisa diserahkan ke manajemen.
43. Sebagai **tour leader**, saya ingin laporan mencantumkan tour leader yang ditugaskan pada batch, bukan siapa pun yang terakhir menyentuh checklist, agar pertanggungjawaban jelas.
44. Sebagai **tour leader**, saya ingin membuka dashboard checklist tanpa memicu penulisan data setiap sepuluh detik, agar perangkat dan server tidak terbebani di tengah kesibukan bandara.
45. Sebagai **tour leader**, saya ingin unduhan laporan PDF tidak gagal diam-diam saat basis data sedang bermasalah, agar saya tahu harus mencoba lagi.
46. Sebagai **admin**, saya ingin daftar dokumen dipaginasi, agar halaman review tetap responsif setelah data bertahun-tahun menumpuk.
47. Sebagai **admin**, saya ingin ringkasan "sudah disetujui dari total" pada halaman review dihitung dari seluruh dokumen peserta, bukan dari hasil filter yang sedang aktif, agar angkanya tidak menyesatkan.
48. Sebagai **admin**, saya ingin nama panjang pada laporan PDF terpotong tanpa merusak karakter, agar laporan tetap rapi.

### Kualitas Data & OCR

49. Sebagai **admin**, saya ingin nama hasil OCR terbaca benar meskipun baris teks memiliki spasi di depan, agar saya tidak perlu mengoreksi manual setiap dokumen.
50. Sebagai **peserta**, saya ingin berkas yang saya unggah disimpan dengan nama yang aman, agar nama berkas tidak bisa dipakai memanipulasi permintaan ke penyimpanan.

### Pengembangan & Verifikasi

51. Sebagai **developer**, saya ingin semua masukan pengguna divalidasi oleh validator yang sudah terpasang, agar janji §19.3 benar-benar berlaku dan nilai tidak sah ditolak sebelum menyentuh basis data.
52. Sebagai **developer**, saya ingin panic pada goroutine latar tidak mematikan seluruh proses, agar satu galat tidak menjatuhkan layanan bagi semua pengguna.
53. Sebagai **developer**, saya ingin entity domain memiliki method sebagaimana digambarkan class diagram §14.4, agar diagram dapat ditunjukkan berlaku di kode.
54. Sebagai **developer**, saya ingin ada seam pengujian di lapisan HTTP, agar RBAC dan alur antar-lapisan terlindungi dari regresi.
55. Sebagai **developer**, saya ingin coverage unit test backend mencapai ambang §21.10, agar exit criteria pengujian terpenuhi secara jujur.
56. Sebagai **penguji skripsi**, saya ingin setiap klaim terukur di PRD dapat saya verifikasi langsung di repositori, agar dokumen dan sistem bisa dinilai sebagai satu kesatuan.
57. Sebagai **developer**, saya ingin berkas biner hasil kompilasi benar-benar diabaikan Git, agar tidak ada artefak puluhan megabita yang tidak sengaja masuk riwayat.
58. Sebagai **developer**, saya ingin utilitas yang sama tidak diduplikasi di banyak paket, agar perubahan format cukup dilakukan di satu tempat.

## Implementation Decisions

### Arah penyelarasan

- **PRD adalah sumber kebenaran.** Setiap ketidakcocokan ditutup dengan mengubah kode. Dokumen tidak diedit sebagai bagian dari pekerjaan ini.
- Satu pengecualian yang perlu dicatat: **FR-RPT-02 menyatakan endpoint analytics dapat diakses semua peran staf.** Endpoint statistik dan analytics karena itu **tetap** berada pada grup tanpa pembatasan peran. Yang dipindahkan ke grup ops adalah endpoint penandatanganan URL file privat, yang tidak diatur FR manapun dan tidak seharusnya terbuka untuk seluruh peran.

### Kontrol akses

- Endpoint penandatanganan URL privat berhenti menerima nama bucket dan path bebas dari klien. Penanda tangan menerima **pengenal sumber daya domain** (mis. pengenal dokumen atau bukti bayar), lalu me-resolve bucket dan path dari basis data setelah memeriksa kepemilikan. Pada jalur portal, kepemilikan diverifikasi terhadap identitas portal pada token; pada jalur staf, terhadap peran.
- Middleware JWT portal mendapat penjagaan simetris dengan middleware staf: token tanpa pengenal peserta ditolak, sebagaimana token tanpa pengenal pengguna dan peran ditolak di sisi staf.
- Manajemen pengguna menolak operasi yang akan menghapus super admin terakhir, dan memvalidasi peran terhadap himpunan peran resmi §5.3.

### Ketahanan runtime

- Instance Echo memasang middleware pemulihan panic dan pembatas ukuran body.
- Setiap goroutine latar yang dijalankan aplikasi dibungkus pemulih panic bersama, karena runtime HTTP tidak memulihkan goroutine yang dibuat aplikasi. Pola pembungkus ini disediakan satu kali dan dipakai seluruh pemanggil, bukan disalin di tiap tempat.
- Penyimpanan token reset password mendapat penyapu kedaluwarsa berkala, mengikuti pola yang sudah dipakai pembatas laju.

### Validasi masukan

- Fungsi pengikat JSON bersama memanggil validator setelah dekode, sehingga seluruh handler mendapat validasi tanpa perubahan per-handler.
- Struktur payload permintaan diberi tag validasi. Prioritas: peran pengguna, tipe kamar, status invoice, status dokumen, status lead, format email, dan format nomor telepon.
- Galat validasi dipetakan ke respons `400` dengan pesan yang menyebut field bermasalah.

### Pembayaran gateway

- Pemrosesan notifikasi gateway menjadi **idempoten terhadap identitas transaksi**, bukan terhadap status invoice. Kunci idempotensi disimpan sehingga notifikasi berulang untuk transaksi yang sama tidak pernah menghasilkan bukti bayar kedua. Ini menutup kasus pembayaran sebagian yang sebelumnya lolos karena invoice belum berstatus lunas.
- Pengenal order gateway tidak lagi ditimpa. Relasi invoice ke order menjadi satu-ke-banyak sehingga pembayaran atas sesi yang dibuka lebih awal tetap dapat dicocokkan.
- Penjagaan status fraud tidak lagi dikecualikan berdasarkan metode pembayaran.
- Galat basis data saat memproses webhook dibedakan dari kondisi "tidak ditemukan", dan dikembalikan sebagai galat sehingga gateway mengirim ulang.
- Penyelesaian bukti bayar memverifikasi bahwa bukti tersebut milik invoice yang dirujuk.
- Pembuatan transaksi pembayaran di antarmuka menjadi aksi eksplisit sekali jalan, bukan kueri yang dapat diambil ulang otomatis oleh pustaka data.
- Kosakata status invoice di frontend disamakan dengan skema, termasuk status menunggu konfirmasi gateway, dan komponen status tidak lagi mengasumsikan seluruh status terdaftar di peta lokalnya.

### Integritas konversi lead

- Pembuatan identitas portal dan data peserta dijalankan dalam **satu transaksi basis data**. Kegagalan pada tahap manapun mengembalikan seluruh perubahan.
- Repository memperoleh kemampuan menjalankan beberapa operasi dalam satu transaksi. Antarmuka repository yang ada diperluas dengan mekanisme unit-of-work, bukan dengan membocorkan tipe basis data ke lapisan aplikasi.
- Password sementara dikirim ke WhatsApp peserta sebagai bagian dari alur konversi, memenuhi FR-PORTAL-01. Nilai mentahnya tetap dikembalikan ke admin sebagai cadangan.

### Jejak audit & hapus lunak

- Ditambahkan tabel riwayat status lead yang mencatat status lama, status baru, waktu, dan pelaku. Operasi pengubahan status pada repository berhenti membuang parameter pelaku dan menulis baris riwayat dalam transaksi yang sama.
- Pekerjaan penjadwal yang mengubah status secara massal menulis riwayat dengan atribusi sistem.
- Kolom hapus lunak dilengkapi pada tabel transaksional yang belum memilikinya, sehingga klaim §13.1 berlaku menyeluruh.
- Repository invoice mulai menyaring baris terhapus lunak, menyamai perilaku repository lain.

### Notifikasi

- Jalur penyelesaian pembayaran yang dipakai bersama oleh konfirmasi manual dan gateway mengirim notifikasi permintaan dokumen via WhatsApp, memenuhi FR-AUTO-04 dan diagram §14.5.3. Endpoint konfirmasi lama tidak lagi menjadi satu-satunya jalur yang mengirimnya.
- Pengingat pembayaran memperoleh jenis pesan tersendiri untuk hari ke-6, dan menerima nomor invoice serta pengenal invoice yang benar sebagai referensi.
- Pencarian penerima email admin mencakup peran super admin. Logika ini disatukan menjadi satu utilitas bersama, menggantikan tiga pemanggilan yang berbeda-beda cakupannya.
- Dedup notifikasi jatuh tempo dicatat terlepas dari ada tidaknya nomor WhatsApp penerima, mengikuti pola penanda yang sudah dipakai peringatan kuota.
- Jumlah template direkonsiliasi terhadap tabel §17.1–17.3. Pelaksana perlu membaca tabel tersebut untuk menentukan template mana yang belum ada — analisis kami baru sampai pada selisih agregat pada baris WhatsApp→Peserta, belum pada identitas template yang kurang.

### Domain model

- Entity domain memperoleh method perilaku sebagaimana class diagram §14.4. Yang murni predikat dan turunan diletakkan pada entity; yang memerlukan I/O tetap di lapisan aplikasi dan **tidak** dipindahkan ke domain.
- Method yang bersifat orkestrasi pada diagram diinterpretasikan sebagai perilaku entity yang setara dan dapat diuji tanpa basis data. Pemetaan konkretnya dicatat pada berkas pelaksanaan agar diagram dan kode dapat dicocokkan baris per baris.

### Fitur portal & laporan yang belum lengkap

- Kartu riwayat perjalanan menampilkan penanda status penyelesaian. Status "dibatalkan" belum punya padanan di skema; keputusan apakah menambah kolom status peserta atau menurunkan penanda dari data yang ada diserahkan ke pelaksanaan, dan harus dicatat sebagai keputusan.
- Unduhan artefak perjalanan lama diperluas ke itinerary.
- Pra-pengisian formulir konsultasi diperluas ke email dan tipe kamar dari perjalanan terakhir. Data ini diambil dari endpoint portal, bukan dari penyimpanan peramban, agar tidak bergantung pada nilai yang mungkin usang.
- Laporan pasca-handling mengambil nama paket, tanggal keberangkatan, dan tour leader yang ditugaskan dari batch, bukan dari hasil penelusuran checklist.
- Daftar dokumen memperoleh paginasi pada filter, repository, dan antarmuka.
- Inisialisasi checklist bandara dipindahkan keluar dari endpoint baca.

### Kualitas

- Pembuatan PDF menggunakan penerjemah karakter yang sesuai dengan font inti, menyamai pola yang sudah benar pada modul laporan.
- Ekstraksi berlabel pada OCR memakai baris yang sama untuk pencocokan dan pemotongan.
- Nama berkas unggahan disanitasi termasuk bagian ekstensinya, dan komponen path dikodekan saat disusun menjadi URL.
- Pengunduhan berkas di frontend memasang elemen pemicu ke dokumen dan menunda pelepasan URL objek.
- Utilitas format mata uang, pemotongan teks aman-Unicode, dan pembacaan variabel lingkungan dengan nilai bawaan disatukan masing-masing menjadi satu implementasi bersama.
- Parsing respons JSON penyimpanan memakai pustaka JSON standar.
- Aturan pengabaian berkas biner pada Git diperbaiki sehingga benar-benar cocok.
- Seluruh berkas yang disentuh diformat sesuai formatter bawaan bahasa.

## Testing Decisions

### Apa yang membuat test bagus di sini

Test menguji **perilaku yang dijanjikan PRD kepada pengguna**, bukan detail implementasi. Satu test yang baik dapat dibaca sebagai kalimat dari user story: peran ini mengirim permintaan itu, dan menerima hasil ini. Test tidak menyentuh nama fungsi internal, urutan pemanggilan, atau struktur data privat — sehingga refactor tidak memecahkannya, tapi perubahan perilaku memecahkannya.

### Seam utama: lapisan HTTP

Seam ini **sudah ada di kode produksi** dan tidak menuntut refactor: fungsi pendaftaran rute menerima satu struct berisi seluruh dependensi, dan seluruh repository di dalamnya sudah berupa interface. Adapter eksternal (WhatsApp, email, penyimpanan, gateway, chatbot, OCR) sudah memiliki mode nir-operasi saat konfigurasinya kosong, sehingga aman dipakai apa adanya dalam test.

Test membangun instance server dengan repository palsu dan **service aplikasi yang asli**, lalu menembakkan permintaan HTTP dan memeriksa respons. Konsekuensinya satu test menembus routing, middleware, handler, service aplikasi, dan domain sekaligus — inilah seam tertinggi yang tersedia, dan alasan utama seam ini dipilih dibanding memperbanyak test di lapisan service.

Seam ini menutup hampir seluruh keputusan implementasi di atas: RBAC per grup, pemisahan token, validasi masukan, pembatas body, pemulihan panic, idempotensi webhook, transaksi konversi, jejak audit, notifikasi, dan paginasi.

**Dua celah yang diketahui:** endpoint dashboard/analytics dan laporan memakai koneksi basis data secara langsung sehingga tidak dapat dipalsukan; dan pemanggilan keluar ke gateway pembayaran memakai alamat dasar yang tidak dapat disetel dari test. Untuk yang kedua, alamat dasar dibuat dapat disetel saat konstruksi agar sisi pembuatan transaksi bisa diuji terhadap server uji lokal — perubahan kecil yang sepadan karena menutup satu kelompok user story keuangan.

### Seam kedua: kontrak repository terhadap Postgres

Diperlukan karena penyelarasan ini menambah pekerjaan level skema — tabel riwayat status, kolom hapus lunak, dan penyaringan baris terhapus. Jaminan itu tidak dapat diverifikasi lewat repository palsu.

Seam ini dijaga **sekecil mungkin**: hanya test kontrak repository, dijalankan terhadap layanan basis data yang sudah ada di komposisi kontainer proyek, dan diberi penanda agar dapat dilewati saat basis data tidak tersedia. Tidak ada logika bisnis yang diuji di sini.

### Seam ketiga: fungsi murni

Seam yang sudah ada dipertahankan apa adanya untuk fungsi murni — pemetaan destinasi ke kode negara, normalisasi nomor telepon, parsing teks OCR, format mata uang, dan predikat domain baru dari class diagram §14.4. Murah, cepat, dan sudah menjadi pola di repo.

### Prior art

Test service aplikasi yang ada sudah memakai repository palsu yang mengimplementasikan interface domain — pola fake inilah yang dipakai ulang dan diperluas untuk seam HTTP, bukan pustaka mock baru. Test fungsi murni yang ada sudah memakai tabel kasus map masukan-ke-harapan; gaya itu dipertahankan.

### Target

Coverage backend mencapai ambang §21.10. Angka dikejar lewat seam HTTP terlebih dahulu, karena satu test di sana menaikkan coverage di banyak paket sekaligus dan sekaligus melindungi perilaku yang benar-benar dijanjikan dokumen. Menaikkan angka dengan menumpuk test fungsi murni secara eksplisit **tidak** diterima sebagai cara memenuhi target ini.

## Out of Scope

- **Penyuntingan PRD.** Arah penyelarasan sudah diputuskan: kode yang menyesuaikan. Bila ada penyimpangan yang ternyata lebih masuk akal diselesaikan dengan mengubah dokumen, itu diangkat sebagai keputusan terpisah, bukan dikerjakan diam-diam di sini.
- **Bagian PRD yang belum diverifikasi.** Bagian 1 (Konsep & Bisnis), Bagian 5 (Desain UI), §14.3 Use Case, §14.6 Activity Diagram, serta empat dari enam sequence diagram belum ditelusuri terhadap kode. Penyimpangan di sana, bila ada, belum tercakup spec ini.
- **Test frontend.** Repo belum punya infrastruktur test frontend sama sekali. Membangunnya adalah pekerjaan tersendiri; cacat antarmuka dalam spec ini diverifikasi manual.
- **Test performa.** Exit criteria §21.10 juga menyebut waktu respons di bawah 500ms untuk 95% permintaan. Itu memerlukan perkakas pengukuran beban tersendiri.
- **UAT dan skor SUS.** Melibatkan responden manusia, di luar jangkauan perubahan kode.
- **Migrasi data historis.** Tabel riwayat status lead dimulai kosong; perubahan status yang sudah terjadi sebelum tabel ada tidak direkonstruksi.
- **Perombakan arsitektur.** Ketergantungan service peserta pada service invoice, dan handler yang memakai koneksi basis data langsung, dibiarkan apa adanya kecuali menghalangi seam pengujian.

## Further Notes

- **Urutan pengerjaan yang disarankan.** Seam HTTP dibangun lebih dulu, sebelum perbaikan perilaku apa pun. Setiap cacat kemudian diperbaiki dengan test yang gagal dahulu di seam itu. Ini membuat setiap perbaikan terbukti, dan membuat angka coverage naik sebagai efek samping yang jujur alih-alih sebagai target yang dikejar terpisah.
- **Tiga temuan bersifat merusak diam-diam** dan sebaiknya didahulukan setelah seam siap: penghitungan ganda pembayaran gateway, akun portal yatim yang mengunci peserta permanen, dan penandatanganan URL file privat lintas peserta. Ketiganya tidak menimbulkan galat yang terlihat — sistem tampak bekerja normal sambil salah.
- **Matriks RBAC §5.3 sudah cocok persis dengan kode.** Keempat belas barisnya diverifikasi terhadap grup rute: analytics dan profil sendiri terbuka untuk empat peran staf; CRM dan peserta mencakup konsultan dengan pembatasan kepemilikan; NIK, dokumen, invoice, paket, laporan, dan chatbot terbatas ke super admin dan admin; airport handling mencakup tour leader; manajemen pengguna hanya super admin. Tidak ada baris yang meleset. Pekerjaan kontrol akses dalam spec ini karena itu **tidak** menyentuh pembagian grup yang sudah ada — hanya menambal celah di luarnya.
- **Satu klaim review sebelumnya perlu diralat.** Temuan bahwa grup admin dasar tanpa pembatasan peran adalah cacat ternyata hanya benar sebagian. FR-RPT-02 menyatakan analytics terbuka untuk semua peran staf, dan baris pertama matriks §5.3 mengonfirmasinya. Yang benar-benar salah tempat hanya endpoint penandatanganan URL.
- **Endpoint penandatanganan URL tidak diatur PRD sama sekali.** Endpoint ini tidak muncul di matriks §5.3 maupun di FR manapun, sehingga cacat kepemilikannya bukan pelanggaran dokumen melainkan endpoint yang hak aksesnya memang belum pernah dirancang. Ini satu-satunya item dalam spec ini yang tidak punya rujukan PRD untuk disandari — perilaku targetnya **diputuskan**, bukan dibaca. Setelah diputuskan, hasilnya perlu diangkat sebagai usulan baris tambahan pada §5.3 agar dokumen dan kode kembali sejajar.
- **Tiga hal butuh keputusan saat pelaksanaan** dan harus dicatat, bukan diputuskan diam-diam: padanan status "dibatalkan" pada kartu riwayat perjalanan yang belum ada di skema; identitas template notifikasi yang kurang pada rekapitulasi §17.3; dan tingkat akses untuk endpoint penandatanganan URL sebagaimana catatan di atas.
- **PRD tidak masuk kontrol versi.** Berkas PRD diabaikan Git, sehingga tidak ikut tertinjau saat perubahan kode direview. Bila dokumen ini akan terus dipakai sebagai sumber kebenaran, keputusan itu layak ditinjau ulang.
