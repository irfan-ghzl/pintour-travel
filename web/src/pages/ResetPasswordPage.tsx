import { useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { MapPin, CheckCircle, AlertCircle, ArrowLeft } from 'lucide-react'
import axios from 'axios'

export default function ResetPasswordPage() {
  const [params] = useSearchParams()
  const token = params.get('token') ?? ''
  const navigate = useNavigate()

  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState('')
  const [done, setDone] = useState(false)

  const mutation = useMutation({
    mutationFn: () =>
      axios.post('/api/v1/auth/reset-password', { token, password }),
    onSuccess: () => {
      setDone(true)
      setTimeout(() => navigate('/login'), 3000)
    },
    onError: (e: any) => {
      setError(e.response?.data?.message ?? 'Token tidak valid atau sudah kadaluarsa')
    },
  })

  if (!token) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
        <div className="bg-white rounded-2xl shadow-sm border w-full max-w-sm p-8 text-center">
          <AlertCircle className="text-red-500 mx-auto mb-3" size={36} />
          <h2 className="text-lg font-bold text-gray-800 mb-2">Token Tidak Ada</h2>
          <p className="text-sm text-gray-500 mb-6">
            Link reset password tidak valid. Silakan request ulang dari halaman lupa password.
          </p>
          <Link
            to="/forgot-password"
            className="inline-flex items-center gap-2 px-4 py-2 bg-emerald-600 text-white rounded-lg text-sm"
          >
            Request Link Baru
          </Link>
        </div>
      </div>
    )
  }

  if (done) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
        <div className="bg-white rounded-2xl shadow-sm border w-full max-w-sm p-8 text-center">
          <div className="w-14 h-14 bg-emerald-50 rounded-full flex items-center justify-center mx-auto mb-4">
            <CheckCircle className="text-emerald-600" size={28} />
          </div>
          <h2 className="text-lg font-bold text-gray-800 mb-2">Password Berhasil Diubah</h2>
          <p className="text-sm text-gray-500">
            Anda akan dialihkan ke halaman login dalam 3 detik...
          </p>
        </div>
      </div>
    )
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    if (password.length < 8) {
      setError('Password minimal 8 karakter')
      return
    }
    if (password !== confirm) {
      setError('Password dan konfirmasi tidak sama')
      return
    }
    mutation.mutate()
  }

  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
      <div className="bg-white rounded-2xl shadow-sm border w-full max-w-sm p-8">
        <div className="text-center mb-6">
          <div className="w-12 h-12 bg-emerald-700 rounded-xl flex items-center justify-center mx-auto mb-3">
            <MapPin className="text-white" size={22} />
          </div>
          <h1 className="text-xl font-bold text-gray-800">Buat Password Baru</h1>
          <p className="text-sm text-gray-500 mt-1">
            Masukkan password baru untuk akun Anda
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">Password Baru</label>
            <input
              required
              type="password"
              minLength={8}
              placeholder="Minimal 8 karakter"
              className="w-full border rounded-lg px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-400"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">Konfirmasi Password</label>
            <input
              required
              type="password"
              minLength={8}
              placeholder="Ulangi password baru"
              className="w-full border rounded-lg px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-400"
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
            />
          </div>

          {error && (
            <p className="flex items-center gap-1.5 text-sm text-red-600 bg-red-50 rounded-lg px-3 py-2">
              <AlertCircle size={14} /> {error}
            </p>
          )}

          <button
            type="submit"
            disabled={mutation.isPending}
            className="w-full py-2.5 bg-emerald-600 text-white rounded-lg font-semibold text-sm hover:bg-emerald-700 transition-colors disabled:opacity-50"
          >
            {mutation.isPending ? 'Memproses...' : 'Ubah Password'}
          </button>
        </form>

        <div className="text-center mt-5">
          <Link to="/login" className="text-sm text-gray-500 hover:text-emerald-600 flex items-center justify-center gap-1">
            <ArrowLeft size={13} /> Kembali ke Login
          </Link>
        </div>
      </div>
    </div>
  )
}
