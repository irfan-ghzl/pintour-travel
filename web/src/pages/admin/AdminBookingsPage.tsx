import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { ChevronDown, ChevronUp, X, Plus, Trash2 } from 'lucide-react'
import api from '../../utils/api'
import { Booking, BookingsResponse, BookingParticipant } from '../../types'
import Spinner from '../../components/Spinner'

const PAYMENT_STATUSES = ['pending', 'dp', 'lunas', 'refund']
const BOOKING_STATUSES = ['confirmed', 'cancelled', 'completed']

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
    cancelled: 'bg-red-100 text-red-700',
    completed: 'bg-green-100 text-green-700',
  }
  return map[s] ?? 'bg-gray-100 text-gray-500'
}

const fmtRp = (n: number) =>
  new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(n)

const EMPTY_PARTICIPANT = { full_name: '', id_type: 'ktp', id_number: '', date_of_birth: '', phone: '' }

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
          {BOOKING_STATUSES.map((s) => <option key={s} value={s}>{s}</option>)}
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
                        <option key={s} value={s}>{s}</option>
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
                    <td colSpan={9} className="px-8 py-4">
                      <p className="text-xs font-semibold text-gray-500 uppercase mb-3">Daftar Peserta (Manifest)</p>
                      {!detailData ? (
                        <p className="text-xs text-gray-400">Memuat...</p>
                      ) : detailData.participants && detailData.participants.length > 0 ? (
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
                            {detailData.participants.map((p: BookingParticipant) => (
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
