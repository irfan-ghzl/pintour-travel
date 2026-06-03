import { useQuery } from '@tanstack/react-query'
import { Clock, FileText, BookOpen, CheckCircle, AlertCircle, Receipt, Shield, ChevronRight } from 'lucide-react'
import { Link } from 'react-router-dom'
import { portalApi } from '../../utils/api'
import type { PortalMeResponse, Invoice, Document } from '../../types'

interface Task {
  id: string
  label: string
  done: boolean
  link?: string
  urgent?: boolean
}

export default function PortalDashboardPage() {
  const { data, isLoading } = useQuery({
    queryKey: ['portal-me'],
    queryFn: () =>
      portalApi.get<{ success: boolean; data: PortalMeResponse }>('/portal/me')
        .then(r => r.data.data),
  })

  const { data: invoices } = useQuery({
    queryKey: ['portal-invoices-dash'],
    queryFn: () =>
      portalApi.get<{ success: boolean; data: Invoice[] }>('/portal/invoices')
        .then(r => r.data.data ?? []),
  })

  const { data: docs } = useQuery({
    queryKey: ['portal-documents-dash'],
    queryFn: () =>
      portalApi.get<{ success: boolean; data: Document[] }>('/portal/documents')
        .then(r => r.data.data ?? []),
  })

  if (isLoading) return (
    <div className="space-y-4 animate-pulse">
      <div className="h-28 bg-gray-200 rounded-xl" />
      <div className="h-20 bg-gray-200 rounded-xl" />
      <div className="h-20 bg-gray-200 rounded-xl" />
    </div>
  )

  const participant = data?.participant
  const countdown = data?.days_to_depart
  const briefingActive = data?.briefing_active

  // FR-PORTAL-02: build task checklist
  const invList = invoices ?? []
  const docList = docs ?? []

  const hasUnpaidInvoice = invList.some(inv => inv.status === 'diterbitkan' || inv.status === 'menunggu_bayar')
  const hasPaidInvoice = invList.some(inv => inv.status === 'dibayar' || inv.status === 'lunas')
  const docTypes = ['passport', 'ktp', 'rekening_koran']
  const docStatus = docTypes.map(t => {
    const latest = docList.filter(d => d.document_type === t).at(-1)
    return { type: t, doc: latest }
  })
  const allDocsApproved = docStatus.every(d => d.doc?.status === 'disetujui')

  const tasks: Task[] = [
    {
      id: 'pay',
      label: hasPaidInvoice ? 'Pembayaran lunas' : 'Bayar invoice & upload bukti transfer',
      done: hasPaidInvoice,
      urgent: hasUnpaidInvoice,
      link: '/portal/invoices',
    },
    ...docStatus.map(d => ({
      id: `doc-${d.type}`,
      label: `Upload ${
        d.type === 'passport' ? 'Paspor' :
        d.type === 'ktp' ? 'KTP' : 'Rekening Koran'
      }${d.doc?.status === 'ditolak' ? ' (perlu upload ulang)' : ''}`,
      done: d.doc?.status === 'disetujui',
      urgent: d.doc?.status === 'ditolak',
      link: '/portal/documents',
    })),
    {
      id: 'briefing',
      label: 'Baca materi briefing digital',
      done: !!participant?.briefing_viewed,
      urgent: briefingActive && !participant?.briefing_viewed,
      link: briefingActive ? '/portal/briefing' : undefined,
    },
  ]

  const doneCount = tasks.filter(t => t.done).length

  return (
    <div className="space-y-4">
      {/* Welcome card */}
      <div className="bg-emerald-700 text-white rounded-xl p-5">
        <p className="text-emerald-200 text-sm mb-1">Selamat datang,</p>
        <h2 className="text-xl font-bold mb-1">{participant?.name}</h2>
        <p className="text-emerald-200 text-sm">{participant?.package_name}</p>
      </div>

      {/* Countdown */}
      {countdown !== undefined && countdown >= 0 ? (
        <div className="bg-white rounded-xl border p-4 flex items-center gap-4">
          <div className="w-14 h-14 rounded-xl bg-emerald-50 flex items-center justify-center shrink-0">
            <Clock className="text-emerald-600" size={28} />
          </div>
          <div>
            <p className="text-3xl font-bold text-gray-800">{countdown}</p>
            <p className="text-sm text-gray-500">hari lagi keberangkatan</p>
            {participant?.batch_departure_date && (
              <p className="text-xs text-gray-400">
                {new Date(participant.batch_departure_date).toLocaleDateString('id-ID', {
                  weekday: 'long', day: 'numeric', month: 'long', year: 'numeric',
                })}
              </p>
            )}
          </div>
        </div>
      ) : null}

      {/* FR-PORTAL-02: Daftar Tugas / Checklist */}
      <div className="bg-white rounded-xl border p-4">
        <div className="flex items-center justify-between mb-3">
          <h3 className="font-semibold text-sm text-gray-800">📋 Daftar Tugas</h3>
          <span className="text-xs text-gray-500">{doneCount}/{tasks.length} selesai</span>
        </div>
        <div className="h-1.5 bg-gray-100 rounded-full overflow-hidden mb-3">
          <div
            className="h-full bg-emerald-500 transition-all"
            style={{ width: tasks.length > 0 ? `${(doneCount / tasks.length) * 100}%` : '0%' }}
          />
        </div>
        <ul className="space-y-1.5">
          {tasks.map(t => {
            const inner = (
              <div className={`flex items-center gap-3 text-sm py-1.5 ${t.link ? 'cursor-pointer hover:bg-gray-50 -mx-2 px-2 rounded' : ''}`}>
                <div className={`w-5 h-5 rounded-full flex items-center justify-center shrink-0 ${
                  t.done ? 'bg-emerald-500 text-white'
                    : t.urgent ? 'bg-red-100 text-red-600' : 'bg-gray-100 text-gray-400'
                }`}>
                  {t.done ? <CheckCircle size={14} /> : t.urgent ? <AlertCircle size={12} /> : '○'}
                </div>
                <span className={`flex-1 ${t.done ? 'text-gray-400 line-through' : t.urgent ? 'text-red-600 font-medium' : 'text-gray-700'}`}>
                  {t.label}
                </span>
                {t.link && !t.done && <ChevronRight size={14} className="text-gray-400" />}
              </div>
            )
            return t.link ? (
              <li key={t.id}><Link to={t.link}>{inner}</Link></li>
            ) : (
              <li key={t.id}>{inner}</li>
            )
          })}
        </ul>
        {doneCount === tasks.length && tasks.length > 0 && (
          <p className="mt-3 text-xs text-emerald-600 font-medium text-center">
            🎉 Semua persiapan selesai! Anda siap untuk perjalanan.
          </p>
        )}
      </div>

      {/* Quick access */}
      <h3 className="font-semibold text-gray-700 text-sm">Akses Cepat</h3>
      <div className="grid grid-cols-2 gap-3">
        <Link
          to="/portal/invoices"
          className="bg-white rounded-xl border p-4 flex flex-col gap-2 hover:border-emerald-300 transition-colors"
        >
          <Receipt className="text-purple-500" size={24} />
          <p className="font-medium text-sm text-gray-800">Invoice</p>
          <p className="text-xs text-gray-500">Lihat tagihan & upload bukti</p>
        </Link>

        <Link
          to="/portal/documents"
          className="bg-white rounded-xl border p-4 flex flex-col gap-2 hover:border-emerald-300 transition-colors"
        >
          <FileText className="text-blue-500" size={24} />
          <p className="font-medium text-sm text-gray-800">Dokumen Saya</p>
          <p className="text-xs text-gray-500">Upload & cek status</p>
        </Link>

        <Link
          to="/portal/itinerary"
          className="bg-white rounded-xl border p-4 flex flex-col gap-2 hover:border-emerald-300 transition-colors"
        >
          <CheckCircle className="text-emerald-500" size={24} />
          <p className="font-medium text-sm text-gray-800">Itinerary</p>
          <p className="text-xs text-gray-500">Jadwal hari per hari</p>
        </Link>

        <Link
          to="/portal/briefing"
          className={`bg-white rounded-xl border p-4 flex flex-col gap-2 transition-colors ${
            briefingActive ? 'border-emerald-300 bg-emerald-50' : 'opacity-60'
          }`}
        >
          <BookOpen className={briefingActive ? 'text-emerald-600' : 'text-gray-400'} size={24} />
          <p className="font-medium text-sm text-gray-800">Briefing Digital</p>
          <p className="text-xs text-gray-500">
            {briefingActive ? '✓ Aktif – H-14' : 'Tersedia H-14'}
          </p>
        </Link>

        <Link
          to="/portal/insurance"
          className="bg-white rounded-xl border p-4 flex flex-col gap-2 hover:border-emerald-300 transition-colors"
        >
          <Shield className="text-orange-400" size={24} />
          <p className="font-medium text-sm text-gray-800">Asuransi</p>
          <p className="text-xs text-gray-500">Info Zurich Travel</p>
        </Link>

        <Link
          to="/portal/profile"
          className="bg-white rounded-xl border p-4 flex flex-col gap-2 hover:border-emerald-300 transition-colors"
        >
          <AlertCircle className="text-gray-500" size={24} />
          <p className="font-medium text-sm text-gray-800">Profil & Privasi</p>
          <p className="text-xs text-gray-500">Hak atas data pribadi</p>
        </Link>
      </div>

      {/* Status summary */}
      <div className="bg-white rounded-xl border p-4">
        <h3 className="font-semibold text-sm text-gray-700 mb-3">Status Persiapan</h3>
        <div className="space-y-2">
          <div className="flex items-center justify-between text-sm">
            <span className="text-gray-600">Status Portal</span>
            <span className={`font-medium ${participant?.is_active ? 'text-green-600' : 'text-orange-500'}`}>
              {participant?.is_active ? '✓ Aktif' : 'Menunggu Aktivasi'}
            </span>
          </div>
          <div className="flex items-center justify-between text-sm">
            <span className="text-gray-600">Dokumen Lengkap</span>
            <span className={`font-medium ${allDocsApproved ? 'text-green-600' : 'text-orange-500'}`}>
              {allDocsApproved ? '✓ Disetujui' : 'Belum Lengkap'}
            </span>
          </div>
          <div className="flex items-center justify-between text-sm">
            <span className="text-gray-600">Briefing Digital</span>
            <span className={`font-medium ${participant?.briefing_viewed ? 'text-green-600' : 'text-gray-400'}`}>
              {participant?.briefing_viewed ? '✓ Sudah Dilihat' : 'Belum Dilihat'}
            </span>
          </div>
        </div>
      </div>
    </div>
  )
}
