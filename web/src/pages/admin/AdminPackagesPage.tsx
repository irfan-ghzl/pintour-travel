import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Plus, Pencil, Trash2, Calendar } from 'lucide-react'
import toast from 'react-hot-toast'
import api from '../../utils/api'
import { formatDate } from '../../utils/date'
import type { Package, PackageBatch, PaginatedResponse } from '../../types'

const CATEGORIES = ['reguler', 'umroh', 'halal', 'honeymoon'] as const
type Category = typeof CATEGORIES[number]

const EMPTY_PKG: Partial<Package> = {
  name: '', slug: '', destination: '', category: 'reguler', duration_days: 7,
  description: '', base_price: 0, is_active: true,
  itinerary: [], requirements: [], facilities: [],
  terms_conditions: '', visa_info: '',
}

export default function AdminPackagesPage() {
  const [page, setPage] = useState(1)
  const [catFilter, setCatFilter] = useState<Category | ''>('')
  const [showForm, setShowForm] = useState(false)
  const [editing, setEditing] = useState<Package | null>(null)
  const [showBatches, setShowBatches] = useState<Package | null>(null)
  const qc = useQueryClient()

  const params = new URLSearchParams({ page: String(page), per_page: '20' })
  if (catFilter) params.set('category', catFilter)

  const { data, isLoading } = useQuery({
    queryKey: ['admin-packages', catFilter, page],
    queryFn: () => api.get<PaginatedResponse<Package>>(`/admin/packages?${params}`).then(r => r.data),
  })

  const deleteMut = useMutation({
    mutationFn: (id: string) => api.delete(`/admin/packages/${id}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['admin-packages'] }); toast.success('Paket dihapus') },
    onError: () => toast.error('Gagal menghapus paket'),
  })

  const packages = data?.data ?? []
  const meta = data?.meta

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-gray-800">Paket Wisata</h2>
        <button
          onClick={() => { setEditing(null); setShowForm(true) }}
          className="flex items-center gap-2 px-4 py-2 bg-emerald-600 text-white text-sm rounded-lg hover:bg-emerald-700"
        >
          <Plus size={16} /> Tambah Paket
        </button>
      </div>

      {/* Category filter */}
      <div className="flex gap-2 flex-wrap">
        {(['', ...CATEGORIES] as const).map((c) => (
          <button
            key={c}
            onClick={() => { setCatFilter(c); setPage(1) }}
            className={`px-3 py-1.5 rounded-full text-xs font-medium border capitalize ${
              catFilter === c ? 'bg-emerald-700 text-white border-emerald-700' : 'bg-white text-gray-600 border-gray-200'
            }`}
          >
            {c === '' ? 'Semua' : c}
          </button>
        ))}
      </div>

      {/* Table */}
      <div className="bg-white rounded-xl border overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 text-xs text-gray-500 uppercase tracking-wider">
            <tr>
              <th className="px-4 py-3 text-left">Nama Paket</th>
              <th className="px-4 py-3 text-left">Destinasi</th>
              <th className="px-4 py-3 text-left">Kategori</th>
              <th className="px-4 py-3 text-right">Harga</th>
              <th className="px-4 py-3 text-left">Durasi</th>
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
            ) : packages.length === 0 ? (
              <tr><td colSpan={7} className="text-center py-12 text-gray-400">Belum ada paket</td></tr>
            ) : packages.map((pkg) => (
              <tr key={pkg.id} className="hover:bg-gray-50">
                <td className="px-4 py-3">
                  <p className="font-medium text-gray-800">{pkg.name}</p>
                  <p className="text-xs text-gray-400">{pkg.slug}</p>
                </td>
                <td className="px-4 py-3 text-gray-600">{pkg.destination}</td>
                <td className="px-4 py-3 capitalize text-gray-600">{pkg.category}</td>
                <td className="px-4 py-3 text-right font-medium text-gray-800">
                  Rp {pkg.base_price.toLocaleString('id-ID')}
                </td>
                <td className="px-4 py-3 text-gray-600">{pkg.duration_days} hari</td>
                <td className="px-4 py-3">
                  <span className={`px-2 py-0.5 rounded-full text-xs ${pkg.is_active ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'}`}>
                    {pkg.is_active ? 'Aktif' : 'Nonaktif'}
                  </span>
                </td>
                <td className="px-4 py-3">
                  <div className="flex items-center gap-1">
                    <button onClick={() => setShowBatches(pkg)} className="p-1 text-blue-500 hover:text-blue-600" title="Jadwal">
                      <Calendar size={15} />
                    </button>
                    <button onClick={() => { setEditing(pkg); setShowForm(true) }} className="p-1 text-gray-400 hover:text-gray-600">
                      <Pencil size={15} />
                    </button>
                    <button
                      onClick={() => window.confirm('Hapus paket ini?') && deleteMut.mutate(pkg.id)}
                      className="p-1 text-red-400 hover:text-red-600"
                    >
                      <Trash2 size={15} />
                    </button>
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
            className="px-3 py-1.5 border rounded text-sm disabled:opacity-40">Berikutnya ���</button>
        </div>
      )}

      {/* Package form modal */}
      {showForm && (
        <PackageFormModal
          initial={editing}
          onClose={() => setShowForm(false)}
          onSaved={() => { setShowForm(false); qc.invalidateQueries({ queryKey: ['admin-packages'] }) }}
        />
      )}

      {/* Batch list modal */}
      {showBatches && (
        <BatchModal pkg={showBatches} onClose={() => setShowBatches(null)} />
      )}
    </div>
  )
}

// ── PackageFormModal ──────────────────────────────────────────────────────────

function PackageFormModal({ initial, onClose, onSaved }: {
  initial: Package | null
  onClose: () => void
  onSaved: () => void
}) {
  const [form, setForm] = useState<Partial<Package>>(initial ?? { ...EMPTY_PKG })

  const saveMut = useMutation({
    mutationFn: () =>
      initial
        ? api.put(`/admin/packages/${initial.id}`, form)
        : api.post('/admin/packages', form),
    onSuccess: () => { toast.success(initial ? 'Paket diperbarui' : 'Paket ditambahkan'); onSaved() },
    onError: (e: any) => toast.error(e.response?.data?.message ?? 'Gagal menyimpan'),
  })

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4 overflow-y-auto">
      <div className="bg-white rounded-xl w-full max-w-lg p-6 space-y-4 my-4">
        <div className="flex justify-between">
          <h3 className="font-semibold">{initial ? 'Edit Paket' : 'Tambah Paket'}</h3>
          <button onClick={onClose} className="text-gray-400">✕</button>
        </div>
        <div className="grid grid-cols-2 gap-3">
          <div className="col-span-2">
            <label className="text-xs font-medium text-gray-600 block mb-1">Nama Paket *</label>
            <input className="w-full border rounded-lg px-3 py-2 text-sm" value={form.name ?? ''}
              onChange={(e) => setForm({ ...form, name: e.target.value })} />
          </div>
          <div>
            <label className="text-xs font-medium text-gray-600 block mb-1">Destinasi *</label>
            <input className="w-full border rounded-lg px-3 py-2 text-sm" value={form.destination ?? ''}
              onChange={(e) => setForm({ ...form, destination: e.target.value })} />
          </div>
          <div>
            <label className="text-xs font-medium text-gray-600 block mb-1">Kategori</label>
            <select className="w-full border rounded-lg px-3 py-2 text-sm" value={form.category ?? 'reguler'}
              onChange={(e) => setForm({ ...form, category: e.target.value as Category })}>
              {CATEGORIES.map(c => <option key={c} value={c} className="capitalize">{c}</option>)}
            </select>
          </div>
          <div>
            <label className="text-xs font-medium text-gray-600 block mb-1">Harga Dasar (Rp) *</label>
            <input type="number" className="w-full border rounded-lg px-3 py-2 text-sm" value={form.base_price ?? 0}
              onChange={(e) => setForm({ ...form, base_price: Number(e.target.value) })} />
          </div>
          <div>
            <label className="text-xs font-medium text-gray-600 block mb-1">Durasi (hari) *</label>
            <input type="number" min={1} className="w-full border rounded-lg px-3 py-2 text-sm"
              value={form.duration_days ?? 7}
              onChange={(e) => setForm({ ...form, duration_days: Number(e.target.value) })} />
          </div>
          <div className="col-span-2">
            <label className="text-xs font-medium text-gray-600 block mb-1">Deskripsi</label>
            <textarea rows={3} className="w-full border rounded-lg px-3 py-2 text-sm resize-none"
              value={form.description ?? ''}
              onChange={(e) => setForm({ ...form, description: e.target.value })} />
          </div>
          {/* FR-CAT-03: Fasilitas yang disertakan */}
          <div className="col-span-2">
            <label className="text-xs font-medium text-gray-600 block mb-1">
              Fasilitas yang Disertakan (1 baris per fasilitas)
            </label>
            <textarea rows={3} className="w-full border rounded-lg px-3 py-2 text-sm resize-none"
              placeholder="Hotel bintang 4&#10;Tiket pesawat PP&#10;Makan 3x sehari&#10;Tour leader berpengalaman"
              value={(form.facilities ?? []).join('\n')}
              onChange={(e) => setForm({ ...form, facilities: e.target.value.split('\n').filter(s => s.trim()) })} />
          </div>
          {/* FR-CAT-03: Persyaratan */}
          <div className="col-span-2">
            <label className="text-xs font-medium text-gray-600 block mb-1">
              Persyaratan Dokumen (1 baris per item)
            </label>
            <textarea rows={2} className="w-full border rounded-lg px-3 py-2 text-sm resize-none"
              placeholder="Paspor minimal masa berlaku 6 bulan&#10;KTP&#10;Rekening koran 3 bulan terakhir"
              value={(form.requirements ?? []).join('\n')}
              onChange={(e) => setForm({ ...form, requirements: e.target.value.split('\n').filter(s => s.trim()) })} />
          </div>
          {/* FR-CAT-03: Informasi Visa */}
          <div className="col-span-2">
            <label className="text-xs font-medium text-gray-600 block mb-1">Informasi Visa</label>
            <textarea rows={2} className="w-full border rounded-lg px-3 py-2 text-sm resize-none"
              placeholder="Visa Schengen diurus oleh tim Pintour, pembayaran terpisah Rp 1.500.000"
              value={form.visa_info ?? ''}
              onChange={(e) => setForm({ ...form, visa_info: e.target.value })} />
          </div>
          {/* FR-CAT-03: Syarat & Ketentuan */}
          <div className="col-span-2">
            <label className="text-xs font-medium text-gray-600 block mb-1">Syarat & Ketentuan</label>
            <textarea rows={2} className="w-full border rounded-lg px-3 py-2 text-sm resize-none"
              placeholder="Pembayaran DP 50% saat booking, pelunasan H-14"
              value={form.terms_conditions ?? ''}
              onChange={(e) => setForm({ ...form, terms_conditions: e.target.value })} />
          </div>
          <div className="col-span-2 flex items-center gap-2">
            <input type="checkbox" id="is_active" checked={form.is_active ?? true}
              onChange={(e) => setForm({ ...form, is_active: e.target.checked })} />
            <label htmlFor="is_active" className="text-sm text-gray-700">Tampilkan di katalog publik</label>
          </div>
        </div>
        <button
          onClick={() => saveMut.mutate()}
          disabled={!form.name || !form.destination || !form.base_price || saveMut.isPending}
          className="w-full py-2.5 bg-emerald-600 text-white rounded-lg text-sm font-medium disabled:opacity-50"
        >
          {saveMut.isPending ? 'Menyimpan...' : (initial ? 'Perbarui Paket' : 'Tambah Paket')}
        </button>
      </div>
    </div>
  )
}

