import { TourPackage } from '../types'
import { Link } from 'react-router-dom'
import { Clock, Users, Tag } from 'lucide-react'

interface Props {
  pkg: TourPackage
}

export default function PackageCard({ pkg }: Props) {
  return (
    <div className="card hover:shadow-lg transition-shadow duration-300 flex flex-col">
      {/* Cover image */}
      {pkg.cover_image_url ? (
        <img
          src={pkg.cover_image_url}
          alt={pkg.title}
          className="w-full h-52 object-cover"
        />
      ) : (
        <div className="w-full h-52 bg-gradient-to-br from-primary-200 to-primary-400 flex items-center justify-center">
          <span className="text-white text-4xl font-bold opacity-30">{pkg.title.charAt(0)}</span>
        </div>
      )}

      <div className="p-5 flex flex-col flex-1">
        {/* Destination badge */}
        {pkg.destination_name && (
          <span className="inline-flex items-center gap-1 text-xs font-medium text-primary-700 bg-primary-50 px-2.5 py-0.5 rounded-full mb-2 w-fit">
            <Tag className="w-3 h-3" />
            {pkg.destination_name}, {pkg.destination_country}
          </span>
        )}

        <h3 className="font-bold text-gray-900 text-lg leading-snug mb-1">{pkg.title}</h3>

        {pkg.description && (
          <p className="text-sm text-gray-500 line-clamp-2 mb-3">{pkg.description}</p>
        )}

        <div className="flex items-center gap-4 text-sm text-gray-500 mt-auto pt-3 border-t border-gray-100">
          <span className="flex items-center gap-1.5">
            <Clock className="w-4 h-4" />
            {pkg.duration_days} hari
          </span>
          {pkg.min_participants > 1 && (
            <span className="flex items-center gap-1.5">
              <Users className="w-4 h-4" />
              min. {pkg.min_participants} orang
            </span>
          )}
          <span className="ml-auto font-semibold text-primary-700 text-base">
            {pkg.price_label ?? `Rp ${pkg.price.toLocaleString('id-ID')}`}
          </span>
        </div>

        <Link
          to={`/packages/${pkg.slug}`}
          className="btn-primary mt-4 w-full justify-center text-sm"
        >
          Lihat Detail
        </Link>
      </div>
    </div>
  )
}
