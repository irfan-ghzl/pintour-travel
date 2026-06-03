import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Luggage, Ticket, FileCheck, Loader2, Send, FileText } from 'lucide-react'
import toast from 'react-hot-toast'
import api from '../../utils/api'
import type { AirportChecklist, AirportChecklistResponse } from '../../types'

type FilterStatus = '' | 'pending' | 'done'

export default function AdminAirportPage() {
  const [batchID, setBatchID] = useState('')
  const [statusFilter, setStatusFilter] = useState<FilterStatus>('')
  const [inputBatch, setInputBatch] = useState('')
  const [showConfirmModal, setShowConfirmModal] = useState(false)
  const [confirmForm, setConfirmForm] = useState({
    gather_point: '', gather_time: '', gate: '', checkin_time: '',
  })
  const qc = useQueryClient()

  const { data, isLoading, isFetching } = useQuery({
    queryKey: ['airport-checklist', batchID, statusFilter],
    queryFn: () => {
      const params = new URLSearchParams({ batch_id: batchID })
      if (statusFilter) params.set('status', statusFilter)
      return api.get<{ success: boolean; data: AirportChecklistResponse }>(`/admin/airport/checklist?${params}`)
        .then(r => r.data.data)
    },
    enabled: !!batchID,
    refetchInterval: batchID ? 10000 : false,
  })

  const checkMut = useMutation({
    mutationFn: ({ type, pid }: { type: 'baggage' | 'ticket' | 'passport'; pid: string }) =>
      api.patch(`/admin/airport/participants/${pid}/${type}?batch_id=${batchID}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['airport-checklist'] }),
    onError: () => toast.error('Gagal update checklist'),
  })

  const confirmMut = useMutation({
    mutationFn: () => api.post('/admin/airport/confirm-departure', {
      batch_id: batchID, ...confirmForm,
    }),
    onSuccess: (res) => {
      toast.success(res.data?.data?.message ?? 'WA konfirmasi berhasil dikirim')
      setShowConfirmModal(false)
    },
    onError: (e: any) => toast.error(e.response?.data?.message ?? 'Gagal mengirim konfirmasi'),
  })

  const { data: reportData } = useQuery({
    queryKey: ['airport-report', batchID],
    queryFn: () =>
      api.get(`/admin/airport/report?batch_id=${batchID}`).then(r => r.data),
    enabled: false,
  })

  const checklists = data?.checklists ?? []
  const progress = data?.progress

  function isDone(c: AirportChecklist) {
    return c.baggage_checked && c.ticket_distributed && c.passport_returned
  }

  const allDone = progress && progress.pending_count === 0 && progress.total_pax > 0

  return (
    <div className="space-y-4 pb-20 md:pb-6">
      <div className="flex items-center gap-3 flex-wrap">
        <h2 className="text-lg font-semibold text-gray-800">Airport Handling</h2>
        {isFetching && <Loader2 size={16} className="animate-spin text-gray-400" />}
        <span className="text-xs text-gray-400">Auto-refresh 10 detik</span>
      </div>

      {/* Batch input */}
      <div className="flex gap-2">
        <input
          className="flex-1 border rounded-lg px-3 py-2 text-sm max-w-xs"
          placeholder="Masukkan Batch ID..."
          value={inputBatch}
          onChange={e => setInputBatch(e.target.value)}
          onKeyDown={e => e.key === 'Enter' && setBatchID(inputBatch.trim())}
        />
        <button
          onClick={() => setBatchID(inputBatch.trim())}
          className="px-4 py-2 bg-emerald-600 text-white text-sm rounded-lg hover:bg-emerald-700"
        >
          Muat
        </button>
      </div>

      {!batchID && (
        <div className="text-center py-12 text-gray-400">
          <Loader2 className="mx-auto mb-3 opacity-30" size={40} />
          <p>Masukkan Batch ID untuk memuat checklist peserta</p>
        </div>
      )}

      {isLoading && batchID && (
        <div className="flex justify-center py-12">
          <Loader2 className="animate-spin text-emerald-600" size={32} />
        </div>
      )}

      {batchID && !isLoading && progress && (
        <>
          {/* Progress summary */}
          <div className="grid grid-cols-3 gap-3">
            <div className="bg-white rounded-xl border p-4 text-center">
              <p className="text-2xl font-bold text-gray-800">{progress.total_pax}</p>
              <p className="text-xs text-gray-500">Total Peserta</p>
            </div>
            <div className="bg-white rounded-xl border p-4 text-center">
              <p className="text-2xl font-bold text-emerald-600">{progress.done_count}</p>
              <p className="text-xs text-gray-500">Selesai</p>
            </div>
            <div className="bg-white rounded-xl border p-4 text-center">
              <p className="text-2xl font-bold text-orange-500">{progress.pending_count}</p>
              <p className="text-xs text-gray-500">Menunggu</p>
            </div>
          </div>

          {/* Progress bar */}
          <div className="bg-white rounded-xl border p-4">
            <div className="flex justify-between text-xs text-gray-500 mb-1.5">
              <span>Progress handling</span>
              <span>
                {progress.total_pax > 0 ? Math.round(progress.done_count / progress.total_pax * 100) : 0}%
              </span>
            </div>
            <div className="h-2.5 bg-gray-100 rounded-full overflow-hidden">
              <div
                className="h-full bg-emerald-500 rounded-full transition-all duration-500"
                style={{ width: `${progress.total_pax > 0 ? progress.done_count / progress.total_pax * 100 : 0}%` }}
              />
            </div>
          </div>

          {/* Actions: Laporan + Konfirmasi */}
          <div className="flex gap-2 flex-wrap">
            <a
              href={`/api/v1/admin/airport/report?batch_id=${batchID}`}
              target="_blank"
              className="flex items-center gap-1.5 px-3 py-2 border rounded-lg text-sm text-gray-600 hover:bg-gray-50"
            >
              <FileText size={14} /> Laporan JSON
            </a>
            {/* FR-AIR-06: laporan PDF */}
            <a
              href={`/api/v1/admin/airport/report/pdf?batch_id=${batchID}`}
              target="_blank"
              className="flex items-center gap-1.5 px-3 py-2 bg-blue-600 text-white text-sm rounded-lg hover:bg-blue-700"
            >
              <FileText size={14} /> Unduh PDF Laporan
            </a>
            {allDone && (
              <button
                onClick={() => setShowConfirmModal(true)}
                className="flex items-center gap-1.5 px-3 py-2 bg-emerald-600 text-white text-sm rounded-lg hover:bg-emerald-700"
              >
                <Send size={14} /> Kirim Konfirmasi Keberangkatan
              </button>
            )}
          </div>

          {/* Filter */}
          <div className="flex gap-2 flex-wrap">
            {([['', 'Semua'], ['pending', 'Belum Selesai'], ['done', 'Selesai']] as const).map(([v, l]) => (
              <button
                key={v}
                onClick={() => setStatusFilter(v)}
                className={`px-3 py-1.5 rounded-full text-xs font-medium border ${
                  statusFilter === v ? 'bg-emerald-700 text-white border-emerald-700' : 'bg-white text-gray-600 border-gray-200'
                }`}
              >
                {l}
              </button>
            ))}
          </div>

          {/* Checklist cards */}
          <div className="space-y-3">
            {checklists.length === 0 ? (
              <div className="text-center py-10 text-gray-400">Tidak ada peserta sesuai filter</div>
            ) : checklists.map(c => (
              <div
                key={c.id}
                className={`bg-white rounded-xl border p-4 ${isDone(c) ? 'border-emerald-200 bg-emerald-50/50' : ''}`}
              >
                <div className="flex items-center justify-between mb-3">
                  <div>
                    <p className="font-semibold text-sm text-gray-800">{c.participant_name}</p>
                    <p className="text-xs text-gray-500">{c.participant_phone}</p>
                    {c.handled_by_name && (
                      <p className="text-xs text-gray-400">Ditangani: {c.handled_by_name}</p>
                    )}
                  </div>
                  {isDone(c) && (
                    <span className="px-2 py-0.5 bg-emerald-100 text-emerald-700 text-xs rounded-full font-medium">
                      ✓ Selesai
                    </span>
                  )}
                </div>

                <div className="grid grid-cols-3 gap-2">
                  {[
                    { type: 'baggage', label: 'Bagasi', icon: Luggage, done: c.baggage_checked, time: c.baggage_checked_at },
                    { type: 'ticket', label: 'Tiket', icon: Ticket, done: c.ticket_distributed, time: c.ticket_distributed_at },
                    { type: 'passport', label: 'Paspor', icon: FileCheck, done: c.passport_returned, time: c.passport_returned_at },
                  ].map(item => (
                    <button
                      key={item.type}
                      onClick={() => !item.done && checkMut.mutate({ type: item.type as any, pid: c.participant_id })}
                      disabled={item.done || checkMut.isPending}
                      className={`flex flex-col items-center gap-1 p-3 rounded-lg border transition-colors ${
                        item.done
                          ? 'bg-emerald-50 border-emerald-200 text-emerald-700'
                          : 'bg-gray-50 border-gray-200 text-gray-500 hover:bg-orange-50 hover:border-orange-300 active:scale-95'
                      }`}
                    >
                      <item.icon size={20} />
                      <span className="text-xs font-medium">{item.label}</span>
                      {item.time && (
                        <span className="text-xs opacity-60">
                          {new Date(item.time).toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' })}
                        </span>
                      )}
                    </button>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </>
      )}

      {/* Confirm departure modal */}
      {showConfirmModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl w-full max-w-sm p-6 space-y-4">
            <div className="flex justify-between">
              <h3 className="font-semibold">Kirim Konfirmasi Keberangkatan</h3>
              <button onClick={() => setShowConfirmModal(false)} className="text-gray-400">✕</button>
            </div>
            <p className="text-xs text-gray-500">
              WA akan dikirim ke semua peserta batch ini dengan informasi keberangkatan.
            </p>
            <div className="space-y-3">
              {[
                { key: 'gather_point', label: 'Titik Kumpul', placeholder: 'contoh: Gate A2, Terminal 3' },
                { key: 'gather_time', label: 'Waktu Kumpul', placeholder: 'contoh: 04:00 WIB' },
                { key: 'gate', label: 'Nomor Gate', placeholder: 'contoh: Gate B7' },
                { key: 'checkin_time', label: 'Waktu Check-in', placeholder: 'contoh: 05:00 WIB' },
              ].map(f => (
                <div key={f.key}>
                  <label className="text-xs font-medium text-gray-600 block mb-1">{f.label}</label>
                  <input
                    className="w-full border rounded-lg px-3 py-2 text-sm"
                    placeholder={f.placeholder}
                    value={(confirmForm as any)[f.key]}
                    onChange={e => setConfirmForm({ ...confirmForm, [f.key]: e.target.value })}
                  />
                </div>
              ))}
            </div>
            <button
              onClick={() => confirmMut.mutate()}
              disabled={confirmMut.isPending}
              className="w-full py-2 bg-emerald-600 text-white rounded-lg text-sm font-medium disabled:opacity-50"
            >
              {confirmMut.isPending ? 'Mengirim WA...' : `Kirim ke ${progress?.total_pax ?? 0} Peserta`}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
