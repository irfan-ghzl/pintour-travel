import { useState } from 'react'
import { Users, ClipboardList, FileText, Plane, FileSpreadsheet, FileDown } from 'lucide-react'
import toast from 'react-hot-toast'
import { exportReport, type ReportType, type ReportFormat } from '../../utils/api'

const REPORTS: { type: ReportType; title: string; desc: string; icon: typeof Users }[] = [
  { type: 'leads', title: 'Leads', desc: 'Semua leads CRM dengan status, paket, dan konsultan.', icon: Users },
  { type: 'participants', title: 'Peserta', desc: 'Daftar peserta dengan status bayar & dokumen.', icon: ClipboardList },
  { type: 'invoices', title: 'Invoice', desc: 'Invoice dengan nominal, terbayar, sisa, dan status.', icon: FileText },
  { type: 'batches', title: 'Batch Keberangkatan', desc: 'Batch dengan kuota, terjual, dan tour leader.', icon: Plane },
]

export default function ReportPage() {
  const [loading, setLoading] = useState<string | null>(null)

  async function handleExport(type: ReportType, format: ReportFormat) {
    const key = `${type}-${format}`
    setLoading(key)
    try {
      await exportReport(type, format)
      toast.success(`Laporan ${type} (${format}) berhasil diunduh`)
    } catch {
      toast.error('Gagal mengekspor laporan')
    } finally {
      setLoading(null)
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-gray-800">Laporan &amp; Export</h1>
        <p className="text-sm text-gray-500 mt-0.5">Unduh laporan operasional dalam format Excel atau PDF.</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {REPORTS.map(({ type, title, desc, icon: Icon }) => (
          <div key={type} className="bg-white rounded-xl border p-5 flex flex-col">
            <div className="flex items-center gap-3 mb-2">
              <div className="w-10 h-10 rounded-lg bg-emerald-50 text-emerald-600 flex items-center justify-center">
                <Icon size={20} />
              </div>
              <h2 className="font-semibold text-gray-800">{title}</h2>
            </div>
            <p className="text-sm text-gray-500 flex-1 mb-4">{desc}</p>
            <div className="flex gap-2">
              <button
                onClick={() => handleExport(type, 'excel')}
                disabled={loading !== null}
                className="flex-1 inline-flex items-center justify-center gap-1.5 px-3 py-2 rounded-lg
                           bg-emerald-600 text-white text-sm font-medium hover:bg-emerald-700
                           disabled:opacity-50 transition-colors"
              >
                <FileSpreadsheet size={15} />
                {loading === `${type}-excel` ? 'Memproses…' : 'Export Excel'}
              </button>
              <button
                onClick={() => handleExport(type, 'pdf')}
                disabled={loading !== null}
                className="flex-1 inline-flex items-center justify-center gap-1.5 px-3 py-2 rounded-lg
                           border border-gray-300 text-gray-700 text-sm font-medium hover:bg-gray-50
                           disabled:opacity-50 transition-colors"
              >
                <FileDown size={15} />
                {loading === `${type}-pdf` ? 'Memproses…' : 'Export PDF'}
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
