import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { ArrowRight, Star, Users, Globe2 } from 'lucide-react'
import api from '../utils/api'
import PackageCard from '../components/PackageCard'
import Spinner from '../components/Spinner'
import { PackagesResponse, Testimonial } from '../types'

export default function HomePage() {
  const { data: pkgsData, isLoading: pkgsLoading } = useQuery<PackagesResponse>({
    queryKey: ['packages', 1, 6],
    queryFn: () => api.get('/packages?page=1&per_page=6').then((r) => r.data),
  })

  const { data: testimonials } = useQuery<Testimonial[]>({
    queryKey: ['testimonials', 1, 3],
    queryFn: () => api.get('/testimonials?page=1&per_page=3').then((r) => r.data),
  })

  return (
    <>
      {/* Hero */}
      <section className="relative bg-gradient-to-br from-primary-800 via-primary-700 to-primary-600 text-white overflow-hidden">
        <div className="absolute inset-0 opacity-10 bg-[radial-gradient(circle_at_30%_50%,white,transparent_60%)]" />
        <div className="relative max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-28 md:py-36">
          <h1 className="text-4xl md:text-6xl font-extrabold leading-tight max-w-2xl mb-6">
            Jelajahi Dunia Bersama <span className="text-accent-400">Pintour Travel</span>
          </h1>
          <p className="text-lg md:text-xl text-primary-100 max-w-xl mb-10">
            Konsultan wisata personal Anda. Paket tour terbaik, harga transparan,
            dan perjalanan tak terlupakan.
          </p>
          <div className="flex flex-wrap gap-4">
            <Link to="/packages" className="btn-primary text-base px-7 py-3">
              Lihat Paket Wisata
              <ArrowRight className="w-5 h-5" />
            </Link>
            <Link to="/build-my-trip" className="btn-secondary text-base px-7 py-3">
              Build My Trip
            </Link>
          </div>

          {/* Stats */}
          <div className="grid grid-cols-3 gap-6 mt-16 max-w-lg">
            {[
              { icon: Users, value: '1,000+', label: 'Pelanggan Puas' },
              { icon: Globe2, value: '50+', label: 'Destinasi' },
              { icon: Star, value: '4.9', label: 'Rating' },
            ].map(({ icon: Icon, value, label }) => (
              <div key={label} className="text-center">
                <Icon className="w-6 h-6 text-accent-400 mx-auto mb-1" />
                <p className="text-2xl font-bold">{value}</p>
                <p className="text-xs text-primary-200">{label}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Featured packages */}
      <section className="py-20 bg-white">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-end justify-between mb-10">
            <div>
              <h2 className="text-3xl font-bold text-gray-900">Paket Unggulan</h2>
              <p className="text-gray-500 mt-1">Destinasi terpopuler pilihan pelanggan kami</p>
            </div>
            <Link to="/packages" className="text-primary-600 font-semibold text-sm hover:underline flex items-center gap-1">
              Lihat semua <ArrowRight className="w-4 h-4" />
            </Link>
          </div>

          {pkgsLoading ? (
            <Spinner message="Memuat paket..." />
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
              {pkgsData?.data.map((pkg) => (
                <PackageCard key={pkg.id} pkg={pkg} />
              ))}
              {(!pkgsData?.data || pkgsData.data.length === 0) && (
                <p className="text-gray-400 col-span-3 text-center py-10">Belum ada paket tersedia.</p>
              )}
            </div>
          )}
        </div>
      </section>

      {/* Testimonials snippet */}
      {testimonials && testimonials.length > 0 && (
        <section className="py-20 bg-gray-50">
          <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8">
            <h2 className="text-3xl font-bold text-gray-900 mb-10 text-center">Apa Kata Mereka?</h2>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
              {testimonials.map((t) => (
                <div key={t.id} className="card p-6">
                  <div className="flex gap-1 mb-3">
                    {Array.from({ length: 5 }).map((_, i) => (
                      <Star key={i} className={`w-4 h-4 ${i < t.rating ? 'text-yellow-400 fill-yellow-400' : 'text-gray-200'}`} />
                    ))}
                  </div>
                  <p className="text-gray-600 text-sm leading-relaxed mb-4">"{t.content}"</p>
                  <p className="font-semibold text-gray-800 text-sm">— {t.customer_name}</p>
                </div>
              ))}
            </div>
            <div className="text-center mt-8">
              <Link to="/testimonials" className="btn-secondary">
                Lihat Semua Testimonial
              </Link>
            </div>
          </div>
        </section>
      )}

      {/* CTA */}
      <section className="py-20 bg-accent-500">
        <div className="max-w-2xl mx-auto px-4 text-center text-white">
          <h2 className="text-3xl font-bold mb-4">Siap Merencanakan Perjalanan Impian?</h2>
          <p className="text-accent-100 mb-8">
            Ceritakan kebutuhan Anda dan konsultan kami akan menyiapkan penawaran terbaik.
          </p>
          <Link to="/build-my-trip" className="inline-flex items-center gap-2 bg-white text-accent-600 font-bold px-8 py-3.5 rounded-xl shadow hover:bg-accent-50 transition-colors text-base">
            Mulai Sekarang
            <ArrowRight className="w-5 h-5" />
          </Link>
        </div>
      </section>
    </>
  )
}
