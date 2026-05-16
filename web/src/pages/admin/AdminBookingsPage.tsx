import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { ChevronDown, ChevronUp, X, Plus, Trash2, CheckCircle, ExternalLink } from 'lucide-react'
import api from '../../utils/api'
import { Booking, BookingsResponse, BookingParticipant, Payment, ParticipantDocument } from '../../types'
import Spinner from '../../components/Spinner'

const PAYMENT_STATUSES = ['pending', 'dp', 'lunas', 'refund']
const BOOKING_STATUSES = ['confirmed', 'awaiting_docs', 'visa_process', 'ready_to_depart', 'departed', 'completed', 'cancelled']
const BOOKING_STATUS_LABELS: Record<string, string> = {
  confirmed: 'Confirmed',
  awaiting_docs: 'Tunggu Dok.',
  visa_process: 'Proses Visa',
  ready_to_depart: 'Siap Berangkat',
  departed: 'Berangkat',
  completed: 'Selesai',
  cancelled: 'Dibatalkan',
}

const PAYMENT_TYPES = ['dp', 'pelunasan', 'full']
const DOC_TYPES = ['passport', 'ktp', 'bank_statement', 'visa_support', 'photo', 'other']

function paymentColor(s: string): string {
  const map: Record<string, string> = {
    pending: 'bg-gray-100 text-gray-600',
    dp: 'bg-yellow-100 text-yellow-700',
    lunas: 'bg-green-100 text-green-700',
    refund: 'bg-red-100 text-red-700',
  }
  return map[s] ?? 'bg-gray-100 text-gray-500'
}

function bookingColor(s: string): string {
  const map: Record<string, string> = {
    confirmed: 'bg-blue-100 text-blue-700',
    awaiting_docs: 'bg-orange-100 text-orange-700',
    visa_process: 'bg-purple-100 text-purple-700',
    ready_to_depart: 'bg-teal-100 text-teal-700',
    departed: 'bg-indigo-100 text-indigo-700',
    completed: 'bg-green-100 text-green-700',
    cancelled: 'bg-red-100 text-red-700',
  }
  return map[s] ?? 'bg-gray-100 text-gray-500'
}

const fmtRp = (n: number) =>
  new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(n)

const fmtDate = (s: string) => (s ? new Date(s).toLocaleDateString('id-ID', { day: '2-digit', month: 'short', year: 'numeric' }) : '-')

const EMPTY_PARTICIPANT = { full_name: '', id_type: 'ktp', id_number: '', date_of_birth: '', phone: '' }

// ── BookingDetailPanel ────────────────────────────────────────────────────────

