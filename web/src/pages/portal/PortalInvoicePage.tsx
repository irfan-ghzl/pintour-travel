import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { FileText, Upload, CheckCircle, Clock, Download, CreditCard } from 'lucide-react'
import toast from 'react-hot-toast'
import { portalApi } from '../../utils/api'
import FileUpload from '../../components/FileUpload'
import type { Invoice, InvoiceStatus, UploadProofRequest } from '../../types'

const STATUS_COLORS: Record<InvoiceStatus, string> = {
  diterbitkan: 'bg-blue-100 text-blue-700',
  menunggu_bayar: 'bg-yellow-100 text-yellow-700',
  menunggu_konfirmasi_gateway: 'bg-amber-100 text-amber-700',
  dibayar: 'bg-purple-100 text-purple-700',
  lunas: 'bg-green-100 text-green-700',
}

const STATUS_LABELS: Record<InvoiceStatus, string> = {
  diterbitkan: 'Diterbitkan',
  menunggu_bayar: 'Menunggu Bayar',
  menunggu_konfirmasi_gateway: 'Menunggu Konfirmasi',
  dibayar: 'Dibayar',
  lunas: 'Lunas ✓',
}

// A status the schema gains before this page learns about it must not blank the
// participant's invoice page — they still need to see the amount and the due
// date. Lookups go through these so an unknown value degrades to plain text.
const UNKNOWN_STATUS_COLOR = 'bg-gray-100 text-gray-700'

function statusColor(status: InvoiceStatus): string {
  return STATUS_COLORS[status] ?? UNKNOWN_STATUS_COLOR
}

function statusLabel(status: InvoiceStatus): string {
  return STATUS_LABELS[status] ?? String(status).replace(/_/g, ' ')
}

