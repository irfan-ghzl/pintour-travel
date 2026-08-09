import { useState } from 'react'
import { XCircle } from 'lucide-react'

// RejectModal collects a rejection reason (min 10 chars) for a destructive
// review action (§5.8). Used for document/payment rejection flows.
export default function RejectModal({
  open,
  title = 'Tolak',
  description,
  placeholder = 'Tuliskan alasan penolakan...',
  loading = false,
  onCancel,
  onSubmit,
}: {
  open: boolean
  title?: string
  description?: string
  placeholder?: string
  loading?: boolean
  onCancel: () => void
  onSubmit: (reason: string) => void
}) {
  const [reason, setReason] = useState('')
  if (!open) return null

  const valid = reason.trim().length >= 10

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-[60] p-4">
      <div className="bg-white rounded-xl w-full max-w-sm p-5 space-y-3">
        <div className="flex items-center gap-2 text-red-600">
          <XCircle size={18} />
          <h3 className="font-semibold">{title}</h3>
        </div>
        {description && <p className="text-sm text-gray-500">{description}</p>}
        <textarea
          rows={3}
          autoFocus
          className="w-full border rounded-lg px-3 py-2 text-sm resize-none focus:ring-1 focus:ring-red-400"
          placeholder={placeholder}
          value={reason}
          onChange={(e) => setReason(e.target.value)}
        />
        <p className="text-xs text-gray-400">{reason.trim().length}/10 karakter minimum</p>
        <div className="flex gap-2 justify-end">
          <button onClick={onCancel} className="px-3 py-1.5 border rounded-lg text-sm text-gray-600">
            Batal
          </button>
          <button
            onClick={() => onSubmit(reason.trim())}
            disabled={!valid || loading}
            className="px-3 py-1.5 bg-red-600 text-white rounded-lg text-sm font-medium disabled:opacity-50"
          >
            {loading ? 'Memproses…' : title}
          </button>
        </div>
      </div>
    </div>
  )
}
