import { useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { listAdminBatches, listPackageBatches, type BatchPage } from '../utils/api'
import { formatDate } from '../utils/date'
import type { PackageBatch } from '../types'

// Satu-satunya pemilih keberangkatan di antarmuka admin.
//
// Sebelumnya empat halaman meminta operator mengetikkan UUID batch yang tidak
// pernah dicetak aplikasi di mana pun, dan dialog konversi lead punya
// dropdown-nya sendiri. Menyatukannya di sini berarti "bagaimana sebuah
// keberangkatan disebut" hanya ditulis satu kali: tanggal, nama paketnya, dan
// berapa orang yang sudah terdaftar.
//
// Batch penuh atau ditutup tetap terlihat. Menghilangkannya dari daftar membuat
// admin menyangka batch-nya tidak ada; menampilkannya dengan alasannya membuat
// ketidaktersediaan itu terbaca.

export interface BatchPickerProps {
  value: string
  // onChange menerima batch utuhnya, bukan hanya id — pemanggil yang perlu
  // menampilkan nama paket atau tanggalnya tidak perlu mencarinya lagi.
  onChange: (batchID: string, batch?: PackageBatch) => void
  // packageId membatasi pilihan ke satu paket. Dialog konversi lead memakainya;
  // halaman lain tidak punya paket di tangan lebih dulu.
  packageId?: string
  // upcomingOnly menyingkirkan keberangkatan yang sudah lewat. Tidak berlaku
  // saat packageId diberikan: daftar satu paket memang sudah pendek.
  upcomingOnly?: boolean
  // disableUnavailable mencegah batch penuh/ditutup dipilih. Dinyalakan untuk
  // penugasan (konversi lead), dimatikan untuk penyaringan — menyaring peserta
  // pada batch yang sudah penuh tetap masuk akal.
  disableUnavailable?: boolean
  // autoSelectNearest memilih keberangkatan terdekat sendiri saat belum ada
  // pilihan, sehingga halaman bandara tidak pernah terbuka kosong.
  autoSelectNearest?: boolean
  placeholder?: string
  emptyLabel?: string
  className?: string
  id?: string
}

// useBatchOptions adalah pembacaan di balik pemilih, terbuka bagi halaman yang
// perlu tahu keadaannya sendiri — halaman bandara membedakan "sedang memuat"
// dari "memang tidak ada keberangkatan mendatang", dan keduanya tidak boleh
// tampil sebagai layar kosong tanpa penjelasan. Kunci kueri-nya sama dengan yang
// dipakai pemilih, jadi keduanya berbagi satu hasil, bukan dua permintaan.
export function useBatchOptions({
  packageId,
  upcomingOnly = true,
}: { packageId?: string; upcomingOnly?: boolean }) {
  const scoped = !!packageId
  return useQuery({
    queryKey: ['batch-picker', packageId ?? 'semua', scoped ? false : upcomingOnly],
    queryFn: async (): Promise<BatchPage> => {
      // Daftar satu paket tidak pernah dipotong dan tidak membawa cacah peserta:
      // pembacaan yang sama melayani katalog publik.
      if (scoped) {
        const items = await listPackageBatches(packageId!)
        return { items, total: items.length }
      }
      return listAdminBatches(upcomingOnly)
    },
    // Daftar keberangkatan berubah jauh lebih jarang daripada halaman dibuka,
    // dan memuatnya tidak boleh menahan daftar utamanya.
    staleTime: 5 * 60_000,
  })
}

// batchLabel adalah cara sistem menyebut sebuah keberangkatan kepada manusia.
//
// Isiannya disebut "kursi terisi", bukan "peserta", karena halaman bandara
// menampilkan angka lain di sebelahnya: checklist bandara hanya dibuat untuk
// peserta yang portalnya sudah aktif, sehingga peserta yang belum melunasi
// terhitung di sini tapi belum di sana. Keduanya benar, dan menyebut keduanya
// "peserta" membuat perbedaan itu terbaca sebagai salah hitung.
export function batchLabel(b: PackageBatch, withPackageName: boolean): string {
  // Tanggal dan nama paket adalah identitas keberangkatan itu; sisanya
  // keterangan, dipisahkan titik tengah.
  const head =
    withPackageName && b.package_name
      ? `${formatDate(b.departure_date)} — ${b.package_name}`
      : formatDate(b.departure_date)

  const notes: string[] = []
  // Isian hanya disebut bila memang dihitung. Daftar satu paket tidak
  // menghitungnya — pembacaan yang sama melayani katalog publik — dan "0/12"
  // di sana akan terbaca "belum ada yang mendaftar" padahal yang benar "tidak
  // dihitung". Kuotanya sendiri tetap disebut, karena orang yang menugaskan
  // peserta ke sebuah batch memang perlu tahu sebesar apa batch itu.
  notes.push(
    b.participant_count !== undefined
      ? `${b.participant_count}/${b.quota} kursi terisi`
      : `kuota ${b.quota}`,
  )
  if (b.status !== 'tersedia') notes.push(b.status)

  return [head, ...notes].join(' · ')
}

export default function BatchPicker({
  value,
  onChange,
  packageId,
  upcomingOnly = true,
  disableUnavailable = false,
  autoSelectNearest = false,
  placeholder = 'Semua keberangkatan',
  emptyLabel = 'Belum ada keberangkatan yang bisa dipilih.',
  className = '',
  id,
}: BatchPickerProps) {
  const scoped = !!packageId
  const { data: page, isLoading, isError } = useBatchOptions({ packageId, upcomingOnly })

  const options = page?.items ?? []
  // Server meng-clamp ukuran halaman, jadi daftar panjang memang terpotong.
  // Yang tidak boleh adalah terpotong tanpa memberi tahu.
  const hidden = (page?.total ?? 0) - options.length
  const selectable = (b: PackageBatch) => !disableUnavailable || b.status === 'tersedia'

  // Keberangkatan terdekat lebih dulu adalah urutan yang dikirim server, jadi
  // "yang pertama bisa dipilih" sama dengan "yang paling dekat".
  const nearest = options.find(selectable)
  useEffect(() => {
    if (!autoSelectNearest || value || !nearest) return
    onChange(nearest.id, nearest)
  }, [autoSelectNearest, value, nearest, onChange])

  if (isError) {
    return (
      <p className={`text-xs text-red-600 ${className}`}>
        Gagal memuat daftar keberangkatan. Muat ulang halaman untuk mencoba lagi.
      </p>
    )
  }

  return (
    <div className={className}>
      <select
        id={id}
        className="w-full border rounded-lg px-3 py-2 text-sm disabled:bg-gray-50 disabled:text-gray-400"
        value={value}
        disabled={isLoading}
        onChange={(e) => onChange(e.target.value, options.find((b) => b.id === e.target.value))}
      >
        <option value="">{isLoading ? 'Memuat keberangkatan…' : placeholder}</option>
        {options.map((b) => (
          <option key={b.id} value={b.id} disabled={!selectable(b)}>
            {batchLabel(b, !scoped)}
          </option>
        ))}
      </select>
      {!isLoading && options.length === 0 && (
        <p className="text-xs text-gray-500 mt-1">{emptyLabel}</p>
      )}
      {hidden > 0 && (
        <p className="text-xs text-amber-700 mt-1">
          Menampilkan {options.length} keberangkatan terdekat; {hidden} lainnya tidak muat di daftar
          ini.
        </p>
      )}
    </div>
  )
}
