import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Search, UserPlus } from 'lucide-react'
import toast from 'react-hot-toast'
import api from '../../utils/api'
import type { Participant, PaginatedResponse, ConvertLeadRequest, Lead } from '../../types'

export default function AdminParticipantsPage() {
  const [search, setSearch] = useState('')
  const [batchID, setBatchID] = useState('')
  const [page, setPage] = useState(1)
  const [showConvert, setShowConvert] = useState(false)
  const [convertForm, setConvertForm] = useState<ConvertLeadRequest>({
    lead_id: '', batch_id: '', room_type: 'double',
  })
  const qc = useQueryClient()

  const params = new URLSearchParams({ page: String(page), per_page: '20' })
  if (search) params.set('search', search)
  if (batchID) params.set('batch_id', batchID)

  const { data, isLoading } = useQuery({
    queryKey: ['participants', search, batchID, page],
    queryFn: () => api.get<PaginatedResponse<Participant>>(`/admin/participants?${params}`).then(r => r.data),
  })

  // v2.0 F4 — peek at the lead being converted to warn before submit when the
  // customer already has a portal account (returning customer).
  const leadId = convertForm.lead_id.trim()
  const { data: convertLead } = useQuery({
    queryKey: ['convert-lead-peek', leadId],
    queryFn: () =>
      api.get<{ success: boolean; data: { lead: Lead; previous_trips: Participant[] | null } }>(`/admin/leads/${leadId}`)
        .then(r => r.data.data),
    enabled: showConvert && leadId.length >= 30,
    retry: false,
  })

  const convertMutation = useMutation({
    mutationFn: (req: ConvertLeadRequest) =>
      api.post('/admin/participants/convert', req).then(r => r.data),
    onSuccess: (res) => {
      qc.invalidateQueries({ queryKey: ['participants'] })
      // v2.0 F1 — returning customer reuses the existing portal account (no new password).
      if (res.data?.reused_account) {
        toast.success('Peserta lama terdeteksi — akun portal lama digunakan, password tidak berubah.', { duration: 6000 })
      } else {
        toast.success(`Peserta berhasil dibuat! Password sementara: ${res.data?.temp_password}`, { duration: 6000 })
      }
      setShowConvert(false)
    },
    onError: (e: any) => toast.error(e.response?.data?.message ?? 'Konversi gagal'),
  })

  const participants = data?.data ?? []
  const meta = data?.meta

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-gray-800">Peserta</h2>
        <button
          onClick={() => setShowConvert(true)}
          className="flex items-center gap-2 px-4 py-2 bg-emerald-600 text-white text-sm rounded-lg hover:bg-emerald-700"
        >
          <UserPlus size={16} /> Konversi Leads
        </button>
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

      {/* Convert modal */}
      {showConvert && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl w-full max-w-sm p-6 space-y-4">
            <div className="flex justify-between">
              <h3 className="font-semibold text-gray-800">Konversi Leads ke Peserta</h3>
              <button onClick={() => setShowConvert(false)} className="text-gray-400">✕</button>
            </div>
            {/* v2.0 F4 — peringatan returning customer sebelum submit */}
            {convertLead?.lead?.is_returning && (
              <div className="bg-amber-50 border border-amber-200 rounded-lg px-3 py-2.5 text-xs text-amber-800">
                <p className="font-semibold mb-0.5">⚠️ Pelanggan Lama Terdeteksi</p>
                <p>
                  Nomor <span className="font-medium">{convertLead.lead.phone}</span> sudah memiliki akun portal.
                  Akun lama akan digunakan — password <span className="font-medium">tidak berubah</span>, tidak perlu kirim password baru.
                </p>
                {convertLead.previous_trips && convertLead.previous_trips.length > 0 && (
                  <p className="mt-1 text-amber-700">
                    {convertLead.previous_trips.length} tour sebelumnya: {convertLead.previous_trips.map(t => t.package_name).join(', ')}
                  </p>
                )}
              </div>
            )}
            <div className="space-y-3">
              <div>
                <label className="text-xs font-medium text-gray-600 block mb-1">Lead ID (paste dari daftar leads)</label>
                <input
                  className="w-full border rounded-lg px-3 py-2 text-sm"
                  placeholder="UUID lead..."
                  value={convertForm.lead_id}
                  onChange={(e) => setConvertForm({ ...convertForm, lead_id: e.target.value })}
                />
              </div>
              <div>
                <label className="text-xs font-medium text-gray-600 block mb-1">Batch ID</label>
                <input
                  className="w-full border rounded-lg px-3 py-2 text-sm"
                  placeholder="UUID batch keberangkatan..."
                  value={convertForm.batch_id}
                  onChange={(e) => setConvertForm({ ...convertForm, batch_id: e.target.value })}
                />
              </div>
              <div>
                <label className="text-xs font-medium text-gray-600 block mb-1">Tipe Kamar</label>
                <select
                  className="w-full border rounded-lg px-3 py-2 text-sm"
                  value={convertForm.room_type}
                  onChange={(e) => setConvertForm({ ...convertForm, room_type: e.target.value as any })}
                >
                  <option value="single">Single</option>
                  <option value="double">Double</option>
                  <option value="triple">Triple</option>
                </select>
              </div>
            </div>
            <button
              onClick={() => convertMutation.mutate(convertForm)}
              disabled={!convertForm.lead_id || !convertForm.batch_id || convertMutation.isPending}
              className="w-full py-2 bg-emerald-600 text-white rounded-lg text-sm font-medium disabled:opacity-50"
            >
              {convertMutation.isPending ? 'Memproses...' : 'Konversi Sekarang'}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
