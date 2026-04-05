import { useQuery } from '@tanstack/react-query'
import { Star } from 'lucide-react'
import api from '../utils/api'
import Spinner from '../components/Spinner'
import { Testimonial } from '../types'
import { useState } from 'react'

export default function TestimonialsPage() {
  const [page, setPage] = useState(1)
  const perPage = 9

  const { data, isLoading } = useQuery<Testimonial[]>({
    queryKey: ['testimonials', page, perPage],
    queryFn: () => api.get(`/testimonials?page=${page}&per_page=${perPage}`).then((r) => r.data),
  })

  return (
    <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-14">
      <h1 className="text-3xl font-bold text-gray-900 mb-2">Cerita Perjalanan Pelanggan</h1>
      <p className="text-gray-500 mb-10">Pengalaman nyata dari pelanggan setia kami.</p>

      {isLoading ? (
        <Spinner message="Memuat testimonial..." />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {data?.map((t) => (
            <div key={t.id} className="card p-6 flex flex-col gap-3">
              {t.photo_url && (
                <img
                  src={t.photo_url}
                  alt={t.customer_name}
                  className="w-full h-44 object-cover rounded-xl"
                />
              )}
              <div className="flex gap-1">
                {Array.from({ length: 5 }).map((_, i) => (
                  <Star key={i} className={`w-4 h-4 ${i < t.rating ? 'text-yellow-400 fill-yellow-400' : 'text-gray-200'}`} />
                ))}
              </div>
              <p className="text-gray-600 text-sm leading-relaxed flex-1">"{t.content}"</p>
              <p className="font-semibold text-gray-800 text-sm">— {t.customer_name}</p>
            </div>
          ))}
          {(!data || data.length === 0) && (
            <p className="col-span-3 text-center py-20 text-gray-400">Belum ada testimonial.</p>
          )}
        </div>
      )}
    </div>
  )
}
