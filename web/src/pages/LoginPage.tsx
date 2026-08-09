import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { MapPin } from 'lucide-react'
import api from '../utils/api'
import { authStorage } from '../utils/auth'
import type { LoginRequest, LoginResponse } from '../types'

export default function LoginPage() {
  const navigate = useNavigate()
  const [form, setForm] = useState<LoginRequest>({ email: '', password: '' })
  const [error, setError] = useState('')

  const mutation = useMutation<{ data: { data: LoginResponse } }, Error, LoginRequest>({
    mutationFn: (data) => api.post('/auth/login', data),
    onSuccess: (res) => {
      // Backend wraps the payload as { success, data }, so the LoginResponse
      // (incl. role) lives at res.data.data — unwrapping res.data stored {}.
      authStorage.setSession(res.data.data)
      navigate('/admin')
    },
    onError: () => setError('Email atau password salah.'),
  })

  return (
    <div className="min-h-screen bg-emerald-800 flex items-center justify-center px-4">
      <div className="w-full max-w-sm bg-white rounded-2xl shadow-2xl p-8">
        <div className="text-center mb-8">
          <div className="w-12 h-12 bg-emerald-700 rounded-xl flex items-center justify-center mx-auto mb-3">
            <MapPin className="text-white" size={24} />
          </div>
          <h1 className="text-xl font-bold text-gray-800">Pintour Admin</h1>
          <p className="text-sm text-gray-500 mt-1">Masukkan kredensial untuk mengakses dashboard</p>
        </div>

        <form
          onSubmit={(e) => { e.preventDefault(); setError(''); mutation.mutate(form) }}
          className="space-y-4"
        >
          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">Email</label>
            <input
              required
              type="email"
              placeholder="admin@pintour.app"
              className="w-full border rounded-lg px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-400"
              value={form.email}
              onChange={(e) => setForm({ ...form, email: e.target.value })}
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">Password</label>
            <input
              required
              type="password"
              placeholder="••••••••"
              className="w-full border rounded-lg px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-400"
              value={form.password}
              onChange={(e) => setForm({ ...form, password: e.target.value })}
            />
          </div>

          {error && (
            <p className="text-sm text-red-600 bg-red-50 rounded-lg px-3 py-2">{error}</p>
          )}

          <button
            type="submit"
            disabled={mutation.isPending}
            className="w-full py-2.5 bg-emerald-600 text-white rounded-lg font-semibold text-sm hover:bg-emerald-700 transition-colors disabled:opacity-50"
          >
            {mutation.isPending ? 'Masuk...' : 'Masuk ke Dashboard'}
          </button>
        </form>

        {/* §15.4 auth-forgot-password link */}
        <div className="text-center mt-5">
          <Link
            to="/forgot-password"
            className="text-sm text-emerald-600 hover:underline"
          >
            Lupa password?
          </Link>
        </div>
      </div>
    </div>
  )
}
