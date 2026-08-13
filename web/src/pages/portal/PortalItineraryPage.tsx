import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { MapPin, Clock, ChevronDown, ChevronUp, Calendar } from 'lucide-react'
import { portalApi } from '../../utils/api'
import { formatDate, formatDateLong, WITH_WEEKDAY } from '../../utils/date'
import { paymentRequiredMessage } from '../../hooks/usePortalMe'
import PortalLockedNotice from '../../components/PortalLockedNotice'
import type { ItineraryDay } from '../../types'

interface ItineraryResponse {
  package_name: string
  destination: string
  duration_days: number
  departure_date: string | null
  return_date: string | null
  itinerary: ItineraryDay[]
  requirements: string[]
}

export default function PortalItineraryPage() {
  const [expanded, setExpanded] = useState<number | null>(1)

  const { data, isLoading, error } = useQuery({
    queryKey: ['portal-itinerary'],
    queryFn: () =>
      portalApi.get<{ success: boolean; data: ItineraryResponse }>('/portal/itinerary')
        .then(r => r.data.data),
    // A locked page is a settled answer, not a hiccup: retrying a 403 four times
    // only delays the explanation the participant is owed.
    retry: (_count, err) => paymentRequiredMessage(err) === null,
  })

  // Reached by typing the URL, or by a menu that has not caught up yet. Either
  // way the answer is the reason, not "gagal memuat".
  const locked = paymentRequiredMessage(error)
  if (locked) return <PortalLockedNotice title="Itinerary" reason={locked} />

  if (isLoading) return (
    <div className="space-y-3 animate-pulse">
      <div className="h-20 bg-gray-200 rounded-xl" />
      {[1, 2, 3, 4].map(i => <div key={i} className="h-14 bg-gray-200 rounded-xl" />)}
    </div>
  )

  if (error || !data) return (
    <div className="text-center py-16 text-gray-400">
      <p>Gagal memuat itinerary. Coba refresh halaman.</p>
    </div>
  )

  const itinerary = Array.isArray(data.itinerary) ? data.itinerary : []
  const requirements = Array.isArray(data.requirements) ? data.requirements : []

  return (
    <div className="space-y-4">
      {/* Header info */}
      <div className="bg-white rounded-xl border p-4">
        <h2 className="font-bold text-gray-800 mb-1">{data.package_name}</h2>
        <div className="flex flex-wrap gap-3 text-sm text-gray-500">
          <span className="flex items-center gap-1"><MapPin size={13} /> {data.destination}</span>
          <span className="flex items-center gap-1"><Clock size={13} /> {data.duration_days} hari</span>
          {data.departure_date && (
            <span className="flex items-center gap-1">
              <Calendar size={13} />
              {formatDateLong(data.departure_date)}
            </span>
          )}
        </div>
      </div>

      {/* Departure info */}
      {data.departure_date && (
        <div className="bg-emerald-50 border border-emerald-200 rounded-xl p-4">
          <p className="text-sm font-medium text-emerald-700">
            ✈️ Keberangkatan:{' '}
            <strong>
              {formatDate(data.departure_date, WITH_WEEKDAY)}
            </strong>
          </p>
          {data.return_date && (
            <p className="text-sm text-emerald-600 mt-0.5">
              🏠 Kepulangan:{' '}
              {formatDate(data.return_date, WITH_WEEKDAY)}
            </p>
          )}
        </div>
      )}

      {/* Itinerary per day */}
      <h3 className="font-semibold text-gray-700 text-sm">Jadwal Perjalanan</h3>

      {itinerary.length === 0 ? (
        <div className="bg-white rounded-xl border p-8 text-center text-gray-400">
          <p>Itinerary belum tersedia.</p>
          <p className="text-xs mt-1">Hubungi tim Pintour untuk informasi lebih lanjut.</p>
        </div>
      ) : (
        <div className="space-y-2">
          {itinerary.map((day) => (
            <div key={day.day} className="bg-white rounded-xl border overflow-hidden">
              <button
                onClick={() => setExpanded(expanded === day.day ? null : day.day)}
                className="w-full flex items-center gap-3 px-4 py-3 text-left hover:bg-gray-50 transition-colors"
              >
                <span className="w-8 h-8 rounded-full bg-emerald-100 text-emerald-700 text-sm font-bold flex items-center justify-center shrink-0">
                  {day.day}
                </span>
                <div className="flex-1 min-w-0">
                  <p className="font-medium text-sm text-gray-800 truncate">{day.title}</p>
                </div>
                {expanded === day.day
                  ? <ChevronUp size={16} className="text-gray-400 shrink-0" />
                  : <ChevronDown size={16} className="text-gray-400 shrink-0" />
                }
              </button>
              {expanded === day.day && (
                <div className="px-4 pb-4 border-t bg-gray-50">
                  {day.description && (
                    <p className="text-sm text-gray-600 mt-3 mb-2">{day.description}</p>
                  )}
                  {Array.isArray(day.activities) && day.activities.length > 0 && (
                    <ul className="space-y-1.5">
                      {day.activities.map((activity, i) => (
                        <li key={i} className="flex items-start gap-2 text-sm text-gray-600">
                          <MapPin size={12} className="text-emerald-500 mt-0.5 shrink-0" />
                          {activity}
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Requirements */}
      {requirements.length > 0 && (
        <div className="bg-white rounded-xl border p-4">
          <h3 className="font-semibold text-gray-800 text-sm mb-3">📋 Persyaratan & Dokumen</h3>
          <ul className="space-y-2">
            {requirements.map((req, i) => (
              <li key={i} className="flex items-start gap-2 text-sm text-gray-600">
                <span className="text-emerald-500 mt-0.5 shrink-0">✓</span>
                {req}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  )
}
