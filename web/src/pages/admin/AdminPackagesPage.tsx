import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Pencil, Trash2, Plus } from 'lucide-react'
import api from '../../utils/api'
import { TourPackage, PackagesResponse } from '../../types'
import Spinner from '../../components/Spinner'

export default function AdminPackagesPage() {
  const [page, setPage] = useState(1)
  const [showForm, setShowForm] = useState(false)
  const [editing, setEditing] = useState<TourPackage | null>(null)
  const qc = useQueryClient()

  const { data, isLoading } = useQuery<PackagesResponse>({
    queryKey: ['admin-packages', page],
    queryFn: () => api.get(`/packages?page=${page}&per_page=10`).then((r) => r.data),
  })

  const deleteMut = useMutation({
    mutationFn: (id: string) => api.delete(`/admin/packages/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-packages'] }),
  })

  if (isLoading) return <Spinner message="Memuat paket..." />

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Manajemen Paket Wisata</h1>
        <button onClick={() => { setEditing(null); setShowForm(true) }} className="btn-primary text-sm">
          <Plus className="w-4 h-4" /> Tambah Paket
        </button>
      </div>

      {showForm && (
        <PackageForm
          initial={editing}
          onClose={() => setShowForm(false)}
          onSaved={() => { setShowForm(false); qc.invalidateQueries({ queryKey: ['admin-packages'] }) }}
        />
      )}

      <div className="card overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 text-gray-500 text-xs uppercase tracking-wide">
            <tr>
              <th className="px-5 py-3 text-left">Judul</th>
              <th className="px-5 py-3 text-left">Durasi</th>
              <th className="px-5 py-3 text-left">Harga</th>
              <th className="px-5 py-3 text-left">Status</th>
              <th className="px-5 py-3 text-right">Aksi</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-50">
            {data?.data.map((pkg) => (
              <tr key={pkg.id} className="hover:bg-gray-50 transition-colors">
                <td className="px-5 py-3 font-medium text-gray-800">{pkg.title}</td>
                <td className="px-5 py-3 text-gray-500">{pkg.duration_days} hari</td>
                <td className="px-5 py-3 text-gray-700">
                  {pkg.price_label ?? `Rp ${pkg.price.toLocaleString('id-ID')}`}
                </td>
                <td className="px-5 py-3">
                  <span className={`text-xs font-medium px-2.5 py-0.5 rounded-full ${pkg.is_active ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'}`}>
                    {pkg.is_active ? 'Aktif' : 'Nonaktif'}
                  </span>
                </td>
                <td className="px-5 py-3 text-right flex justify-end gap-2">
                  <button
                    onClick={() => { setEditing(pkg); setShowForm(true) }}
                    className="p-1.5 text-gray-400 hover:text-primary-600 transition-colors"
                    title="Edit"
                  >
                    <Pencil className="w-4 h-4" />
                  </button>
                  <button
                    onClick={() => { if (confirm('Hapus paket ini?')) deleteMut.mutate(pkg.id) }}
                    className="p-1.5 text-gray-400 hover:text-red-600 transition-colors"
                    title="Hapus"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </td>
              </tr>
            ))}
            {(!data?.data || data.data.length === 0) && (
              <tr><td colSpan={5} className="px-5 py-10 text-center text-gray-400">Belum ada paket.</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}

interface PackageFormProps {
  initial: TourPackage | null
  onClose: () => void
  onSaved: () => void
}

function PackageForm({ initial, onClose, onSaved }: PackageFormProps) {
  const [form, setForm] = useState({
    title: initial?.title ?? '',
    slug: initial?.slug ?? '',
    description: initial?.description ?? '',
    price: initial?.price ?? 0,
    price_label: initial?.price_label ?? '',
    duration_days: initial?.duration_days ?? 1,
    min_participants: initial?.min_participants ?? 1,
    package_type: initial?.package_type ?? 'regular',
    is_active: initial?.is_active ?? true,
    cover_image_url: initial?.cover_image_url ?? '',
  })

  const saveMut = useMutation({
    mutationFn: () =>
      initial
        ? api.put(`/admin/packages/${initial.id}`, form)
        : api.post('/admin/packages', form),
    onSuccess: onSaved,
  })

  return (
    <div className="card p-6 mb-6">
      <h2 className="font-semibold text-gray-800 mb-4">{initial ? 'Edit Paket' : 'Tambah Paket Baru'}</h2>
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div className="sm:col-span-2">
          <label className="label">Judul</label>
          <input className="input" value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} />
        </div>
        <div>
          <label className="label">Slug (URL)</label>
          <input className="input" value={form.slug} onChange={(e) => setForm({ ...form, slug: e.target.value })} />
        </div>
        <div>
          <label className="label">Tipe Paket</label>
          <select className="input" value={form.package_type} onChange={(e) => setForm({ ...form, package_type: e.target.value })}>
            <option value="regular">Regular</option>
            <option value="hemat">Hemat</option>
            <option value="luxury">Luxury</option>
            <option value="premium">Premium</option>
          </select>
        </div>
        <div>
          <label className="label">Harga (Rp)</label>
          <input type="number" className="input" value={form.price} onChange={(e) => setForm({ ...form, price: Number(e.target.value) })} />
        </div>
        <div>
          <label className="label">Label Harga (opsional)</label>
          <input className="input" value={form.price_label} onChange={(e) => setForm({ ...form, price_label: e.target.value })} placeholder="mulai dari Rp 5 jt/orang" />
        </div>
        <div>
          <label className="label">Durasi (hari)</label>
          <input type="number" className="input" value={form.duration_days} onChange={(e) => setForm({ ...form, duration_days: Number(e.target.value) })} />
        </div>
        <div>
          <label className="label">Min. Peserta</label>
          <input type="number" className="input" value={form.min_participants} onChange={(e) => setForm({ ...form, min_participants: Number(e.target.value) })} />
        </div>
        <div className="sm:col-span-2">
          <label className="label">Cover Image URL</label>
          <input className="input" value={form.cover_image_url} onChange={(e) => setForm({ ...form, cover_image_url: e.target.value })} />
        </div>
        <div className="sm:col-span-2">
          <label className="label">Deskripsi</label>
          <textarea rows={3} className="input resize-none" value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
        </div>
        <div className="flex items-center gap-2">
          <input type="checkbox" id="is_active" checked={form.is_active} onChange={(e) => setForm({ ...form, is_active: e.target.checked })} className="rounded" />
          <label htmlFor="is_active" className="text-sm text-gray-700">Aktif</label>
        </div>
      </div>
      <div className="flex gap-3 mt-5">
        <button onClick={() => saveMut.mutate()} disabled={saveMut.isPending} className="btn-primary text-sm">
          {saveMut.isPending ? 'Menyimpan...' : 'Simpan'}
        </button>
        <button onClick={onClose} className="btn-secondary text-sm">Batal</button>
      </div>
    </div>
  )
}
