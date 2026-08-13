import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Plus, CheckCircle, Eye, XCircle, FileText } from 'lucide-react'
import toast from 'react-hot-toast'
import api, { openSignedFile } from '../../utils/api'
import { formatDate } from '../../utils/date'
import ParticipantPicker from '../../components/ParticipantPicker'
import type {
  Invoice, InvoiceStatus, PaginatedResponse, CreateInvoiceRequest, PaymentProof, Participant,
} from '../../types'

const STATUS_COLORS: Record<InvoiceStatus, string> = {
  diterbitkan: 'bg-blue-100 text-blue-700',
  menunggu_bayar: 'bg-yellow-100 text-yellow-700',
  menunggu_konfirmasi_gateway: 'bg-amber-100 text-amber-700',
  dibayar: 'bg-purple-100 text-purple-700',
  lunas: 'bg-emerald-100 text-emerald-700',
}

export default function AdminInvoicesPage() {
  const [statusFilter, setStatusFilter] = useState<InvoiceStatus | ''>('')
  const [page, setPage] = useState(1)
  const [showCreate, setShowCreate] = useState(false)
  const [selectedInvoice, setSelectedInvoice] = useState<Invoice | null>(null)
  // Invoice utama sudah terbit otomatis saat konversi lead. Formulir ini untuk
  // tagihan tambahan di luar paket — upgrade kamar, penalti, layanan ekstra.
  //
  // Batch tidak lagi ditanyakan: setiap peserta sudah terikat pada satu batch,
  // jadi menurunkannya dari peserta yang dipilih menghapus seluruh kelas
  // kesalahan "peserta A ditagih pada batch B".
  const [invoiceFor, setInvoiceFor] = useState<Participant | null>(null)
  const [form, setForm] = useState<CreateInvoiceRequest>({
    participant_id: '', batch_id: '', amount: 0, due_date: '', notes: '',
  })
  const qc = useQueryClient()

  function pickInvoiceParticipant(p: Participant | null) {
    setInvoiceFor(p)
    setForm((f) => ({ ...f, participant_id: p?.id ?? '', batch_id: p?.batch_id ?? '' }))
  }

  function closeCreate() {
    setShowCreate(false)
    setInvoiceFor(null)
    setForm({ participant_id: '', batch_id: '', amount: 0, due_date: '', notes: '' })
  }

  const params = new URLSearchParams({ page: String(page), per_page: '20' })
  if (statusFilter) params.set('status', statusFilter)

  const { data, isLoading } = useQuery({
    queryKey: ['invoices', statusFilter, page],
    queryFn: () => api.get<PaginatedResponse<Invoice>>(`/admin/invoices?${params}`).then(r => r.data),
  })

  const createMutation = useMutation({
    mutationFn: (req: CreateInvoiceRequest) => api.post('/admin/invoices', req),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['invoices'] })
      toast.success('Invoice berhasil diterbitkan dan dikirim via WA')
      closeCreate()
    },
    onError: (e: any) => toast.error(e.response?.data?.message ?? 'Gagal membuat invoice'),
  })

  const confirmMutation = useMutation({
    mutationFn: (id: string) => api.post(`/admin/invoices/${id}/confirm`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['invoices'] })
      toast.success('Pembayaran dikonfirmasi, portal peserta diaktifkan')
      setSelectedInvoice(null)
    },
    onError: () => toast.error('Gagal konfirmasi'),
  })

  const invoices = data?.data ?? []
  const meta = data?.meta

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-gray-800">Invoice</h2>
        <button
          onClick={() => setShowCreate(true)}
          className="flex items-center gap-2 px-4 py-2 bg-emerald-600 text-white text-sm rounded-lg hover:bg-emerald-700"
        >
          <Plus size={16} /> Buat Invoice
        </button>
      </div>

      {/* Status filter */}
      <div className="flex gap-2 flex-wrap">
        {(['', 'diterbitkan', 'menunggu_bayar', 'menunggu_konfirmasi_gateway', 'dibayar', 'lunas'] as const).map((s) => (
          <button
            key={s}
            onClick={() => { setStatusFilter(s); setPage(1) }}
            className={`px-3 py-1.5 rounded-full text-xs font-medium border ${
              statusFilter === s ? 'bg-emerald-700 text-white border-emerald-700' : 'bg-white text-gray-600 border-gray-200'
            }`}
          >
            {s === '' ? 'Semua' : s.replace('_', ' ')}
          </button>
        ))}
      </div>

      {/* Table */}
      <div className="bg-white rounded-xl border overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 text-xs text-gray-500 uppercase tracking-wider">
            <tr>
              <th className="px-4 py-3 text-left">No. Invoice</th>
              <th className="px-4 py-3 text-left">Peserta</th>
              <th className="px-4 py-3 text-left">Paket</th>
              <th className="px-4 py-3 text-right">Jumlah</th>
              <th className="px-4 py-3 text-left">Jatuh Tempo</th>
              <th className="px-4 py-3 text-left">Status</th>
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
            ) : invoices.length === 0 ? (
              <tr><td colSpan={7} className="text-center py-12 text-gray-400">Belum ada invoice</td></tr>
            ) : invoices.map((inv) => (
              <tr key={inv.id} className="hover:bg-gray-50">
                <td className="px-4 py-3 font-mono text-xs font-semibold text-gray-700">{inv.invoice_number}</td>
                <td className="px-4 py-3">
                  <p className="font-medium text-gray-800">{inv.participant_name}</p>
                  <p className="text-xs text-gray-500">{inv.participant_phone}</p>
                </td>
                <td className="px-4 py-3 text-gray-600 text-xs max-w-xs truncate">{inv.package_name}</td>
                <td className="px-4 py-3 text-right font-medium text-gray-800">
                  Rp {inv.amount.toLocaleString('id-ID')}
                </td>
                <td className="px-4 py-3 text-gray-600 text-xs">
                  {formatDate(inv.due_date)}
                </td>
                <td className="px-4 py-3">
                  <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${STATUS_COLORS[inv.status] ?? 'bg-gray-100 text-gray-700'}`}>
                    {inv.status.replace('_', ' ')}
                  </span>
                </td>
                <td className="px-4 py-3">
                  <div className="flex items-center gap-1">
                    <button
                      onClick={() => setSelectedInvoice(inv)}
                      className="p-1 text-gray-400 hover:text-gray-600"
                      title="Detail"
                    >
                      <Eye size={16} />
                    </button>
                    {(inv.status === 'menunggu_bayar' || inv.status === 'dibayar') && (
                      <button
                        onClick={() => confirmMutation.mutate(inv.id)}
                        className="p-1 text-emerald-600 hover:text-emerald-700"
                        title="Konfirmasi Pembayaran"
                      >
                        <CheckCircle size={16} />
                      </button>
                    )}
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
          <span className="px-3 py-1.5 text-sm">Hal {page}/{meta.total_pages}</span>
          <button disabled={page >= meta.total_pages} onClick={() => setPage(p => p + 1)}
            className="px-3 py-1.5 border rounded text-sm disabled:opacity-40">Berikutnya →</button>
        </div>
      )}

      {/* Create invoice modal */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl w-full max-w-md p-6 space-y-4">
            <div className="flex justify-between">
              <div>
                <h3 className="font-semibold">Buat Invoice Tambahan</h3>
                <p className="text-xs text-gray-500 mt-0.5">
                  Untuk tagihan di luar paket. Invoice paket sudah terbit otomatis saat konversi lead.
                </p>
              </div>
              <button onClick={closeCreate} className="text-gray-400">✕</button>
            </div>
            <div className="space-y-3">
              <div>
                <label htmlFor="invoice-participant" className="text-xs font-medium text-gray-600 block mb-1">
                  Peserta
                </label>
                <ParticipantPicker
                  id="invoice-participant"
                  selected={invoiceFor}
                  onChange={pickInvoiceParticipant}
                />
                {/* Keterangan, bukan masukan: batch mengikuti peserta yang
                    dipilih, sehingga tidak mungkin dipasangkan ke batch yang
                    bukan miliknya. */}
                {invoiceFor && (
                  <p className="text-xs text-gray-500 mt-1.5">
                    {invoiceFor.package_name || 'Paket tidak diketahui'} &middot; berangkat{' '}
                    {formatDate(invoiceFor.batch_departure_date)}
                  </p>
                )}
              </div>
              <div>
                <label className="text-xs font-medium text-gray-600 block mb-1">Total Tagihan (Rp)</label>
                <input type="number" className="w-full border rounded-lg px-3 py-2 text-sm"
                  value={form.amount} onChange={(e) => setForm({ ...form, amount: Number(e.target.value) })} />
              </div>
              <div>
                <label className="text-xs font-medium text-gray-600 block mb-1">Jatuh Tempo</label>
                <input type="date" className="w-full border rounded-lg px-3 py-2 text-sm"
                  value={form.due_date} onChange={(e) => setForm({ ...form, due_date: e.target.value })} />
              </div>
              <div>
                <label className="text-xs font-medium text-gray-600 block mb-1">Catatan (opsional)</label>
                <textarea rows={2} className="w-full border rounded-lg px-3 py-2 text-sm resize-none"
                  value={form.notes} onChange={(e) => setForm({ ...form, notes: e.target.value })} />
              </div>
            </div>
            <button
              onClick={() => createMutation.mutate(form)}
              disabled={!form.participant_id || !form.batch_id || !form.amount || !form.due_date || createMutation.isPending}
              className="w-full py-2 bg-emerald-600 text-white rounded-lg text-sm font-medium disabled:opacity-50"
            >
              {createMutation.isPending ? 'Menerbitkan...' : 'Terbitkan Invoice'}
            </button>
          </div>
        </div>
      )}

      {/* Invoice detail + Payment Proof Review (AC-INV-02) */}
      {selectedInvoice && (
        <InvoiceDetailModal
          invoice={selectedInvoice}
          onClose={() => setSelectedInvoice(null)}
          onConfirm={() => confirmMutation.mutate(selectedInvoice.id)}
          confirmPending={confirmMutation.isPending}
        />
      )}
    </div>
  )
}

// ─── Invoice Detail + Payment Proof Review Modal ─────────────────────────────

function InvoiceDetailModal({
  invoice, onClose, onConfirm, confirmPending,
}: {
  invoice: Invoice
  onClose: () => void
  onConfirm: () => void
  confirmPending: boolean
}) {
  const qc = useQueryClient()
  const [rejectProofID, setRejectProofID] = useState<string | null>(null)
  const [rejectNotes, setRejectNotes] = useState('')

  // Fetch invoice detail with payment proofs
  const { data } = useQuery({
    queryKey: ['invoice-detail', invoice.id],
    queryFn: () =>
      api.get<{ success: boolean; data: { invoice: Invoice; payment_proofs: PaymentProof[] } }>(
        `/admin/invoices/${invoice.id}`
      ).then(r => r.data.data),
  })

  const reviewMut = useMutation({
    mutationFn: ({ proofID, status, notes }: { proofID: string; status: string; notes: string }) =>
      api.patch(`/admin/invoices/${invoice.id}/proofs/${proofID}/review`, { status, notes }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['invoice-detail', invoice.id] })
      qc.invalidateQueries({ queryKey: ['invoices'] })
      toast.success('Bukti transfer direview')
      setRejectProofID(null)
      setRejectNotes('')
    },
    onError: () => toast.error('Gagal mereview bukti'),
  })

  const proofs = data?.payment_proofs ?? []

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4 overflow-y-auto">
      <div className="bg-white rounded-xl w-full max-w-lg p-6 space-y-4 max-h-[90vh] overflow-y-auto my-4">
        <div className="flex justify-between items-start">
          <h3 className="font-semibold">{invoice.invoice_number}</h3>
          <button onClick={onClose} className="text-gray-400">✕</button>
        </div>

        {/* Invoice info */}
        <div className="text-sm space-y-1 text-gray-700">
          <p><span className="font-medium">Peserta:</span> {invoice.participant_name}</p>
          <p><span className="font-medium">WA:</span> {invoice.participant_phone}</p>
          <p><span className="font-medium">Paket:</span> {invoice.package_name}</p>
          <p><span className="font-medium">Jumlah:</span> Rp {invoice.amount.toLocaleString('id-ID')}</p>
          <p><span className="font-medium">Jatuh Tempo:</span> {formatDate(invoice.due_date)}</p>
          <p><span className="font-medium">Status:</span> {invoice.status}</p>
          {invoice.confirmed_at && (
            <p><span className="font-medium">Dikonfirmasi:</span> {new Date(invoice.confirmed_at).toLocaleDateString('id-ID')}</p>
          )}
        </div>

        {/* Payment Proofs Review (AC-INV-02) */}
        <div className="border-t pt-3">
          <h4 className="font-semibold text-sm text-gray-700 mb-2 flex items-center gap-1.5">
            <FileText size={14} /> Bukti Transfer ({proofs.length})
          </h4>
          {proofs.length === 0 ? (
            <p className="text-xs text-gray-400 italic">Peserta belum upload bukti transfer</p>
          ) : (
            <div className="space-y-2">
              {proofs.map(p => (
                <div key={p.id} className="border rounded-lg p-3 text-xs">
                  <div className="flex items-center justify-between mb-1.5">
                    <span className={`px-2 py-0.5 rounded-full font-medium ${
                      p.status === 'disetujui' ? 'bg-green-100 text-green-700' :
                      p.status === 'ditolak' ? 'bg-red-100 text-red-600' :
                      'bg-yellow-100 text-yellow-700'
                    }`}>
                      {p.status}
                    </span>
                    <span className="text-gray-400">
                      Rp {p.amount_claimed.toLocaleString('id-ID')} ·{' '}
                      {new Date(p.uploaded_at).toLocaleDateString('id-ID', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })}
                    </span>
                  </div>
                  {p.notes && <p className="text-gray-500 italic">"{p.notes}"</p>}
                  {p.review_notes && (
                    <p className="text-red-600 mt-1">Catatan admin: {p.review_notes}</p>
                  )}
                  <div className="flex items-center gap-2 mt-2">
                    <button
                      onClick={() =>
                        openSignedFile('payment_proof', p.id, p.file_path).catch(() =>
                          toast.error('Gagal membuka bukti transfer'),
                        )
                      }
                      className="px-2 py-1 border rounded text-xs hover:bg-gray-50"
                    >
                      Lihat Bukti
                    </button>
                    {p.status === 'menunggu' && (
                      <>
                        <button
                          onClick={() => reviewMut.mutate({ proofID: p.id, status: 'disetujui', notes: '' })}
                          disabled={reviewMut.isPending}
                          className="flex items-center gap-1 px-2 py-1 bg-emerald-50 text-emerald-700 rounded text-xs hover:bg-emerald-100"
                        >
                          <CheckCircle size={11} /> Setujui
                        </button>
                        <button
                          onClick={() => setRejectProofID(p.id)}
                          className="flex items-center gap-1 px-2 py-1 bg-red-50 text-red-600 rounded text-xs hover:bg-red-100"
                        >
                          <XCircle size={11} /> Tolak
                        </button>
                      </>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Reject reason input */}
        {rejectProofID && (
          <div className="bg-red-50 border border-red-200 rounded-lg p-3 space-y-2">
            <p className="text-xs font-medium text-red-700">Alasan Penolakan</p>
            <textarea
              rows={2}
              className="w-full border rounded px-2 py-1.5 text-xs resize-none"
              placeholder="contoh: Jumlah transfer tidak sesuai"
              value={rejectNotes}
              onChange={e => setRejectNotes(e.target.value)}
            />
            <div className="flex gap-2">
              <button
                onClick={() => { setRejectProofID(null); setRejectNotes('') }}
                className="px-3 py-1 border rounded text-xs"
              >
                Batal
              </button>
              <button
                onClick={() => reviewMut.mutate({ proofID: rejectProofID!, status: 'ditolak', notes: rejectNotes })}
                disabled={!rejectNotes.trim() || reviewMut.isPending}
                className="px-3 py-1 bg-red-600 text-white rounded text-xs disabled:opacity-50"
              >
                Tolak Bukti
              </button>
            </div>
          </div>
        )}

        {/* Confirm payment */}
        {(invoice.status === 'menunggu_bayar' || invoice.status === 'dibayar') && (
          <button
            onClick={onConfirm}
            disabled={confirmPending}
            className="w-full py-2 bg-emerald-600 text-white rounded-lg text-sm font-medium disabled:opacity-50"
          >
            {confirmPending ? 'Memproses...' : '✓ Konfirmasi Pembayaran Lunas'}
          </button>
        )}
      </div>
    </div>
  )
}
