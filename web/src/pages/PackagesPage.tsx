import { useQuery } from '@tanstack/react-query'
import api from '../utils/api'
import PackageCard from '../components/PackageCard'
import Spinner from '../components/Spinner'
import { Destination, PackagesResponse } from '../types'
import { useState } from 'react'

interface Filters {
  destination_id: string
  package_type: string
  price_min: string
  price_max: string
  duration_days: string
}

const INITIAL_FILTERS: Filters = {
  destination_id: '',
  package_type: '',
  price_min: '',
  price_max: '',
  duration_days: '',
}

export default function PackagesPage() {
  const [page, setPage] = useState(1)
  const perPage = 9
  const [filters, setFilters] = useState<Filters>(INITIAL_FILTERS)
  const [applied, setApplied] = useState<Filters>(INITIAL_FILTERS)

  const { data: destinations } = useQuery<Destination[]>({
    queryKey: ['destinations'],
    queryFn: () => api.get('/destinations').then((r) => r.data),
    staleTime: Infinity,
  })

  const buildQuery = (f: Filters, p: number) => {
    const params = new URLSearchParams({ page: String(p), per_page: String(perPage) })
    if (f.destination_id) params.set('destination_id', f.destination_id)
    if (f.package_type) params.set('package_type', f.package_type)
    if (f.price_min) params.set('price_min', f.price_min)
    if (f.price_max) params.set('price_max', f.price_max)
    if (f.duration_days) params.set('duration_days', f.duration_days)
    return `/packages?${params.toString()}`
  }

  const { data, isLoading, isFetching } = useQuery<PackagesResponse>({
    queryKey: ['packages', page, perPage, applied],
    queryFn: () => api.get(buildQuery(applied, page)).then((r) => r.data),
  })

  const totalPages = data ? Math.ceil(data.total / perPage) : 1

  const handleApply = () => {
    setPage(1)
    setApplied({ ...filters })
  }

  const handleReset = () => {
    setFilters(INITIAL_FILTERS)
    setApplied(INITIAL_FILTERS)
    setPage(1)
  }

  return (
    <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-14">
      <h1 className="text-3xl font-bold text-gray-900 mb-2">Paket Wisata</h1>
      <p className="text-gray-500 mb-8">Pilih destinasi impian Anda dari koleksi paket kami.</p>

      {/* Filter Bar */}
      <div className="bg-white border border-gray-200 rounded-xl p-5 mb-8 shadow-sm">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4">
          {/* Destinasi */}
          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">Destinasi</label>
            <select
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              value={filters.destination_id}
              onChange={(e) => setFilters((f) => ({ ...f, destination_id: e.target.value }))}
            >
              <option value="">Semua Destinasi</option>
              {destinations?.map((d) => (
                <option key={d.id} value={d.id}>
                  {d.name}, {d.country}
                </option>
              ))}
            </select>
          </div>

          {/* Tipe Paket */}
          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">Tipe Paket</label>
            <select
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              value={filters.package_type}
              onChange={(e) => setFilters((f) => ({ ...f, package_type: e.target.value }))}
            >
              <option value="">Semua Tipe</option>
              <option value="regular">Regular</option>
              <option value="private">Private</option>
              <option value="honeymoon">Honeymoon</option>
              <option value="educational">Educational</option>
            </select>
          </div>

          {/* Budget Min */}
          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">Budget Min (Rp)</label>
            <input
              type="number"
              min="0"
              placeholder="0"
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              value={filters.price_min}
              onChange={(e) => setFilters((f) => ({ ...f, price_min: e.target.value }))}
            />
          </div>

          {/* Budget Max */}
          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">Budget Max (Rp)</label>
            <input
              type="number"
              min="0"
              placeholder="Tidak terbatas"
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              value={filters.price_max}
              onChange={(e) => setFilters((f) => ({ ...f, price_max: e.target.value }))}
            />
          </div>

          {/* Durasi */}
          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">Durasi (hari)</label>
            <input
              type="number"
              min="1"
              placeholder="Semua"
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              value={filters.duration_days}
              onChange={(e) => setFilters((f) => ({ ...f, duration_days: e.target.value }))}
            />
          </div>
        </div>

        <div className="flex gap-3 mt-4">
          <button
            onClick={handleApply}
            className="btn-primary py-2 px-5 text-sm"
          >
            Cari Paket
          </button>
          <button
            onClick={handleReset}
            className="btn-secondary py-2 px-5 text-sm"
          >
            Reset
          </button>
        </div>
      </div>

      {isLoading ? (
        <Spinner message="Memuat paket wisata..." />
      ) : (
        <>
          {isFetching && (
            <p className="text-sm text-blue-500 mb-4 animate-pulse">Memperbarui hasil...</p>
          )}
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
            {data?.data.map((pkg) => (
              <PackageCard key={pkg.id} pkg={pkg} />
            ))}
            {(!data?.data || data.data.length === 0) && (
              <p className="col-span-3 text-center py-20 text-gray-400">
                Tidak ada paket yang sesuai dengan filter Anda.
              </p>
            )}
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex justify-center gap-2 mt-12">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page === 1 || isFetching}
                className="btn-secondary py-2 px-4 text-sm disabled:opacity-40"
              >
                Sebelumnya
              </button>
              <span className="flex items-center px-4 text-sm text-gray-500">
                Halaman {page} / {totalPages}
              </span>
              <button
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page === totalPages || isFetching}
                className="btn-secondary py-2 px-4 text-sm disabled:opacity-40"
              >
                Berikutnya
              </button>
            </div>
          )}
        </>
      )}
    </div>
  )
}

