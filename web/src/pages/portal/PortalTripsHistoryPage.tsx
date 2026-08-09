import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { Plane, Download, History, Plus } from 'lucide-react'
import { getMyTrips, downloadTripInvoice, type TripCard } from '../../utils/api'

function paymentBadge(status: string) {
  if (status === 'lunas') return 'bg-green-100 text-green-700'
  if (status === '-') return 'bg-gray-100 text-gray-500'
  return 'bg-orange-100 text-orange-600'
}

function TripRow({ trip, past }: { trip: TripCard; past: boolean }) {
  return (
    <div className="flex items-center gap-3 p-3 rounded-lg border bg-white">
      <div className={`w-10 h-10 rounded-lg flex items-center justify-center shrink-0 ${past ? 'bg-gray-100' : 'bg-emerald-100'}`}>
        <Plane size={18} className={past ? 'text-gray-500' : 'text-emerald-600'} />
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium text-gray-800 truncate">{trip.package_name}</p>
        <p className="text-xs text-gray-500">
          {trip.departure_date
            ? new Date(trip.departure_date).toLocaleDateString('id-ID', { day: 'numeric', month: 'long', year: 'numeric' })
            : 'Tanggal belum diatur'}
          {' · '}
          <span className="capitalize">{trip.room_type}</span>
        </p>
      </div>
      <span className={`px-2 py-0.5 rounded-full text-xs font-medium capitalize shrink-0 ${paymentBadge(trip.payment_status)}`}>
        {trip.payment_status === 'lunas' ? 'Lunas' : trip.payment_status === '-' ? 'Belum ada invoice' : trip.payment_status.replace('_', ' ')}
      </span>
      <button
        onClick={() => downloadTripInvoice(trip.participant_id)}
        className="flex items-center gap-1 text-xs text-emerald-700 hover:text-emerald-800 px-2 py-1 rounded hover:bg-emerald-50 shrink-0"
        title="Unduh invoice"
      >
        <Download size={14} /> Invoice
      </button>
    </div>
  )
}

export default function PortalTripsHistoryPage() {
  const { data, isLoading } = useQuery({ queryKey: ['portal-my-trips'], queryFn: getMyTrips })

  if (isLoading) return (
    <div className="space-y-3 animate-pulse">
      <div className="h-20 bg-gray-200 rounded-xl" />
      <div className="h-20 bg-gray-200 rounded-xl" />
    </div>
  )

  const active = data?.active ?? []
  const history = data?.history ?? []

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-gray-800 flex items-center gap-2">
          <History size={18} className="text-emerald-600" /> Riwayat Perjalanan
        </h2>
        <Link
          to="/"
          className="flex items-center gap-1.5 px-3 py-1.5 bg-emerald-600 text-white text-xs rounded-lg hover:bg-emerald-700"
        >
          <Plus size={14} /> Tour Baru
        </Link>
      </div>

      {/* Tour Aktif */}
      <section>
        <h3 className="text-sm font-semibold text-gray-700 mb-2">Tour Aktif</h3>
        {active.length === 0 ? (
          <p className="text-sm text-gray-400 bg-white border rounded-lg p-4 text-center">Tidak ada tour aktif saat ini.</p>
        ) : (
          <div className="space-y-2">
            {active.map(t => <TripRow key={t.participant_id} trip={t} past={false} />)}
          </div>
        )}
      </section>

      {/* Riwayat */}
      <section>
        <h3 className="text-sm font-semibold text-gray-700 mb-2">Tour Lampau</h3>
        {history.length === 0 ? (
          <p className="text-sm text-gray-400 bg-white border rounded-lg p-4 text-center">Belum ada tour yang selesai.</p>
        ) : (
          <div className="space-y-2">
            {history.map(t => <TripRow key={t.participant_id} trip={t} past={true} />)}
          </div>
        )}
      </section>

      {/* CTA konsultasi */}
      <Link
        to="/"
        className="block bg-gradient-to-r from-emerald-600 to-emerald-700 text-white rounded-xl p-4 text-center text-sm font-medium hover:from-emerald-700 hover:to-emerald-800 transition-colors"
      >
        Tertarik tour lagi? Lihat paket terbaru — akun Anda tetap sama
      </Link>
    </div>
  )
}
