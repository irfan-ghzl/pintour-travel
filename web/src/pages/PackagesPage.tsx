import { useQuery } from '@tanstack/react-query'
import api from '../utils/api'
import PackageCard from '../components/PackageCard'
import Spinner from '../components/Spinner'
import { PackagesResponse } from '../types'
import { useState } from 'react'

export default function PackagesPage() {
  const [page, setPage] = useState(1)
  const perPage = 9

  const { data, isLoading, isFetching } = useQuery<PackagesResponse>({
    queryKey: ['packages', page, perPage],
    queryFn: () => api.get(`/packages?page=${page}&per_page=${perPage}`).then((r) => r.data),
  })

  const totalPages = data ? Math.ceil(data.total / perPage) : 1

  return (
    <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-14">
      <h1 className="text-3xl font-bold text-gray-900 mb-2">Paket Wisata</h1>
      <p className="text-gray-500 mb-10">Pilih destinasi impian Anda dari koleksi paket kami.</p>

      {isLoading ? (
        <Spinner message="Memuat paket wisata..." />
      ) : (
        <>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
            {data?.data.map((pkg) => (
              <PackageCard key={pkg.id} pkg={pkg} />
            ))}
            {(!data?.data || data.data.length === 0) && (
              <p className="col-span-3 text-center py-20 text-gray-400">Belum ada paket tersedia.</p>
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
