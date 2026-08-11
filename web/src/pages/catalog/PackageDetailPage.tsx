import { useEffect, useRef, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useQuery, useMutation } from '@tanstack/react-query'
import { MapPin, Clock, ChevronDown, ChevronUp, Send } from 'lucide-react'
import toast from 'react-hot-toast'
import api, { getConsultationPrefill } from '../../utils/api'
import { formatDate } from '../../utils/date'
import type { PackageDetailResponse, CreateLeadRequest } from '../../types'

export default function PackageDetailPage() {
  const { slug } = useParams<{ slug: string }>()
  const [expandedDay, setExpandedDay] = useState<number | null>(1)
  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState<CreateLeadRequest>({
    name: '', phone: '', email: '', pax: 1, message: '', package_id: '', source: 'organic',
  })

  // FR-PORTAL-12: a logged-in participant gets the form filled in from the
  // server, not from what the browser happened to keep. localStorage held only
  // a name and phone, and held them from whenever they were last written — a
  // detail an admin corrected since would have been re-submitted stale.
  const { data: prefill } = useQuery({
    queryKey: ['consultation-prefill'],
    queryFn: () => getConsultationPrefill(),
    enabled: !!localStorage.getItem('portal_token'),
    retry: false,
  })
  const isReturning = !!prefill?.phone

  // Applied once, and only over fields still untouched, so the participant stays
  // free to edit anything it filled in.
  const prefilled = useRef(false)
  useEffect(() => {
    if (!prefill || prefilled.current) return
    prefilled.current = true
    setForm(f => ({
      ...f,
      name: f.name || prefill.name,
      phone: f.phone || prefill.phone,
      email: f.email || prefill.email,
      message: f.message || (prefill.room_type ? `Tipe kamar sebelumnya: ${prefill.room_type}` : ''),
      source: 'referral',
    }))
  }, [prefill])
  const [phoneError, setPhoneError] = useState('')
  const [consentGiven, setConsentGiven] = useState(false)
  const [consentError, setConsentError] = useState('')

  const { data, isLoading } = useQuery({
    queryKey: ['package-detail', slug],
    queryFn: () =>
      api.get<{ success: boolean; data: PackageDetailResponse }>(`/packages/${slug}`)
        .then((r) => r.data.data),
  })

  const leadMutation = useMutation({
    mutationFn: (req: CreateLeadRequest) => api.post('/leads', req),
    onSuccess: () => {
      toast.success('Terima kasih! Tim kami akan segera menghubungi Anda via WhatsApp.')
      setShowForm(false)
      setForm({ name: '', phone: '', email: '', pax: 1, message: '', package_id: '', source: 'organic' })
    },
    onError: () => toast.error('Gagal mengirim, coba lagi.'),
  })

  if (isLoading) return (
    <div className="max-w-4xl mx-auto p-6 animate-pulse space-y-4">
      <div className="h-64 bg-gray-200 rounded-xl" />
      <div className="h-8 bg-gray-200 rounded w-2/3" />
      <div className="h-4 bg-gray-200 rounded w-1/3" />
    </div>
  )

  const pkg = data?.package
  const batches = data?.batches ?? []
  const images = data?.images ?? []

  if (!pkg) return <div className="text-center py-20">Paket tidak ditemukan.</div>

  const thumbnail = images.find(i => i.is_thumbnail)?.file_path || pkg.thumbnail

  function validatePhone(val: string) {
    const cleaned = val.trim()
    if (!cleaned.startsWith('08') && !cleaned.startsWith('+62') && !cleaned.startsWith('628')) {
      setPhoneError('Nomor WhatsApp harus diawali 08 atau +62')
      return false
    }
    if (cleaned.replace(/\D/g, '').length < 10) {
      setPhoneError('Nomor WhatsApp terlalu pendek')
      return false
    }
    setPhoneError('')
    return true
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!validatePhone(form.phone)) return
    if (!consentGiven) {
      setConsentError('Anda harus menyetujui kebijakan privasi untuk melanjutkan')
      return
    }
    setConsentError('')
    const phone = form.phone.replace(/^\+/, '').replace(/^0/, '62')
    leadMutation.mutate({ ...form, phone, package_id: pkg!.id, consent_given: consentGiven })
  }

  const itinerary = Array.isArray(pkg.itinerary) ? pkg.itinerary : []
  const requirements = Array.isArray(pkg.requirements) ? pkg.requirements : []
  const facilities = Array.isArray(pkg.facilities) ? pkg.facilities : []

  return (
    <div className="max-w-4xl mx-auto px-4 py-8">
      {/* Hero image */}
      <div className="aspect-video rounded-xl overflow-hidden bg-gray-200 mb-6">
        {thumbnail && <img src={thumbnail} alt={pkg.name} className="w-full h-full object-cover" />}
      </div>

      <div className="flex flex-col md:flex-row gap-8">
        {/* Left — details */}
        <div className="flex-1">
          <div className="flex items-start justify-between gap-2 mb-2">
            <span className="bg-emerald-100 text-emerald-700 text-xs px-2 py-0.5 rounded-full capitalize">
              {pkg.category}
            </span>
          </div>
          <h1 className="text-2xl font-bold text-gray-800 mb-1">{pkg.name}</h1>
          <p className="text-gray-500 flex items-center gap-1 mb-4">
            <MapPin size={14} /> {pkg.destination} &bull; <Clock size={14} /> {pkg.duration_days} hari
          </p>

          {pkg.description && (
            <p className="text-gray-600 text-sm leading-relaxed mb-6">{pkg.description}</p>
          )}

          {/* Itinerary */}
          {itinerary.length > 0 && (
            <section className="mb-6">
              <h2 className="font-semibold text-gray-800 mb-3">Itinerary Perjalanan</h2>
              <div className="space-y-2">
                {itinerary.map((day) => (
                  <div key={day.day} className="border rounded-lg overflow-hidden">
                    <button
                      onClick={() => setExpandedDay(expandedDay === day.day ? null : day.day)}
                      className="w-full flex items-center justify-between px-4 py-3 text-left hover:bg-gray-50"
                    >
                      <span className="font-medium text-sm">Hari {day.day}: {day.title}</span>
                      {expandedDay === day.day ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
                    </button>
                    {expandedDay === day.day && (
                      <div className="px-4 pb-4 text-sm text-gray-600">
                        <p className="mb-2">{day.description}</p>
                        {day.activities?.length > 0 && (
                          <ul className="list-disc list-inside space-y-1 text-gray-500">
                            {day.activities.map((a, i) => <li key={i}>{a}</li>)}
                          </ul>
                        )}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </section>
          )}

          {/* FR-CAT-03: Fasilitas yang disertakan */}
          {facilities.length > 0 && (
            <section className="mb-6">
              <h2 className="font-semibold text-gray-800 mb-3">Fasilitas yang Disertakan</h2>
              <ul className="grid grid-cols-1 sm:grid-cols-2 gap-1 text-sm text-gray-600">
                {facilities.map((f, i) => (
                  <li key={i} className="flex items-start gap-2">
                    <span className="text-emerald-600 mt-0.5">✓</span> {f}
                  </li>
                ))}
              </ul>
            </section>
          )}

          {/* Requirements */}
          {requirements.length > 0 && (
            <section className="mb-6">
              <h2 className="font-semibold text-gray-800 mb-3">Persyaratan & Dokumen</h2>
              <ul className="space-y-1 text-sm text-gray-600">
                {requirements.map((r, i) => (
                  <li key={i} className="flex items-start gap-2">
                    <span className="text-emerald-600 mt-0.5">✓</span> {r}
                  </li>
                ))}
              </ul>
            </section>
          )}

          {/* FR-CAT-03: Informasi visa */}
          {pkg.visa_info && (
            <section className="mb-6">
              <h2 className="font-semibold text-gray-800 mb-3">Informasi Visa</h2>
              <p className="text-sm text-gray-600 whitespace-pre-line">{pkg.visa_info}</p>
            </section>
          )}

          {/* FR-CAT-03: Syarat & Ketentuan */}
          {pkg.terms_conditions && (
            <section className="mb-6">
              <h2 className="font-semibold text-gray-800 mb-3">Syarat & Ketentuan</h2>
              <p className="text-sm text-gray-600 whitespace-pre-line">{pkg.terms_conditions}</p>
            </section>
          )}
        </div>

        {/* Right — pricing & CTA */}
        <div className="md:w-80 shrink-0">
          <div className="bg-white rounded-xl border p-5 shadow-sm sticky top-4">
            <p className="text-xs text-gray-400 mb-1">Mulai dari</p>
            <p className="text-2xl font-bold text-emerald-700 mb-4">
              Rp {pkg.base_price.toLocaleString('id-ID')}
            </p>

            {batches.length > 0 && (
              <div className="mb-4">
                <p className="text-xs font-medium text-gray-500 mb-2">Jadwal Keberangkatan</p>
                <div className="space-y-2">
                  {batches.filter(b => b.status === 'tersedia').slice(0, 3).map((b) => (
                    <div key={b.id} className="flex items-center justify-between text-xs bg-gray-50 rounded px-3 py-2">
                      <span>{formatDate(b.departure_date)}</span>
                      <span className="text-emerald-600 font-medium">Tersedia</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            <button
              onClick={() => setShowForm(!showForm)}
              className="w-full py-3 bg-emerald-600 text-white rounded-lg font-semibold hover:bg-emerald-700 transition-colors flex items-center justify-center gap-2"
            >
              <Send size={16} /> Konsultasi Sekarang
            </button>

            {/* Lead form */}
            {showForm && (
              <form onSubmit={handleSubmit} className="mt-4 space-y-3">
                {/* v2.0 F3 — returning customer: data sudah terisi dari akun portal */}
                {isReturning && (
                  <div className="bg-blue-50 border border-blue-100 rounded-lg px-3 py-2 text-xs text-blue-700">
                    ⭐ Selamat datang kembali! Data Anda sudah terisi otomatis. Anda tetap memakai akun portal yang sama — tidak perlu daftar ulang.
                  </div>
                )}
                <input
                  required
                  placeholder="Nama lengkap *"
                  className="w-full border rounded-lg px-3 py-2 text-sm"
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                />
                <div>
                  <input
                    required
                    placeholder="No. WhatsApp (08xxx atau +62xxx) *"
                    className={`w-full border rounded-lg px-3 py-2 text-sm ${phoneError ? 'border-red-400 bg-red-50' : ''}`}
                    value={form.phone}
                    onChange={(e) => { setForm({ ...form, phone: e.target.value }); if (phoneError) validatePhone(e.target.value) }}
                    onBlur={() => form.phone && validatePhone(form.phone)}
                  />
                  {phoneError && (
                    <p className="text-xs text-red-500 mt-1">{phoneError}</p>
                  )}
                </div>
                <input
                  placeholder="Email (opsional)"
                  type="email"
                  className="w-full border rounded-lg px-3 py-2 text-sm"
                  value={form.email}
                  onChange={(e) => setForm({ ...form, email: e.target.value })}
                />
                <div className="flex items-center gap-2">
                  <label className="text-xs text-gray-500 shrink-0">Jumlah peserta</label>
                  <input
                    type="number" min={1} max={50}
                    className="w-20 border rounded-lg px-3 py-2 text-sm"
                    value={form.pax}
                    onChange={(e) => setForm({ ...form, pax: Number(e.target.value) })}
                  />
                </div>
                <textarea
                  placeholder="Pesan atau pertanyaan..."
                  rows={3}
                  className="w-full border rounded-lg px-3 py-2 text-sm resize-none"
                  value={form.message}
                  onChange={(e) => setForm({ ...form, message: e.target.value })}
                />
                {/* §25.2 Informed consent — checkbox tidak pre-checked */}
                <div className="space-y-1">
                  <label className="flex items-start gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      className="mt-0.5 shrink-0"
                      checked={consentGiven}
                      onChange={(e) => { setConsentGiven(e.target.checked); if (e.target.checked) setConsentError('') }}
                    />
                    <span className="text-xs text-gray-600">
                      Saya menyetujui pengumpulan dan penggunaan data pribadi saya sesuai{' '}
                      <a href="/privacy-policy" target="_blank" className="text-emerald-600 underline">
                        Kebijakan Privasi
                      </a>{' '}
                      Pintour untuk keperluan konsultasi dan pemesanan perjalanan.
                    </span>
                  </label>
                  {consentError && (
                    <p className="text-xs text-red-500">{consentError}</p>
                  )}
                </div>
                <button
                  type="submit"
                  disabled={leadMutation.isPending}
                  className="w-full py-2 bg-emerald-600 text-white rounded-lg text-sm font-medium disabled:opacity-50"
                >
                  {leadMutation.isPending ? 'Mengirim...' : 'Kirim Permintaan'}
                </button>
              </form>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
