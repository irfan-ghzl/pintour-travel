import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Search, RefreshCw, ChevronRight, UserPlus, MessageSquare, Activity } from 'lucide-react'
import toast from 'react-hot-toast'
import api from '../../utils/api'
import BatchPicker from '../../components/BatchPicker'
import type { Lead, LeadStatus, LeadNote, LeadStatusChange, PaginatedResponse, WANotification, Participant, ConvertLeadRequest } from '../../types'

const STATUS_COLORS: Record<LeadStatus, string> = {
  baru: 'bg-blue-100 text-blue-700',
  dihubungi: 'bg-yellow-100 text-yellow-700',
  konsultasi: 'bg-purple-100 text-purple-700',
  deal: 'bg-emerald-100 text-emerald-700',
  tidak_deal: 'bg-red-100 text-red-700',
  peserta: 'bg-gray-100 text-gray-600',
}

const STATUS_FLOW: Partial<Record<LeadStatus, LeadStatus>> = {
  baru: 'dihubungi',
  dihubungi: 'konsultasi',
  konsultasi: 'deal',
}

// isStale flags leads still 'baru' after 24h (§5.5) — pairs with the backend
// checkStaleLeads job that nudges the assigned consultant.
function isStale(lead: Lead) {
  return lead.status === 'baru' && Date.now() - new Date(lead.created_at).getTime() > 24 * 60 * 60 * 1000
}

const STATUS_OPTIONS: { value: LeadStatus | ''; label: string }[] = [
  { value: '', label: 'Semua Status' },
  { value: 'baru', label: 'Baru' },
  { value: 'dihubungi', label: 'Dihubungi' },
  { value: 'konsultasi', label: 'Konsultasi' },
  { value: 'deal', label: 'Deal' },
  { value: 'tidak_deal', label: 'Tidak Deal' },
  { value: 'peserta', label: 'Peserta' },
]

