import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Plus, Eye } from 'lucide-react'
import api from '../../utils/api'
import { QuotationsResponse, Quotation, QuotationItem } from '../../types'
import Spinner from '../../components/Spinner'

export default function AdminQuotationsPage() {
  const [page, setPage] = useState(1)
  const [showForm, setShowForm] = useState(false)
  const [viewing, setViewing] = useState<Quotation | null>(null)
  const qc = useQueryClient()

  const { data, isLoading } = useQuery<QuotationsResponse>({
    queryKey: ['admin-quotations', page],
    queryFn: () => api.get(`/admin/quotations?page=${page}&per_page=10`).then((r) => r.data),
  })

  if (isLoading) return <Spinner message="Memuat penawaran..." />

  const totalPages = data ? Math.ceil(data.total / 10) : 1

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Quotation Builder</h1>
        <button onClick={() => setShowForm(true)} className="btn-primary text-sm">
          <Plus className="w-4 h-4" /> Buat Penawaran
        </button>
      </div>

      {showForm && (
        <QuotationForm
          onClose={() => setShowForm(false)}
          onSaved={() => { setShowForm(false); qc.invalidateQueries({ queryKey: ['admin-quotations'] }) }}
        />
      )}

      {viewing && (
        <QuotationDetail quotation={viewing} onClose={() => setViewing(null)} />
      )}

      <div className="card overflow-x-auto">
        <table className="w-full text-sm min-w-max">
          <thead className="bg-gray-50 text-gray-500 text-xs uppercase tracking-wide">
            <tr>
              <th className="px-5 py-3 text-left">Judul</th>
              <th className="px-5 py-3 text-left">Pelanggan</th>
              <th className="px-5 py-3 text-left">Total</th>
              <th className="px-5 py-3 text-left">Status</th>
              <th className="px-5 py-3 text-left">Tanggal</th>
              <th className="px-5 py-3 text-right">Aksi</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-50">
            {data?.data.map((q: Quotation) => (
              <tr key={q.id} className="hover:bg-gray-50 transition-colors">
                <td className="px-5 py-3 font-medium text-gray-800">{q.title}</td>
                <td className="px-5 py-3 text-gray-600">{q.customer_name}</td>
                <td className="px-5 py-3 text-gray-700 font-semibold">
                  Rp {q.total_price.toLocaleString('id-ID')}
                </td>
                <td className="px-5 py-3">
                  <span className={`text-xs font-medium px-2.5 py-0.5 rounded-full ${
                    q.status === 'sent' ? 'bg-blue-100 text-blue-700' :
                    q.status === 'accepted' ? 'bg-green-100 text-green-700' :
                    'bg-gray-100 text-gray-500'
                  }`}>{q.status}</span>
                </td>
                <td className="px-5 py-3 text-gray-400 text-xs">
                  {new Date(q.created_at).toLocaleDateString('id-ID')}
                </td>
                <td className="px-5 py-3 text-right">
                  <button
                    onClick={() => {
                      api.get(`/admin/quotations/${q.id}`).then((r) => setViewing(r.data))
                    }}
                    className="p-1.5 text-gray-400 hover:text-primary-600"
                    title="Lihat detail"
                  >
                    <Eye className="w-4 h-4" />
                  </button>
                </td>
              </tr>
            ))}
            {(!data?.data || data.data.length === 0) && (
              <tr><td colSpan={6} className="px-5 py-10 text-center text-gray-400">Belum ada penawaran.</td></tr>
            )}
          </tbody>
        </table>
      </div>

      {totalPages > 1 && (
        <div className="flex justify-center gap-2 mt-6">
          <button onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page === 1} className="btn-secondary py-2 px-4 text-sm disabled:opacity-40">
            Sebelumnya
          </button>
          <span className="flex items-center px-4 text-sm text-gray-500">{page} / {totalPages}</span>
          <button onClick={() => setPage((p) => Math.min(totalPages, p + 1))} disabled={page === totalPages} className="btn-secondary py-2 px-4 text-sm disabled:opacity-40">
            Berikutnya
          </button>
        </div>
      )}
    </div>
  )
}

