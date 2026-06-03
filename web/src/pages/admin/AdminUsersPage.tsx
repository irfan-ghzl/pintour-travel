import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Plus, Pencil, UserX, KeyRound, Shield } from 'lucide-react'
import toast from 'react-hot-toast'
import api from '../../utils/api'
import type { User, UserRole } from '../../types'

const ROLES: { value: UserRole; label: string; color: string }[] = [
  { value: 'super_admin', label: 'Super Admin', color: 'bg-red-100 text-red-700' },
  { value: 'admin', label: 'Admin', color: 'bg-blue-100 text-blue-700' },
  { value: 'konsultan', label: 'Konsultan', color: 'bg-purple-100 text-purple-700' },
  { value: 'tour_leader', label: 'Tour Leader', color: 'bg-green-100 text-green-700' },
]

function getRoleBadge(role: string) {
  const r = ROLES.find(x => x.value === role)
  return r ? (
    <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${r.color}`}>{r.label}</span>
  ) : (
    <span className="px-2 py-0.5 rounded-full text-xs bg-gray-100 text-gray-600">{role}</span>
  )
}

export default function AdminUsersPage() {
  const [roleFilter, setRoleFilter] = useState('')
  const [showCreate, setShowCreate] = useState(false)
  const [editUser, setEditUser] = useState<User | null>(null)
  const [resetUser, setResetUser] = useState<User | null>(null)
  const [newPassword, setNewPassword] = useState('')
  const [form, setForm] = useState({ name: '', email: '', password: '', role: 'konsultan' as UserRole, phone: '' })
  const qc = useQueryClient()

  const params = roleFilter ? `?role=${roleFilter}` : ''
  const { data, isLoading } = useQuery({
    queryKey: ['admin-users', roleFilter],
    queryFn: () => api.get<{ success: boolean; data: User[] }>(`/admin/users${params}`).then(r => r.data.data ?? []),
  })

  const createMut = useMutation({
    mutationFn: () => api.post('/admin/users', form),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['admin-users'] }); toast.success('Akun berhasil dibuat'); setShowCreate(false) },
    onError: (e: any) => toast.error(e.response?.data?.message ?? 'Gagal membuat akun'),
  })

  const updateMut = useMutation({
    mutationFn: (u: User) => api.put(`/admin/users/${u.id}`, { name: u.name, email: u.email, role: u.role, phone: u.phone }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['admin-users'] }); toast.success('Akun diperbarui'); setEditUser(null) },
    onError: () => toast.error('Gagal memperbarui akun'),
  })

  const deactivateMut = useMutation({
    mutationFn: (id: string) => api.patch(`/admin/users/${id}/deactivate`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['admin-users'] }); toast.success('Akun dinonaktifkan') },
    onError: () => toast.error('Gagal menonaktifkan akun'),
  })

  const resetPwMut = useMutation({
    mutationFn: ({ id, password }: { id: string; password: string }) =>
      api.patch(`/admin/users/${id}/reset-password`, { password }),
    onSuccess: () => { toast.success('Password berhasil direset'); setResetUser(null); setNewPassword('') },
    onError: () => toast.error('Gagal reset password'),
  })

  const users = data ?? []

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-gray-800 flex items-center gap-2">
          <Shield size={20} className="text-emerald-600" /> Manajemen Pengguna
        </h2>
        <button
          onClick={() => { setForm({ name: '', email: '', password: '', role: 'konsultan', phone: '' }); setShowCreate(true) }}
          className="flex items-center gap-2 px-4 py-2 bg-emerald-600 text-white text-sm rounded-lg hover:bg-emerald-700"
        >
          <Plus size={16} /> Tambah Pengguna
        </button>
      </div>

      {/* Role filter */}
      <div className="flex gap-2 flex-wrap">
        {[{ value: '', label: 'Semua' }, ...ROLES].map(r => (
          <button
            key={r.value}
            onClick={() => setRoleFilter(r.value)}
            className={`px-3 py-1.5 rounded-full text-xs font-medium border ${
              roleFilter === r.value ? 'bg-emerald-700 text-white border-emerald-700' : 'bg-white text-gray-600 border-gray-200'
            }`}
          >
            {r.label}
          </button>
        ))}
      </div>

      {/* Table */}
      <div className="bg-white rounded-xl border overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 text-xs text-gray-500 uppercase tracking-wider">
            <tr>
              <th className="px-4 py-3 text-left">Nama</th>
              <th className="px-4 py-3 text-left">Email</th>
              <th className="px-4 py-3 text-left">Telepon</th>
              <th className="px-4 py-3 text-left">Role</th>
              <th className="px-4 py-3 text-left">Status</th>
              <th className="px-4 py-3"></th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {isLoading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <tr key={i} className="animate-pulse">
                  <td colSpan={6} className="px-4 py-3"><div className="h-4 bg-gray-100 rounded" /></td>
                </tr>
              ))
            ) : users.length === 0 ? (
              <tr><td colSpan={6} className="text-center py-10 text-gray-400">Tidak ada pengguna</td></tr>
            ) : users.map(u => (
              <tr key={u.id} className="hover:bg-gray-50">
                <td className="px-4 py-3 font-medium text-gray-800">{u.name}</td>
                <td className="px-4 py-3 text-gray-600">{u.email}</td>
                <td className="px-4 py-3 text-gray-500 text-xs">{u.phone || '—'}</td>
                <td className="px-4 py-3">{getRoleBadge(u.role)}</td>
                <td className="px-4 py-3">
                  <span className={`text-xs ${u.is_active ? 'text-green-600' : 'text-red-500'}`}>
                    {u.is_active ? '● Aktif' : '○ Nonaktif'}
                  </span>
                </td>
                <td className="px-4 py-3">
                  <div className="flex items-center gap-1">
                    <button onClick={() => setEditUser({ ...u })} className="p-1 text-gray-400 hover:text-blue-600" title="Edit">
                      <Pencil size={14} />
                    </button>
                    <button onClick={() => setResetUser(u)} className="p-1 text-gray-400 hover:text-orange-500" title="Reset Password">
                      <KeyRound size={14} />
                    </button>
                    {u.is_active && (
                      <button
                        onClick={() => window.confirm(`Nonaktifkan akun ${u.name}?`) && deactivateMut.mutate(u.id)}
                        className="p-1 text-gray-400 hover:text-red-500"
                        title="Nonaktifkan"
                      >
                        <UserX size={14} />
                      </button>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Create modal */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl w-full max-w-sm p-6 space-y-4">
            <div className="flex justify-between"><h3 className="font-semibold">Tambah Pengguna</h3>
              <button onClick={() => setShowCreate(false)} className="text-gray-400">✕</button></div>
            <div className="space-y-3">
              {[
                { key: 'name', label: 'Nama Lengkap', placeholder: 'Nama lengkap' },
                { key: 'email', label: 'Email', placeholder: 'email@example.com', type: 'email' },
                { key: 'password', label: 'Password', placeholder: 'Minimal 8 karakter', type: 'password' },
                { key: 'phone', label: 'No. WA', placeholder: '628123456789' },
              ].map(f => (
                <div key={f.key}>
                  <label className="text-xs font-medium text-gray-600 block mb-1">{f.label} *</label>
                  <input type={f.type ?? 'text'} className="w-full border rounded-lg px-3 py-2 text-sm"
                    placeholder={f.placeholder}
                    value={(form as any)[f.key]}
                    onChange={e => setForm({ ...form, [f.key]: e.target.value })} />
                </div>
              ))}
              <div>
                <label className="text-xs font-medium text-gray-600 block mb-1">Role *</label>
                <select className="w-full border rounded-lg px-3 py-2 text-sm"
                  value={form.role} onChange={e => setForm({ ...form, role: e.target.value as UserRole })}>
                  {ROLES.map(r => <option key={r.value} value={r.value}>{r.label}</option>)}
                </select>
              </div>
            </div>
            <button
              onClick={() => createMut.mutate()}
              disabled={!form.name || !form.email || !form.password || createMut.isPending}
              className="w-full py-2 bg-emerald-600 text-white rounded-lg text-sm font-medium disabled:opacity-50"
            >
              {createMut.isPending ? 'Membuat...' : 'Buat Akun'}
            </button>
          </div>
        </div>
      )}

      {/* Edit modal */}
      {editUser && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl w-full max-w-sm p-6 space-y-4">
            <div className="flex justify-between"><h3 className="font-semibold">Edit Pengguna</h3>
              <button onClick={() => setEditUser(null)} className="text-gray-400">✕</button></div>
            <div className="space-y-3">
              {(['name', 'email', 'phone'] as const).map(k => (
                <div key={k}>
                  <label className="text-xs font-medium text-gray-600 block mb-1 capitalize">{k === 'phone' ? 'No. WA' : k}</label>
                  <input className="w-full border rounded-lg px-3 py-2 text-sm"
                    value={editUser[k] ?? ''}
                    onChange={e => setEditUser({ ...editUser, [k]: e.target.value })} />
                </div>
              ))}
              <div>
                <label className="text-xs font-medium text-gray-600 block mb-1">Role</label>
                <select className="w-full border rounded-lg px-3 py-2 text-sm"
                  value={editUser.role}
                  onChange={e => setEditUser({ ...editUser, role: e.target.value as UserRole })}>
                  {ROLES.map(r => <option key={r.value} value={r.value}>{r.label}</option>)}
                </select>
              </div>
            </div>
            <button onClick={() => updateMut.mutate(editUser)}
              disabled={updateMut.isPending}
              className="w-full py-2 bg-emerald-600 text-white rounded-lg text-sm font-medium disabled:opacity-50">
              {updateMut.isPending ? 'Menyimpan...' : 'Simpan Perubahan'}
            </button>
          </div>
        </div>
      )}

      {/* Reset password modal */}
      {resetUser && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl w-full max-w-sm p-6 space-y-4">
            <div className="flex justify-between"><h3 className="font-semibold">Reset Password</h3>
              <button onClick={() => setResetUser(null)} className="text-gray-400">✕</button></div>
            <p className="text-sm text-gray-600">Reset password untuk: <strong>{resetUser.name}</strong></p>
            <div>
              <label className="text-xs font-medium text-gray-600 block mb-1">Password Baru *</label>
              <input type="password" className="w-full border rounded-lg px-3 py-2 text-sm"
                placeholder="Minimal 8 karakter"
                value={newPassword}
                onChange={e => setNewPassword(e.target.value)} />
            </div>
            <button
              onClick={() => resetPwMut.mutate({ id: resetUser.id, password: newPassword })}
              disabled={newPassword.length < 8 || resetPwMut.isPending}
              className="w-full py-2 bg-orange-500 text-white rounded-lg text-sm font-medium disabled:opacity-50">
              {resetPwMut.isPending ? 'Mereset...' : 'Reset Password'}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
