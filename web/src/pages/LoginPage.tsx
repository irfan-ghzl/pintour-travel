import { useForm } from 'react-hook-form'
import { useMutation } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { MapPin } from 'lucide-react'
import api from '../utils/api'
import { authStorage } from '../utils/auth'
import { LoginRequest, LoginResponse } from '../types'

export default function LoginPage() {
  const navigate = useNavigate()
  const { register, handleSubmit, formState: { errors } } = useForm<LoginRequest>()

  const mutation = useMutation<LoginResponse, Error, LoginRequest>({
    mutationFn: (data) => api.post('/auth/login', data).then((r) => r.data),
    onSuccess: (data) => {
      authStorage.setSession(data)
      navigate('/admin')
    },
  })

  return (
    <div className="min-h-screen bg-gradient-to-br from-primary-800 to-primary-600 flex items-center justify-center px-4">
      <div className="w-full max-w-md bg-white rounded-2xl shadow-2xl p-8">
        <div className="flex items-center gap-2 mb-8">
          <MapPin className="w-7 h-7 text-accent-500" />
          <span className="text-xl font-bold text-primary-700">Pintour Admin</span>
        </div>

        <h1 className="text-2xl font-bold text-gray-900 mb-1">Masuk</h1>
        <p className="text-gray-500 text-sm mb-8">Masukkan kredensial Anda untuk mengakses dashboard.</p>

        <form onSubmit={handleSubmit((data) => mutation.mutate(data))} className="space-y-5">
          <div>
            <label className="label">Email</label>
            <input
              {...register('email', { required: 'Email wajib diisi' })}
              type="email"
              className="input"
              placeholder="admin@pintour.com"
            />
            {errors.email && <p className="text-red-500 text-xs mt-1">{errors.email.message}</p>}
          </div>

          <div>
            <label className="label">Password</label>
            <input
              {...register('password', { required: 'Password wajib diisi' })}
              type="password"
              className="input"
              placeholder="••••••••"
            />
            {errors.password && <p className="text-red-500 text-xs mt-1">{errors.password.message}</p>}
          </div>

          {mutation.isError && (
            <p className="text-red-500 text-sm">Email atau password salah.</p>
          )}

          <button
            type="submit"
            disabled={mutation.isPending}
            className="btn-primary w-full justify-center py-3"
          >
            {mutation.isPending ? 'Masuk...' : 'Masuk ke Dashboard'}
          </button>
        </form>
      </div>
    </div>
  )
}
