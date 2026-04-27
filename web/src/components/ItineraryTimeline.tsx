import { ItineraryItem } from '../types'
import { MapPin, Clock } from 'lucide-react'

interface Props {
  items: ItineraryItem[]
}

// Group items by day
function groupByDay(items: ItineraryItem[]): Map<number, ItineraryItem[]> {
  const map = new Map<number, ItineraryItem[]>()
  for (const item of items) {
    const dayItems = map.get(item.day_number) ?? []
    dayItems.push(item)
    map.set(item.day_number, dayItems)
  }
  return map
}

export default function ItineraryTimeline({ items }: Props) {
  if (!items.length) {
    return <p className="text-gray-400 text-sm">Itinerary belum tersedia.</p>
  }

  const grouped = groupByDay(items)
  const days = Array.from(grouped.keys()).sort((a, b) => a - b)

  return (
    <div className="space-y-8">
      {days.map((day) => {
        const dayItems = grouped.get(day)!
        return (
          <div key={day}>
            {/* Day header */}
            <div className="flex items-center gap-3 mb-4">
              <div className="w-10 h-10 rounded-full bg-primary-600 text-white flex items-center justify-center font-bold text-sm flex-shrink-0">
                D{day}
              </div>
              <h4 className="font-semibold text-gray-800 text-base">Hari ke-{day}</h4>
            </div>

            {/* Timeline items */}
            <div className="ml-5 border-l-2 border-primary-100 pl-6 space-y-5">
              {dayItems.map((item) => (
                <div key={item.id} className="relative">
                  {/* Dot */}
                  <div className="absolute -left-[33px] w-4 h-4 rounded-full bg-white border-2 border-primary-400 mt-0.5" />

                  <div className="bg-gray-50 rounded-xl p-4 border border-gray-100">
                    <div className="flex items-start justify-between gap-2 flex-wrap">
                      <h5 className="font-semibold text-gray-900 text-sm">{item.title}</h5>
                      {(item.start_time || item.end_time) && (
                        <span className="flex items-center gap-1 text-xs text-gray-400 flex-shrink-0">
                          <Clock className="w-3 h-3" />
                          {item.start_time}{item.end_time ? ` – ${item.end_time}` : ''}
                        </span>
                      )}
                    </div>

                    {item.description && (
                      <p className="text-xs text-gray-500 mt-1.5 leading-relaxed">{item.description}</p>
                    )}

                    {item.location && (
                      <div className="flex items-center gap-1 mt-2 text-xs text-primary-600">
                        <MapPin className="w-3 h-3" />
                        {item.location}
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )
      })}
    </div>
  )
}
