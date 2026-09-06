import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { MapPin } from 'lucide-react'
import axios from 'axios'
import type { PortalLoginRequest, PortalLoginResponse } from '../../types'

export default function PortalLoginPage() {
  const [form, setForm] = useState<PortalLoginRequest>({ phone: '', password: '' })
  const [error, setError] = useState('')
  const navigate = useNavigate()

  const loginMutation = useMutation({
    mutationFn: (req: PortalLoginRequest) =>
      axios.post<{ success: boolean; data: PortalLoginResponse }>('/api/v1/portal/login', req)
        .then(r => r.data.data),
    onSuccess: (data, vars) => {
      localStorage.setItem('portal_token', data.token)
      localStorage.setItem('portal_participant', JSON.stringify({
        id: data.participant_id,
        name: data.name,
        package_name: data.package_name,
        phone: vars.phone, // v2.0 F3 — for prefilling the next-tour consultation form
      }))
      navigate('/portal')
    },
    onError: (e: any) => {
      setError(e.response?.data?.message ?? 'Nomor WA atau password salah')
    },
  })

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    const phone = form.phone.replace(/^0/, '62').replace(/^\+/, '')
    loginMutation.mutate({ ...form, phone })
  }

  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
      <div className="bg-white rounded-2xl shadow-sm border w-full max-w-sm p-8">
        {/* Logo */}
        <div className="text-center mb-8">
          <div className="w-12 h-12 bg-emerald-700 rounded-xl flex items-center justify-center mx-auto mb-3">
            <MapPin className="text-white" size={24} />
          </div>
          <h1 className="text-xl font-bold text-gray-800">Portal Peserta</h1>
          <p className="text-sm text-gray-500 mt-1">Masuk untuk mengakses perjalanan Anda</p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">Nomor WhatsApp</label>
            <input
              required
              type="tel"
              placeholder="08123456789"
              className="w-full border rounded-lg px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-400"
              value={form.phone}
              onChange={(e) => setForm({ ...form, phone: e.target.value })}
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">Password</label>
            <input
              required
              type="password"
              placeholder="Password yang dikirim via WA"
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
            disabled={loginMutation.isPending}
            className="w-full py-2.5 bg-emerald-600 text-white rounded-lg font-semibold text-sm hover:bg-emerald-700 transition-colors disabled:opacity-50"
          >
            {loginMutation.isPending ? 'Memproses...' : 'Masuk ke Portal'}
          </button>
        </form>

        {/* The old line said the password arrives "setelah pembayaran
            dikonfirmasi", which was never how it worked: it is sent the moment
            the consultant converts the booking, precisely so the participant can
            log in and pay. Repeating it here told anyone who could not get in
            that they were simply too early. */}
        <p className="text-center text-xs text-gray-400 mt-6">
          Password dikirim via WhatsApp saat pendaftaran Anda dikonfirmasi konsultan.<br />
          Hubungi tim Pintour jika mengalami kendala.
        </p>
      </div>
    </div>
  )
}
