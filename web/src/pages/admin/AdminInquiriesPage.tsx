import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { ExternalLink } from 'lucide-react'
import api from '../../utils/api'
import { InquiriesResponse, Inquiry } from '../../types'
import Spinner from '../../components/Spinner'

const STATUSES = ['new', 'contacted', 'in_progress', 'quoted', 'booked', 'closed']

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

export default function AdminInquiriesPage() {
  const [page, setPage] = useState(1)
  const [filterStatus, setFilterStatus] = useState('')
  const qc = useQueryClient()

  const { data, isLoading } = useQuery<InquiriesResponse>({
    queryKey: ['admin-inquiries', page, filterStatus],
    queryFn: () => {
      const q = filterStatus ? `&status=${filterStatus}` : ''
      return api.get(`/admin/inquiries?page=${page}&per_page=15${q}`).then((r) => r.data)
    },
  })

  const statusMut = useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      api.patch(`/admin/inquiries/${id}/status`, { status }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-inquiries'] }),
  })

  if (isLoading) return <Spinner message="Memuat konsultasi..." />

  const totalPages = data ? Math.ceil(data.total / 15) : 1

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Lead & Konsultasi</h1>
        <select
          value={filterStatus}
          onChange={(e) => { setFilterStatus(e.target.value); setPage(1) }}
          className="input w-auto text-sm"
        >
          <option value="">Semua Status</option>
          {STATUSES.map((s) => (
            <option key={s} value={s}>{s}</option>
          ))}
        </select>
      </div>

      <div className="card overflow-x-auto">
        <table className="w-full text-sm min-w-max">
          <thead className="bg-gray-50 text-gray-500 text-xs uppercase tracking-wide">
            <tr>
              <th className="px-5 py-3 text-left">Nama</th>
              <th className="px-5 py-3 text-left">Destinasi</th>
              <th className="px-5 py-3 text-left">Orang</th>
              <th className="px-5 py-3 text-left">Budget</th>
              <th className="px-5 py-3 text-left">Tanggal</th>
              <th className="px-5 py-3 text-left">Status</th>
              <th className="px-5 py-3 text-left">WA</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-50">
            {data?.data.map((inq: Inquiry) => (
              <tr key={inq.id} className="hover:bg-gray-50 transition-colors">
                <td className="px-5 py-3">
                  <p className="font-medium text-gray-800">{inq.full_name}</p>
                  {inq.email && <p className="text-xs text-gray-400">{inq.email}</p>}
                </td>
                <td className="px-5 py-3 text-gray-600">{inq.destination ?? '—'}</td>
                <td className="px-5 py-3 text-gray-600">{inq.num_people}</td>
                <td className="px-5 py-3 text-gray-600">
                  {inq.budget ? `Rp ${inq.budget.toLocaleString('id-ID')}` : '—'}
                </td>
                <td className="px-5 py-3 text-gray-400 text-xs">
                  {new Date(inq.created_at).toLocaleDateString('id-ID')}
                </td>
                <td className="px-5 py-3">
                  <select
                    value={inq.status}
                    onChange={(e) => statusMut.mutate({ id: inq.id, status: e.target.value })}
                    className={`text-xs font-medium px-2 py-0.5 rounded-full border-0 cursor-pointer ${statusColor(inq.status)}`}
                  >
                    {STATUSES.map((s) => <option key={s} value={s}>{s}</option>)}
                  </select>
                </td>
                <td className="px-5 py-3">
                  {inq.wa_link && (
                    <a
                      href={inq.wa_link}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-green-600 hover:text-green-700"
                      title="Buka WA"
                    >
                      <ExternalLink className="w-4 h-4" />
                    </a>
                  )}
                </td>
              </tr>
            ))}
            {(!data?.data || data.data.length === 0) && (
              <tr><td colSpan={7} className="px-5 py-10 text-center text-gray-400">Belum ada konsultasi.</td></tr>
            )}
          </tbody>
        </table>
      </div>

      {totalPages > 1 && (
        <div className="flex justify-center gap-2 mt-6">
          <button onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page === 1} className="btn-secondary py-2 px-4 text-sm disabled:opacity-40">
            Sebelumnya
          </button>
          <span className="flex items-center px-4 text-sm text-gray-500">
            {page} / {totalPages}
          </span>
          <button onClick={() => setPage((p) => Math.min(totalPages, p + 1))} disabled={page === totalPages} className="btn-secondary py-2 px-4 text-sm disabled:opacity-40">
            Berikutnya
          </button>
        </div>
      )}
    </div>
  )
}
