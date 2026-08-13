# 04 — Kolom "Status Portal" di halaman peserta menyebut hal yang salah

**What to build:** Admin membuka halaman Peserta dan membaca status pembayaran
tiap peserta, bukan status portal. Kolomnya berjudul **Pembayaran** dan berisi
**Lunas** atau **Belum Lunas**.

**Blocked by:** None — can start immediately.

**Status:** ready-for-agent

## Kenapa labelnya salah

Kolom itu berjudul "Status Portal" dan menampilkan "Aktif" / "Belum Aktif",
tetapi yang dibacanya adalah `is_active`. Sejak tiket 01, `is_active` tidak lagi
menentukan apakah peserta bisa masuk portal — login sudah tidak menanyakan
pembayaran sama sekali. Yang dijaganya sekarang adalah isi perjalanan:
itinerary, briefing, dan kontak tour leader.

Akibatnya label itu memberi tahu admin sesuatu yang tidak benar. Ditemukan
persis begitu: admin melihat "Belum Aktif" pada peserta yang saat itu sudah
masuk portal dan sedang membayar di dalamnya.

Kapan nilainya berubah, supaya penggantinya menyebut hal yang tepat:

| Pembayaran | Status invoice | `is_active` |
| --- | --- | --- |
| Sebagian | `dibayar` | tetap tidak aktif |
| Lunas | `lunas` | aktif |

Jadi yang sebenarnya dilaporkan kolom itu adalah **lunas atau belum**, bukan
portal terbuka atau tertutup.

## Kenapa ditunda

Cacatnya kosmetik dan tidak menghalangi pekerjaan siapa pun: peserta tetap bisa
masuk dan membayar, admin tetap bisa mengonfirmasi. Yang keliru hanya kata yang
dibaca operator. Dikesampingkan atas keputusan pemilik produk agar tidak
mendahului pekerjaan yang lebih berdampak menjelang sidang, dan dicatat di sini
supaya tidak hilang.

Perlu diingat saat mengerjakannya: kolom `is_active` sendiri **tidak** diganti
nama. Mengganti namanya menjadi sesuatu seperti `is_paid` memang lebih jujur,
tapi ia tersebar di repository, penjadwal, dan laporan — refactor lebar
tersendiri, dan bukan bagian tiket ini.

- [ ] Judul kolom pada halaman Peserta menjadi "Pembayaran"
- [ ] Isinya "Lunas" atau "Belum Lunas", bukan "Aktif" / "Belum Aktif"
- [ ] Tidak ada lagi teks di antarmuka admin yang menyiratkan peserta terkunci
      di luar portal sebelum membayar
- [ ] Kolom `is_active` tidak diganti nama, dan tidak ada perubahan backend
