import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { CheckCircle, XCircle, FileText } from 'lucide-react'
import toast from 'react-hot-toast'
import api from '../../utils/api'
import type { Document, DocumentStatus } from '../../types'

const STATUS_COLORS: Record<DocumentStatus, string> = {
  menunggu: 'bg-yellow-100 text-yellow-700',
  disetujui: 'bg-green-100 text-green-700',
  ditolak: 'bg-red-100 text-red-700',
}

const DOC_TYPE_LABELS: Record<string, string> = {
  passport: 'Paspor', ktp: 'KTP', rekening_koran: 'Rekening Koran',
  visa_support: 'Dokumen Visa', lainnya: 'Lainnya',
}

export default function AdminDocumentsPage() {
  const [statusFilter, setStatusFilter] = useState<DocumentStatus | ''>('menunggu')
  const [participantID, setParticipantID] = useState('')
  const [rejectDoc, setRejectDoc] = useState<Document | null>(null)
  const [rejectReason, setRejectReason] = useState('')
  const qc = useQueryClient()

  const params = new URLSearchParams()
  if (statusFilter) params.set('status', statusFilter)
  if (participantID) params.set('participant_id', participantID)

  const { data: docsData, isLoading } = useQuery({
    queryKey: ['documents', statusFilter, participantID],
    queryFn: () => {
      const url = participantID
        ? `/admin/participants/${participantID}/documents`
        : `/admin/documents?${params}`
      return api.get<{ success: boolean; data: Document[] }>(url).then(r => r.data.data)
    },
  })

  const reviewMutation = useMutation({
    mutationFn: ({ id, status, reason }: { id: string; status: string; reason?: string }) =>
      api.patch(`/admin/documents/${id}/review`, { status, rejection_reason: reason ?? '' }),
    onSuccess: (_, vars) => {
      qc.invalidateQueries({ queryKey: ['documents'] })
      toast.success(vars.status === 'disetujui' ? 'Dokumen disetujui' : 'Dokumen ditolak, notif terkirim')
      setRejectDoc(null)
      setRejectReason('')
    },
    onError: () => toast.error('Gagal mereview dokumen'),
  })

  const docs = docsData ?? []

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold text-gray-800">Review Dokumen Peserta</h2>

      {/* Filters */}
      <div className="flex gap-3 flex-wrap">
        <input
          className="border rounded-lg px-3 py-2 text-sm w-56"
          placeholder="Filter Participant ID..."
          value={participantID}
          onChange={(e) => setParticipantID(e.target.value)}
        />
        <div className="flex gap-2">
          {(['', 'menunggu', 'disetujui', 'ditolak'] as const).map((s) => (
            <button
              key={s}
              onClick={() => setStatusFilter(s)}
              className={`px-3 py-1.5 rounded-full text-xs font-medium border ${
                statusFilter === s ? 'bg-emerald-700 text-white border-emerald-700' : 'bg-white text-gray-600 border-gray-200'
              }`}
            >
              {s === '' ? 'Semua' : s}
            </button>
          ))}
        </div>
      </div>

      {/* Document list */}
      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="h-16 bg-gray-100 rounded-xl animate-pulse" />
          ))}
        </div>
      ) : docs.length === 0 ? (
        <div className="text-center py-16 text-gray-400">
          <FileText className="mx-auto mb-3 opacity-40" size={40} />
          <p>Tidak ada dokumen {statusFilter === 'menunggu' ? 'yang menunggu review' : ''}</p>
        </div>
      ) : (
        <div className="bg-white rounded-xl border divide-y">
          {docs.map((doc) => (
            <div key={doc.id} className="px-4 py-3 flex items-center gap-3">
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-0.5">
                  <p className="font-medium text-sm text-gray-800">{doc.participant_name}</p>
                  <span className="text-xs text-gray-400">·</span>
                  <span className="text-xs text-gray-600">{DOC_TYPE_LABELS[doc.document_type] ?? doc.document_type}</span>
                </div>
                <p className="text-xs text-gray-500 truncate">{doc.file_name}</p>
                {doc.rejection_reason && (
                  <p className="text-xs text-red-500 mt-0.5">Alasan: {doc.rejection_reason}</p>
                )}
              </div>

              <span className={`px-2 py-0.5 rounded-full text-xs font-medium shrink-0 ${STATUS_COLORS[doc.status]}`}>
                {doc.status}
              </span>

              <a
                href={doc.file_path}
                target="_blank"
                rel="noopener noreferrer"
                className="px-2 py-1 text-xs border rounded hover:bg-gray-50 shrink-0"
              >
                Lihat
              </a>

              {doc.status === 'menunggu' && (
                <div className="flex gap-1 shrink-0">
                  <button
                    onClick={() => reviewMutation.mutate({ id: doc.id, status: 'disetujui' })}
                    className="p-1.5 text-green-600 hover:bg-green-50 rounded"
                    title="Setujui"
                  >
                    <CheckCircle size={18} />
                  </button>
                  <button
                    onClick={() => setRejectDoc(doc)}
                    className="p-1.5 text-red-500 hover:bg-red-50 rounded"
                    title="Tolak"
                  >
                    <XCircle size={18} />
                  </button>
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Reject modal */}
      {rejectDoc && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl w-full max-w-sm p-6 space-y-4">
            <h3 className="font-semibold">Tolak Dokumen</h3>
            <p className="text-sm text-gray-600">
              Dokumen: <strong>{DOC_TYPE_LABELS[rejectDoc.document_type]}</strong> dari <strong>{rejectDoc.participant_name}</strong>
            </p>
            <div>
              <label className="text-xs font-medium text-gray-600 block mb-1">Alasan Penolakan *</label>
              <textarea
                rows={3}
                className="w-full border rounded-lg px-3 py-2 text-sm resize-none"
                placeholder="Contoh: Foto buram, harap upload ulang dengan kualitas lebih baik"
                value={rejectReason}
                onChange={(e) => setRejectReason(e.target.value)}
              />
            </div>
            <div className="flex gap-3">
              <button
                onClick={() => { setRejectDoc(null); setRejectReason('') }}
                className="flex-1 py-2 border rounded-lg text-sm"
              >
                Batal
              </button>
              <button
                onClick={() => reviewMutation.mutate({ id: rejectDoc.id, status: 'ditolak', reason: rejectReason })}
                disabled={!rejectReason.trim() || reviewMutation.isPending}
                className="flex-1 py-2 bg-red-600 text-white rounded-lg text-sm disabled:opacity-50"
              >
                Tolak Dokumen
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