// ─── Quotation Form ───────────────────────────────────────────────────────────

interface ItemRow { description: string; category: string; quantity: number; unit_price: number }

function QuotationForm({ onClose, onSaved }: { onClose: () => void; onSaved: () => void }) {
  const [form, setForm] = useState({ title: '', customer_name: '', customer_email: '', customer_phone: '', valid_until: '', notes: '' })
  const [items, setItems] = useState<ItemRow[]>([{ description: '', category: 'Transportasi', quantity: 1, unit_price: 0 }])

  const total = items.reduce((sum, it) => sum + it.quantity * it.unit_price, 0)

  const saveMut = useMutation({
    mutationFn: () => api.post('/admin/quotations', { ...form, items }),
    onSuccess: onSaved,
  })

  const updateItem = (idx: number, field: keyof ItemRow, value: string | number) => {
    setItems((prev) => prev.map((it, i) => i === idx ? { ...it, [field]: value } : it))
  }

  return (
    <div className="card p-6 mb-6">
      <h2 className="font-semibold text-gray-800 mb-5">Buat Penawaran Baru</h2>
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mb-5">
        <div className="sm:col-span-2">
          <label className="label">Judul Penawaran</label>
          <input className="input" value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} />
        </div>
        <div>
          <label className="label">Nama Pelanggan</label>
          <input className="input" value={form.customer_name} onChange={(e) => setForm({ ...form, customer_name: e.target.value })} />
        </div>
        <div>
          <label className="label">Email Pelanggan</label>
          <input type="email" className="input" value={form.customer_email} onChange={(e) => setForm({ ...form, customer_email: e.target.value })} />
        </div>
        <div>
          <label className="label">WhatsApp Pelanggan</label>
          <input className="input" value={form.customer_phone} onChange={(e) => setForm({ ...form, customer_phone: e.target.value })} />
        </div>
        <div>
          <label className="label">Berlaku Hingga</label>
          <input type="date" className="input" value={form.valid_until} onChange={(e) => setForm({ ...form, valid_until: e.target.value })} />
        </div>
      </div>

      {/* Line items */}
      <div className="mb-4">
        <div className="flex items-center justify-between mb-2">
          <h3 className="text-sm font-medium text-gray-700">Rincian Harga</h3>
          <button
            type="button"
            onClick={() => setItems([...items, { description: '', category: 'Lainnya', quantity: 1, unit_price: 0 }])}
            className="text-xs text-primary-600 hover:underline"
          >
            + Tambah baris
          </button>
        </div>
        <div className="overflow-x-auto rounded-lg border border-gray-200">
          <table className="w-full text-xs">
            <thead className="bg-gray-50 text-gray-500 uppercase tracking-wide">
              <tr>
                <th className="px-3 py-2 text-left">Deskripsi</th>
                <th className="px-3 py-2 text-left">Kategori</th>
                <th className="px-3 py-2 text-left w-16">Qty</th>
                <th className="px-3 py-2 text-left">Harga Satuan</th>
                <th className="px-3 py-2 text-left">Total</th>
                <th className="w-6" />
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {items.map((item, idx) => (
                <tr key={idx}>
                  <td className="px-3 py-2"><input className="input py-1 text-xs" value={item.description} onChange={(e) => updateItem(idx, 'description', e.target.value)} /></td>
                  <td className="px-3 py-2">
                    <select className="input py-1 text-xs" value={item.category} onChange={(e) => updateItem(idx, 'category', e.target.value)}>
                      {['Transportasi', 'Akomodasi', 'Makan', 'Tiket Masuk', 'Tour Guide', 'Lainnya'].map((c) => <option key={c}>{c}</option>)}
                    </select>
                  </td>
                  <td className="px-3 py-2"><input type="number" min="1" className="input py-1 text-xs w-16" value={item.quantity} onChange={(e) => updateItem(idx, 'quantity', Number(e.target.value))} /></td>
                  <td className="px-3 py-2"><input type="number" min="0" className="input py-1 text-xs" value={item.unit_price} onChange={(e) => updateItem(idx, 'unit_price', Number(e.target.value))} /></td>
                  <td className="px-3 py-2 font-medium">Rp {(item.quantity * item.unit_price).toLocaleString('id-ID')}</td>
                  <td className="px-3 py-2">
                    <button type="button" onClick={() => setItems(items.filter((_, i) => i !== idx))} className="text-red-400 hover:text-red-600 text-base leading-none">×</button>
                  </td>
                </tr>
              ))}
            </tbody>
            <tfoot>
              <tr className="bg-gray-50">
                <td colSpan={4} className="px-3 py-2 text-right font-semibold text-sm text-gray-700">Total</td>
                <td className="px-3 py-2 font-bold text-primary-700 text-sm">Rp {total.toLocaleString('id-ID')}</td>
                <td />
              </tr>
            </tfoot>
          </table>
        </div>
      </div>

      <div className="mb-4">
        <label className="label">Catatan</label>
        <textarea rows={2} className="input resize-none text-sm" value={form.notes} onChange={(e) => setForm({ ...form, notes: e.target.value })} />
      </div>

      <div className="flex gap-3">
        <button onClick={() => saveMut.mutate()} disabled={saveMut.isPending} className="btn-primary text-sm">
          {saveMut.isPending ? 'Menyimpan...' : 'Simpan Penawaran'}
        </button>
        <button onClick={onClose} className="btn-secondary text-sm">Batal</button>
      </div>
    </div>
  )
}