export default function AdminLeadsPage() {
  const [status, setStatus] = useState<LeadStatus | ''>('')
  const [search, setSearch] = useState('')
  const [dateFrom, setDateFrom] = useState('')
  const [dateTo, setDateTo] = useState('')
  const [page, setPage] = useState(1)
  const [selectedLead, setSelectedLead] = useState<Lead | null>(null)
  const [noteText, setNoteText] = useState('')
  const [activeTab, setActiveTab] = useState<'detail' | 'log' | 'wa'>('detail')
  // Konversi dibuka dari lead yang bersangkutan, bukan dari halaman Peserta.
  // Menyimpan lead-nya utuh (bukan sekadar id) supaya dialog bisa menyebut
  // namanya dan menyaring batch ke paket yang memang diminati lead itu.
  const [convertLead, setConvertLead] = useState<Lead | null>(null)
  const [convertRoom, setConvertRoom] = useState<ConvertLeadRequest['room_type']>('double')
  const [convertBatch, setConvertBatch] = useState('')
  const qc = useQueryClient()

  const params = new URLSearchParams({ page: String(page), per_page: '20' })
  if (status) params.set('status', status)
  if (search) params.set('search', search)
  if (dateFrom) params.set('date_from', dateFrom)
  if (dateTo) params.set('date_to', dateTo)

  const { data, isLoading } = useQuery({
    queryKey: ['leads', status, search, dateFrom, dateTo, page],
    queryFn: () => api.get<PaginatedResponse<Lead>>(`/admin/leads?${params}`).then(r => r.data),
  })

  // Activity log + WA log for selected lead (FR-CRM-03)
  const { data: leadDetail } = useQuery({
    queryKey: ['lead-detail', selectedLead?.id],
    queryFn: () =>
      api.get<{ success: boolean; data: {
        lead: Lead
        activity_log: LeadNote[]
        status_history: LeadStatusChange[]
        wa_log: WANotification[]
        previous_trips: Participant[] | null
      } }>(
        `/admin/leads/${selectedLead!.id}`
      ).then(r => r.data.data),
    enabled: !!selectedLead?.id,
  })

  const statusMut = useMutation({
    mutationFn: ({ id, newStatus }: { id: string; newStatus: string }) =>
      api.patch(`/admin/leads/${id}/status`, { status: newStatus }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['leads'] })
      qc.invalidateQueries({ queryKey: ['lead-detail'] })
      toast.success('Status diperbarui')
    },
    onError: () => toast.error('Gagal memperbarui status'),
  })

  const convertMut = useMutation({
    mutationFn: (req: ConvertLeadRequest) =>
      api.post('/admin/participants/convert', req).then(r => r.data),
    onSuccess: (res) => {
      qc.invalidateQueries({ queryKey: ['leads'] })
      qc.invalidateQueries({ queryKey: ['lead-detail'] })
      if (res.data?.reused_account) {
        toast.success('Peserta lama terdeteksi — akun portal lama dipakai, password tidak berubah.', { duration: 6000 })
      } else {
        toast.success(`Peserta dibuat. Password sementara: ${res.data?.temp_password}`, { duration: 6000 })
      }
      setConvertLead(null)
      setConvertBatch('')
    },
    onError: (e: any) => toast.error(e.response?.data?.message ?? 'Konversi gagal'),
  })

  const noteMut = useMutation({
    mutationFn: ({ id, note }: { id: string; note: string }) =>
      api.post(`/admin/leads/${id}/notes`, { note }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['lead-detail'] })
      toast.success('Catatan disimpan')
      setNoteText('')
    },
    onError: () => toast.error('Gagal menyimpan catatan'),
  })

  const leads = data?.data ?? []
  const meta = data?.meta
  const activityLog = leadDetail?.activity_log ?? []
  const statusHistory = leadDetail?.status_history ?? []
  const waLog = leadDetail?.wa_log ?? []

  // FR-CRM-02: the status trail and the consultant's notes are one timeline for
  // the reader even though they are two records for the system — reading either
  // alone loses the thread of what happened to the lead.
  const timeline = [
    ...activityLog.map(n => ({
      id: `note-${n.id}`,
      kind: 'note' as const,
      at: n.created_at,
      who: n.user_name,
      text: n.note,
    })),
    ...statusHistory.map(s => ({
      id: `status-${s.id}`,
      kind: 'status' as const,
      at: s.changed_at,
      who: s.changed_by_name,
      text: s.from_status
        ? `Status: ${s.from_status} → ${s.to_status}`
        : `Status: ${s.to_status}`,
    })),
  ].sort((a, b) => new Date(b.at).getTime() - new Date(a.at).getTime())

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-gray-800">CRM Leads</h2>
        <span className="text-sm text-gray-500">{meta?.total ?? 0} total</span>
      </div>

      {/* Filters */}
      <div className="flex gap-3 flex-wrap">
        <div className="relative">
          <Search className="absolute left-3 top-2.5 text-gray-400" size={14} />
          <input
            className="pl-8 pr-4 py-2 border rounded-lg text-sm w-56"
            placeholder="Cari nama / nomor WA..."
            value={search}
            onChange={e => { setSearch(e.target.value); setPage(1) }}
          />
        </div>
        <select
          className="border rounded-lg px-3 py-2 text-sm"
          value={status}
          onChange={e => { setStatus(e.target.value as LeadStatus | ''); setPage(1) }}
        >
          {STATUS_OPTIONS.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
        </select>
        {/* FR-CRM-05: filter rentang tanggal masuk */}
        <div className="flex items-center gap-1">
          <input
            type="date"
            className="border rounded-lg px-2 py-2 text-sm"
            value={dateFrom}
            onChange={e => { setDateFrom(e.target.value); setPage(1) }}
            title="Dari tanggal"
          />
          <span className="text-xs text-gray-400">–</span>
          <input
            type="date"
            className="border rounded-lg px-2 py-2 text-sm"
            value={dateTo}
            onChange={e => { setDateTo(e.target.value); setPage(1) }}
            title="Sampai tanggal"
          />
        </div>
        {(dateFrom || dateTo) && (
          <button
            onClick={() => { setDateFrom(''); setDateTo(''); setPage(1) }}
            className="text-xs text-emerald-600 hover:underline"
          >
            × Reset tanggal
          </button>
        )}
      </div>

      {/* Table */}
      <div className="bg-white rounded-xl border overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 text-xs text-gray-500 uppercase tracking-wider">
            <tr>
              <th className="px-4 py-3 text-left">Leads</th>
              <th className="px-4 py-3 text-left">Paket</th>
              <th className="px-4 py-3 text-left">Pax</th>
              <th className="px-4 py-3 text-left">Status</th>
              <th className="px-4 py-3 text-left">Konsultan</th>
              <th className="px-4 py-3 text-left">Masuk</th>
              <th className="px-4 py-3"></th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {isLoading ? (
              Array.from({ length: 8 }).map((_, i) => (
                <tr key={i} className="animate-pulse">
                  <td colSpan={7} className="px-4 py-3"><div className="h-4 bg-gray-100 rounded" /></td>
                </tr>
              ))
            ) : leads.length === 0 ? (
              <tr><td colSpan={7} className="text-center py-12 text-gray-400">Tidak ada leads</td></tr>
            ) : leads.map(lead => (
              <tr key={lead.id} className="hover:bg-gray-50 transition-colors">
                <td className="px-4 py-3">
                  <div className="flex items-center gap-1.5">
                    <p className="font-medium text-gray-800">{lead.name}</p>
                    {lead.is_returning && (
                      <span
                        className="inline-flex items-center px-1.5 py-0.5 rounded-full text-[10px] font-semibold bg-blue-100 text-blue-700"
                        title="Sudah pernah ikut tour — gunakan akun portal yang sama"
                      >
                        ⭐ Pelanggan Lama
                      </span>
                    )}
                  </div>
                  <p className="text-gray-500 text-xs">{lead.phone}</p>
                </td>
                <td className="px-4 py-3 text-gray-600 max-w-xs truncate text-xs">{lead.package_name}</td>
                <td className="px-4 py-3 text-gray-600 text-xs">{lead.pax} orang</td>
                <td className="px-4 py-3">
                  <div className="flex items-center gap-1.5">
                    <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${STATUS_COLORS[lead.status]}`}>
                      {lead.status.replace('_', ' ')}
                    </span>
                    {isStale(lead) && (
                      <span
                        className="inline-flex items-center px-1.5 py-0.5 rounded-full text-[10px] font-semibold bg-red-100 text-red-700 animate-pulse"
                        title="Belum direspons lebih dari 24 jam"
                      >
                        ⚠️ 24 jam
                      </span>
                    )}
                  </div>
                </td>
                <td className="px-4 py-3 text-gray-500 text-xs">{lead.assignee_name ?? '—'}</td>
                <td className="px-4 py-3 text-gray-400 text-xs">
                  {new Date(lead.created_at).toLocaleDateString('id-ID', { day: 'numeric', month: 'short' })}
                </td>
                <td className="px-4 py-3">
                  <div className="flex items-center gap-1.5">
                    {STATUS_FLOW[lead.status] && (
                      <button
                        onClick={() => statusMut.mutate({ id: lead.id, newStatus: STATUS_FLOW[lead.status]! })}
                        disabled={statusMut.isPending}
                        className="flex items-center gap-1 px-2 py-1 text-xs bg-emerald-50 text-emerald-700 rounded hover:bg-emerald-100"
                        title={`Ubah ke: ${STATUS_FLOW[lead.status]}`}
                      >
                        <RefreshCw size={11} />
                        {STATUS_FLOW[lead.status]}
                      </button>
                    )}
                    {lead.status === 'deal' && (
                      <button
                        onClick={() => { setConvertLead(lead); setConvertBatch(''); setConvertRoom('double') }}
                        className="flex items-center gap-1 px-2 py-1 text-xs bg-blue-600 text-white rounded font-medium hover:bg-blue-700"
                      >
                        <UserPlus size={11} />
                        Konversi
                      </button>
                    )}
                    <button
                      onClick={() => { setSelectedLead(lead); setActiveTab('detail') }}
                      className="p-1 text-gray-400 hover:text-gray-600"
                    >
                      <ChevronRight size={16} />
                    </button>
                  </div>
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

      {/* Lead detail panel */}
      {selectedLead && (
        <div className="fixed inset-0 bg-black/50 flex items-end md:items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl w-full max-w-md p-5 space-y-4 max-h-[90vh] overflow-y-auto">
            <div className="flex justify-between items-start">
              <div>
                <h3 className="font-semibold text-gray-800">{selectedLead.name}</h3>
                <p className="text-sm text-gray-500">{selectedLead.phone}</p>
              </div>
              <button onClick={() => setSelectedLead(null)} className="text-gray-400 hover:text-gray-600">✕</button>
            </div>

            {/* Tabs */}
            <div className="flex border-b text-sm">
              {[
                { id: 'detail' as const, label: 'Detail', icon: MessageSquare, count: 0 },
                { id: 'log' as const, label: 'Log Aktivitas', icon: Activity, count: timeline.length },
                { id: 'wa' as const, label: 'Riwayat WA', icon: MessageSquare, count: waLog.length },
              ].map(tab => (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={`flex items-center gap-1.5 px-3 py-2 border-b-2 transition-colors ${
                    activeTab === tab.id ? 'border-emerald-600 text-emerald-700 font-medium' : 'border-transparent text-gray-500'
                  }`}
                >
                  <tab.icon size={13} /> {tab.label}
                  {tab.count > 0 && (
                    <span className="bg-gray-200 text-gray-600 text-xs px-1 py-0.5 rounded-full">{tab.count}</span>
                  )}
                </button>
              ))}
            </div>

            {activeTab === 'detail' && (
              <div className="space-y-4">
                <div className="text-sm space-y-1.5 text-gray-600">
                  <p><span className="font-medium">Paket:</span> {selectedLead.package_name}</p>
                  <p><span className="font-medium">Pax:</span> {selectedLead.pax} orang</p>
                  <p><span className="font-medium">Sumber:</span> {selectedLead.source}</p>
                  <p><span className="font-medium">Masuk:</span> {new Date(selectedLead.created_at).toLocaleDateString('id-ID')}</p>
                  {selectedLead.message && (
                    <div className="bg-gray-50 rounded-lg p-2">
                      <p className="font-medium text-xs text-gray-500 mb-1">Pesan:</p>
                      <p className="text-sm">{selectedLead.message}</p>
                    </div>
                  )}
                </div>

                {/* v2.0 F4 — Riwayat tour sebelumnya (returning customer) */}
                {leadDetail?.previous_trips && leadDetail.previous_trips.length > 0 && (
                  <div className="bg-blue-50 border border-blue-100 rounded-lg p-3">
                    <p className="font-medium text-xs text-blue-700 mb-2 flex items-center gap-1">
                      ⭐ Pelanggan Lama — {leadDetail.previous_trips.length} tour sebelumnya
                    </p>
                    <div className="space-y-1.5">
                      {leadDetail.previous_trips.map(t => (
                        <div key={t.id} className="text-xs text-gray-600 flex justify-between gap-2">
                          <span className="truncate">{t.package_name}</span>
                          <span className="text-gray-400 shrink-0">
                            {t.batch_departure_date
                              ? new Date(t.batch_departure_date).toLocaleDateString('id-ID', { month: 'short', year: 'numeric' })
                              : '—'}
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {/* Quick status update */}
                <div className="flex gap-2 flex-wrap">
                  {(['dihubungi', 'konsultasi', 'deal', 'tidak_deal'] as LeadStatus[]).map(s => (
                    <button
                      key={s}
                      onClick={() => statusMut.mutate({ id: selectedLead.id, newStatus: s })}
                      disabled={selectedLead.status === s || statusMut.isPending}
                      className={`px-2.5 py-1 text-xs rounded-full border disabled:opacity-40 ${
                        selectedLead.status === s
                          ? STATUS_COLORS[s] + ' font-semibold border-transparent'
                          : 'bg-white text-gray-600 border-gray-200 hover:bg-gray-50'
                      }`}
                    >
                      {s.replace('_', ' ')}
                    </button>
                  ))}
                </div>

                {/* Add note */}
                <div>
                  <p className="text-xs font-medium text-gray-500 mb-1.5">Tambah Catatan</p>
                  <textarea
                    rows={3}
                    className="w-full border rounded-lg px-3 py-2 text-sm resize-none"
                    placeholder="Catatan konsultasi..."
                    value={noteText}
                    onChange={e => setNoteText(e.target.value)}
                  />
                  <button
                    onClick={() => noteMut.mutate({ id: selectedLead.id, note: noteText })}
                    disabled={!noteText.trim() || noteMut.isPending}
                    className="mt-1.5 px-4 py-1.5 bg-emerald-600 text-white text-sm rounded-lg disabled:opacity-50"
                  >
                    {noteMut.isPending ? 'Menyimpan...' : 'Simpan Catatan'}
                  </button>
                </div>

                {selectedLead.status === 'deal' && (
                  <div className="border-t pt-3">
                    <p className="text-xs text-gray-500 mb-2">
                      Leads siap dikonversi ke peserta. Buka halaman Peserta untuk proses konversi.
                    </p>
                    <a
                      href="/admin/participants"
                      className="inline-flex items-center gap-1 px-3 py-1.5 bg-blue-600 text-white text-xs rounded-lg"
                    >
                      <UserPlus size={12} /> Pergi ke Halaman Peserta
                    </a>
                  </div>
                )}
              </div>
            )}

            {activeTab === 'log' && (
              <div className="space-y-2 max-h-80 overflow-y-auto">
                {timeline.length === 0 ? (
                  <p className="text-sm text-gray-400 text-center py-6">Belum ada aktivitas tercatat</p>
                ) : (
                  timeline.map(entry => (
                    <div key={entry.id} className="flex gap-2 text-sm">
                      <div
                        className={`w-1.5 h-1.5 rounded-full mt-1.5 shrink-0 ${
                          entry.kind === 'status' ? 'bg-sky-400' : 'bg-emerald-400'
                        }`}
                      />
                      <div>
                        <p className={entry.kind === 'status' ? 'text-gray-500 text-xs' : 'text-gray-700'}>
                          {entry.text}
                        </p>
                        <p className="text-xs text-gray-400 mt-0.5">
                          {entry.who} · {new Date(entry.at).toLocaleDateString('id-ID', {
                            day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit',
                          })}
                        </p>
                      </div>
                    </div>
                  ))
                )}
              </div>
            )}

            {/* FR-CRM-03: WA notification history linked to this lead */}
            {activeTab === 'wa' && (
              <div className="space-y-2 max-h-80 overflow-y-auto">
                {waLog.length === 0 ? (
                  <p className="text-sm text-gray-400 text-center py-6">Belum ada pesan WA tercatat</p>
                ) : (
                  [...waLog].map(notif => (
                    <div key={notif.id} className="border rounded-lg p-3 text-xs">
                      <div className="flex items-center justify-between mb-1">
                        <span className={`px-2 py-0.5 rounded-full font-medium ${
                          notif.status === 'sent' ? 'bg-green-100 text-green-700' :
                          notif.status === 'failed' ? 'bg-red-100 text-red-600' :
                          'bg-yellow-100 text-yellow-700'
                        }`}>
                          {notif.status}
                        </span>
                        <span className="text-gray-400">
                          {notif.sent_at
                            ? new Date(notif.sent_at).toLocaleDateString('id-ID', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })
                            : new Date(notif.created_at).toLocaleDateString('id-ID', { day: 'numeric', month: 'short' })
                          }
                        </span>
                      </div>
                      <p className="font-medium text-gray-700 mb-1">{notif.message_type}</p>
                      <p className="text-gray-500 whitespace-pre-wrap line-clamp-3">{notif.message_content}</p>
                    </div>
                  ))
                )}
              </div>
            )}
          </div>
        </div>
      )}

      {convertLead && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl w-full max-w-sm p-6 space-y-4">
            <div className="flex justify-between items-start">
              <div>
                <h3 className="font-semibold text-gray-800">Konversi ke Peserta</h3>
                <p className="text-xs text-gray-500 mt-0.5">
                  {convertLead.name} &middot; {convertLead.phone} &middot; {convertLead.pax} pax
                </p>
              </div>
              <button onClick={() => setConvertLead(null)} className="text-gray-400">&#10005;</button>
            </div>

            <p className="text-xs text-gray-600 bg-gray-50 rounded-lg px-3 py-2">
              Paket: <span className="font-medium">{convertLead.package_name}</span>
            </p>

            {convertLead.is_returning && (
              <div className="bg-amber-50 border border-amber-200 rounded-lg px-3 py-2.5 text-xs text-amber-800">
                <p className="font-semibold mb-0.5">Pelanggan Lama Terdeteksi</p>
                <p>
                  Nomor <span className="font-medium">{convertLead.phone}</span> sudah memiliki akun portal.
                  Akun lama akan dipakai &mdash; password <span className="font-medium">tidak berubah</span>.
                </p>
              </div>
            )}

            <div className="space-y-3">
              <div>
                <label htmlFor="convert-batch" className="text-xs font-medium text-gray-600 block mb-1">
                  Batch Keberangkatan
                </label>
                {/* Pemilih yang sama dengan keempat halaman lain. Dibatasi ke
                    paket yang diminati lead, sehingga "peserta ditempatkan pada
                    batch paket lain" tetap tidak mungkin — dan batch penuh atau
                    ditutup tetap terlihat beserta alasannya. */}
                <BatchPicker
                  id="convert-batch"
                  value={convertBatch}
                  onChange={(id) => setConvertBatch(id)}
                  packageId={convertLead.package_id}
                  disableUnavailable
                  placeholder="Pilih tanggal keberangkatan..."
                  emptyLabel="Paket ini belum punya batch keberangkatan. Tambahkan dulu di menu Paket."
                />
              </div>
              <div>
                <label className="text-xs font-medium text-gray-600 block mb-1">Tipe Kamar</label>
                <select
                  className="w-full border rounded-lg px-3 py-2 text-sm"
                  value={convertRoom}
                  onChange={(e) => setConvertRoom(e.target.value as ConvertLeadRequest['room_type'])}
                >
                  <option value="single">Single</option>
                  <option value="double">Double</option>
                  <option value="triple">Triple</option>
                </select>
              </div>
            </div>

            <button
              onClick={() => convertMut.mutate({ lead_id: convertLead.id, batch_id: convertBatch, room_type: convertRoom })}
              disabled={!convertBatch || convertMut.isPending}
              className="w-full py-2 bg-emerald-600 text-white rounded-lg text-sm font-medium disabled:opacity-50"
            >
              {convertMut.isPending ? 'Memproses...' : 'Konversi Sekarang'}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
