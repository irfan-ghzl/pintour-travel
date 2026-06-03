import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Pencil, User } from 'lucide-react'
import toast from 'react-hot-toast'
import api from '../../utils/api'
import type { TourLeader } from '../../types'

export default function AdminTourLeadersPage() {
  const [editing, setEditing] = useState<TourLeader | null>(null)
  const qc = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: ['tour-leaders'],
    queryFn: () =>
      api.get<{ success: boolean; data: TourLeader[] }>('/admin/tour-leaders')
        .then(r => r.data.data ?? []),
  })

  const saveMut = useMutation({
    mutationFn: (tl: TourLeader) =>
      api.put(`/admin/tour-leaders/${tl.user_id}`, tl),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['tour-leaders'] })
      toast.success('Profil tour leader diperbarui')
      setEditing(null)
    },
    onError: () => toast.error('Gagal menyimpan profil'),
  })

  const leaders = data ?? []

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-gray-800 flex items-center gap-2">
          <User size={20} className="text-emerald-600" /> Profil Tour Leader
        </h2>
      </div>

      <p className="text-sm text-gray-500">
        Profil tour leader ditampilkan kepada peserta di halaman Briefing Digital.
        Pastikan data lengkap sebelum H-14 keberangkatan.
      </p>

      {/* Leader cards */}
      {isLoading ? (
        <div className="space-y-3">
          {[1, 2, 3].map(i => <div key={i} className="h-24 bg-gray-100 rounded-xl animate-pulse" />)}
        </div>
      ) : leaders.length === 0 ? (
        <div className="bg-white rounded-xl border p-10 text-center text-gray-400">
          <User className="mx-auto mb-3 opacity-40" size={36} />
          <p>Belum ada tour leader terdaftar</p>
          <p className="text-xs mt-1">
            Tambahkan pengguna dengan role "Tour Leader" di halaman Manajemen Pengguna,
            lalu isi profil mereka di sini.
          </p>
        </div>
      ) : (
        <div className="space-y-3">
          {leaders.map(tl => (
            <div key={tl.id} className="bg-white rounded-xl border p-4 flex items-start gap-4">
              {tl.photo_path ? (
                <img src={tl.photo_path} alt={tl.name}
                  className="w-14 h-14 rounded-full object-cover border-2 border-emerald-200 shrink-0" />
              ) : (
                <div className="w-14 h-14 rounded-full bg-emerald-100 flex items-center justify-center shrink-0">
                  <User className="text-emerald-600" size={24} />
                </div>
              )}
              <div className="flex-1 min-w-0">
                <div className="flex items-start justify-between gap-2">
                  <div>
                    <p className="font-semibold text-gray-800">{tl.name}</p>
                    <p className="text-xs text-gray-500">{tl.email}</p>
                  </div>
                  <button
                    onClick={() => setEditing({ ...tl })}
                    className="p-1.5 text-gray-400 hover:text-blue-600 shrink-0"
                  >
                    <Pencil size={15} />
                  </button>
                </div>
                <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-gray-500">
                  {tl.specialization && <span>🎯 {tl.specialization}</span>}
                  {tl.experience_years > 0 && <span>⭐ {tl.experience_years} tahun pengalaman</span>}
                  {tl.emergency_phone && <span>📞 {tl.emergency_phone}</span>}
                </div>
                {tl.bio && (
                  <p className="mt-1.5 text-xs text-gray-600 line-clamp-2">{tl.bio}</p>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Edit modal */}
      {editing && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4 overflow-y-auto">
          <div className="bg-white rounded-xl w-full max-w-md p-6 space-y-4 my-4">
            <div className="flex justify-between items-center">
              <h3 className="font-semibold">Edit Profil Tour Leader</h3>
              <button onClick={() => setEditing(null)} className="text-gray-400">✕</button>
            </div>
            <p className="text-sm text-gray-500">
              Nama: <strong>{editing.name}</strong>
            </p>
            <div className="space-y-3">
              <div>
                <label className="text-xs font-medium text-gray-600 block mb-1">Bio / Deskripsi Singkat</label>
                <textarea rows={3} className="w-full border rounded-lg px-3 py-2 text-sm resize-none"
                  placeholder="Deskripsi singkat pengalaman tour leader..."
                  value={editing.bio ?? ''}
                  onChange={e => setEditing({ ...editing, bio: e.target.value })} />
              </div>
              <div>
                <label className="text-xs font-medium text-gray-600 block mb-1">URL Foto Profil</label>
                <input className="w-full border rounded-lg px-3 py-2 text-sm"
                  placeholder="https://... (link foto)"
                  value={editing.photo_path ?? ''}
                  onChange={e => setEditing({ ...editing, photo_path: e.target.value })} />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-xs font-medium text-gray-600 block mb-1">Tahun Pengalaman</label>
                  <input type="number" min="0" className="w-full border rounded-lg px-3 py-2 text-sm"
                    value={editing.experience_years ?? 0}
                    onChange={e => setEditing({ ...editing, experience_years: Number(e.target.value) })} />
                </div>
                <div>
                  <label className="text-xs font-medium text-gray-600 block mb-1">No. Darurat</label>
                  <input className="w-full border rounded-lg px-3 py-2 text-sm"
                    placeholder="628123456789"
                    value={editing.emergency_phone ?? ''}
                    onChange={e => setEditing({ ...editing, emergency_phone: e.target.value })} />
                </div>
              </div>
              <div>
                <label className="text-xs font-medium text-gray-600 block mb-1">Spesialisasi</label>
                <input className="w-full border rounded-lg px-3 py-2 text-sm"
                  placeholder="contoh: Umroh, Eropa, Jepang..."
                  value={editing.specialization ?? ''}
                  onChange={e => setEditing({ ...editing, specialization: e.target.value })} />
              </div>
            </div>
            <button
              onClick={() => saveMut.mutate(editing)}
              disabled={saveMut.isPending}
              className="w-full py-2 bg-emerald-600 text-white rounded-lg text-sm font-medium disabled:opacity-50"
            >
              {saveMut.isPending ? 'Menyimpan...' : 'Simpan Profil'}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
