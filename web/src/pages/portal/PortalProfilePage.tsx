import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { User, Download, Trash2 } from 'lucide-react'
import toast from 'react-hot-toast'
import { portalApi } from '../../utils/api'
import type { PortalMeResponse } from '../../types'

export default function PortalProfilePage() {
  const qc = useQueryClient()
  const [form, setForm] = useState({ name: '', email: '' })
  const [showDeleteRequest, setShowDeleteRequest] = useState(false)
  const [deleteReason, setDeleteReason] = useState('')

  const { data, isLoading } = useQuery({
    queryKey: ['portal-me'],
    queryFn: () =>
      portalApi.get<{ success: boolean; data: PortalMeResponse }>('/portal/me')
        .then(r => {
          const p = r.data.data.participant
          setForm({ name: p.name, email: p.email })
          return r.data.data
        }),
  })

  const updateMut = useMutation({
    mutationFn: () => portalApi.put('/portal/profile', form),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['portal-me'] })
      toast.success('Profil berhasil diperbarui')
    },
    onError: () => toast.error('Gagal memperbarui profil'),
  })

  const deleteMut = useMutation({
    mutationFn: () =>
      portalApi.post('/portal/account-deletion-request', { reason: deleteReason }),
    onSuccess: (res: any) => {
      toast.success(res.data?.data?.message ?? 'Permintaan dikirim')
      setShowDeleteRequest(false)
    },
    onError: () => toast.error('Gagal mengirim permintaan'),
  })

  const p = data?.participant

  if (isLoading) return (
    <div className="space-y-4 animate-pulse">
      <div className="h-20 bg-gray-200 rounded-xl" />
      <div className="h-32 bg-gray-200 rounded-xl" />
    </div>
  )

  return (
    <div className="space-y-4">
      <h2 className="font-semibold text-gray-800 flex items-center gap-2">
        <User size={18} /> Profil Saya
      </h2>

      {/* Info card */}
      <div className="bg-white rounded-xl border p-4">
        <div className="flex items-center gap-3 mb-4">
          <div className="w-12 h-12 rounded-full bg-emerald-100 flex items-center justify-center">
            <User className="text-emerald-600" size={22} />
          </div>
          <div>
            <p className="font-semibold text-gray-800">{p?.name}</p>
            <p className="text-sm text-gray-500">{p?.phone}</p>
          </div>
        </div>

        <form
          onSubmit={(e) => { e.preventDefault(); updateMut.mutate() }}
          className="space-y-3"
        >
          <div>
            <label className="text-xs font-medium text-gray-600 block mb-1">Nama Lengkap</label>
            <input
              className="w-full border rounded-lg px-3 py-2 text-sm"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
            />
          </div>
          <div>
            <label className="text-xs font-medium text-gray-600 block mb-1">Email</label>
            <input
              type="email"
              className="w-full border rounded-lg px-3 py-2 text-sm"
              placeholder="email@example.com"
              value={form.email}
              onChange={(e) => setForm({ ...form, email: e.target.value })}
            />
          </div>
          <div>
            <label className="text-xs font-medium text-gray-600 block mb-1">Nomor WhatsApp</label>
            <input
              disabled
              className="w-full border rounded-lg px-3 py-2 text-sm bg-gray-50 text-gray-400"
              value={p?.phone ?? ''}
            />
            <p className="text-xs text-gray-400 mt-1">Nomor WA tidak dapat diubah</p>
          </div>
          <button
            type="submit"
            disabled={updateMut.isPending}
            className="w-full py-2 bg-emerald-600 text-white rounded-lg text-sm font-medium disabled:opacity-50"
          >
            {updateMut.isPending ? 'Menyimpan...' : 'Simpan Perubahan'}
          </button>
        </form>
      </div>

      {/* Data rights §25.5 */}
      <div className="bg-white rounded-xl border p-4 space-y-3">
        <h3 className="font-semibold text-sm text-gray-800">Hak atas Data Pribadi Anda</h3>
        <p className="text-xs text-gray-500">
          Sesuai UU PDP No. 27/2022, Anda berhak mengakses, mengunduh, dan meminta
          penghapusan data pribadi Anda.
        </p>
        <div className="flex gap-2 flex-wrap">
          <a
            href="/api/v1/portal/my-data"
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-1.5 px-3 py-2 border rounded-lg text-xs text-gray-600 hover:bg-gray-50"
          >
            <Download size={13} /> Unduh Data Saya
          </a>
          <button
            onClick={() => setShowDeleteRequest(true)}
            className="flex items-center gap-1.5 px-3 py-2 border border-red-200 rounded-lg text-xs text-red-600 hover:bg-red-50"
          >
            <Trash2 size={13} /> Minta Penghapusan Data
          </button>
        </div>
      </div>

      {/* Delete request modal */}
      {showDeleteRequest && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl w-full max-w-sm p-6 space-y-4">
            <h3 className="font-semibold text-gray-800">Permintaan Penghapusan Data</h3>
            <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-3 text-xs text-yellow-700">
              ⚠️ Permintaan ini akan diproses oleh tim Pintour dalam 14 hari kerja.
              Data yang terkait dengan kewajiban hukum akan tetap disimpan sesuai regulasi.
            </div>
            <div>
              <label className="text-xs font-medium text-gray-600 block mb-1">Alasan (opsional)</label>
              <textarea
                rows={3}
                className="w-full border rounded-lg px-3 py-2 text-sm resize-none"
                placeholder="Alasan permintaan penghapusan..."
                value={deleteReason}
                onChange={(e) => setDeleteReason(e.target.value)}
              />
            </div>
            <div className="flex gap-2">
              <button
                onClick={() => setShowDeleteRequest(false)}
                className="flex-1 py-2 border rounded-lg text-sm"
              >
                Batal
              </button>
              <button
                onClick={() => deleteMut.mutate()}
                disabled={deleteMut.isPending}
                className="flex-1 py-2 bg-red-600 text-white rounded-lg text-sm disabled:opacity-50"
              >
                {deleteMut.isPending ? 'Mengirim...' : 'Kirim Permintaan'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
