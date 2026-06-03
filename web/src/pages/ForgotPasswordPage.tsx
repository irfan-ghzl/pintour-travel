import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { MapPin, ArrowLeft, CheckCircle } from 'lucide-react'
import axios from 'axios'

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState('')
  const [sent, setSent] = useState(false)

  const mutation = useMutation({
    mutationFn: (email: string) =>
      axios.post('/api/v1/auth/forgot-password', { email }),
    onSuccess: () => setSent(true),
  })

  if (sent) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
        <div className="bg-white rounded-2xl shadow-sm border w-full max-w-sm p-8 text-center">
          <div className="w-14 h-14 bg-emerald-50 rounded-full flex items-center justify-center mx-auto mb-4">
            <CheckCircle className="text-emerald-600" size={28} />
          </div>
          <h2 className="text-lg font-bold text-gray-800 mb-2">Email Terkirim</h2>
          <p className="text-sm text-gray-500 mb-6">
            Jika email <strong>{email}</strong> terdaftar, link reset password akan segera dikirimkan.
            Periksa folder inbox atau spam.
          </p>
          <Link
            to="/login"
            className="inline-flex items-center gap-2 text-sm text-emerald-600 hover:underline"
          >
            <ArrowLeft size={14} /> Kembali ke halaman login
          </Link>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
      <div className="bg-white rounded-2xl shadow-sm border w-full max-w-sm p-8">
        <div className="text-center mb-8">
          <div className="w-12 h-12 bg-emerald-700 rounded-xl flex items-center justify-center mx-auto mb-3">
            <MapPin className="text-white" size={22} />
          </div>
          <h1 className="text-xl font-bold text-gray-800">Reset Password</h1>
          <p className="text-sm text-gray-500 mt-1">
            Masukkan email yang terdaftar, kami akan kirim link reset password.
          </p>
        </div>

        <form
          onSubmit={(e) => { e.preventDefault(); mutation.mutate(email) }}
          className="space-y-4"
        >
          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">Alamat Email</label>
            <input
              required
              type="email"
              placeholder="email@example.com"
              className="w-full border rounded-lg px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-400"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </div>

          {mutation.isError && (
            <p className="text-sm text-red-600 bg-red-50 rounded-lg px-3 py-2">
              Terjadi kesalahan, coba lagi.
            </p>
          )}

          <button
            type="submit"
            disabled={mutation.isPending}
            className="w-full py-2.5 bg-emerald-600 text-white rounded-lg font-semibold text-sm hover:bg-emerald-700 transition-colors disabled:opacity-50"
          >
            {mutation.isPending ? 'Mengirim...' : 'Kirim Link Reset'}
          </button>
        </form>

        <div className="text-center mt-6">
          <Link to="/login" className="text-sm text-gray-500 hover:text-emerald-600 flex items-center justify-center gap-1">
            <ArrowLeft size={13} /> Kembali ke Login
          </Link>
        </div>
      </div>
    </div>
  )
}