function BookingDetailPanel({ booking }: { booking: Booking }) {
  const [activeTab, setActiveTab] = useState<'manifest' | 'payments' | 'documents' | 'departure'>('manifest')
  const qc = useQueryClient()

  const [payForm, setPayForm] = useState({ payment_type: 'dp', amount: '', paid_at: '', proof_url: '', notes: '' })
  const [showPayForm, setShowPayForm] = useState(false)
  const [docForm, setDocForm] = useState({ participant_id: '', doc_type: 'passport', file_url: '', notes: '' })
  const [showDocForm, setShowDocForm] = useState(false)
  const [leaderID, setLeaderID] = useState(booking.tour_leader_id ?? '')
  const [waLink, setWaLink] = useState(booking.wa_group_link ?? '')
  const [briefing, setBriefing] = useState(booking.briefing_done ?? false)

  const { data: payments = [] } = useQuery<Payment[]>({
    queryKey: ['booking-payments', booking.id],
    queryFn: () => api.get(`/admin/bookings/${booking.id}/payments`).then((r) => r.data),
    enabled: activeTab === 'payments',
  })

  const { data: documents = [] } = useQuery<ParticipantDocument[]>({
    queryKey: ['booking-documents', booking.id],
    queryFn: () => api.get(`/admin/bookings/${booking.id}/documents`).then((r) => r.data),
    enabled: activeTab === 'documents',
  })

  const createPayMut = useMutation({
    mutationFn: (body: typeof payForm) =>
      api.post(`/admin/bookings/${booking.id}/payments`, {
        payment_type: body.payment_type,
        amount: Number(body.amount),
        paid_at: body.paid_at,
        proof_url: body.proof_url || undefined,
        notes: body.notes || undefined,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['booking-payments', booking.id] })
      setShowPayForm(false)
      setPayForm({ payment_type: 'dp', amount: '', paid_at: '', proof_url: '', notes: '' })
    },
  })

  const verifyPayMut = useMutation({
    mutationFn: (pid: string) => api.patch(`/admin/payments/${pid}/verify`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['booking-payments', booking.id] }),
  })

  const deletePayMut = useMutation({
    mutationFn: (pid: string) => api.delete(`/admin/payments/${pid}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['booking-payments', booking.id] }),
  })

  const createDocMut = useMutation({
    mutationFn: (body: typeof docForm) =>
      api.post(`/admin/participants/${body.participant_id}/documents`, {
        doc_type: body.doc_type,
        file_url: body.file_url,
        notes: body.notes || undefined,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['booking-documents', booking.id] })
      setShowDocForm(false)
      setDocForm({ participant_id: '', doc_type: 'passport', file_url: '', notes: '' })
    },
  })

  const verifyDocMut = useMutation({
    mutationFn: (did: string) => api.patch(`/admin/documents/${did}/verify`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['booking-documents', booking.id] }),
  })

  const deleteDocMut = useMutation({
    mutationFn: (did: string) => api.delete(`/admin/documents/${did}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['booking-documents', booking.id] }),
  })

  const setLeaderMut = useMutation({
    mutationFn: () => api.patch(`/admin/bookings/${booking.id}/leader`, { tour_leader_id: leaderID }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-booking-detail', booking.id] }),
  })

  const setWAMut = useMutation({
    mutationFn: () => api.patch(`/admin/bookings/${booking.id}/wa-group`, { wa_group_link: waLink }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-booking-detail', booking.id] }),
  })

  const setBriefingMut = useMutation({
    mutationFn: (done: boolean) => api.patch(`/admin/bookings/${booking.id}/briefing`, { briefing_done: done }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-booking-detail', booking.id] }),
  })

  const tabs = [
    { key: 'manifest' as const, label: 'Manifest' },
    { key: 'payments' as const, label: 'Pembayaran' },
    { key: 'documents' as const, label: 'Dokumen' },
    { key: 'departure' as const, label: 'Keberangkatan' },
  ]

  return (
    <div>
      <div className="flex gap-1 mb-4 border-b">
        {tabs.map((t) => (
          <button
            key={t.key}
            onClick={() => setActiveTab(t.key)}
            className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
              activeTab === t.key ? 'border-blue-600 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700'
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {activeTab === 'manifest' && (
        <div>
          {booking.participants && booking.participants.length > 0 ? (
            <table className="w-full text-sm">
              <thead>
                <tr className="text-xs text-gray-500 uppercase">
                  <th className="text-left pr-6 py-1">Nama</th>
                  <th className="text-left pr-6 py-1">Jenis ID</th>
                  <th className="text-left pr-6 py-1">No. ID</th>
                  <th className="text-left pr-6 py-1">Tgl Lahir</th>
                  <th className="text-left py-1">Telepon</th>
                </tr>
              </thead>
              <tbody>
                {booking.participants.map((p: BookingParticipant) => (
                  <tr key={p.id} className="text-gray-700">
                    <td className="pr-6 py-1 font-medium">{p.full_name}</td>
                    <td className="pr-6 py-1 uppercase text-xs">{p.id_type}</td>
                    <td className="pr-6 py-1 font-mono text-xs">{p.id_number}</td>
                    <td className="pr-6 py-1">{p.date_of_birth?.slice(0, 10) ?? '-'}</td>
                    <td className="py-1">{p.phone ?? '-'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <p className="text-xs text-gray-400">Belum ada peserta terdaftar.</p>
          )}
        </div>
      )}

      {activeTab === 'payments' && (
        <div>
          <div className="flex items-center justify-between mb-3">
            <p className="text-xs font-semibold text-gray-500 uppercase">Riwayat Pembayaran</p>
            <button onClick={() => setShowPayForm((v) => !v)} className="text-xs text-blue-600 hover:underline flex items-center gap-1">
              <Plus className="w-3 h-3" /> Tambah
            </button>
          </div>
          {showPayForm && (
            <div className="bg-white border rounded-lg p-4 mb-4 grid grid-cols-2 gap-3">
              <div>
                <label className="label text-xs">Jenis</label>
                <select className="input text-sm" value={payForm.payment_type} onChange={(e) => setPayForm((f) => ({ ...f, payment_type: e.target.value }))}>
                  {PAYMENT_TYPES.map((t) => <option key={t} value={t}>{t.toUpperCase()}</option>)}
                </select>
              </div>
              <div>
                <label className="label text-xs">Jumlah (Rp)</label>
                <input className="input text-sm" type="number" min="0" value={payForm.amount} onChange={(e) => setPayForm((f) => ({ ...f, amount: e.target.value }))} />
              </div>
              <div>
                <label className="label text-xs">Tanggal Bayar</label>
                <input className="input text-sm" type="datetime-local" value={payForm.paid_at} onChange={(e) => setPayForm((f) => ({ ...f, paid_at: e.target.value }))} />
              </div>
              <div>
                <label className="label text-xs">Bukti Transfer (URL)</label>
                <input className="input text-sm" placeholder="https://..." value={payForm.proof_url} onChange={(e) => setPayForm((f) => ({ ...f, proof_url: e.target.value }))} />
              </div>
              <div className="col-span-2">
                <label className="label text-xs">Catatan</label>
                <input className="input text-sm" value={payForm.notes} onChange={(e) => setPayForm((f) => ({ ...f, notes: e.target.value }))} />
              </div>
              <div className="col-span-2 flex gap-2">
                <button onClick={() => createPayMut.mutate(payForm)} disabled={createPayMut.isPending || !payForm.amount || !payForm.paid_at} className="btn-primary py-1.5 px-4 text-xs disabled:opacity-60">
                  {createPayMut.isPending ? 'Menyimpan...' : 'Simpan'}
                </button>
                <button onClick={() => setShowPayForm(false)} className="btn-secondary py-1.5 px-3 text-xs">Batal</button>
              </div>
            </div>
          )}
          {payments.length === 0 ? (
            <p className="text-xs text-gray-400">Belum ada pembayaran.</p>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="text-xs text-gray-500 uppercase">
                  <th className="text-left pr-4 py-1">Jenis</th>
                  <th className="text-left pr-4 py-1">Jumlah</th>
                  <th className="text-left pr-4 py-1">Tanggal</th>
                  <th className="text-left pr-4 py-1">Bukti</th>
                  <th className="text-left pr-4 py-1">Verifikasi</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {payments.map((p) => (
                  <tr key={p.id} className="text-gray-700 border-t border-gray-100">
                    <td className="pr-4 py-1.5 uppercase text-xs font-medium">{p.payment_type}</td>
                    <td className="pr-4 py-1.5 font-semibold">{fmtRp(p.amount)}</td>
                    <td className="pr-4 py-1.5 text-xs">{fmtDate(p.paid_at)}</td>
                    <td className="pr-4 py-1.5">
                      {p.proof_url ? (
                        <a href={p.proof_url} target="_blank" rel="noopener noreferrer" className="text-blue-600 hover:underline flex items-center gap-1 text-xs">
                          Lihat <ExternalLink className="w-3 h-3" />
                        </a>
                      ) : <span className="text-gray-400 text-xs">-</span>}
                    </td>
                    <td className="pr-4 py-1.5">
                      {p.verified_at ? (
                        <span className="text-xs text-green-600 flex items-center gap-1"><CheckCircle className="w-3 h-3" /> Verified</span>
                      ) : (
                        <button onClick={() => verifyPayMut.mutate(p.id)} className="text-xs text-blue-600 hover:underline">Verifikasi</button>
                      )}
                    </td>
                    <td className="py-1.5">
                      <button onClick={() => { if (confirm('Hapus pembayaran ini?')) deletePayMut.mutate(p.id) }} className="text-gray-400 hover:text-red-600">
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}

      {activeTab === 'documents' && (
        <div>
          <div className="flex items-center justify-between mb-3">
            <p className="text-xs font-semibold text-gray-500 uppercase">Dokumen Peserta</p>
            <button onClick={() => setShowDocForm((v) => !v)} className="text-xs text-blue-600 hover:underline flex items-center gap-1">
              <Plus className="w-3 h-3" /> Upload Dokumen
            </button>
          </div>
          {showDocForm && (
            <div className="bg-white border rounded-lg p-4 mb-4 grid grid-cols-2 gap-3">
              <div>
                <label className="label text-xs">Peserta</label>
                <select className="input text-sm" value={docForm.participant_id} onChange={(e) => setDocForm((f) => ({ ...f, participant_id: e.target.value }))}>
                  <option value="">-- Pilih Peserta --</option>
                  {(booking.participants ?? []).map((p) => (
                    <option key={p.id} value={p.id}>{p.full_name}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="label text-xs">Jenis Dokumen</label>
                <select className="input text-sm" value={docForm.doc_type} onChange={(e) => setDocForm((f) => ({ ...f, doc_type: e.target.value }))}>
                  {DOC_TYPES.map((t) => <option key={t} value={t}>{t}</option>)}
                </select>
              </div>
              <div className="col-span-2">
                <label className="label text-xs">URL File</label>
                <input className="input text-sm" placeholder="https://..." value={docForm.file_url} onChange={(e) => setDocForm((f) => ({ ...f, file_url: e.target.value }))} />
              </div>
              <div className="col-span-2">
                <label className="label text-xs">Catatan</label>
                <input className="input text-sm" value={docForm.notes} onChange={(e) => setDocForm((f) => ({ ...f, notes: e.target.value }))} />
              </div>
              <div className="col-span-2 flex gap-2">
                <button onClick={() => createDocMut.mutate(docForm)} disabled={createDocMut.isPending || !docForm.participant_id || !docForm.file_url} className="btn-primary py-1.5 px-4 text-xs disabled:opacity-60">
                  {createDocMut.isPending ? 'Menyimpan...' : 'Simpan'}
                </button>
                <button onClick={() => setShowDocForm(false)} className="btn-secondary py-1.5 px-3 text-xs">Batal</button>
              </div>
            </div>
          )}
          {documents.length === 0 ? (
            <p className="text-xs text-gray-400">Belum ada dokumen terupload.</p>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="text-xs text-gray-500 uppercase">
                  <th className="text-left pr-4 py-1">Peserta</th>
                  <th className="text-left pr-4 py-1">Jenis</th>
                  <th className="text-left pr-4 py-1">File</th>
                  <th className="text-left pr-4 py-1">Status</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {documents.map((d) => {
                  const participant = (booking.participants ?? []).find((p) => p.id === d.participant_id)
                  return (
                    <tr key={d.id} className="text-gray-700 border-t border-gray-100">
                      <td className="pr-4 py-1.5 text-xs">{participant?.full_name ?? d.participant_id.slice(0, 8)}</td>
                      <td className="pr-4 py-1.5 uppercase text-xs font-medium">{d.doc_type}</td>
                      <td className="pr-4 py-1.5">
                        <a href={d.file_url} target="_blank" rel="noopener noreferrer" className="text-blue-600 hover:underline flex items-center gap-1 text-xs">
                          Lihat <ExternalLink className="w-3 h-3" />
                        </a>
                      </td>
                      <td className="pr-4 py-1.5">
                        {d.verified ? (
                          <span className="text-xs text-green-600 flex items-center gap-1"><CheckCircle className="w-3 h-3" /> Verified</span>
                        ) : (
                          <button onClick={() => verifyDocMut.mutate(d.id)} className="text-xs text-blue-600 hover:underline">Verifikasi</button>
                        )}
                      </td>
                      <td className="py-1.5">
                        <button onClick={() => { if (confirm('Hapus dokumen ini?')) deleteDocMut.mutate(d.id) }} className="text-gray-400 hover:text-red-600">
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          )}
        </div>
      )}

      {activeTab === 'departure' && (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-6">
          <div>
            <label className="label text-xs">Tour Leader (User ID)</label>
            <div className="flex gap-2">
              <input className="input text-sm flex-1" placeholder="UUID tour leader..." value={leaderID} onChange={(e) => setLeaderID(e.target.value)} />
              <button onClick={() => setLeaderMut.mutate()} disabled={!leaderID || setLeaderMut.isPending} className="btn-primary py-1.5 px-3 text-xs disabled:opacity-60">
                Simpan
              </button>
            </div>
          </div>
          <div>
            <label className="label text-xs">Link Grup WhatsApp</label>
            <div className="flex gap-2">
              <input className="input text-sm flex-1" placeholder="https://chat.whatsapp.com/..." value={waLink} onChange={(e) => setWaLink(e.target.value)} />
              <button onClick={() => setWAMut.mutate()} disabled={!waLink || setWAMut.isPending} className="btn-primary py-1.5 px-3 text-xs disabled:opacity-60">
                Simpan
              </button>
            </div>
            {booking.wa_group_link && (
              <a href={booking.wa_group_link} target="_blank" rel="noopener noreferrer" className="text-xs text-blue-600 hover:underline mt-1 flex items-center gap-1">
                Buka Grup <ExternalLink className="w-3 h-3" />
              </a>
            )}
          </div>
          <div className="sm:col-span-2">
            <label className="flex items-center gap-3 cursor-pointer select-none">
              <input
                type="checkbox"
                className="w-4 h-4 rounded"
                checked={briefing}
                onChange={(e) => {
                  setBriefing(e.target.checked)
                  setBriefingMut.mutate(e.target.checked)
                }}
              />
              <span className="text-sm text-gray-700">Briefing keberangkatan sudah dilakukan</span>
            </label>
          </div>
        </div>
      )}
    </div>
  )
}

interface ParticipantForm {
  full_name: string
  id_type: string
  id_number: string
  date_of_birth: string
  phone: string
}

interface BookingForm {
  tour_package_id: string
  customer_name: string
  customer_email: string
  customer_phone: string
  departure_date: string
  num_people: number
  total_price: number
  notes: string
  participants: ParticipantForm[]
}

const EMPTY_FORM: BookingForm = {
  tour_package_id: '',
  customer_name: '',
  customer_email: '',
  customer_phone: '',
  departure_date: '',
  num_people: 1,
  total_price: 0,
  notes: '',
  participants: [{ ...EMPTY_PARTICIPANT }],
}

export default function AdminBookingsPage() {
  const [page, setPage] = useState(1)
  const [filterPayment, setFilterPayment] = useState('')
  const [filterBooking, setFilterBooking] = useState('')
  const [expandedId, setExpandedId] = useState<string | null>(null)
  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState<BookingForm>({ ...EMPTY_FORM })
  const qc = useQueryClient()

  const { data, isLoading } = useQuery<BookingsResponse>({
    queryKey: ['admin-bookings', page, filterPayment, filterBooking],
    queryFn: () => {
      const params = new URLSearchParams({ page: String(page), per_page: '15' })
      if (filterPayment) params.set('payment_status', filterPayment)
      if (filterBooking) params.set('booking_status', filterBooking)
      return api.get(`/admin/bookings?${params.toString()}`).then((r) => r.data)
    },
  })

  const { data: detailData } = useQuery<Booking>({
    queryKey: ['admin-booking-detail', expandedId],
    queryFn: () => api.get(`/admin/bookings/${expandedId}`).then((r) => r.data),
    enabled: !!expandedId,
  })

  const payMut = useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      api.patch(`/admin/bookings/${id}/payment-status`, { payment_status: status }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin-bookings'] })
      qc.invalidateQueries({ queryKey: ['admin-booking-detail', expandedId] })
    },
  })

  const bookMut = useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      api.patch(`/admin/bookings/${id}/booking-status`, { booking_status: status }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin-bookings'] })
      qc.invalidateQueries({ queryKey: ['admin-booking-detail', expandedId] })
    },
  })

  const deleteMut = useMutation({
    mutationFn: (id: string) => api.delete(`/admin/bookings/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-bookings'] }),
  })

  const createMut = useMutation({
    mutationFn: (body: BookingForm) =>
      api.post('/admin/bookings', {
        ...body,
        tour_package_id: body.tour_package_id || undefined,
        participants: body.participants.filter((p) => p.full_name && p.id_number),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin-bookings'] })
      setShowForm(false)
      setForm({ ...EMPTY_FORM })
    },
  })

  const totalPages = data ? Math.ceil(data.total / 15) : 1

  const handleAddParticipant = () =>
    setForm((f) => ({ ...f, participants: [...f.participants, { ...EMPTY_PARTICIPANT }] }))

  const handleRemoveParticipant = (idx: number) =>
    setForm((f) => ({ ...f, participants: f.participants.filter((_, i) => i !== idx) }))

  const handleParticipantChange = (idx: number, field: keyof ParticipantForm, value: string) =>
    setForm((f) => {
      const ps = [...f.participants]
      ps[idx] = { ...ps[idx], [field]: value }
      return { ...f, participants: ps }
    })

  if (isLoading) return <Spinner message="Memuat data booking..." />

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Manifest & Pembayaran</h1>
        <button onClick={() => setShowForm(true)} className="btn-primary py-2 px-4 text-sm flex items-center gap-2">
          <Plus className="w-4 h-4" /> Tambah Booking
        </button>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-3 mb-5">
        <select
          value={filterPayment}
          onChange={(e) => { setFilterPayment(e.target.value); setPage(1) }}
          className="input w-auto text-sm"
        >
          <option value="">Semua Pembayaran</option>
          {PAYMENT_STATUSES.map((s) => <option key={s} value={s}>{s.toUpperCase()}</option>)}
        </select>
        <select
          value={filterBooking}
          onChange={(e) => { setFilterBooking(e.target.value); setPage(1) }}
          className="input w-auto text-sm"
        >
          <option value="">Semua Status Booking</option>
          {BOOKING_STATUSES.map((s) => <option key={s} value={s}>{BOOKING_STATUS_LABELS[s] ?? s}</option>)}
        </select>
      </div>

      {/* Table */}
      <div className="card overflow-x-auto">
        <table className="w-full text-sm min-w-max">
          <thead className="bg-gray-50 text-gray-500 text-xs uppercase tracking-wide">
            <tr>
              <th className="px-5 py-3 text-left">Kode</th>
              <th className="px-5 py-3 text-left">Pelanggan</th>
              <th className="px-5 py-3 text-left">Paket</th>
              <th className="px-5 py-3 text-left">Berangkat</th>
              <th className="px-5 py-3 text-left">Orang</th>
              <th className="px-5 py-3 text-left">Total</th>
              <th className="px-5 py-3 text-left">Pembayaran</th>
              <th className="px-5 py-3 text-left">Status</th>
              <th className="px-5 py-3 text-left"></th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-50">
            {data?.data.map((b: Booking) => (
              <>
                <tr key={b.id} className="hover:bg-gray-50 transition-colors">
                  <td className="px-5 py-3 font-mono text-xs text-gray-600">{b.booking_code}</td>
                  <td className="px-5 py-3">
                    <p className="font-medium text-gray-800">{b.customer_name}</p>
                    {b.customer_phone && <p className="text-xs text-gray-400">{b.customer_phone}</p>}
                  </td>
                  <td className="px-5 py-3 text-gray-600 max-w-[160px] truncate">{b.package_title ?? '-'}</td>
                  <td className="px-5 py-3 text-gray-600">{b.departure_date?.slice(0, 10)}</td>
                  <td className="px-5 py-3 text-center">{b.num_people}</td>
                  <td className="px-5 py-3 text-gray-800 font-medium">{fmtRp(b.total_price)}</td>
                  <td className="px-5 py-3">
                    <select
                      className={`text-xs font-semibold rounded-full px-2 py-1 border-0 cursor-pointer ${paymentColor(b.payment_status)}`}
                      value={b.payment_status}
                      onChange={(e) => payMut.mutate({ id: b.id, status: e.target.value })}
                    >
                      {PAYMENT_STATUSES.map((s) => (
                        <option key={s} value={s}>{s.toUpperCase()}</option>
                      ))}
                    </select>
                  </td>
                  <td className="px-5 py-3">
                    <select
                      className={`text-xs font-semibold rounded-full px-2 py-1 border-0 cursor-pointer ${bookingColor(b.booking_status)}`}
                      value={b.booking_status}
                      onChange={(e) => bookMut.mutate({ id: b.id, status: e.target.value })}
                    >
                      {BOOKING_STATUSES.map((s) => (
                        <option key={s} value={s}>{BOOKING_STATUS_LABELS[s] ?? s}</option>
                      ))}
                    </select>
                  </td>
                  <td className="px-5 py-3">
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => setExpandedId(expandedId === b.id ? null : b.id)}
                        className="text-gray-400 hover:text-blue-600 transition-colors"
                        title="Lihat peserta"
                      >
                        {expandedId === b.id ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
                      </button>
                      <button
                        onClick={() => {
                          if (confirm(`Hapus booking ${b.booking_code}?`)) deleteMut.mutate(b.id)
                        }}
                        className="text-gray-400 hover:text-red-600 transition-colors"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </td>
                </tr>
                {expandedId === b.id && (
                  <tr key={`${b.id}-detail`} className="bg-blue-50">
                    <td colSpan={9} className="px-8 py-5">
                      {!detailData ? (
                        <p className="text-xs text-gray-400">Memuat...</p>
                      ) : (
                        <BookingDetailPanel booking={detailData} />
                      )}
                    </td>
                  </tr>
                )}
              </>
            ))}
            {(!data?.data || data.data.length === 0) && (
              <tr>
                <td colSpan={9} className="text-center py-16 text-gray-400">Belum ada data booking.</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex justify-center gap-2 mt-6">
          <button onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page === 1} className="btn-secondary py-2 px-4 text-sm disabled:opacity-40">Sebelumnya</button>
          <span className="flex items-center px-4 text-sm text-gray-500">Halaman {page} / {totalPages}</span>
          <button onClick={() => setPage((p) => Math.min(totalPages, p + 1))} disabled={page === totalPages} className="btn-secondary py-2 px-4 text-sm disabled:opacity-40">Berikutnya</button>
        </div>
      )}

      {/* Create Booking Modal */}
      {showForm && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl shadow-xl w-full max-w-2xl max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between px-6 py-4 border-b">
              <h2 className="text-lg font-bold text-gray-900">Tambah Booking Baru</h2>
              <button onClick={() => setShowForm(false)}><X className="w-5 h-5 text-gray-400 hover:text-gray-700" /></button>
            </div>
            <form
              className="p-6 space-y-4"
              onSubmit={(e) => { e.preventDefault(); createMut.mutate(form) }}
            >
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label className="label">Nama Pelanggan *</label>
                  <input className="input" required value={form.customer_name} onChange={(e) => setForm((f) => ({ ...f, customer_name: e.target.value }))} />
                </div>
                <div>
                  <label className="label">Telepon</label>
                  <input className="input" value={form.customer_phone} onChange={(e) => setForm((f) => ({ ...f, customer_phone: e.target.value }))} />
                </div>
                <div>
                  <label className="label">Email</label>
                  <input className="input" type="email" value={form.customer_email} onChange={(e) => setForm((f) => ({ ...f, customer_email: e.target.value }))} />
                </div>
                <div>
                  <label className="label">Tanggal Berangkat *</label>
                  <input className="input" type="date" required value={form.departure_date} onChange={(e) => setForm((f) => ({ ...f, departure_date: e.target.value }))} />
                </div>
                <div>
                  <label className="label">Jumlah Orang</label>
                  <input className="input" type="number" min="1" value={form.num_people} onChange={(e) => setForm((f) => ({ ...f, num_people: Number(e.target.value) }))} />
                </div>
                <div>
                  <label className="label">Total Harga (Rp)</label>
                  <input className="input" type="number" min="0" value={form.total_price} onChange={(e) => setForm((f) => ({ ...f, total_price: Number(e.target.value) }))} />
                </div>
              </div>
              <div>
                <label className="label">Catatan</label>
                <textarea className="input" rows={2} value={form.notes} onChange={(e) => setForm((f) => ({ ...f, notes: e.target.value }))} />
              </div>

              {/* Participants */}
              <div>
                <div className="flex items-center justify-between mb-3">
                  <p className="text-sm font-semibold text-gray-700">Daftar Peserta</p>
                  <button type="button" onClick={handleAddParticipant} className="text-xs text-blue-600 hover:underline flex items-center gap-1">
                    <Plus className="w-3 h-3" /> Tambah Peserta
                  </button>
                </div>
                {form.participants.map((p, idx) => (
                  <div key={idx} className="border rounded-lg p-3 mb-3 bg-gray-50">
                    <div className="flex items-center justify-between mb-2">
                      <p className="text-xs font-medium text-gray-500">Peserta {idx + 1}</p>
                      {form.participants.length > 1 && (
                        <button type="button" onClick={() => handleRemoveParticipant(idx)}>
                          <X className="w-3.5 h-3.5 text-gray-400 hover:text-red-500" />
                        </button>
                      )}
                    </div>
                    <div className="grid grid-cols-2 gap-2">
                      <input className="input text-sm" placeholder="Nama Lengkap *" value={p.full_name} onChange={(e) => handleParticipantChange(idx, 'full_name', e.target.value)} />
                      <select className="input text-sm" value={p.id_type} onChange={(e) => handleParticipantChange(idx, 'id_type', e.target.value)}>
                        <option value="ktp">KTP</option>
                        <option value="passport">Passport</option>
                        <option value="sim">SIM</option>
                      </select>
                      <input className="input text-sm" placeholder="Nomor ID *" value={p.id_number} onChange={(e) => handleParticipantChange(idx, 'id_number', e.target.value)} />
                      <input className="input text-sm" type="date" placeholder="Tgl Lahir" value={p.date_of_birth} onChange={(e) => handleParticipantChange(idx, 'date_of_birth', e.target.value)} />
                      <input className="input text-sm col-span-2" placeholder="Telepon" value={p.phone} onChange={(e) => handleParticipantChange(idx, 'phone', e.target.value)} />
                    </div>
                  </div>
                ))}
              </div>

              <div className="flex gap-3 pt-2">
                <button type="submit" disabled={createMut.isPending} className="btn-primary py-2 px-6 text-sm disabled:opacity-60">
                  {createMut.isPending ? 'Menyimpan...' : 'Simpan Booking'}
                </button>
                <button type="button" onClick={() => setShowForm(false)} className="btn-secondary py-2 px-4 text-sm">Batal</button>
              </div>
              {createMut.isError && (
                <p className="text-sm text-red-600">Gagal menyimpan booking. Periksa kembali data Anda.</p>
              )}
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
