import { useState, useRef } from 'react'
import { Upload, X, CheckCircle, AlertCircle } from 'lucide-react'
import { portalApi } from '../utils/api'

interface Props {
  /** Endpoint relatif ke /api/v1, contoh: "/portal/upload/document" */
  endpoint: string
  /** Callback dengan path yang dikembalikan setelah upload sukses */
  onUploaded: (path: string, fileName: string) => void
  /** Pesan placeholder */
  placeholder?: string
  /** Tipe file yang diterima */
  accept?: string
  /** Ukuran maksimum (default 5MB) */
  maxSize?: number
}

/**
 * FileUpload menggunakan Supabase Storage via backend endpoint.
 * Auto-fallback ke text input jika storage tidak terkonfigurasi (503).
 */
export default function FileUpload({
  endpoint,
  onUploaded,
  placeholder = 'Pilih file...',
  accept = '.pdf,.jpg,.jpeg,.png',
  maxSize = 5 * 1024 * 1024,
}: Props) {
  const [uploading, setUploading] = useState(false)
  const [error, setError] = useState('')
  const [uploaded, setUploaded] = useState<{ path: string; name: string } | null>(null)
  const [fallback, setFallback] = useState(false)
  const [fallbackUrl, setFallbackUrl] = useState('')
  const [fallbackName, setFallbackName] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)

  async function handleFile(file: File) {
    setError('')
    if (file.size > maxSize) {
      setError(`Ukuran file melebihi ${(maxSize / 1024 / 1024).toFixed(0)}MB`)
      return
    }
    setUploading(true)
    const formData = new FormData()
    formData.append('file', file)
    try {
      const res = await portalApi.post<{ success: boolean; data: { path: string; file_name: string } }>(
        endpoint,
        formData,
        { headers: { 'Content-Type': 'multipart/form-data' } },
      )
      const path = res.data.data.path
      const name = res.data.data.file_name ?? file.name
      setUploaded({ path, name })
      onUploaded(path, name)
    } catch (e: any) {
      if (e.response?.status === 503) {
        // Storage tidak terkonfigurasi — fallback ke URL manual
        setFallback(true)
      } else {
        setError(e.response?.data?.message ?? 'Upload gagal')
      }
    } finally {
      setUploading(false)
    }
  }

  function handleFallbackSubmit() {
    if (!fallbackUrl || !fallbackName) {
      setError('URL dan nama file harus diisi')
      return
    }
    setUploaded({ path: fallbackUrl, name: fallbackName })
    onUploaded(fallbackUrl, fallbackName)
  }

  function reset() {
    setUploaded(null)
    setFallbackUrl('')
    setFallbackName('')
    if (inputRef.current) inputRef.current.value = ''
  }

  if (uploaded) {
    return (
      <div className="flex items-center justify-between gap-2 bg-emerald-50 border border-emerald-200 rounded-lg px-3 py-2">
        <div className="flex items-center gap-2 min-w-0">
          <CheckCircle className="text-emerald-600 shrink-0" size={16} />
          <span className="text-xs text-emerald-700 truncate">{uploaded.name}</span>
        </div>
        <button onClick={reset} className="text-emerald-600 hover:text-emerald-800 shrink-0">
          <X size={14} />
        </button>
      </div>
    )
  }

  if (fallback) {
    return (
      <div className="space-y-2 bg-yellow-50 border border-yellow-200 rounded-lg p-3">
        <p className="text-xs text-yellow-700">
          ⚠️ Storage belum terkonfigurasi. Silakan upload file ke Google Drive / Cloudinary
          dan paste link di sini.
        </p>
        <input
          className="w-full border rounded px-2 py-1.5 text-xs"
          placeholder="URL file..."
          value={fallbackUrl}
          onChange={(e) => setFallbackUrl(e.target.value)}
        />
        <input
          className="w-full border rounded px-2 py-1.5 text-xs"
          placeholder="Nama file..."
          value={fallbackName}
          onChange={(e) => setFallbackName(e.target.value)}
        />
        <button
          onClick={handleFallbackSubmit}
          className="px-3 py-1.5 bg-emerald-600 text-white text-xs rounded"
        >
          Gunakan URL
        </button>
      </div>
    )
  }

  return (
    <div className="space-y-1">
      <label className="flex items-center justify-center gap-2 border-2 border-dashed border-gray-300 rounded-lg px-3 py-4 cursor-pointer hover:border-emerald-400 hover:bg-emerald-50/50 transition-colors">
        <Upload className="text-gray-400" size={18} />
        <span className="text-sm text-gray-500">
          {uploading ? 'Mengupload...' : placeholder}
        </span>
        <input
          ref={inputRef}
          type="file"
          className="hidden"
          accept={accept}
          disabled={uploading}
          onChange={(e) => {
            const file = e.target.files?.[0]
            if (file) handleFile(file)
          }}
        />
      </label>
      {error && (
        <p className="flex items-center gap-1 text-xs text-red-600">
          <AlertCircle size={12} /> {error}
        </p>
      )}
      <p className="text-xs text-gray-400">
        Maks {(maxSize / 1024 / 1024).toFixed(0)}MB · Format: {accept}
      </p>
    </div>
  )
}
