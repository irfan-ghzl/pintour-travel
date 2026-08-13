import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Search } from 'lucide-react'
import api from '../../utils/api'
import type { Participant, PaginatedResponse } from '../../types'

export default function AdminParticipantsPage() {
  const [search, setSearch] = useState('')
  const [batchID, setBatchID] = useState('')
  const [page, setPage] = useState(1)

  const params = new URLSearchParams({ page: String(page), per_page: '20' })
  if (search) params.set('search', search)
  if (batchID) params.set('batch_id', batchID)

  const { data, isLoading } = useQuery({
    queryKey: ['participants', search, batchID, page],
    queryFn: () => api.get<PaginatedResponse<Participant>>(`/admin/participants?${params}`).then(r => r.data),
  })

  const participants = data?.data ?? []
  const meta = data?.meta

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-gray-800">Peserta</h2>
        <p className="text-xs text-gray-500">
          Peserta dibuat dari halaman Leads &mdash; tombol Konversi pada lead berstatus deal.
        </p>
      </div>

      {/* Filters */}
      <div className="flex gap-3 flex-wrap">
        <div className="relative">
          <Search className="absolute left-3 top-2.5 text-gray-400" size={14} />
          <input
            className="pl-8 pr-4 py-2 border rounded-lg text-sm w-56"
            placeholder="Cari nama / nomor WA..."
            value={search}
            onChange={(e) => { setSearch(e.target.value); setPage(1) }}
          />
        </div>
        <input
          className="border rounded-lg px-3 py-2 text-sm"
          placeholder="Filter Batch ID..."
          value={batchID}
          onChange={(e) => { setBatchID(e.target.value); setPage(1) }}
        />
      </div>

      {/* Table */}
      <div className="bg-white rounded-xl border overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 text-xs text-gray-500 uppercase tracking-wider">
            <tr>
              <th className="px-4 py-3 text-left">Peserta</th>
              <th className="px-4 py-3 text-left">Paket</th>
              <th className="px-4 py-3 text-left">Tipe Kamar</th>
              <th className="px-4 py-3 text-left">Keberangkatan</th>
              <th className="px-4 py-3 text-left">Status Portal</th>
              <th className="px-4 py-3 text-left">Briefing</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {isLoading ? (
              Array.from({ length: 8 }).map((_, i) => (
                <tr key={i} className="animate-pulse">
                  <td colSpan={6} className="px-4 py-3"><div className="h-4 bg-gray-100 rounded" /></td>
                </tr>
              ))
            ) : participants.length === 0 ? (
              <tr><td colSpan={6} className="text-center py-12 text-gray-400">Belum ada peserta</td></tr>
            ) : participants.map((p) => (
              <tr key={p.id} className="hover:bg-gray-50">
                <td className="px-4 py-3">
                  <p className="font-medium text-gray-800">{p.name}</p>
                  <p className="text-xs text-gray-500">{p.phone}</p>
                </td>
                <td className="px-4 py-3 text-gray-600 text-xs">{p.package_name}</td>
                <td className="px-4 py-3 capitalize text-gray-600">{p.room_type}</td>
                <td className="px-4 py-3 text-gray-600 text-xs">
                  {p.batch_departure_date
                    ? new Date(p.batch_departure_date).toLocaleDateString('id-ID', { day: 'numeric', month: 'short', year: 'numeric' })
                    : '—'}
                </td>
                <td className="px-4 py-3">
                  <span className={`px-2 py-0.5 rounded-full text-xs ${p.is_active ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'}`}>
                    {p.is_active ? 'Aktif' : 'Belum Aktif'}
                  </span>
                </td>
                <td className="px-4 py-3">
                  <span className={`text-xs ${p.briefing_viewed ? 'text-green-600' : 'text-gray-400'}`}>
                    {p.briefing_viewed ? '✓ Dilihat' : '—'}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {meta && meta.total_pages > 1 && (
        <div className="flex justify-center gap-2">
          <button disabled={page <= 1} onClick={() => setPage(p => p - 1)}
            className="px-3 py-1.5 border rounded text-sm disabled:opacity-40">← Sebelumnya</button>
          <span className="px-3 py-1.5 text-sm text-gray-600">Hal {page}/{meta.total_pages}</span>
          <button disabled={page >= meta.total_pages} onClick={() => setPage(p => p + 1)}
            className="px-3 py-1.5 border rounded text-sm disabled:opacity-40">Berikutnya →</button>
        </div>
      )}

    </div>
  )
}
