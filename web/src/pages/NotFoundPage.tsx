import { Link } from 'react-router-dom'
import { MapPin } from 'lucide-react'

export default function NotFoundPage() {
  return (
    <div className="min-h-screen flex flex-col items-center justify-center text-center px-4">
      <MapPin className="w-16 h-16 text-primary-300 mb-6" />
      <h1 className="text-6xl font-extrabold text-gray-900 mb-3">404</h1>
      <p className="text-gray-500 mb-8 text-lg">Halaman tidak ditemukan.</p>
      <Link to="/" className="btn-primary">Kembali ke Beranda</Link>
    </div>
  )
}
