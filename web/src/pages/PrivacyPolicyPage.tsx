import { Link } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'

const sections = [
  {
    title: '1. Data yang Kami Kumpulkan',
    content: `Pintour mengumpulkan data berikut untuk keperluan pemesanan dan pengurusan perjalanan:
• Identitas: nama lengkap, nomor telepon, alamat email
• Dokumen perjalanan: paspor, KTP, rekening koran
• Data transaksi: invoice, bukti pembayaran
• Data perjalanan: paket yang dipesan, tanggal keberangkatan, itinerary`,
  },
  {
    title: '2. Tujuan Penggunaan Data',
    content: `Data Anda digunakan hanya untuk:
• Memproses pemesanan paket wisata
• Pengurusan dokumen visa dan keimigrasian
• Komunikasi terkait perjalanan (konfirmasi, pengingat, informasi)
• Pemenuhan kewajiban hukum (perpajakan, keimigrasian)`,
  },
  {
    title: '3. Pihak yang Memiliki Akses',
    content: `Data Anda hanya diakses oleh:
• Tim internal Pintour (admin, konsultan, tour leader) dengan akses terbatas sesuai peran
• Pihak ketiga yang diperlukan untuk penyelenggaraan perjalanan (maskapai, kedutaan, hotel) — hanya data yang relevan
• Kami tidak menjual data Anda kepada pihak ketiga manapun`,
  },
  {
    title: '4. Durasi Penyimpanan Data',
    content: `• Data leads (tidak jadi peserta): disimpan 1 tahun kemudian dihapus
• Data peserta (selesai perjalanan): disimpan 5 tahun untuk keperluan arsip
• Dokumen sensitif (paspor, KTP): dihapus 1 tahun setelah perjalanan selesai
• Rekening koran: dihapus 6 bulan setelah perjalanan selesai
• Log sistem: disimpan 3 tahun`,
  },
  {
    title: '5. Hak Anda atas Data Pribadi',
    content: `Sesuai UU PDP No. 27/2022, Anda berhak:
• Mengakses data pribadi yang tersimpan (login ke portal → Profil → Unduh Data Saya)
• Memperbarui data yang tidak akurat (melalui portal peserta)
• Meminta penghapusan data (portal peserta → Profil → Minta Penghapusan Data)
• Menolak pengiriman notifikasi marketing
• Mendapatkan salinan data dalam format yang dapat dibaca mesin`,
  },
  {
    title: '6. Keamanan Data',
    content: `Kami melindungi data Anda dengan:
• Enkripsi HTTPS untuk semua komunikasi
• Autentikasi JWT dengan masa berlaku terbatas
• Dokumen sensitif disimpan di storage private (tidak dapat diakses tanpa autentikasi)
• Akses data dibatasi berdasarkan peran pengguna (role-based access control)
• Audit log untuk semua operasi sensitif`,
  },
  {
    title: '7. Kontak Penanggung Jawab Data',
    content: `Untuk pertanyaan terkait privasi atau penggunaan data, hubungi:
• Email: privacy@pintour.app
• WhatsApp: +62 811-xxxx-xxxx
• Alamat: [Alamat kantor Pintour]

Kami akan merespons permintaan terkait privasi dalam 14 hari kerja.`,
  },
]

export default function PrivacyPolicyPage() {
  return (
    <div className="min-h-screen bg-gray-50">
      <div className="max-w-3xl mx-auto px-4 py-10">
        {/* Header */}
        <div className="mb-8">
          <Link to="/" className="inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-emerald-600 mb-4">
            <ArrowLeft size={14} /> Kembali ke Beranda
          </Link>
          <h1 className="text-2xl font-bold text-gray-800">Kebijakan Privasi</h1>
          <p className="text-sm text-gray-500 mt-1">
            Terakhir diperbarui: Mei 2026 · Berlaku untuk seluruh layanan Pintour Travel
          </p>
        </div>

        {/* Intro */}
        <div className="bg-emerald-50 border border-emerald-200 rounded-xl p-4 mb-6 text-sm text-emerald-700">
          Pintour Travel berkomitmen melindungi privasi dan keamanan data pribadi Anda sesuai dengan
          <strong> Undang-Undang Pelindungan Data Pribadi (UU PDP No. 27/2022)</strong>.
          Dengan menggunakan layanan kami, Anda menyetujui kebijakan privasi ini.
        </div>

        {/* Sections */}
        <div className="space-y-5">
          {sections.map((sec) => (
            <div key={sec.title} className="bg-white rounded-xl border p-5">
              <h2 className="font-semibold text-gray-800 mb-3">{sec.title}</h2>
              <p className="text-sm text-gray-600 whitespace-pre-line leading-relaxed">
                {sec.content}
              </p>
            </div>
          ))}
        </div>

        {/* Informed consent note */}
        <div className="bg-blue-50 border border-blue-200 rounded-xl p-4 mt-5 text-sm text-blue-700">
          <p className="font-medium mb-1">📋 Persetujuan Data</p>
          <p>
            Dengan mengisi form konsultasi di website kami, Anda memberikan persetujuan eksplisit
            untuk pengumpulan dan pemrosesan data sesuai kebijakan ini.
            Anda dapat mencabut persetujuan kapan saja melalui portal peserta.
          </p>
        </div>

        <p className="text-center text-xs text-gray-400 mt-8">
          © {new Date().getFullYear()} Pintour Travel. Kebijakan ini dapat diperbarui sewaktu-waktu.
          Perubahan signifikan akan diberitahukan melalui WhatsApp atau email terdaftar.
        </p>
      </div>
    </div>
  )
}
