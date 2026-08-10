import { Link } from 'react-router-dom'
import { ShieldX } from 'lucide-react'

export default function ForbiddenPage() {
  return (
    <div className="min-h-[60vh] flex flex-col items-center justify-center text-center px-4">
      <ShieldX className="w-16 h-16 text-red-300 mb-6" />
      <h1 className="text-6xl font-extrabold text-gray-900 mb-3">403</h1>
      <p className="text-gray-500 mb-2 text-lg">Akses ditolak.</p>
      <p className="text-gray-400 mb-8 text-sm">
        Peran akun Anda tidak memiliki izin untuk membuka halaman ini.
      </p>
      <Link
        to="/admin"
        className="px-4 py-2 bg-emerald-600 text-white rounded-lg font-semibold text-sm hover:bg-emerald-700 transition-colors"
      >
        Kembali ke Dashboard
      </Link>
    </div>
  )
}