// ── BatchModal ────────────────────────────────────────────────────────────────

function BatchModal({ pkg, onClose }: { pkg: Package; onClose: () => void }) {
  const qc = useQueryClient()
  const [showAdd, setShowAdd] = useState(false)
  const [form, setForm] = useState({
    departure_date: '', return_date: '', quota: 20,
    price_single: 0, price_double: 0, price_triple: 0, status: 'tersedia',
    tour_leader_id: '', wa_group_link: '',
  })

  const { data: batches } = useQuery({
    queryKey: ['batches', pkg.id],
    queryFn: () => api.get<{ success: boolean; data: PackageBatch[] }>(`/admin/packages/${pkg.id}/batches`).then(r => r.data.data),
  })

  // Fetch tour leaders for assignment dropdown
  const { data: tourLeaders } = useQuery({
    queryKey: ['tour-leaders-list'],
    queryFn: () => api.get<{ success: boolean; data: any[] }>('/admin/tour-leaders').then(r => r.data.data ?? []),
  })

  const addMut = useMutation({
    mutationFn: () => {
      const payload = { ...form, tour_leader_id: form.tour_leader_id || null }
      return api.post(`/admin/packages/${pkg.id}/batches`, payload)
    },
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['batches', pkg.id] }); toast.success('Batch ditambahkan'); setShowAdd(false) },
    onError: () => toast.error('Gagal menambah batch'),
  })

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-xl w-full max-w-lg p-6 space-y-4 max-h-[90vh] overflow-y-auto">
        <div className="flex justify-between">
          <h3 className="font-semibold">Jadwal Keberangkatan — {pkg.name}</h3>
          <button onClick={onClose} className="text-gray-400">✕</button>
        </div>

        <div className="space-y-2">
          {(batches ?? []).length === 0 ? (
            <p className="text-sm text-gray-400 text-center py-4">Belum ada jadwal</p>
          ) : (batches ?? []).map((b) => (
            <div key={b.id} className="flex items-center justify-between bg-gray-50 rounded-lg px-3 py-2 text-sm">
              <div>
                <span className="font-medium">{formatDate(b.departure_date)}</span>
                <span className="text-gray-400 mx-2">→</span>
                <span>{formatDate(b.return_date, { day: 'numeric', month: 'short' })}</span>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-gray-500">{b.quota} pax</span>
                <span className={`px-2 py-0.5 rounded text-xs ${b.status === 'tersedia' ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-600'}`}>
                  {b.status}
                </span>
              </div>
            </div>
          ))}
        </div>

        <button onClick={() => setShowAdd(!showAdd)}
          className="flex items-center gap-2 px-3 py-2 bg-emerald-50 text-emerald-700 text-sm rounded-lg hover:bg-emerald-100">
          <Plus size={14} /> Tambah Jadwal
        </button>

        {showAdd && (
          <div className="border rounded-lg p-4 space-y-3 bg-gray-50">
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="text-xs text-gray-600 block mb-1">Tgl Berangkat</label>
                <input type="date" className="w-full border rounded px-2 py-1.5 text-sm"
                  value={form.departure_date} onChange={(e) => setForm({ ...form, departure_date: e.target.value })} />
              </div>
              <div>
                <label className="text-xs text-gray-600 block mb-1">Tgl Pulang</label>
                <input type="date" className="w-full border rounded px-2 py-1.5 text-sm"
                  value={form.return_date} onChange={(e) => setForm({ ...form, return_date: e.target.value })} />
              </div>
              <div>
                <label className="text-xs text-gray-600 block mb-1">Kuota</label>
                <input type="number" className="w-full border rounded px-2 py-1.5 text-sm"
                  value={form.quota} onChange={(e) => setForm({ ...form, quota: Number(e.target.value) })} />
              </div>
              <div>
                <label className="text-xs text-gray-600 block mb-1">Harga Double (Rp)</label>
                <input type="number" className="w-full border rounded px-2 py-1.5 text-sm"
                  value={form.price_double} onChange={(e) => setForm({ ...form, price_double: Number(e.target.value), price_single: Number(e.target.value) * 1.2 | 0, price_triple: Number(e.target.value) * 0.9 | 0 })} />
              </div>
              {/* Tour Leader assignment per batch */}
              <div className="col-span-2">
                <label className="text-xs text-gray-600 block mb-1">Tour Leader</label>
                <select
                  className="w-full border rounded px-2 py-1.5 text-sm"
                  value={form.tour_leader_id}
                  onChange={(e) => setForm({ ...form, tour_leader_id: e.target.value })}
                >
                  <option value="">— Pilih tour leader —</option>
                  {(tourLeaders ?? []).map((tl: any) => (
                    <option key={tl.user_id} value={tl.user_id}>
                      {tl.name} {tl.specialization ? `· ${tl.specialization}` : ''}
                    </option>
                  ))}
                </select>
              </div>
              {/* §6.1: grup WA briefing */}
              <div className="col-span-2">
                <label className="text-xs text-gray-600 block mb-1">Link Grup WA Briefing</label>
                <input
                  type="url"
                  className="w-full border rounded px-2 py-1.5 text-sm"
                  placeholder="https://chat.whatsapp.com/..."
                  value={form.wa_group_link}
                  onChange={(e) => setForm({ ...form, wa_group_link: e.target.value })}
                />
              </div>
            </div>
            <button onClick={() => addMut.mutate()} disabled={!form.departure_date || !form.return_date || addMut.isPending}
              className="w-full py-2 bg-emerald-600 text-white text-sm rounded-lg disabled:opacity-50">
              {addMut.isPending ? 'Menambahkan...' : 'Tambah Jadwal'}
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
