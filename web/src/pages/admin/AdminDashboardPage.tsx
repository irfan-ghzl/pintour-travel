import { useQuery } from '@tanstack/react-query'
import { Package, MessageSquare, FileText, TrendingUp } from 'lucide-react'
import api from '../../utils/api'

export default function AdminDashboardPage() {
  const { data: pkgs } = useQuery({
    queryKey: ['admin-packages-count'],
    queryFn: () => api.get('/packages?per_page=1').then((r) => r.data),
  })
  const { data: inquiries } = useQuery({
    queryKey: ['admin-inquiries-count'],
    queryFn: () => api.get('/admin/inquiries?per_page=1').then((r) => r.data),
  })
  const { data: quotations } = useQuery({
    queryKey: ['admin-quotations-count'],
    queryFn: () => api.get('/admin/quotations?per_page=1').then((r) => r.data),
  })

  const stats = [
    { label: 'Paket Aktif', value: pkgs?.total ?? '—', icon: Package, color: 'text-primary-600 bg-primary-50' },
    { label: 'Total Konsultasi', value: inquiries?.total ?? '—', icon: MessageSquare, color: 'text-accent-600 bg-accent-50' },
    { label: 'Total Penawaran', value: quotations?.total ?? '—', icon: FileText, color: 'text-purple-600 bg-purple-50' },
    { label: 'Pertumbuhan', value: '+12%', icon: TrendingUp, color: 'text-green-600 bg-green-50' },
  ]

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900 mb-8">Dashboard</h1>

      <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-5">
        {stats.map(({ label, value, icon: Icon, color }) => (
          <div key={label} className="card p-6 flex items-center gap-4">
            <div className={`w-12 h-12 rounded-xl flex items-center justify-center ${color}`}>
              <Icon className="w-6 h-6" />
            </div>
            <div>
              <p className="text-2xl font-bold text-gray-900">{value}</p>
              <p className="text-sm text-gray-500">{label}</p>
            </div>
          </div>
        ))}
      </div>

      {/* Recent inquiries */}
      <div className="mt-10 card">
        <div className="px-6 py-4 border-b border-gray-100">
          <h2 className="font-semibold text-gray-800">Konsultasi Terbaru</h2>
        </div>
        <RecentInquiries />
      </div>
    </div>
  )
}

function RecentInquiries() {
  const { data, isLoading } = useQuery({
    queryKey: ['recent-inquiries'],
    queryFn: () => api.get('/admin/inquiries?per_page=5').then((r) => r.data),
  })

  if (isLoading) return <div className="px-6 py-8 text-sm text-gray-400">Memuat...</div>

  return (
    <div className="divide-y divide-gray-50">
      {data?.data?.length === 0 && (
        <p className="px-6 py-8 text-sm text-gray-400">Belum ada konsultasi masuk.</p>
      )}
      {data?.data?.map((inq: { id: string; full_name: string; destination?: string; status: string; created_at: string }) => (
        <div key={inq.id} className="px-6 py-4 flex items-center justify-between gap-4 flex-wrap">
          <div>
            <p className="font-medium text-gray-800 text-sm">{inq.full_name}</p>
            <p className="text-xs text-gray-400">{inq.destination ?? '—'}</p>
          </div>
          <div className="flex items-center gap-3">
            <span className={`text-xs font-medium px-2.5 py-0.5 rounded-full ${statusColor(inq.status)}`}>
              {inq.status}
            </span>
            <span className="text-xs text-gray-400">
              {new Date(inq.created_at).toLocaleDateString('id-ID')}
            </span>
          </div>
        </div>
      ))}
    </div>
  )
}

function statusColor(status: string): string {
  const map: Record<string, string> = {
    new: 'bg-blue-100 text-blue-700',
    contacted: 'bg-yellow-100 text-yellow-700',
    in_progress: 'bg-orange-100 text-orange-700',
    quoted: 'bg-purple-100 text-purple-700',
    booked: 'bg-green-100 text-green-700',
    closed: 'bg-gray-100 text-gray-500',
  }
  return map[status] ?? 'bg-gray-100 text-gray-500'
}
