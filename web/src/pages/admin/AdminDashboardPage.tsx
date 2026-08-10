import { useQuery } from '@tanstack/react-query'
import {
  Package, Users, FileText, ClipboardList, TrendingUp, FolderCheck,
  Wallet, CalendarClock, Plane,
} from 'lucide-react'
import { Link } from 'react-router-dom'
import api, { getDashboardAnalytics } from '../../utils/api'
import { authStorage } from '../../utils/auth'
import ProgressBar from '../../components/ProgressBar'

function rupiah(n: number) {
  return 'Rp ' + (n ?? 0).toLocaleString('id-ID')
}

export default function AdminDashboardPage() {
  const user = authStorage.getUser()

  const { data: analytics } = useQuery({
    queryKey: ['dashboard-analytics'],
    queryFn: getDashboardAnalytics,
  })

  const { data: docData } = useQuery({
    queryKey: ['dashboard-docs'],
    // per_page=1 because only meta.total is read: the tile used to pull every
    // pending document over the wire so it could take .length of the array.
    queryFn: () => api.get('/admin/documents?status=menunggu&per_page=1').then((r) => r.data),
  })

  const ls = analytics?.leads_summary
  const rs = analytics?.revenue_summary
  const bs = analytics?.batch_summary
  const maxMonthly = Math.max(1, ...(analytics?.monthly_leads ?? []).map((m) => m.count))

  const stats = [
    {
      label: 'Total Leads', value: ls?.total ?? '—',
      icon: Users, color: 'text-blue-600 bg-blue-50',
      link: '/admin/leads', hint: `${ls?.baru ?? 0} baru`,
    },
    {
      label: 'Revenue Diterima', value: rs ? rupiah(rs.total_paid) : '—',
      icon: Wallet, color: 'text-emerald-600 bg-emerald-50',
      link: '/admin/invoices', hint: rs ? `${rupiah(rs.total_pending)} pending` : '',
    },
    {
      label: 'Peserta Aktif', value: bs?.total_participants ?? '—',
      icon: ClipboardList, color: 'text-purple-600 bg-purple-50',
      link: '/admin/participants', hint: 'Portal aktif',
    },
    {
      label: 'Batch Aktif', value: bs?.total_active_batches ?? '—',
      icon: Plane, color: 'text-orange-600 bg-orange-50',
      link: '/admin/packages', hint: 'Status tersedia',
    },
  ]

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-gray-800">Selamat datang, {user?.name ?? 'Admin'} 👋</h1>
        <p className="text-sm text-gray-500 mt-0.5 capitalize">{user?.role?.replace('_', ' ') ?? ''}</p>
      </div>

      {/* Row 1 — StatCards */}
      <div className="grid grid-cols-2 xl:grid-cols-4 gap-4">
        {stats.map(({ label, value, icon: Icon, color, link, hint }) => (
          <Link key={label} to={link}
            className="bg-white rounded-xl border p-5 hover:shadow-sm transition-shadow block">
            <div className={`w-10 h-10 rounded-lg ${color} flex items-center justify-center mb-3`}>
              <Icon size={20} />
            </div>
            <p className="text-2xl font-bold text-gray-800">{value}</p>
            <p className="text-sm font-medium text-gray-700">{label}</p>
            <p className="text-xs text-gray-400">{hint}</p>
          </Link>
        ))}
      </div>

      {/* Row 2 — Leads funnel + nearest departure */}
      <div className="grid grid-cols-1 lg:grid-cols-5 gap-4">
        <div className="lg:col-span-3 bg-white rounded-xl border p-5">
          <h2 className="font-semibold text-gray-700 mb-4">Funnel Konversi Leads</h2>
          {ls ? (
            <div className="space-y-3">
              <ProgressBar label="Total Leads" value={ls.total} max={ls.total || 1} color="bg-blue-500" />
              <ProgressBar label="Dalam Proses" value={ls.proses} max={ls.total || 1} color="bg-yellow-500" />
              <ProgressBar label="Deal" value={ls.deal} max={ls.total || 1} color="bg-emerald-500" />
              <div className="pt-2 border-t flex items-center justify-between text-sm">
                <span className="text-gray-600">Conversion Rate</span>
                <span className="font-bold text-emerald-600">{ls.conversion_rate.toFixed(1)}%</span>
              </div>
            </div>
          ) : (
            <p className="text-sm text-gray-400">Memuat…</p>
          )}
        </div>

        <div className="lg:col-span-2 bg-white rounded-xl border p-5">
          <div className="flex items-center gap-2 mb-3 text-gray-700">
            <CalendarClock size={18} className="text-orange-500" />
            <h2 className="font-semibold">Keberangkatan Terdekat</h2>
          </div>
          {bs?.nearest_departure ? (
            <div>
              <p className="font-medium text-gray-800">{bs.nearest_departure.package_name}</p>
              <p className="text-sm text-gray-500">{bs.nearest_departure.departure_date}</p>
              <p className="text-3xl font-bold text-orange-600 mt-3">
                {bs.nearest_departure.days_remaining}
                <span className="text-sm font-normal text-gray-500"> hari lagi</span>
              </p>
              <p className="text-xs text-gray-400 mt-1">
                {bs.nearest_departure.participant_count} peserta terdaftar
              </p>
            </div>
          ) : (
            <p className="text-sm text-gray-400">Belum ada batch mendatang.</p>
          )}
        </div>
      </div>

      {/* Row 3 — Monthly leads chart */}
      <div className="bg-white rounded-xl border p-5">
        <h2 className="font-semibold text-gray-700 mb-4">Leads 6 Bulan Terakhir</h2>
        {analytics?.monthly_leads?.length ? (
          <div className="flex items-end gap-3 h-40">
            {analytics.monthly_leads.map((m) => (
              <div key={m.month} className="flex-1 flex flex-col items-center justify-end gap-1">
                <span className="text-xs font-medium text-gray-600">{m.count}</span>
                <div
                  className="w-full bg-emerald-500 rounded-t transition-all"
                  style={{ height: `${(m.count / maxMonthly) * 100}%`, minHeight: m.count > 0 ? '4px' : '0' }}
                />
                <span className="text-[11px] text-gray-400">{m.month}</span>
              </div>
            ))}
          </div>
        ) : (
          <p className="text-sm text-gray-400">Belum ada data.</p>
        )}
      </div>

      {/* Quick links */}
      <div className="bg-white rounded-xl border p-5">
        <h2 className="font-semibold text-gray-700 mb-4">Aksi Cepat</h2>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
          {[
            { to: '/admin/leads', label: 'Lihat Leads Baru', icon: Users },
            { to: '/admin/invoices', label: 'Buat Invoice', icon: FileText },
            { to: '/admin/documents', label: `Review Dokumen${docData?.meta?.total ? ` (${docData.meta.total})` : ''}`, icon: FolderCheck },
            { to: '/admin/participants', label: 'Kelola Peserta', icon: ClipboardList },
            { to: '/admin/packages', label: 'Kelola Paket', icon: Package },
            { to: '/admin/reports', label: 'Laporan & Export', icon: TrendingUp },
          ].map(({ to, label, icon: Icon }) => (
            <Link key={to} to={to}
              className="flex items-center gap-2 px-3 py-2.5 rounded-lg border text-sm text-gray-700 hover:bg-gray-50 transition-colors">
              <Icon size={16} className="text-emerald-600" />
              {label}
            </Link>
          ))}
        </div>
      </div>
    </div>
  )
}
