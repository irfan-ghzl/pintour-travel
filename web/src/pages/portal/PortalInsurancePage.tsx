import { Shield, ExternalLink, Phone, FileText, CheckCircle } from 'lucide-react'

const coverageItems = [
  'Kecelakaan selama perjalanan (darat, udara, laut)',
  'Biaya pengobatan darurat di luar negeri',
  'Evakuasi medis darurat',
  'Penundaan dan pembatalan penerbangan',
  'Kehilangan atau kerusakan bagasi',
  'Repatriasi jenazah',
  'Tanggung jawab hukum pihak ketiga',
]

export default function PortalInsurancePage() {
  return (
    <div className="space-y-4 pb-6">
      {/* Header */}
      <div className="bg-emerald-700 text-white rounded-xl p-5">
        <div className="flex items-center gap-3 mb-2">
          <Shield size={28} />
          <div>
            <h2 className="font-bold text-lg">Asuransi Perjalanan</h2>
            <p className="text-emerald-200 text-sm">Zurich Travel Insurance</p>
          </div>
        </div>
        <p className="text-emerald-100 text-sm">
          Pintour merekomendasikan Zurich Travel Insurance untuk perlindungan optimal selama perjalanan Anda.
        </p>
      </div>

      {/* Coverage */}
      <div className="bg-white rounded-xl border p-5">
        <h3 className="font-semibold text-gray-800 mb-3">🛡️ Manfaat Perlindungan</h3>
        <ul className="space-y-2">
          {coverageItems.map((item, i) => (
            <li key={i} className="flex items-start gap-2 text-sm text-gray-600">
              <CheckCircle size={16} className="text-emerald-500 mt-0.5 shrink-0" />
              {item}
            </li>
          ))}
        </ul>
      </div>

      {/* Plans */}
      <div className="bg-white rounded-xl border p-5">
        <h3 className="font-semibold text-gray-800 mb-3">📋 Paket yang Tersedia</h3>
        <div className="space-y-3">
          {[
            { name: 'Basic', price: 'Rp 150.000 – Rp 300.000', desc: 'Perlindungan standar untuk perjalanan jangka pendek' },
            { name: 'Classic', price: 'Rp 300.000 – Rp 600.000', desc: 'Perlindungan lebih lengkap termasuk pembatalan penerbangan' },
            { name: 'Premium', price: 'Rp 600.000 – Rp 1.200.000', desc: 'Perlindungan komprehensif dengan limit klaim tertinggi' },
          ].map((plan) => (
            <div key={plan.name} className="flex items-start justify-between gap-3 p-3 rounded-lg bg-gray-50 border">
              <div>
                <p className="font-semibold text-sm text-gray-800">{plan.name}</p>
                <p className="text-xs text-gray-500 mt-0.5">{plan.desc}</p>
              </div>
              <div className="text-right shrink-0">
                <p className="text-xs text-gray-400">mulai dari</p>
                <p className="text-sm font-bold text-emerald-700">{plan.price}</p>
              </div>
            </div>
          ))}
        </div>
        <p className="text-xs text-gray-400 mt-3">
          * Harga bervariasi berdasarkan destinasi, durasi, dan usia peserta.
        </p>
      </div>

      {/* How to purchase */}
      <div className="bg-white rounded-xl border p-5">
        <h3 className="font-semibold text-gray-800 mb-3">📝 Cara Pembelian</h3>
        <div className="space-y-3 text-sm text-gray-600">
          <div className="flex gap-3">
            <span className="w-6 h-6 rounded-full bg-emerald-100 text-emerald-700 text-xs flex items-center justify-center font-bold shrink-0">1</span>
            <p>Hubungi admin Pintour atau kunjungi website Zurich untuk simulasi premi</p>
          </div>
          <div className="flex gap-3">
            <span className="w-6 h-6 rounded-full bg-emerald-100 text-emerald-700 text-xs flex items-center justify-center font-bold shrink-0">2</span>
            <p>Isi formulir data peserta sesuai paspor</p>
          </div>
          <div className="flex gap-3">
            <span className="w-6 h-6 rounded-full bg-emerald-100 text-emerald-700 text-xs flex items-center justify-center font-bold shrink-0">3</span>
            <p>Lakukan pembayaran premi (transfer bank atau kartu kredit)</p>
          </div>
          <div className="flex gap-3">
            <span className="w-6 h-6 rounded-full bg-emerald-100 text-emerald-700 text-xs flex items-center justify-center font-bold shrink-0">4</span>
            <p>Polis asuransi akan dikirim ke email Anda dalam 1–2 hari kerja</p>
          </div>
        </div>
      </div>

      {/* Contact */}
      <div className="bg-blue-50 border border-blue-200 rounded-xl p-4 space-y-3">
        <h3 className="font-semibold text-blue-800 text-sm">📞 Hubungi Kami</h3>
        <div className="space-y-2">
          <a
            href="https://www.zurich.co.id"
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-2 text-sm text-blue-600 hover:underline"
          >
            <ExternalLink size={14} /> Website Zurich Indonesia
          </a>
          <div className="flex items-center gap-2 text-sm text-blue-600">
            <Phone size={14} /> Hotline Zurich: 1500-394
          </div>
          <div className="flex items-center gap-2 text-sm text-blue-600">
            <FileText size={14} /> Untuk bantuan, hubungi admin Pintour via WhatsApp
          </div>
        </div>
      </div>

      <p className="text-xs text-gray-400 text-center px-4">
        Asuransi perjalanan sangat direkomendasikan untuk semua peserta, terutama perjalanan ke luar negeri.
        Pintour tidak memproses klaim asuransi secara langsung.
      </p>
    </div>
  )
}
