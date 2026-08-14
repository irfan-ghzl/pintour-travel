# Dokumen dummy untuk uji upload peserta

PDF satu halaman untuk mengisi formulir upload di halaman **Portal › Dokumen**
dan antrean **Review Dokumen** di sisi admin.

Nama berkasnya sengaja sama dengan nilai `document_type` yang dipakai aplikasi
(lihat `document.Types` di `internal/domain/document/entity.go`), jadi tinggal
dicocokkan dengan slot uploadnya:

| Berkas | Jenis dokumen di formulir |
| --- | --- |
| `passport.pdf` | Paspor |
| `ktp.pdf` | KTP |
| `rekening_koran.pdf` | Rekening Koran (3 bulan) |
| `visa_support.pdf` | Dokumen Pendukung Visa |
| `lainnya.pdf` | Dokumen Lainnya |

Isinya generik — tidak menyebut nama, nomor, atau tanggal peserta tertentu —
supaya satu set ini bisa dipakai berulang untuk peserta uji mana pun tanpa
perlu dibuat ulang. Setiap halaman menyatakan dirinya dokumen uji coba, agar
salinan yang terlanjur tersimpan di basis data atau bucket tidak pernah
tertukar dengan berkas identitas sungguhan.

Ukurannya jauh di bawah batas 5MB dan formatnya PDF, jadi keduanya lolos
validasi dan yang teruji adalah alur unggahnya, bukan penolakan berkasnya.
