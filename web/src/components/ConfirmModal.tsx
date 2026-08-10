import { AlertTriangle } from 'lucide-react'
import clsx from 'clsx'

// ConfirmModal confirms a destructive or important action (§5.8).
export default function ConfirmModal({
  open,
  title,
  message,
  confirmLabel = 'Konfirmasi',
  cancelLabel = 'Batal',
  danger = false,
  loading = false,
  onCancel,
  onConfirm,
}: {
  open: boolean
  title: string
  message?: string
  confirmLabel?: string
  cancelLabel?: string
  danger?: boolean
  loading?: boolean
  onCancel: () => void
  onConfirm: () => void
}) {
  if (!open) return null
  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-[60] p-4">
      <div className="bg-white rounded-xl w-full max-w-sm p-5 space-y-3">
        <div className={clsx('flex items-center gap-2', danger ? 'text-red-600' : 'text-emerald-600')}>
          <AlertTriangle size={18} />
          <h3 className="font-semibold text-gray-800">{title}</h3>
        </div>
        {message && <p className="text-sm text-gray-500">{message}</p>}
        <div className="flex gap-2 justify-end pt-1">
          <button onClick={onCancel} className="px-3 py-1.5 border rounded-lg text-sm text-gray-600">
            {cancelLabel}
          </button>
          <button
            onClick={onConfirm}
            disabled={loading}
            className={clsx(
              'px-3 py-1.5 rounded-lg text-sm font-medium text-white disabled:opacity-50',
              danger ? 'bg-red-600 hover:bg-red-700' : 'bg-emerald-600 hover:bg-emerald-700',
            )}
          >
            {loading ? 'Memproses…' : confirmLabel}
          </button>
        </div>
      </div>
    </div>
  )
}
