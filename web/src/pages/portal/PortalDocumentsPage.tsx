import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Upload, CheckCircle, XCircle, Clock, AlertCircle } from 'lucide-react'
import toast from 'react-hot-toast'
import { portalApi } from '../../utils/api'
import FileUpload from '../../components/FileUpload'
import type { Document, DocumentType, CountryRequirement, UploadDocumentRequest } from '../../types'

const STATUS_ICON: Record<string, React.ReactNode> = {
  menunggu: <Clock className="text-yellow-500" size={16} />,
  disetujui: <CheckCircle className="text-green-500" size={16} />,
  ditolak: <XCircle className="text-red-500" size={16} />,
}

const DOC_LABELS: Record<string, string> = {
  passport: 'Paspor',
  ktp: 'KTP',
  rekening_koran: 'Rekening Koran (3 bulan)',
  visa_support: 'Dokumen Pendukung Visa',
  lainnya: 'Dokumen Lainnya',
}

const DEFAULT_DOC_TYPES: DocumentType[] = ['passport', 'ktp', 'rekening_koran', 'visa_support', 'lainnya']

export default function PortalDocumentsPage() {
  const [showUpload, setShowUpload] = useState(false)
  const [uploadForm, setUploadForm] = useState<UploadDocumentRequest>({
    document_type: 'passport', file_path: '', file_name: '',
  })
  const qc = useQueryClient()

  const { data: docs } = useQuery({
    queryKey: ['portal-documents'],
    queryFn: () =>
      portalApi.get<{ success: boolean; data: Document[] }>('/portal/documents')
        .then(r => r.data.data ?? []),
  })

  const { data: countryReqs } = useQuery({
    queryKey: ['portal-country-requirements'],
    queryFn: () =>
      portalApi.get<{ success: boolean; data: CountryRequirement[] }>('/portal/documents/requirements')
        .then(r => r.data.data ?? []),
  })

  const uploadMutation = useMutation({
    mutationFn: (req: UploadDocumentRequest) => portalApi.post('/portal/documents', req),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['portal-documents'] })
      toast.success('Dokumen berhasil diupload, menunggu review admin')
      setShowUpload(false)
      setUploadForm({ document_type: 'passport', file_path: '', file_name: '' })
    },
    onError: () => toast.error('Gagal upload dokumen'),
  })

  const docsList = docs ?? []

  // Build required types: from country requirements if available, else default
  const requiredTypes: { type: string; label: string; required: boolean; desc: string }[] =
    countryReqs && countryReqs.length > 0
      ? countryReqs.map(r => ({
          type: r.document_type,
          label: DOC_LABELS[r.document_type] ?? r.document_type,
          required: r.is_required,
          desc: r.description,
        }))
      : DEFAULT_DOC_TYPES.map(t => ({
          type: t,
          label: DOC_LABELS[t],
          required: t !== 'lainnya',
          desc: '',
        }))

  function getLatestDoc(type: string) {
    return docsList.filter(d => d.document_type === type).at(-1)
  }

  const uploadedRequired = requiredTypes
    .filter(t => t.required)
    .filter(t => {
      const d = getLatestDoc(t.type)
      return d?.status === 'disetujui'
    }).length

  const totalRequired = requiredTypes.filter(t => t.required).length

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="font-semibold text-gray-800">Dokumen Saya</h2>
        <button
          onClick={() => setShowUpload(true)}
          className="flex items-center gap-1.5 px-3 py-2 bg-emerald-600 text-white text-sm rounded-lg hover:bg-emerald-700"
        >
          <Upload size={14} /> Upload
        </button>
      </div>

      {/* Progress */}
      <div className="bg-white rounded-xl border p-4">
        <div className="flex justify-between text-xs text-gray-500 mb-1.5">
          <span>Kelengkapan dokumen wajib</span>
          <span className="font-semibold text-gray-700">{uploadedRequired}/{totalRequired}</span>
        </div>
        <div className="h-2 bg-gray-100 rounded-full overflow-hidden">
          <div
            className="h-full bg-emerald-500 rounded-full transition-all"
            style={{ width: totalRequired > 0 ? `${uploadedRequired / totalRequired * 100}%` : '0%' }}
          />
        </div>
        <p className="text-xs text-gray-400 mt-1.5">
          Lengkapi semua dokumen wajib sebelum H-14 keberangkatan
        </p>
      </div>

      {/* Document checklist */}
      <div className="space-y-2">
        {requiredTypes.map(({ type, label, required, desc }) => {
          const latest = getLatestDoc(type)
          return (
            <div key={type} className="bg-white rounded-xl border p-4">
              <div className="flex items-start justify-between gap-2 mb-1">
                <div className="flex-1">
                  <div className="flex items-center gap-1.5">
                    <p className="font-medium text-sm text-gray-800">{label}</p>
                    {required && (
                      <span className="text-xs text-red-500 font-medium">*wajib</span>
                    )}
                  </div>
                  {desc && <p className="text-xs text-gray-500 mt-0.5">{desc}</p>}
                </div>
                {latest
                  ? STATUS_ICON[latest.status] ?? <AlertCircle className="text-gray-300" size={16} />
                  : <AlertCircle className="text-gray-300" size={16} />
                }
              </div>

              {latest ? (
                <div className="mt-2">
                  <div className="flex items-center gap-2 text-xs">
                    <span className="text-gray-500 truncate">{latest.file_name}</span>
                  </div>
                  {latest.status === 'ditolak' && latest.rejection_reason && (
                    <div className="mt-1.5 bg-red-50 rounded px-2 py-1.5 text-xs text-red-600">
                      ✗ Ditolak: {latest.rejection_reason}
                    </div>
                  )}
                  {latest.status === 'menunggu' && (
                    <p className="text-xs text-yellow-600 mt-1">⏳ Menunggu review admin</p>
                  )}
                  {latest.status === 'disetujui' && (
                    <p className="text-xs text-green-600 mt-1">✓ Disetujui</p>
                  )}
                  {(latest.status === 'ditolak') && (
                    <button
                      onClick={() => {
                        setUploadForm({ document_type: type as DocumentType, file_path: '', file_name: '' })
                        setShowUpload(true)
                      }}
                      className="mt-2 text-xs text-emerald-600 underline"
                    >
                      Upload ulang
                    </button>
                  )}
                </div>
              ) : (
                <button
                  onClick={() => {
                    setUploadForm({ document_type: type as DocumentType, file_path: '', file_name: '' })
                    setShowUpload(true)
                  }}
                  className="mt-2 text-xs text-emerald-600 flex items-center gap-1 hover:underline"
                >
                  <Upload size={11} /> Upload dokumen ini
                </button>
              )}
            </div>
          )
        })}
      </div>

      {/* Tips */}
      <div className="bg-blue-50 rounded-xl p-4 text-xs text-blue-700 space-y-1">
        <p className="font-medium">📋 Ketentuan Upload</p>
        <p>• Format: JPG, PNG, atau PDF (maksimal 5MB per file)</p>
        <p>• Foto harus jelas, tidak buram, dan semua sudut terlihat</p>
        <p>• Paspor: upload semua halaman yang berisi data diri</p>
        <p>• Rekening koran: minimal 3 bulan terakhir</p>
      </div>

      {/* Upload modal */}
      {showUpload && (
        <div className="fixed inset-0 bg-black/50 flex items-end md:items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl w-full max-w-sm p-6 space-y-4">
            <div className="flex justify-between items-center">
              <h3 className="font-semibold">Upload Dokumen</h3>
              <button onClick={() => setShowUpload(false)} className="text-gray-400">✕</button>
            </div>
            <div className="space-y-3">
              <div>
                <label className="text-xs font-medium text-gray-600 block mb-1">Jenis Dokumen</label>
                <select
                  className="w-full border rounded-lg px-3 py-2 text-sm"
                  value={uploadForm.document_type}
                  onChange={e => setUploadForm({ ...uploadForm, document_type: e.target.value as DocumentType })}
                >
                  {DEFAULT_DOC_TYPES.map(t => (
                    <option key={t} value={t}>{DOC_LABELS[t]}</option>
                  ))}
                </select>
              </div>

              {/* §16.2 Supabase Storage upload (fallback URL manual) */}
              <FileUpload
                endpoint="/portal/upload/document"
                placeholder="Pilih file paspor/KTP/dokumen..."
                onUploaded={(path, name) => {
                  setUploadForm({ ...uploadForm, file_path: path, file_name: name })
                }}
              />
            </div>
            <button
              onClick={() => uploadMutation.mutate(uploadForm)}
              disabled={!uploadForm.file_path || !uploadForm.file_name || uploadMutation.isPending}
              className="w-full py-2.5 bg-emerald-600 text-white rounded-lg text-sm font-medium disabled:opacity-50"
            >
              {uploadMutation.isPending ? 'Mengirim...' : 'Submit Dokumen'}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
