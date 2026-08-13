import { useMemo, useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { CheckCircle, XCircle, FileText, ScanLine } from 'lucide-react'
import toast from 'react-hot-toast'
import api, {
  approveDocument, rejectDocument, openSignedFile,
  // TODO(ocr-v2.0-F3): getDocumentOCR, applyParticipantNIK — aktifkan ketika billing on
  getDocumentOCR, applyParticipantNIK,
} from '../../utils/api'
import type {
  Document, DocumentStatus, DocumentSummary, PaginatedResponse, Participant,
} from '../../types'
import StatusBadge from '../../components/StatusBadge'
import RejectModal from '../../components/RejectModal'
import ProgressBar from '../../components/ProgressBar'
import ParticipantPicker from '../../components/ParticipantPicker'

const DOC_TYPE_LABELS: Record<string, string> = {
  passport: 'Paspor', ktp: 'KTP', rekening_koran: 'Rekening Koran',
  visa_support: 'Dokumen Visa', lainnya: 'Lainnya',
}
const DOC_TYPES = Object.keys(DOC_TYPE_LABELS)

export default function AdminDocumentsPage() {
  const [statusFilter, setStatusFilter] = useState<DocumentStatus | ''>('menunggu')
  const [typeFilter, setTypeFilter] = useState('')
  // Antrean dipersempit dengan memilih peserta lewat nama atau nomor WhatsApp.
  // Parameter yang dikirim tetap participant_id; hanya cara memperolehnya yang
  // berubah — tidak ada lagi UUID yang harus disalin dari basis data.
  const [participant, setParticipant] = useState<Participant | null>(null)
  const participantID = participant?.id ?? ''
  const [rejectDoc, setRejectDoc] = useState<Document | null>(null)
  const [ocrDoc, setOcrDoc] = useState<Document | null>(null)
  const [page, setPage] = useState(1)
  const qc = useQueryClient()

  const params = new URLSearchParams({ page: String(page), per_page: '20' })
  if (statusFilter) params.set('status', statusFilter)
  if (participantID) params.set('participant_id', participantID)

  // Paginated like every other list in the system. It used to fetch every
  // document ever uploaded in order to show twenty of them.
  const { data: docsPage, isLoading } = useQuery({
    queryKey: ['documents', statusFilter, participantID, page],
    queryFn: () =>
      api.get<PaginatedResponse<Document> & { summaries?: Record<string, DocumentSummary> }>(
        `/admin/documents?${params}`,
      ).then(r => r.data),
  })
  const docsData = docsPage?.data
  const meta = docsPage?.meta

  const reviewMutation = useMutation({
    mutationFn: ({ id, status, reason }: { id: string; status: 'disetujui' | 'ditolak'; reason?: string }) =>
      status === 'disetujui' ? approveDocument(id) : rejectDocument(id, reason ?? ''),
    onSuccess: (_, vars) => {
      qc.invalidateQueries({ queryKey: ['documents'] })
      toast.success(vars.status === 'disetujui' ? 'Dokumen disetujui' : 'Dokumen ditolak, notif terkirim')
      setRejectDoc(null)
    },
    onError: () => toast.error('Gagal mereview dokumen'),
  })

  // Client-side document_type filter (backend list endpoint only supports status/participant).
  const docs = useMemo(
    () => (docsData ?? []).filter((d) => !typeFilter || d.document_type === typeFilter),
    [docsData, typeFilter],
  )

  // Per-participant progress: "X dari Y dokumen disetujui" (§5.3).
  //
  // The server counts this, over ALL of a participant's documents, for every
  // participant on the page. Counting the rows on screen instead made the figure
  // agree with the active filter and with nothing else — someone two of three
  // documents through read "0 of 1" while the page was filtered to the one still
  // pending.
  const progress = docsPage?.summaries ?? {}

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold text-gray-800">Review Dokumen Peserta</h2>

      {/* Filters */}
      <div className="flex gap-3 flex-wrap items-start">
        <ParticipantPicker
          className="w-64"
          selected={participant}
          onChange={(p) => { setParticipant(p); setPage(1) }}
          placeholder="Saring ke satu peserta..."
        />
        <select
          className="border rounded-lg px-3 py-2 text-sm"
          value={typeFilter}
          onChange={(e) => setTypeFilter(e.target.value)}
        >
          <option value="">Semua Jenis</option>
          {DOC_TYPES.map((t) => <option key={t} value={t}>{DOC_TYPE_LABELS[t]}</option>)}
        </select>
        <div className="flex gap-2">
          {(['', 'menunggu', 'disetujui', 'ditolak'] as const).map((s) => (
            <button
              key={s}
              onClick={() => { setStatusFilter(s); setPage(1) }}
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
          {docs.map((doc) => {
            const prog = progress[doc.participant_id]
            return (
              <div key={doc.id} className="px-4 py-3 flex items-center gap-3">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-0.5">
                    <p className="font-medium text-sm text-gray-800">{doc.participant_name}</p>
                    <span className="text-xs text-gray-400">·</span>
                    <span className="text-xs text-gray-600">{DOC_TYPE_LABELS[doc.document_type] ?? doc.document_type}</span>
                    {prog && (
                      <span className="text-[11px] text-gray-400">({prog.disetujui}/{prog.total} disetujui)</span>
                    )}
                  </div>
                  <p className="text-xs text-gray-500 truncate">{doc.file_name}</p>
                  {doc.rejection_reason && (
                    <p className="text-xs text-red-500 mt-0.5">Alasan: {doc.rejection_reason}</p>
                  )}
                </div>

                <StatusBadge status={doc.status} className="shrink-0" />

                <button
                  onClick={() =>
                    openSignedFile('document', doc.id, doc.file_path).catch(() =>
                      toast.error('Gagal membuka dokumen'),
                    )
                  }
                  className="px-2 py-1 text-xs border rounded hover:bg-gray-50 shrink-0"
                >
                  Lihat
                </button>

                {(doc.document_type === 'passport' || doc.document_type === 'ktp') && (
                  <button
                    onClick={() => setOcrDoc(doc)}
                    className="flex items-center gap-1 px-2 py-1 text-xs border rounded text-blue-600 hover:bg-blue-50 shrink-0"
                    title="Lihat hasil OCR"
                  >
                    <ScanLine size={13} /> OCR
                  </button>
                )}

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
            )
          })}
        </div>
      )}

      {meta && meta.total_pages > 1 && (
        <div className="flex justify-center gap-2">
          <button disabled={page <= 1} onClick={() => setPage(p => p - 1)}
            className="px-3 py-1.5 border rounded text-sm disabled:opacity-40">← Sebelumnya</button>
          <span className="px-3 py-1.5 text-sm">Hal {page}/{meta.total_pages}</span>
          <button disabled={page >= meta.total_pages} onClick={() => setPage(p => p + 1)}
            className="px-3 py-1.5 border rounded text-sm disabled:opacity-40">Berikutnya →</button>
        </div>
      )}

      {/* Reject modal */}
      <RejectModal
        open={!!rejectDoc}
        title="Tolak Dokumen"
        description={
          rejectDoc
            ? `Dokumen ${DOC_TYPE_LABELS[rejectDoc.document_type] ?? rejectDoc.document_type} dari ${rejectDoc.participant_name}`
            : undefined
        }
        loading={reviewMutation.isPending}
        onCancel={() => setRejectDoc(null)}
        onSubmit={(reason) => rejectDoc && reviewMutation.mutate({ id: rejectDoc.id, status: 'ditolak', reason })}
      />

      {ocrDoc && <OCRModal doc={ocrDoc} onClose={() => setOcrDoc(null)} />}
    </div>
  )
}

// TODO(ocr-v2.0-F3): OCRModal — aktifkan ketika GCP Vision billing on.
function OCRModal({ doc, onClose }: { doc: Document; onClose: () => void }) {
  const { data: ocr, isLoading, error } = useQuery({
    queryKey: ['ocr-result', doc.id],
    queryFn: () => getDocumentOCR(doc.id),
    retry: false,
  })

  const applyMut = useMutation({
    mutationFn: (nik: string) => applyParticipantNIK(doc.participant_id, nik),
    onSuccess: () => toast.success('Data berhasil diterapkan ke profil peserta'),
    onError: () => toast.error('Gagal menerapkan data'),
  })

  const conf = ocr?.confidence ?? 0
  const data = ocr?.extracted_data
  const statusLabel = () => {
    if (!ocr) return ''
    if (conf >= 0.85 && ocr.validation_passed) return '✅ Valid — Siap Disetujui'
    if (conf >= 0.85 && !ocr.validation_passed) return '⚠️ Data Tidak Valid'
    return '⚠️ Perlu Review Manual'
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-[60] p-4">
      <div className="bg-white rounded-xl w-full max-w-sm p-5 space-y-3">
        <div className="flex items-center justify-between">
          <h3 className="font-semibold flex items-center gap-1.5"><ScanLine size={16} className="text-blue-600" /> Hasil Analisis OCR</h3>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600">✕</button>
        </div>

        {isLoading ? (
          <p className="text-sm text-gray-400 py-6 text-center">🔄 Sedang dianalisis…</p>
        ) : error || !ocr ? (
          <p className="text-sm text-gray-500 bg-gray-50 rounded-lg p-3">
            Hasil OCR belum tersedia. Dokumen mungkin masih diproses, atau OCR tidak aktif. Admin tetap bisa review manual.
          </p>
        ) : (
          <>
            <ProgressBar label="Akurasi" value={Math.round(conf * 100)} color={conf >= 0.85 ? 'bg-emerald-500' : 'bg-yellow-500'} />
            <table className="w-full text-sm">
              <tbody>
                <Row k="Nama" v={data?.name} />
                <Row k={doc.document_type === 'ktp' ? 'NIK' : 'No. Paspor'} v={data?.document_number} />
                <Row k="Tgl Lahir" v={data?.birth_date} />
                {doc.document_type === 'passport' && <Row k="Berlaku s/d" v={data?.expiry_date} />}
                {doc.document_type === 'passport' && <Row k="Kewarganegaraan" v={data?.nationality} />}
              </tbody>
            </table>

            <div className={`rounded-lg p-3 text-sm ${ocr.validation_passed ? 'bg-emerald-50 text-emerald-700' : 'bg-yellow-50 text-yellow-700'}`}>
              <p className="font-medium">{statusLabel()}</p>
              {ocr.validation_notes && <p className="text-xs mt-0.5">{ocr.validation_notes}</p>}
            </div>

            {doc.document_type === 'ktp' && ocr.validation_passed && data?.document_number && (
              <button
                onClick={() => applyMut.mutate(data.document_number!)}
                disabled={applyMut.isPending}
                className="w-full py-2 bg-emerald-600 text-white rounded-lg text-sm font-medium disabled:opacity-50"
              >
                {applyMut.isPending ? 'Menerapkan…' : 'Terapkan ke Data Peserta'}
              </button>
            )}

            <p className="text-[11px] text-gray-400">
              Hasil OCR adalah bantuan otomatis. Keputusan final tetap pada reviewer.
            </p>
          </>
        )}
      </div>
    </div>
  )
}

function Row({ k, v }: { k: string; v?: string }) {
  return (
    <tr className="border-b last:border-0">
      <td className="py-1.5 text-gray-500 w-1/3">{k}</td>
      <td className="py-1.5 font-medium text-gray-800">{v || '—'}</td>
    </tr>
  )
}
