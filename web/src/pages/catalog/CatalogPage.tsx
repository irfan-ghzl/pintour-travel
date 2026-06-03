import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { MapPin, Clock, Users, Search, SlidersHorizontal } from 'lucide-react'
import api from '../../utils/api'
import type { Package, PaginatedResponse } from '../../types'

const CATEGORIES = [
  { value: '', label: 'Semua' },
  { value: 'reguler', label: 'Reguler' },
  { value: 'umroh', label: 'Umroh' },
  { value: 'halal', label: 'Halal' },
  { value: 'honeymoon', label: 'Honeymoon' },
]

// Generate months for the next 12 months (FR-CAT-02)
function getUpcomingMonths() {
  const months = []
  const now = new Date()
  for (let i = 0; i < 12; i++) {
    const d = new Date(now.getFullYear(), now.getMonth() + i, 1)
    const value = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`
    const label = d.toLocaleDateString('id-ID', { month: 'long', year: 'numeric' })
    months.push({ value, label })
  }
  return months
}
const MONTHS = getUpcomingMonths()

const DURATIONS = [
  { value: '', label: 'Semua' },
  { value: '1-5', label: '1–5 hari' },
  { value: '6-9', label: '6–9 hari' },
  { value: '10-14', label: '10–14 hari' },
  { value: '15-99', label: '>14 hari' },
]

export default function CatalogPage() {
  const [category, setCategory] = useState('')
  const [destination, setDestination] = useState('')
  const [search, setSearch] = useState('')
  const [departureMonth, setDepartureMonth] = useState('')
  const [duration, setDuration] = useState('')
  const [page, setPage] = useState(1)

  const params = new URLSearchParams()
  if (category) params.set('category', category)
  if (destination || search) params.set('destination', destination || search)
  if (departureMonth) params.set('departure_month', departureMonth)
  // FR-CAT-02 filter durasi
  if (duration) {
    const [min, max] = duration.split('-')
    params.set('duration_min', min)
    params.set('duration_max', max)
  }
  params.set('page', String(page))
  params.set('per_page', '12')

  const { data, isLoading } = useQuery({
    queryKey: ['packages', category, destination, search, departureMonth, duration, page],
    queryFn: () =>
      api.get<PaginatedResponse<Package>>(`/packages?${params}`).then((r) => r.data),
  })

  const packages = data?.data ?? []
  const meta = data?.meta

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Hero */}
      <section className="bg-emerald-700 text-white py-16 px-4 text-center">
        <h1 className="text-3xl md:text-4xl font-bold mb-2">Katalog Paket Wisata</h1>
        <p className="text-emerald-200 mb-6">Temukan perjalanan impian Anda bersama Pintour</p>
        <div className="max-w-xl mx-auto flex gap-2">
          <div className="flex-1 relative">
            <Search className="absolute left-3 top-3 text-gray-400" size={16} />
            <input
              className="w-full pl-9 pr-4 py-2.5 rounded-lg text-gray-800 text-sm"
              placeholder="Cari destinasi..."
              value={search}
              onChange={(e) => { setSearch(e.target.value); setPage(1) }}
            />
          </div>
        </div>
      </section>

      <div className="max-w-7xl mx-auto px-4 py-8">
        {/* Filter bar — FR-CAT-02 */}
        <div className="flex items-center gap-3 mb-4 flex-wrap">
          <SlidersHorizontal size={16} className="text-gray-500" />
          {CATEGORIES.map((c) => (
            <button
              key={c.value}
              onClick={() => { setCategory(c.value); setPage(1) }}
              className={`px-4 py-1.5 rounded-full text-sm font-medium border transition-colors ${
                category === c.value
                  ? 'bg-emerald-700 text-white border-emerald-700'
                  : 'bg-white text-gray-600 border-gray-200 hover:border-emerald-400'
              }`}
            >
              {c.label}
            </button>
          ))}
        </div>

        {/* Filter durasi + bulan keberangkatan — FR-CAT-02 */}
        <div className="flex items-center gap-4 mb-6 flex-wrap">
          <div className="flex items-center gap-2">
            <span className="text-xs text-gray-500 font-medium">Durasi:</span>
            <select
              className="border rounded-lg px-3 py-1.5 text-sm bg-white"
              value={duration}
              onChange={(e) => { setDuration(e.target.value); setPage(1) }}
            >
              {DURATIONS.map(d => (
                <option key={d.value} value={d.value}>{d.label}</option>
              ))}
            </select>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-xs text-gray-500 font-medium">Bulan:</span>
            <select
              className="border rounded-lg px-3 py-1.5 text-sm bg-white"
              value={departureMonth}
              onChange={(e) => { setDepartureMonth(e.target.value); setPage(1) }}
            >
              <option value="">Semua Bulan</option>
              {MONTHS.map((m) => (
                <option key={m.value} value={m.value}>{m.label}</option>
              ))}
            </select>
          </div>
          {(departureMonth || duration) && (
            <button
              onClick={() => { setDepartureMonth(''); setDuration(''); setPage(1) }}
              className="text-xs text-emerald-600 hover:underline"
            >
              × Reset filter
            </button>
          )}
        </div>

        {/* Package grid */}
        {isLoading ? (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
            {Array.from({ length: 6 }).map((_, i) => (
              <div key={i} className="bg-white rounded-xl h-72 animate-pulse" />
            ))}
          </div>
        ) : packages.length === 0 ? (
          <div className="text-center py-20 text-gray-500">
            <p className="text-lg">Tidak ada paket ditemukan</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
            {packages.map((pkg) => (
              <Link
                key={pkg.id}
                to={`/packages/${pkg.slug}`}
                className="bg-white rounded-xl overflow-hidden shadow-sm border border-gray-100 hover:shadow-md transition-shadow group"
              >
                {/* Thumbnail */}
                <div className="aspect-video bg-gray-200 relative overflow-hidden">
                  {pkg.thumbnail ? (
                    <img
                      src={pkg.thumbnail}
                      alt={pkg.name}
                      className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
                    />
                  ) : (
                    <div className="w-full h-full flex items-center justify-center text-gray-400">
                      <MapPin size={32} />
                    </div>
                  )}
                  <span className="absolute top-2 right-2 bg-emerald-600 text-white text-xs px-2 py-0.5 rounded-full capitalize">
                    {pkg.category}
                  </span>
                </div>
                {/* Info */}
                <div className="p-4">
                  <h3 className="font-semibold text-gray-800 mb-1 line-clamp-2">{pkg.name}</h3>
                  <p className="text-sm text-gray-500 flex items-center gap-1 mb-3">
                    <MapPin size={12} /> {pkg.destination}
                  </p>
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3 text-xs text-gray-500">
                      <span className="flex items-center gap-1"><Clock size={12} /> {pkg.duration_days} hari</span>
                    </div>
                    <div className="text-right">
                      <p className="text-xs text-gray-400">Mulai dari</p>
                      <p className="text-emerald-700 font-bold text-sm">
                        Rp {pkg.base_price.toLocaleString('id-ID')}
                      </p>
                    </div>
                  </div>
                  <button className="mt-3 w-full py-2 bg-emerald-50 text-emerald-700 rounded-lg text-sm font-medium hover:bg-emerald-100 transition-colors">
                    Lihat Detail
                  </button>
                </div>
              </Link>
            ))}
          </div>
        )}

        {/* Pagination */}
        {meta && meta.total_pages > 1 && (
          <div className="flex justify-center gap-2 mt-8">
            <button
              disabled={page <= 1}
              onClick={() => setPage(p => p - 1)}
              className="px-4 py-2 rounded-lg border text-sm disabled:opacity-40"
            >
              ← Sebelumnya
            </button>
            <span className="px-4 py-2 text-sm text-gray-600">
              Halaman {page} dari {meta.total_pages}
            </span>
            <button
              disabled={page >= meta.total_pages}
              onClick={() => setPage(p => p + 1)}
              className="px-4 py-2 rounded-lg border text-sm disabled:opacity-40"
            >
              Berikutnya →
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