// ─── Quotation Detail Modal ───────────────────────────────────────────────────

function QuotationDetail({ quotation, onClose }: { quotation: Quotation; onClose: () => void }) {
  return (
    <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-2xl shadow-2xl w-full max-w-2xl max-h-screen overflow-y-auto p-8">
        <div className="flex items-start justify-between mb-6">
          <div>
            <h2 className="text-xl font-bold text-gray-900">{quotation.title}</h2>
            <p className="text-sm text-gray-500">{quotation.customer_name}</p>
          </div>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600 text-2xl leading-none">×</button>
        </div>

        <table className="w-full text-sm mb-6">
          <thead className="bg-gray-50 text-gray-500 text-xs uppercase">
            <tr>
              <th className="px-3 py-2 text-left">Deskripsi</th>
              <th className="px-3 py-2 text-left">Kategori</th>
              <th className="px-3 py-2 text-center">Qty</th>
              <th className="px-3 py-2 text-right">Satuan</th>
              <th className="px-3 py-2 text-right">Total</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {quotation.items?.map((item: QuotationItem) => (
              <tr key={item.id}>
                <td className="px-3 py-2">{item.description}</td>
                <td className="px-3 py-2 text-gray-500">{item.category}</td>
                <td className="px-3 py-2 text-center">{item.quantity}</td>
                <td className="px-3 py-2 text-right">Rp {item.unit_price.toLocaleString('id-ID')}</td>
                <td className="px-3 py-2 text-right font-semibold">Rp {item.total_price.toLocaleString('id-ID')}</td>
              </tr>
            ))}
          </tbody>
          <tfoot>
            <tr className="bg-primary-50">
              <td colSpan={4} className="px-3 py-3 text-right font-bold text-gray-800">TOTAL</td>
              <td className="px-3 py-3 text-right font-extrabold text-primary-700 text-base">
                Rp {quotation.total_price.toLocaleString('id-ID')}
              </td>
            </tr>
          </tfoot>
        </table>

        {quotation.notes && (
          <p className="text-sm text-gray-500 italic mb-6">Catatan: {quotation.notes}</p>
        )}

        <div className="flex gap-3">
          <button onClick={onClose} className="btn-secondary text-sm">Tutup</button>
          <button
            onClick={() => window.print()}
            className="btn-primary text-sm"
          >
            Cetak / Simpan PDF
          </button>
        </div>
      </div>
    </div>
  )
}
