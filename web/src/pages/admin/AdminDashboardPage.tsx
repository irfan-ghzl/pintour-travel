import { useQuery } from '@tanstack/react-query'
import { Package, Users, FileText, ClipboardList, TrendingUp, FolderCheck } from 'lucide-react'
import { Link } from 'react-router-dom'
import api from '../../utils/api'
import { authStorage } from '../../utils/auth'

export default function AdminDashboardPage() {
  const user = authStorage.getUser()

  const { data: leadsData } = useQuery({
    queryKey: ['dashboard-leads'],
    queryFn: () => api.get('/admin/leads?status=baru&per_page=1').then(r => r.data),
  })
  const { data: paxData } = useQuery({
    queryKey: ['dashboard-participants'],
    queryFn: () => api.get('/admin/participants?per_page=1').then(r => r.data),
  })
  const { data: invData } = useQuery({
    queryKey: ['dashboard-invoices'],
    queryFn: () => api.get('/admin/invoices?per_page=1').then(r => r.data),
  })
  const { data: docData } = useQuery({
    queryKey: ['dashboard-docs'],
    queryFn: () => api.get('/admin/documents?status=menunggu').then(r => r.data),
  })

  const stats = [
    {
      label: 'Leads Baru', value: leadsData?.meta?.total ?? '—',
      icon: Users, color: 'text-blue-600 bg-blue-50',
      link: '/admin/leads', hint: 'Belum dihubungi',
    },
    {
      label: 'Total Peserta', value: paxData?.meta?.total ?? '—',
      icon: ClipboardList, color: 'text-emerald-600 bg-emerald-50',
      link: '/admin/participants', hint: 'Peserta aktif',
    },
    {
      label: 'Invoice Pending', value: invData?.meta?.total ?? '—',
      icon: FileText, color: 'text-orange-600 bg-orange-50',
      link: '/admin/invoices', hint: 'Menunggu pembayaran',
    },
    {
      label: 'Dokumen Review', value: docData?.data?.length ?? '—',
      icon: FolderCheck, color: 'text-purple-600 bg-purple-50',
      link: '/admin/documents', hint: 'Menunggu review',
    },
  ]

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-gray-800">Selamat datang, {user?.name ?? 'Admin'} 👋</h1>
        <p className="text-sm text-gray-500 mt-0.5 capitalize">{user?.role?.replace('_', ' ') ?? ''}</p>
      </div>

      {/* Stats grid */}
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

      {/* Quick links */}
      <div className="bg-white rounded-xl border p-5">
        <h2 className="font-semibold text-gray-700 mb-4">Aksi Cepat</h2>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
          {[
            { to: '/admin/leads', label: 'Lihat Leads Baru', icon: Users },
            { to: '/admin/invoices', label: 'Buat Invoice', icon: FileText },
            { to: '/admin/documents', label: 'Review Dokumen', icon: FolderCheck },
            { to: '/admin/participants', label: 'Kelola Peserta', icon: ClipboardList },
            { to: '/admin/packages', label: 'Kelola Paket', icon: Package },
            { to: '/admin/airport', label: 'Airport Handling', icon: TrendingUp },
          ].map(({ to, label, icon: Icon }) => (
            <Link key={to} to={to}
              className="flex items-center gap-2 px-3 py-2.5 rounded-lg border text-sm text-gray-700 hover:bg-gray-50 transition-colors">
              <Icon size={16} className="text-emerald-600" />
              {label}
            </Link>
          ))}
        </div>
      </div>

      {/* Info banner */}
      <div className="bg-emerald-50 border border-emerald-200 rounded-xl p-4 text-sm text-emerald-700">
        <p className="font-medium mb-1">📋 Alur Operasional</p>
        <p className="text-xs text-emerald-600">
          Leads → Konsultasi → Deal → Konversi ke Peserta → Invoice → Konfirmasi Bayar → Review Dokumen → Briefing → Airport Handling
        </p>
      </div>
    </div>
  )
}
