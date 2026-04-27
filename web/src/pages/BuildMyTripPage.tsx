import { useForm } from 'react-hook-form'
import { useMutation } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'
import { ExternalLink, CheckCircle2 } from 'lucide-react'
import api from '../utils/api'
import { CreateInquiryRequest, CreateInquiryResponse } from '../types'

export default function BuildMyTripPage() {
  const [searchParams] = useSearchParams()
  const defaultPackageId = searchParams.get('package_id') ?? ''
  const defaultPackageTitle = searchParams.get('package_title') ?? ''

  const { register, handleSubmit, formState: { errors } } = useForm<CreateInquiryRequest>({
    defaultValues: {
      num_people: 2,
      tour_package_id: defaultPackageId,
    },
  })

  const mutation = useMutation<CreateInquiryResponse, Error, CreateInquiryRequest>({
    mutationFn: (data) => api.post('/inquiries', data).then((r) => r.data),
  })

  const onSubmit = (data: CreateInquiryRequest) => {
    // Clean empty strings
    const payload = Object.fromEntries(
      Object.entries(data).filter(([, v]) => v !== '' && v !== 0)
    ) as CreateInquiryRequest
    mutation.mutate(payload)
  }

  if (mutation.isSuccess) {
    return (
      <div className="max-w-lg mx-auto px-4 py-20 text-center">
        <CheckCircle2 className="w-16 h-16 text-green-500 mx-auto mb-6" />
        <h2 className="text-2xl font-bold text-gray-900 mb-3">{mutation.data.message}</h2>
        <p className="text-gray-500 mb-8">
          Konsultan kami siap membantu Anda. Klik tombol di bawah untuk memulai percakapan.
        </p>
        <a
          href={mutation.data.wa_link}
          target="_blank"
          rel="noopener noreferrer"
          className="btn-primary text-base px-8 py-3.5"
        >
          <ExternalLink className="w-5 h-5" />
          Chat via WhatsApp
        </a>
      </div>
    )
  }

  return (
    <div className="max-w-2xl mx-auto px-4 sm:px-6 py-14">
      <h1 className="text-3xl font-bold text-gray-900 mb-2">Build My Trip</h1>
      <p className="text-gray-500 mb-10">
        Isi formulir di bawah dan konsultan kami akan menyiapkan penawaran terbaik untuk Anda.
      </p>

      <form onSubmit={handleSubmit(onSubmit)} className="card p-8 space-y-5">
        {/* Full name */}
        <div>
          <label className="label">Nama Lengkap *</label>
          <input
            {...register('full_name', { required: 'Nama lengkap wajib diisi' })}
            className="input"
            placeholder="Masukkan nama Anda"
          />
          {errors.full_name && <p className="text-red-500 text-xs mt-1">{errors.full_name.message}</p>}
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-5">
          {/* Email */}
          <div>
            <label className="label">Email</label>
            <input
              {...register('email')}
              type="email"
              className="input"
              placeholder="email@contoh.com"
            />
          </div>
          {/* Phone */}
          <div>
            <label className="label">Nomor WhatsApp</label>
            <input
              {...register('phone')}
              className="input"
              placeholder="08xxxxxxxxxx"
            />
          </div>
        </div>

        {/* Destination */}
        <div>
          <label className="label">Destinasi yang Diminati</label>
          <input
            {...register('destination')}
            className="input"
            placeholder="Bali, Lombok, Jepang, dll."
            defaultValue={defaultPackageTitle}
          />
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-5">
          {/* Num people */}
          <div>
            <label className="label">Jumlah Orang *</label>
            <input
              {...register('num_people', { required: true, min: 1, valueAsNumber: true })}
              type="number"
              min="1"
              className="input"
            />
          </div>
          {/* Duration */}
          <div>
            <label className="label">Durasi (hari)</label>
            <input
              {...register('duration_days', { min: 1, valueAsNumber: true })}
              type="number"
              min="1"
              className="input"
              placeholder="Misal: 5"
            />
          </div>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-5">
          {/* Budget */}
          <div>
            <label className="label">Budget per Orang (Rp)</label>
            <input
              {...register('budget', { min: 0, valueAsNumber: true })}
              type="number"
              min="0"
              step="100000"
              className="input"
              placeholder="5000000"
            />
          </div>
          {/* Departure date */}
          <div>
            <label className="label">Tanggal Keberangkatan</label>
            <input
              {...register('departure_date')}
              type="date"
              className="input"
            />
          </div>
        </div>

        {/* Notes */}
        <div>
          <label className="label">Catatan Tambahan</label>
          <textarea
            {...register('notes')}
            rows={3}
            className="input resize-none"
            placeholder="Ada permintaan khusus? Tuliskan di sini..."
          />
        </div>

        {mutation.isError && (
          <p className="text-red-500 text-sm">
            Terjadi kesalahan. Silakan coba lagi.
          </p>
        )}

        <button
          type="submit"
          disabled={mutation.isPending}
          className="btn-primary w-full justify-center py-3 text-base"
        >
          {mutation.isPending ? 'Mengirim...' : 'Kirim & Lanjutkan ke WhatsApp'}
          <ExternalLink className="w-5 h-5" />
        </button>
      </form>
    </div>
  )
}
