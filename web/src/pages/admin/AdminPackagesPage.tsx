import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Pencil, Trash2, Plus, Image, X } from 'lucide-react'
import api from '../../utils/api'
import { TourPackage, PackagesResponse } from '../../types'
import Spinner from '../../components/Spinner'

interface GalleryImage { id: string; image_url: string; caption?: string; sort_order: number }

interface ItineraryItem {
  id: string
  day_number: number
  title: string
  description: string
  location: string
  start_time: string
  end_time: string
  activity_type: string
  sort_order: number
}

const EMPTY_ITEM: Omit<ItineraryItem, 'id'> = {
  day_number: 1, title: '', description: '', location: '',
  start_time: '', end_time: '', activity_type: 'sightseeing', sort_order: 0,
}

function ItineraryModal({ pkg, onClose }: { pkg: TourPackage; onClose: () => void }) {
  const qc = useQueryClient()
  const [editingItem, setEditingItem] = useState<ItineraryItem | null>(null)
  const [showAddForm, setShowAddForm] = useState(false)
  const [form, setForm] = useState<Omit<ItineraryItem, 'id'>>(EMPTY_ITEM)

  const { data: items, isLoading } = useQuery<ItineraryItem[]>({
    queryKey: ['admin-itinerary', pkg.id],
    queryFn: () => api.get(`/admin/packages/${pkg.id}/itinerary`).then((r) => r.data),
  })

  const invalidate = () => qc.invalidateQueries({ queryKey: ['admin-itinerary', pkg.id] })

  const addMut = useMutation({
    mutationFn: () => api.post(`/admin/packages/${pkg.id}/itinerary`, form),
    onSuccess: () => { invalidate(); setShowAddForm(false); setForm(EMPTY_ITEM) },
  })

  const updateMut = useMutation({
    mutationFn: () => api.put(`/admin/packages/${pkg.id}/itinerary/${editingItem!.id}`, form),
    onSuccess: () => { invalidate(); setEditingItem(null) },
  })

  const deleteMut = useMutation({
    mutationFn: (itemId: string) => api.delete(`/admin/packages/${pkg.id}/itinerary/${itemId}`),
    onSuccess: invalidate,
  })

  const openEdit = (item: ItineraryItem) => {
    setEditingItem(item)
    setForm({ day_number: item.day_number, title: item.title, description: item.description,
      location: item.location, start_time: item.start_time, end_time: item.end_time,
      activity_type: item.activity_type, sort_order: item.sort_order })
    setShowAddForm(false)
  }

  const openAdd = () => {
    setEditingItem(null)
    setForm(EMPTY_ITEM)
    setShowAddForm(true)
  }

  const activityTypes = ['sightseeing', 'transport', 'kuliner', 'belanja', 'hotel', 'outdoor', 'budaya', 'lainnya']

  const grouped = (items ?? []).reduce<Record<number, ItineraryItem[]>>((acc, it) => {
    ;(acc[it.day_number] ??= []).push(it)
    return acc
  }, {})

  return (
    <div className="fixed inset-0 bg-black/40 animate-backdrop-in flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-2xl shadow-xl w-full max-w-2xl max-h-[90vh] flex flex-col animate-modal-in">
        <div className="flex items-center justify-between px-6 py-4 border-b shrink-0">
          <h2 className="text-lg font-bold text-gray-900">Itinerary: {pkg.title}</h2>
          <button onClick={onClose}><X className="w-5 h-5 text-gray-400 hover:text-gray-700" /></button>
        </div>

        <div className="overflow-y-auto flex-1 p-6">
          {/* Add / Edit form */}
          {(showAddForm || editingItem) && (
            <div className="border rounded-xl p-4 mb-5 bg-blue-50">
              <h3 className="font-semibold text-sm text-blue-800 mb-3">
                {editingItem ? 'Edit Item' : 'Tambah Item Baru'}
              </h3>
              <div className="grid grid-cols-2 gap-3 text-sm">
                <div>
                  <label className="label">Hari ke-</label>
                  <input type="number" min={1} className="input text-sm" value={form.day_number}
                    onChange={(e) => setForm({ ...form, day_number: Number(e.target.value) })} />
                </div>
                <div>
                  <label className="label">Tipe Aktivitas</label>
                  <select className="input text-sm" value={form.activity_type}
                    onChange={(e) => setForm({ ...form, activity_type: e.target.value })}>
                    {activityTypes.map((t) => <option key={t} value={t}>{t}</option>)}
                  </select>
                </div>
                <div className="col-span-2">
                  <label className="label">Judul Aktivitas *</label>
                  <input className="input text-sm" placeholder="cth: Kunjungi Senso-ji Temple" value={form.title}
                    onChange={(e) => setForm({ ...form, title: e.target.value })} />
                </div>
                <div className="col-span-2">
                  <label className="label">Deskripsi</label>
                  <textarea rows={2} className="input text-sm resize-none" value={form.description}
                    onChange={(e) => setForm({ ...form, description: e.target.value })} />
                </div>
                <div>
                  <label className="label">Lokasi</label>
                  <input className="input text-sm" value={form.location}
                    onChange={(e) => setForm({ ...form, location: e.target.value })} />
                </div>
                <div className="grid grid-cols-2 gap-2">
                  <div>
                    <label className="label">Mulai</label>
                    <input type="time" className="input text-sm" value={form.start_time}
                      onChange={(e) => setForm({ ...form, start_time: e.target.value })} />
                  </div>
                  <div>
                    <label className="label">Selesai</label>
                    <input type="time" className="input text-sm" value={form.end_time}
                      onChange={(e) => setForm({ ...form, end_time: e.target.value })} />
                  </div>
                </div>
                <div>
                  <label className="label">Urutan</label>
                  <input type="number" min={0} className="input text-sm" value={form.sort_order}
                    onChange={(e) => setForm({ ...form, sort_order: Number(e.target.value) })} />
                </div>
              </div>
              <div className="flex gap-2 mt-3">
                <button
                  onClick={() => editingItem ? updateMut.mutate() : addMut.mutate()}
                  disabled={!form.title || addMut.isPending || updateMut.isPending}
                  className="btn-primary text-xs px-3 py-1.5 disabled:opacity-50"
                >
                  {addMut.isPending || updateMut.isPending ? 'Menyimpan...' : 'Simpan'}
                </button>
                <button
                  onClick={() => { setShowAddForm(false); setEditingItem(null) }}
                  className="btn-secondary text-xs px-3 py-1.5"
                >Batal</button>
              </div>
            </div>
          )}

          {/* Itinerary list grouped by day */}
          {isLoading ? (
            <p className="text-sm text-gray-400">Memuat itinerary...</p>
          ) : Object.keys(grouped).length > 0 ? (
            <div className="space-y-4">
              {Object.entries(grouped).sort(([a], [b]) => Number(a) - Number(b)).map(([day, dayItems]) => (
                <div key={day}>
                  <div className="flex items-center gap-2 mb-2">
                    <span className="bg-primary-600 text-white text-xs font-bold px-2.5 py-0.5 rounded-full">
                      Hari {day}
                    </span>
                    <div className="flex-1 h-px bg-gray-200" />
                  </div>
                  <div className="space-y-2 pl-2">
                    {dayItems.sort((a, b) => a.sort_order - b.sort_order).map((it) => (
                      <div key={it.id} className="flex items-start justify-between gap-3 bg-gray-50 rounded-lg px-3 py-2.5">
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 flex-wrap">
                            <span className="text-sm font-medium text-gray-800">{it.title}</span>
                            <span className="text-xs bg-gray-200 text-gray-600 px-1.5 py-0.5 rounded-full">{it.activity_type}</span>
                          </div>
                          {(it.start_time || it.location) && (
                            <p className="text-xs text-gray-400 mt-0.5">
                              {it.start_time && it.end_time ? `${it.start_time} – ${it.end_time}` : it.start_time}
                              {it.location && ` · ${it.location}`}
                            </p>
                          )}
                          {it.description && (
                            <p className="text-xs text-gray-500 mt-0.5 line-clamp-2">{it.description}</p>
                          )}
                        </div>
                        <div className="flex gap-1 shrink-0">
                          <button onClick={() => openEdit(it)} className="p-1 text-gray-400 hover:text-blue-600">
                            <Pencil className="w-3.5 h-3.5" />
                          </button>
                          <button
                            onClick={() => { if (confirm('Hapus item ini?')) deleteMut.mutate(it.id) }}
                            className="p-1 text-gray-400 hover:text-red-600"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-gray-400 text-center py-8">Belum ada itinerary. Klik "+ Tambah Item" untuk mulai.</p>
          )}
        </div>

        <div className="px-6 py-3 border-t shrink-0">
          <button onClick={openAdd} className="btn-primary text-sm w-full">
            <Plus className="w-4 h-4" /> Tambah Item
          </button>
        </div>
      </div>
    </div>
  )
}

function GalleryModal({ pkg, onClose }: { pkg: TourPackage; onClose: () => void }) {
  const qc = useQueryClient()
  const [imageUrl, setImageUrl] = useState('')
  const [caption, setCaption] = useState('')

  const { data: gallery, isLoading } = useQuery<GalleryImage[]>({
    queryKey: ['admin-gallery', pkg.id],
    queryFn: () => api.get(`/packages/${pkg.id}/gallery`).then((r) => r.data),
  })

  const addMut = useMutation({
    mutationFn: () => api.post(`/admin/packages/${pkg.id}/gallery`, { image_url: imageUrl, caption }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['admin-gallery', pkg.id] }); setImageUrl(''); setCaption('') },
  })

  const deleteMut = useMutation({
    mutationFn: (imageId: string) => api.delete(`/admin/packages/${pkg.id}/gallery/${imageId}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-gallery', pkg.id] }),
  })

  return (
    <div className="fixed inset-0 bg-black/40 animate-backdrop-in flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-2xl shadow-xl w-full max-w-2xl max-h-[85vh] overflow-y-auto animate-modal-in">
        <div className="flex items-center justify-between px-6 py-4 border-b">
          <h2 className="text-lg font-bold text-gray-900">Galeri: {pkg.title}</h2>
          <button onClick={onClose}><X className="w-5 h-5 text-gray-400 hover:text-gray-700" /></button>
        </div>
        <div className="p-6">
          {/* Add image form */}
          <div className="flex gap-2 mb-6">
            <input
              className="input flex-1 text-sm"
              placeholder="URL Gambar (https://...)"
              value={imageUrl}
              onChange={(e) => setImageUrl(e.target.value)}
            />
            <input
              className="input w-36 text-sm"
              placeholder="Keterangan"
              value={caption}
              onChange={(e) => setCaption(e.target.value)}
            />
            <button
              onClick={() => { if (imageUrl) addMut.mutate() }}
              disabled={!imageUrl || addMut.isPending}
              className="btn-primary text-sm px-3 py-2 disabled:opacity-50"
            >
              <Plus className="w-4 h-4" />
            </button>
          </div>

          {/* Gallery grid */}
          {isLoading ? (
            <p className="text-sm text-gray-400">Memuat galeri...</p>
          ) : gallery && gallery.length > 0 ? (
            <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
              {gallery.map((img) => (
                <div key={img.id} className="relative group rounded-xl overflow-hidden border">
                  <img src={img.image_url} alt={img.caption ?? ''} className="w-full h-32 object-cover" />
                  {img.caption && (
                    <p className="text-xs text-center text-gray-500 px-1 py-1 truncate">{img.caption}</p>
                  )}
                  <button
                    onClick={() => deleteMut.mutate(img.id)}
                    className="absolute top-1.5 right-1.5 bg-red-500 text-white rounded-full p-0.5 opacity-0 group-hover:opacity-100 transition-opacity"
                  >
                    <X className="w-3 h-3" />
                  </button>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-gray-400 text-center py-8">Belum ada foto di galeri ini.</p>
          )}
        </div>
      </div>
    </div>
  )
}

export default function AdminPackagesPage() {
  const [page, _setPage] = useState(1)
  const [showForm, setShowForm] = useState(false)
  const [editing, setEditing] = useState<TourPackage | null>(null)
  const [galleryPkg, setGalleryPkg] = useState<TourPackage | null>(null)
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

      {galleryPkg && (
        <GalleryModal pkg={galleryPkg} onClose={() => setGalleryPkg(null)} />
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
                    onClick={() => setGalleryPkg(pkg)}
                    className="p-1.5 text-gray-400 hover:text-blue-600 transition-colors"
                    title="Kelola Galeri"
                  >
                    <Image className="w-4 h-4" />
                  </button>
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
  const qc = useQueryClient()
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

  // ID paket yang sudah tersimpan (bisa dari initial atau dari hasil create baru)
  const [savedPkgId, setSavedPkgId] = useState<string | null>(initial?.id ?? null)

  // --- itinerary state ---
  const [editingItem, setEditingItem] = useState<ItineraryItem | null>(null)
  const [showAddForm, setShowAddForm] = useState(false)
  const [itineraryForm, setItineraryForm] = useState<Omit<ItineraryItem, 'id'>>(EMPTY_ITEM)
  const activityTypes = ['sightseeing', 'transport', 'kuliner', 'belanja', 'hotel', 'outdoor', 'budaya', 'lainnya']

  const saveMut = useMutation({
    mutationFn: () =>
      initial
        ? api.put(`/admin/packages/${initial.id}`, form)
        : api.post('/admin/packages', form),
    onSuccess: (res) => {
      qc.invalidateQueries({ queryKey: ['admin-packages'] })
      if (initial) {
        // edit: tutup form seperti biasa
        onSaved()
      } else {
        // baru: simpan ID dan tampilkan seksi itinerary, jangan tutup
        setSavedPkgId(res.data.id)
      }
    },
  })

  const { data: itineraryItems, isLoading: itineraryLoading } = useQuery<ItineraryItem[]>({
    queryKey: ['admin-itinerary', savedPkgId],
    queryFn: () => api.get(`/admin/packages/${savedPkgId}/itinerary`).then((r) => r.data),
    enabled: !!savedPkgId,
  })

  const invalidateItinerary = () => qc.invalidateQueries({ queryKey: ['admin-itinerary', savedPkgId] })

  const addItemMut = useMutation({
    mutationFn: () => api.post(`/admin/packages/${savedPkgId}/itinerary`, itineraryForm),
    onSuccess: () => { invalidateItinerary(); setShowAddForm(false); setItineraryForm(EMPTY_ITEM) },
  })

  const updateItemMut = useMutation({
    mutationFn: () => api.put(`/admin/packages/${savedPkgId}/itinerary/${editingItem!.id}`, itineraryForm),
    onSuccess: () => { invalidateItinerary(); setEditingItem(null) },
  })

  const deleteItemMut = useMutation({
    mutationFn: (itemId: string) => api.delete(`/admin/packages/${savedPkgId}/itinerary/${itemId}`),
    onSuccess: invalidateItinerary,
  })

  const openEditItem = (item: ItineraryItem) => {
    setEditingItem(item)
    setItineraryForm({
      day_number: item.day_number, title: item.title, description: item.description,
      location: item.location, start_time: item.start_time, end_time: item.end_time,
      activity_type: item.activity_type, sort_order: item.sort_order,
    })
    setShowAddForm(false)
  }

  const openAddItem = () => { setEditingItem(null); setItineraryForm(EMPTY_ITEM); setShowAddForm(true) }

  const grouped = (itineraryItems ?? []).reduce<Record<number, ItineraryItem[]>>((acc, it) => {
    ;(acc[it.day_number] ??= []).push(it)
    return acc
  }, {})

  const formContent = (
    <div className={initial ? 'bg-white rounded-2xl shadow-xl w-full max-w-2xl my-8 animate-modal-in' : 'bg-white rounded-2xl shadow-md overflow-hidden p-6 mb-6'}>
      {initial ? (
        <div className="flex items-center justify-between px-6 py-4 border-b">
          <h2 className="text-lg font-bold text-gray-900">Edit Paket</h2>
          <button onClick={onClose}><X className="w-5 h-5 text-gray-400 hover:text-gray-700" /></button>
        </div>
      ) : (
        <h2 className="font-semibold text-gray-800 mb-4">Tambah Paket Baru</h2>
      )}
      <div className={initial ? 'p-6' : ''}>
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

      {/* ── Itinerary section (muncul setelah paket disimpan) ── */}
      {savedPkgId && (
        <div className="mt-6 pt-5 border-t">
          <div className="flex items-center justify-between mb-4">
            <h3 className="font-semibold text-gray-800">Itinerary</h3>
            <button onClick={openAddItem} className="btn-primary text-xs px-3 py-1.5">
              <Plus className="w-3.5 h-3.5" /> Tambah Item
            </button>
          </div>

          {/* form tambah / edit item */}
          {(showAddForm || editingItem) && (
            <div className="border rounded-xl p-4 mb-5 bg-blue-50">
              <h4 className="font-semibold text-sm text-blue-800 mb-3">
                {editingItem ? 'Edit Item' : 'Tambah Item Baru'}
              </h4>
              <div className="grid grid-cols-2 gap-3 text-sm">
                <div>
                  <label className="label">Hari ke-</label>
                  <input type="number" min={1} className="input text-sm" value={itineraryForm.day_number}
                    onChange={(e) => setItineraryForm({ ...itineraryForm, day_number: Number(e.target.value) })} />
                </div>
                <div>
                  <label className="label">Tipe Aktivitas</label>
                  <select className="input text-sm" value={itineraryForm.activity_type}
                    onChange={(e) => setItineraryForm({ ...itineraryForm, activity_type: e.target.value })}>
                    {activityTypes.map((t) => <option key={t} value={t}>{t}</option>)}
                  </select>
                </div>
                <div className="col-span-2">
                  <label className="label">Judul Aktivitas *</label>
                  <input className="input text-sm" placeholder="cth: Kunjungi Senso-ji Temple" value={itineraryForm.title}
                    onChange={(e) => setItineraryForm({ ...itineraryForm, title: e.target.value })} />
                </div>
                <div className="col-span-2">
                  <label className="label">Deskripsi</label>
                  <textarea rows={2} className="input text-sm resize-none" value={itineraryForm.description}
                    onChange={(e) => setItineraryForm({ ...itineraryForm, description: e.target.value })} />
                </div>
                <div>
                  <label className="label">Lokasi</label>
                  <input className="input text-sm" value={itineraryForm.location}
                    onChange={(e) => setItineraryForm({ ...itineraryForm, location: e.target.value })} />
                </div>
                <div className="grid grid-cols-2 gap-2">
                  <div>
                    <label className="label">Mulai</label>
                    <input type="time" className="input text-sm" value={itineraryForm.start_time}
                      onChange={(e) => setItineraryForm({ ...itineraryForm, start_time: e.target.value })} />
                  </div>
                  <div>
                    <label className="label">Selesai</label>
                    <input type="time" className="input text-sm" value={itineraryForm.end_time}
                      onChange={(e) => setItineraryForm({ ...itineraryForm, end_time: e.target.value })} />
                  </div>
                </div>
                <div>
                  <label className="label">Urutan</label>
                  <input type="number" min={0} className="input text-sm" value={itineraryForm.sort_order}
                    onChange={(e) => setItineraryForm({ ...itineraryForm, sort_order: Number(e.target.value) })} />
                </div>
              </div>
              <div className="flex gap-2 mt-3">
                <button
                  onClick={() => editingItem ? updateItemMut.mutate() : addItemMut.mutate()}
                  disabled={!itineraryForm.title || addItemMut.isPending || updateItemMut.isPending}
                  className="btn-primary text-xs px-3 py-1.5 disabled:opacity-50"
                >
                  {addItemMut.isPending || updateItemMut.isPending ? 'Menyimpan...' : 'Simpan'}
                </button>
                <button
                  onClick={() => { setShowAddForm(false); setEditingItem(null) }}
                  className="btn-secondary text-xs px-3 py-1.5"
                >Batal</button>
              </div>
            </div>
          )}

          {/* daftar itinerary per hari */}
          {itineraryLoading ? (
            <p className="text-sm text-gray-400">Memuat itinerary...</p>
          ) : Object.keys(grouped).length > 0 ? (
            <div className="space-y-4">
              {Object.entries(grouped).sort(([a], [b]) => Number(a) - Number(b)).map(([day, dayItems]) => (
                <div key={day}>
                  <div className="flex items-center gap-2 mb-2">
                    <span className="bg-primary-600 text-white text-xs font-bold px-2.5 py-0.5 rounded-full">
                      Hari {day}
                    </span>
                    <div className="flex-1 h-px bg-gray-200" />
                  </div>
                  <div className="space-y-2 pl-2">
                    {dayItems.sort((a, b) => a.sort_order - b.sort_order).map((it) => (
                      <div key={it.id} className="flex items-start justify-between gap-3 bg-gray-50 rounded-lg px-3 py-2.5">
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 flex-wrap">
                            <span className="text-sm font-medium text-gray-800">{it.title}</span>
                            <span className="text-xs bg-gray-200 text-gray-600 px-1.5 py-0.5 rounded-full">{it.activity_type}</span>
                          </div>
                          {(it.start_time || it.location) && (
                            <p className="text-xs text-gray-400 mt-0.5">
                              {it.start_time && it.end_time ? `${it.start_time} – ${it.end_time}` : it.start_time}
                              {it.location && ` · ${it.location}`}
                            </p>
                          )}
                          {it.description && (
                            <p className="text-xs text-gray-500 mt-0.5 line-clamp-2">{it.description}</p>
                          )}
                        </div>
                        <div className="flex gap-1 shrink-0">
                          <button onClick={() => openEditItem(it)} className="p-1 text-gray-400 hover:text-blue-600">
                            <Pencil className="w-3.5 h-3.5" />
                          </button>
                          <button
                            onClick={() => { if (confirm('Hapus item ini?')) deleteItemMut.mutate(it.id) }}
                            className="p-1 text-gray-400 hover:text-red-600"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-gray-400 text-center py-6">Belum ada itinerary. Klik "Tambah Item" untuk mulai.</p>
          )}
        </div>
      )}

      <div className="flex gap-3 mt-5">
        {!savedPkgId || initial ? (
          <button onClick={() => saveMut.mutate()} disabled={saveMut.isPending} className="btn-primary text-sm">
            {saveMut.isPending ? 'Menyimpan...' : 'Simpan'}
          </button>
        ) : (
          <button onClick={onSaved} className="btn-primary text-sm">Selesai</button>
        )}
        <button onClick={onClose} className="btn-secondary text-sm">Batal</button>
      </div>
      </div>
    </div>
  )

  if (initial) {
    return (
      <div className="fixed inset-0 bg-black/40 animate-backdrop-in flex items-start justify-center z-50 p-4 overflow-y-auto">
        {formContent}
      </div>
    )
  }
  return formContent
}
