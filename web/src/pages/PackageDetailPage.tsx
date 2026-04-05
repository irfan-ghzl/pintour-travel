import { useParams, Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Clock, Users, Tag, MessageCircle, ArrowLeft } from 'lucide-react'
import api from '../utils/api'
import Spinner from '../components/Spinner'
import ItineraryTimeline from '../components/ItineraryTimeline'
import { TourPackage } from '../types'

export default function PackageDetailPage() {
  const { slug } = useParams<{ slug: string }>()

  const { data: pkg, isLoading, error } = useQuery<TourPackage>({
    queryKey: ['package', slug],
    queryFn: () => api.get(`/packages/${slug}`).then((r) => r.data),
    enabled: !!slug,
  })

  if (isLoading) return <Spinner message="Memuat detail paket..." />
  if (error || !pkg) {
    return (
      <div className="max-w-2xl mx-auto px-4 py-20 text-center">
        <p className="text-gray-500 mb-6">Paket tidak ditemukan.</p>
        <Link to="/packages" className="btn-secondary">
          <ArrowLeft className="w-4 h-4" /> Kembali ke Paket
        </Link>
      </div>
    )
  }

  return (
    <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
      <Link to="/packages" className="inline-flex items-center gap-1 text-sm text-primary-600 hover:underline mb-6">
        <ArrowLeft className="w-4 h-4" /> Kembali ke semua paket
      </Link>

      {/* Cover */}
      {pkg.cover_image_url ? (
        <img
          src={pkg.cover_image_url}
          alt={pkg.title}
          className="w-full h-72 object-cover rounded-2xl mb-8 shadow"
        />
      ) : (
        <div className="w-full h-72 bg-gradient-to-br from-primary-300 to-primary-500 rounded-2xl mb-8 flex items-center justify-center">
          <span className="text-white text-6xl font-extrabold opacity-30">{pkg.title.charAt(0)}</span>
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
        {/* Main content */}
        <div className="md:col-span-2">
          <h1 className="text-3xl font-bold text-gray-900 mb-3">{pkg.title}</h1>

          {pkg.destination_name && (
            <span className="inline-flex items-center gap-1.5 text-sm font-medium text-primary-700 bg-primary-50 px-3 py-1 rounded-full mb-4">
              <Tag className="w-3.5 h-3.5" />
              {pkg.destination_name}, {pkg.destination_country}
            </span>
          )}

          {pkg.description && (
            <p className="text-gray-600 leading-relaxed text-base mb-8">{pkg.description}</p>
          )}

          {/* Itinerary */}
          <h2 className="text-xl font-bold text-gray-800 mb-4">Itinerary Perjalanan</h2>
          <ItineraryTimeline items={pkg.itinerary ?? []} />
        </div>

        {/* Sticky sidebar */}
        <div className="md:col-span-1">
          <div className="card p-6 sticky top-20">
            <p className="text-2xl font-extrabold text-primary-700 mb-1">
              {pkg.price_label ?? `Rp ${pkg.price.toLocaleString('id-ID')}`}
            </p>
            <p className="text-xs text-gray-400 mb-6">/ orang (estimasi)</p>

            <div className="space-y-3 text-sm text-gray-600 mb-6">
              <div className="flex items-center gap-2.5">
                <Clock className="w-4 h-4 text-primary-500" />
                {pkg.duration_days} hari
              </div>
              <div className="flex items-center gap-2.5">
                <Users className="w-4 h-4 text-primary-500" />
                Min. {pkg.min_participants} orang
                {pkg.max_participants && `, maks. ${pkg.max_participants}`}
              </div>
            </div>

            <Link
              to={`/build-my-trip?package_id=${pkg.id}&package_title=${encodeURIComponent(pkg.title)}`}
              className="btn-primary w-full justify-center mb-3"
            >
              <MessageCircle className="w-4 h-4" />
              Konsultasi Paket Ini
            </Link>
            <p className="text-xs text-gray-400 text-center">
              Anda akan diarahkan ke WhatsApp konsultan kami.
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}
