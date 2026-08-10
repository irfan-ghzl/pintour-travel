import clsx from 'clsx'

// StatusBadge renders a colored pill for a status value (prompt §5.8).
// Colors map to the actual backend status vocabularies (invoices, documents,
// leads, payment proofs, batches) — code-wins, no spec-only statuses.

const STATUS_STYLES: Record<string, string> = {
  // invoices
  diterbitkan: 'bg-gray-100 text-gray-700',
  menunggu_bayar: 'bg-yellow-100 text-yellow-800',
  dibayar: 'bg-blue-100 text-blue-800',
  lunas: 'bg-emerald-100 text-emerald-800',
  // leads
  baru: 'bg-blue-100 text-blue-800',
  dihubungi: 'bg-yellow-100 text-yellow-800',
  konsultasi: 'bg-orange-100 text-orange-800',
  deal: 'bg-emerald-100 text-emerald-800',
  tidak_deal: 'bg-gray-200 text-gray-600',
  peserta: 'bg-emerald-100 text-emerald-800',
  // documents / proofs
  menunggu: 'bg-yellow-100 text-yellow-800',
  disetujui: 'bg-emerald-100 text-emerald-800',
  ditolak: 'bg-red-100 text-red-800',
  belum_upload: 'bg-gray-100 text-gray-500',
  // batches
  tersedia: 'bg-emerald-100 text-emerald-800',
  penuh: 'bg-orange-100 text-orange-800',
  ditutup: 'bg-gray-200 text-gray-600',
}

const LABELS: Record<string, string> = {
  menunggu_bayar: 'Menunggu Bayar',
  tidak_deal: 'Tidak Deal',
  belum_upload: 'Belum Upload',
}

export default function StatusBadge({ status, className }: { status: string; className?: string }) {
  const style = STATUS_STYLES[status] ?? 'bg-gray-100 text-gray-700'
  const label = LABELS[status] ?? status.replace(/_/g, ' ')
  return (
    <span
      className={clsx(
        'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium capitalize',
        style,
        className,
      )}
    >
      {label}
    </span>
  )
}