export default function PortalInvoicePage() {
  const [showUpload, setShowUpload] = useState<string | null>(null) // invoice id
  const [uploadForm, setUploadForm] = useState<UploadProofRequest>({
    file_path: '', amount_claimed: 0, notes: '',
  })
  const qc = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: ['portal-invoices'],
    queryFn: () =>
      portalApi.get<{ success: boolean; data: Invoice[] }>('/portal/invoices')
        .then(r => r.data.data),
  })

  const uploadMutation = useMutation({
    mutationFn: ({ invoiceId, req }: { invoiceId: string; req: UploadProofRequest }) =>
      portalApi.post(`/portal/invoices/${invoiceId}/proofs`, req),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['portal-invoices'] })
      toast.success('Bukti transfer dikirim. Admin akan verifikasi dalam 1×24 jam.')
      setShowUpload(null)
      setUploadForm({ file_path: '', amount_claimed: 0, notes: '' })
    },
    onError: () => toast.error('Gagal mengirim bukti transfer'),
  })

  const invoices = data ?? []

  if (isLoading) return (
    <div className="space-y-4 animate-pulse">
      {[1, 2].map(i => <div key={i} className="h-36 bg-gray-200 rounded-xl" />)}
    </div>
  )

  return (
    <div className="space-y-4">
      <h2 className="font-semibold text-gray-800">Invoice Saya</h2>

      {invoices.length === 0 ? (
        <div className="bg-white rounded-xl border p-10 text-center text-gray-400">
          <FileText className="mx-auto mb-3 opacity-40" size={36} />
          <p>Belum ada invoice</p>
          <p className="text-xs mt-1">Invoice akan muncul setelah admin menerbitkan tagihan</p>
        </div>
      ) : invoices.map((inv) => (
        <div key={inv.id} className="bg-white rounded-xl border overflow-hidden">
          {/* Header */}
          <div className="flex items-center justify-between px-4 py-3 border-b bg-gray-50">
            <div>
              <p className="font-mono text-sm font-semibold text-gray-700">{inv.invoice_number}</p>
              <p className="text-xs text-gray-500">{inv.package_name}</p>
            </div>
            <span className={`px-2.5 py-1 rounded-full text-xs font-medium ${statusColor(inv.status)}`}>
              {statusLabel(inv.status)}
            </span>
          </div>

          {/* Body */}
          <div className="p-4 space-y-3">
            <div className="grid grid-cols-2 gap-3 text-sm">
              <div>
                <p className="text-xs text-gray-400">Total Tagihan</p>
                <p className="font-bold text-gray-800 text-base">
                  Rp {inv.amount.toLocaleString('id-ID')}
                </p>
              </div>
              <div>
                <p className="text-xs text-gray-400">Jatuh Tempo</p>
                <p className="font-medium text-gray-700">
                  {new Date(inv.due_date).toLocaleDateString('id-ID', {
                    day: 'numeric', month: 'long', year: 'numeric',
                  })}
                </p>
              </div>
            </div>

            {/* Status indicator */}
            <StatusInfo status={inv.status} />

            {/* Actions */}
            <div className="flex flex-col gap-2 pt-1">
              <div className="flex gap-2">
                {/* Download PDF */}
                <a
                  href={`/api/v1/portal/invoices/${inv.id}/pdf`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-1.5 px-3 py-2 border rounded-lg text-sm text-gray-600 hover:bg-gray-50"
                >
                  <Download size={14} /> Unduh PDF
                </a>

                {/* Bayar online via Midtrans — primary action */}
                {(inv.status === 'diterbitkan' || inv.status === 'menunggu_bayar') && (
                  <Link
                    to={`/portal/payment/${inv.id}`}
                    className="flex items-center gap-1.5 px-3 py-2 bg-emerald-600 text-white text-sm rounded-lg hover:bg-emerald-700"
                  >
                    <CreditCard size={14} /> Bayar Online
                  </Link>
                )}
              </div>

              {/* Upload bukti transfer — opsional untuk transfer manual */}
              {(inv.status === 'diterbitkan' || inv.status === 'menunggu_bayar') && (
                <button
                  onClick={() => setShowUpload(inv.id)}
                  className="flex items-center gap-1 text-xs text-gray-400 hover:text-gray-600 w-fit"
                >
                  <Upload size={11} />
                  Sudah transfer manual? Upload bukti di sini
                </button>
              )}
            </div>
          </div>
        </div>
      ))}

      {/* Upload modal */}
      {showUpload && (
        <div className="fixed inset-0 bg-black/50 flex items-end md:items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl w-full max-w-sm p-6 space-y-4">
            <div className="flex justify-between items-center">
              <h3 className="font-semibold">Upload Bukti Transfer</h3>
              <button onClick={() => setShowUpload(null)} className="text-gray-400 hover:text-gray-600">✕</button>
            </div>

            <div className="bg-amber-50 rounded-lg p-3 text-xs text-amber-700">
              Khusus untuk transfer bank manual. Jika sudah bayar via Midtrans, tutup form ini — verifikasi otomatis dari sistem.
            </div>

            <div className="space-y-3">
              <div>
                <label className="text-xs font-medium text-gray-600 block mb-1">
                  Foto Bukti Transfer *
                </label>
                {/* §16.2 Supabase Storage upload */}
                <FileUpload
                  endpoint="/portal/upload/payment-proof"
                  placeholder="Pilih foto/screenshot bukti transfer..."
                  accept=".jpg,.jpeg,.png,.pdf"
                  onUploaded={(path) => setUploadForm({ ...uploadForm, file_path: path })}
                />
              </div>
              <div>
                <label className="text-xs font-medium text-gray-600 block mb-1">
                  Jumlah yang Ditransfer (Rp) *
                </label>
                <input
                  type="number"
                  className="w-full border rounded-lg px-3 py-2 text-sm"
                  placeholder="0"
                  value={uploadForm.amount_claimed || ''}
                  onChange={e => setUploadForm({ ...uploadForm, amount_claimed: Number(e.target.value) })}
                />
              </div>
              <div>
                <label className="text-xs font-medium text-gray-600 block mb-1">
                  Catatan (opsional)
                </label>
                <textarea
                  rows={2}
                  className="w-full border rounded-lg px-3 py-2 text-sm resize-none"
                  placeholder="Contoh: Transfer BCA atas nama Budi..."
                  value={uploadForm.notes}
                  onChange={e => setUploadForm({ ...uploadForm, notes: e.target.value })}
                />
              </div>
            </div>

            <button
              onClick={() => uploadMutation.mutate({ invoiceId: showUpload!, req: uploadForm })}
              disabled={!uploadForm.file_path || !uploadForm.amount_claimed || uploadMutation.isPending}
              className="w-full py-2.5 bg-emerald-600 text-white rounded-lg text-sm font-medium disabled:opacity-50"
            >
              {uploadMutation.isPending ? 'Mengirim...' : 'Kirim Bukti Transfer'}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

function StatusInfo({ status }: { status: InvoiceStatus }) {
  const info: Record<InvoiceStatus, { icon: React.ReactNode; text: string; color: string }> = {
    diterbitkan: {
      icon: <Clock size={14} />,
      text: 'Klik "Bayar Online" untuk pembayaran otomatis via Midtrans',
      color: 'text-blue-600 bg-blue-50',
    },
    menunggu_bayar: {
      icon: <Clock size={14} />,
      text: 'Silakan selesaikan pembayaran. Verifikasi otomatis setelah pembayaran berhasil.',
      color: 'text-yellow-600 bg-yellow-50',
    },
    menunggu_konfirmasi_gateway: {
      icon: <Clock size={14} />,
      text: 'Pembayaran Anda sedang menunggu konfirmasi dari payment gateway.',
      color: 'text-amber-600 bg-amber-50',
    },
    dibayar: {
      icon: <CheckCircle size={14} />,
      text: 'Pembayaran dikonfirmasi. Portal peserta aktif',
      color: 'text-purple-600 bg-purple-50',
    },
    lunas: {
      icon: <CheckCircle size={14} />,
      text: 'Pembayaran lunas. Selamat bergabung!',
      color: 'text-green-600 bg-green-50',
    },
  }
  // Same reason as statusColor above: a status this map has not caught up with
  // shows a neutral line instead of crashing the page.
  const { icon, text, color } = info[status] ?? {
    icon: <Clock size={14} />,
    text: 'Status pembayaran sedang diproses.',
    color: 'text-gray-600 bg-gray-50',
  }
  return (
    <div className={`flex items-center gap-2 rounded-lg px-3 py-2 text-xs ${color}`}>
      {icon}
      <span>{text}</span>
    </div>
  )
}
