import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Plus, Pencil, Trash2, Globe } from 'lucide-react'
import toast from 'react-hot-toast'
import api from '../../utils/api'
import type { CountryRequirement } from '../../types'

const DOC_TYPES = [
  { value: 'passport', label: 'Paspor' },
  { value: 'ktp', label: 'KTP' },
  { value: 'rekening_koran', label: 'Rekening Koran' },
  { value: 'visa_support', label: 'Dokumen Pendukung Visa' },
  { value: 'lainnya', label: 'Lainnya' },
]

interface FormState {
  country_code: string
  country_name: string
  document_type: string
  is_required: boolean
  description: string
}

const EMPTY_FORM: FormState = {
  country_code: '', country_name: '',
  document_type: 'passport', is_required: true, description: '',
}

export default function AdminCountryRequirementsPage() {
  const [filterCountry, setFilterCountry] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [editing, setEditing] = useState<CountryRequirement | null>(null)
  const [form, setForm] = useState<FormState>(EMPTY_FORM)
  const qc = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: ['country-requirements', filterCountry],
    queryFn: () => {
      const url = filterCountry
        ? `/admin/country-requirements?country_code=${filterCountry}`
        : '/admin/country-requirements'
      return api.get<{ success: boolean; data: CountryRequirement[] }>(url)
        .then(r => r.data.data ?? [])
    },
  })

  const saveMut = useMutation({
    mutationFn: () => {
      const payload = {
        country_code: form.country_code.toUpperCase(),
        country_name: form.country_name,
        document_type: form.document_type,
        is_required: form.is_required,
        description: form.description,
      }
      return editing
        ? api.put(`/admin/country-requirements/${editing.id}`, payload)
        : api.post('/admin/country-requirements', payload)
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['country-requirements'] })
      toast.success(editing ? 'Persyaratan diperbarui' : 'Persyaratan ditambahkan')
      setShowForm(false)
      setEditing(null)
      setForm(EMPTY_FORM)
    },
    onError: (e: any) => toast.error(e.response?.data?.message ?? 'Gagal menyimpan'),
  })

  const deleteMut = useMutation({
    mutationFn: (id: string) => api.delete(`/admin/country-requirements/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['country-requirements'] })
      toast.success('Persyaratan dihapus')
    },
    onError: () => toast.error('Gagal menghapus'),
  })

  function openEdit(r: CountryRequirement) {
    setEditing(r)
    setForm({
      country_code: r.country_code,
      country_name: r.country_name,
      document_type: r.document_type,
      is_required: r.is_required,
      description: r.description ?? '',
    })
    setShowForm(true)
  }

  function openCreate() {
    setEditing(null)
    setForm(EMPTY_FORM)
    setShowForm(true)
  }

  const reqs = data ?? []

  // Group by country
  const grouped = reqs.reduce<Record<string, CountryRequirement[]>>((acc, r) => {
    const key = `${r.country_code}|${r.country_name}`
    if (!acc[key]) acc[key] = []
    acc[key].push(r)
    return acc
  }, {})

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-gray-800 flex items-center gap-2">
          <Globe size={20} className="text-emerald-600" /> Persyaratan Dokumen per Negara
        </h2>
        <button
          onClick={openCreate}
          className="flex items-center gap-2 px-4 py-2 bg-emerald-600 text-white text-sm rounded-lg hover:bg-emerald-700"
        >
          <Plus size={16} /> Tambah
        </button>
      </div>

      <p className="text-sm text-gray-500">
        Atur dokumen yang dibutuhkan peserta untuk setiap negara tujuan.
        Checklist ini muncul di portal peserta sesuai destinasi paket mereka.
      </p>

      <input
        className="border rounded-lg px-3 py-2 text-sm w-48"
        placeholder="Filter kode negara (cth: JP)"
        value={filterCountry}
        onChange={(e) => setFilterCountry(e.target.value.toUpperCase())}
      />

      {isLoading ? (
        <div className="space-y-3">
          {[1, 2, 3].map(i => <div key={i} className="h-20 bg-gray-100 rounded-xl animate-pulse" />)}
        </div>
      ) : Object.keys(grouped).length === 0 ? (
        <div className="bg-white rounded-xl border p-10 text-center text-gray-400">
          <Globe className="mx-auto mb-3 opacity-40" size={36} />
          <p>Belum ada persyaratan dokumen</p>
          <p className="text-xs mt-1">Klik "Tambah" untuk mulai konfigurasi per negara</p>
        </div>
      ) : (
        <div className="space-y-4">
          {Object.entries(grouped).map(([key, items]) => {
            const [code, name] = key.split('|')
            return (
              <div key={key} className="bg-white rounded-xl border overflow-hidden">
                <div className="px-4 py-3 bg-gray-50 border-b flex items-center gap-2">
                  <span className="px-2 py-0.5 bg-emerald-100 text-emerald-700 text-xs font-bold rounded">{code}</span>
                  <span className="font-semibold text-gray-800 text-sm">{name}</span>
                  <span className="text-xs text-gray-400">{items.length} dokumen</span>
                </div>
                <div className="divide-y">
                  {items.map(r => (
                    <div key={r.id} className="px-4 py-3 flex items-start gap-3">
                      <div className="flex-1 min-w-0">
                        <p className="font-medium text-sm text-gray-800">
                          {DOC_TYPES.find(t => t.value === r.document_type)?.label ?? r.document_type}
                          {r.is_required ? (
                            <span className="ml-2 text-xs text-red-500">*wajib</span>
                          ) : (
                            <span className="ml-2 text-xs text-gray-400">opsional</span>
                          )}
                        </p>
                        {r.description && (
                          <p className="text-xs text-gray-500 mt-0.5">{r.description}</p>
                        )}
                      </div>
                      <div className="flex gap-1 shrink-0">
                        <button onClick={() => openEdit(r)} className="p-1 text-gray-400 hover:text-blue-600">
                          <Pencil size={14} />
                        </button>
                        <button
                          onClick={() => window.confirm(`Hapus persyaratan ini?`) && deleteMut.mutate(r.id)}
                          className="p-1 text-gray-400 hover:text-red-500"
                        >
                          <Trash2 size={14} />
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )
          })}
        </div>
      )}

      {/* Form modal */}
      {showForm && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl w-full max-w-sm p-6 space-y-4">
            <div className="flex justify-between">
              <h3 className="font-semibold">
                {editing ? 'Edit Persyaratan' : 'Tambah Persyaratan'}
              </h3>
              <button onClick={() => { setShowForm(false); setEditing(null) }} className="text-gray-400">✕</button>
            </div>

            <div className="space-y-3">
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-xs font-medium text-gray-600 block mb-1">Kode Negara *</label>
                  <input
                    className="w-full border rounded-lg px-3 py-2 text-sm uppercase"
                    placeholder="JP"
                    maxLength={3}
                    value={form.country_code}
                    onChange={e => setForm({ ...form, country_code: e.target.value.toUpperCase() })}
                  />
                </div>
                <div>
                  <label className="text-xs font-medium text-gray-600 block mb-1">Nama Negara *</label>
                  <input
                    className="w-full border rounded-lg px-3 py-2 text-sm"
                    placeholder="Jepang"
                    value={form.country_name}
                    onChange={e => setForm({ ...form, country_name: e.target.value })}
                  />
                </div>
              </div>
              <div>
                <label className="text-xs font-medium text-gray-600 block mb-1">Jenis Dokumen *</label>
                <select
                  className="w-full border rounded-lg px-3 py-2 text-sm"
                  value={form.document_type}
                  onChange={e => setForm({ ...form, document_type: e.target.value })}
                >
                  {DOC_TYPES.map(t => <option key={t.value} value={t.value}>{t.label}</option>)}
                </select>
              </div>
              <div>
                <label className="text-xs font-medium text-gray-600 block mb-1">Keterangan</label>
                <textarea
                  rows={2}
                  className="w-full border rounded-lg px-3 py-2 text-sm resize-none"
                  placeholder="cth: Paspor dengan masa berlaku minimal 6 bulan"
                  value={form.description}
                  onChange={e => setForm({ ...form, description: e.target.value })}
                />
              </div>
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={form.is_required}
                  onChange={e => setForm({ ...form, is_required: e.target.checked })}
                />
                <span className="text-sm text-gray-700">Wajib diisi</span>
              </label>
            </div>

            <button
              onClick={() => saveMut.mutate()}
              disabled={!form.country_code || !form.country_name || saveMut.isPending}
              className="w-full py-2 bg-emerald-600 text-white rounded-lg text-sm font-medium disabled:opacity-50"
            >
              {saveMut.isPending ? 'Menyimpan...' : (editing ? 'Simpan Perubahan' : 'Tambah Persyaratan')}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
