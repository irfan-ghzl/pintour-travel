import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Search, X } from 'lucide-react'
import { searchParticipants } from '../utils/api'
import type { Participant } from '../types'

// Satu-satunya pemilih peserta di antarmuka admin.
//
// Menggantikan kolom `Filter Participant ID...` dan `UUID peserta...`: admin
// mengetikkan sebagian nama atau sebagian nomor WhatsApp, lalu memilih dari
// daftar. Nomor WhatsApp selalu ikut tampil di samping nama supaya dua peserta
// bernama sama tetap bisa dibedakan.
//
// Pencarian berjalan lewat `GET /admin/participants`, yang sudah disaring per
// konsultan — kotak pencarian ini karena itu tidak bisa menjadi jalan pintas
// melewati pembatasan akses.

export interface ParticipantPickerProps {
  // selected adalah peserta yang sedang dipilih, dipegang pemanggil supaya ia
  // bisa membaca batch dan paketnya tanpa mencari ulang.
  selected: Participant | null
  onChange: (participant: Participant | null) => void
  placeholder?: string
  className?: string
  id?: string
}

// jeda tunggu sebelum mengetik dianggap selesai. Cukup pendek untuk terasa
// seketika, cukup panjang untuk tidak memanggil server tiap huruf.
const DEBOUNCE_MS = 300

export default function ParticipantPicker({
  selected,
  onChange,
  placeholder = 'Cari nama atau nomor WhatsApp...',
  className = '',
  id,
}: ParticipantPickerProps) {
  const [term, setTerm] = useState('')
  const [debounced, setDebounced] = useState('')

  useEffect(() => {
    const timer = setTimeout(() => setDebounced(term.trim()), DEBOUNCE_MS)
    return () => clearTimeout(timer)
  }, [term])

  const { data: results, isFetching } = useQuery({
    queryKey: ['participant-picker', debounced],
    queryFn: () => searchParticipants(debounced),
    enabled: debounced.length >= 2,
  })

  if (selected) {
    return (
      <div className={className}>
        <div className="flex items-center gap-2 border rounded-lg px-3 py-2 bg-emerald-50 border-emerald-200">
          <div className="flex-1 min-w-0">
            <p className="text-sm font-medium text-gray-800 truncate">{selected.name}</p>
            <p className="text-xs text-gray-500 truncate">{selected.phone}</p>
          </div>
          <button
            type="button"
            aria-label="Hapus pilihan peserta"
            onClick={() => { onChange(null); setTerm(''); setDebounced('') }}
            className="p-1 text-gray-400 hover:text-gray-600 shrink-0"
          >
            <X size={16} />
          </button>
        </div>
      </div>
    )
  }

  const showResults = debounced.length >= 2 && !isFetching
  const matches = results ?? []

  return (
    <div className={className}>
      <div className="relative">
        <Search className="absolute left-3 top-2.5 text-gray-400" size={14} />
        <input
          id={id}
          className="w-full pl-8 pr-3 py-2 border rounded-lg text-sm"
          placeholder={placeholder}
          value={term}
          onChange={(e) => setTerm(e.target.value)}
        />
      </div>

      {term.trim().length > 0 && term.trim().length < 2 && (
        <p className="text-xs text-gray-400 mt-1">Ketik minimal dua huruf atau dua angka.</p>
      )}
      {isFetching && <p className="text-xs text-gray-400 mt-1">Mencari…</p>}

      {showResults && matches.length === 0 && (
        <p className="text-xs text-gray-500 mt-1">
          Tidak ada peserta yang cocok dengan &ldquo;{debounced}&rdquo;.
        </p>
      )}

      {showResults && matches.length > 0 && (
        <ul className="mt-1 border rounded-lg divide-y max-h-56 overflow-y-auto bg-white">
          {matches.map((p) => (
            <li key={p.id}>
              <button
                type="button"
                onClick={() => onChange(p)}
                className="w-full text-left px-3 py-2 hover:bg-gray-50"
              >
                <span className="text-sm text-gray-800">{p.name}</span>
                <span className="text-xs text-gray-500 ml-2">{p.phone}</span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
