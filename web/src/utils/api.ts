import axios from 'axios'

import type { PackageBatch, PaginatedResponse, Participant } from '../types'
import { saveBlob } from './download'

// Admin/staff API — authentication is carried solely by the httpOnly session
// cookie (PRD §19.1). The JWT is never read from JavaScript, so there is no
// Authorization header to attach here — that is what keeps it safe from XSS.
const api = axios.create({
  baseURL: '/api/v1',
  headers: { 'Content-Type': 'application/json' },
  withCredentials: true, // §19.1: kirim cookie httpOnly otomatis
})

api.interceptors.response.use(
  (res) => res,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('pintour_user')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  },
)

export default api

// Portal API — uses X-Portal-Token header (peserta short-lived token §19.1)
export const portalApi = axios.create({
  baseURL: '/api/v1',
  headers: { 'Content-Type': 'application/json' },
})

portalApi.interceptors.request.use((config) => {
  const token = localStorage.getItem('portal_token')
  if (token) {
    config.headers['X-Portal-Token'] = token
  }
  return config
})

portalApi.interceptors.response.use(
  (res) => res,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('portal_token')
      localStorage.removeItem('portal_participant')
      window.location.href = '/portal/login'
    }
    return Promise.reject(error)
  },
)

// ── Riwayat perjalanan / returning customer (v2.0 F2) ─────────────────────────
export interface TripCard {
  participant_id: string
  package_name: string
  room_type: string
  departure_date: string | null
  is_active: boolean
  payment_status: string
  // FR-PORTAL-09 badge, present on history cards only: "selesai" for a trip
  // paid in full before its departure passed, "dibatalkan" for one that was not.
  completion_status?: 'selesai' | 'dibatalkan'
}
export interface MyTripsResponse {
  active: TripCard[]
  history: TripCard[]
}

export const getMyTrips = (): Promise<MyTripsResponse> =>
  portalApi.get('/portal/my-trips').then((r) => r.data.data ?? { active: [], history: [] })

// downloadTripInvoice fetches the invoice PDF of any tour (incl. past trips) and
// triggers a browser download (F2 — download artefak lama).
export const downloadTripInvoice = async (participantId: string) => {
  const res = await portalApi.get(`/portal/my-trips/${participantId}/invoice-pdf`, {
    responseType: 'blob',
  })
  saveBlob(res.data as Blob, `invoice-${participantId.slice(0, 8)}.pdf`)
}

// ── OCR (v2.0 F6 — self-hosted Tesseract) ─────────────────────────────────────
export interface OCRExtraction {
  document_number?: string
  name?: string
  birth_date?: string
  nationality?: string
  expiry_date?: string
  confidence?: number
}
export interface OCRResult {
  id: string
  document_id: string
  extracted_data: OCRExtraction
  confidence: number
  validation_passed: boolean
  validation_notes: string
  created_at: string
}

export const getDocumentOCR = (documentId: string): Promise<OCRResult> =>
  api.get(`/admin/documents/${documentId}/ocr-result`).then((r) => r.data.data)

export const applyParticipantNIK = (participantId: string, nik: string) =>
  api.patch(`/admin/participants/${participantId}/nik`, { nik }).then((r) => r.data)

// ── Pemilih batch & peserta (admin) ───────────────────────────────────────────
// Empat halaman admin dulu meminta operator mengetikkan UUID yang tidak pernah
// ditampilkan aplikasi di mana pun. Kedua pembacaan di bawah ini adalah sumber
// data pemilih yang menggantikannya.

// BATCH_PAGE_SIZE sama dengan `maxPerPage` di helpers.go — server meng-clamp ke
// angka itu, jadi meminta lebih tidak menghasilkan apa pun selain harapan palsu.
const BATCH_PAGE_SIZE = 100

export interface BatchPage {
  items: PackageBatch[]
  // total adalah cacah seluruh batch yang cocok, bukan yang terkirim. Pemilih
  // membandingkan keduanya supaya pemotongan daftar tidak terjadi diam-diam.
  total: number
}

// listAdminBatches membaca keberangkatan dari seluruh paket sekaligus — halaman
// Peserta, Invoice, dan Airport Handling tidak punya paket di tangan lebih dulu.
// Server mengurutkan terdekat lebih dulu, jadi halaman pertama adalah 100
// keberangkatan paling relevan.
export const listAdminBatches = (upcoming = true): Promise<BatchPage> =>
  api
    .get<PaginatedResponse<PackageBatch>>('/admin/batches', {
      params: { ...(upcoming ? { upcoming: 'true' } : {}), per_page: BATCH_PAGE_SIZE },
    })
    .then((r) => ({ items: r.data.data ?? [], total: r.data.meta?.total ?? 0 }))

// listPackageBatches membaca keberangkatan satu paket. Dipakai dialog konversi
// lead, yang memang harus membatasi pilihan ke paket yang diminati lead itu.
export const listPackageBatches = (packageId: string): Promise<PackageBatch[]> =>
  api
    .get<{ success: boolean; data: PackageBatch[] }>(`/admin/packages/${packageId}/batches`)
    .then((r) => r.data.data ?? [])

// searchParticipants mencari peserta lewat potongan nama atau nomor WhatsApp.
// Endpoint-nya sudah disaring per konsultan, jadi pembatasan akses yang ada ikut
// berlaku tanpa ditulis ulang di sini.
export const searchParticipants = (search: string, perPage = 10): Promise<Participant[]> =>
  api
    .get<PaginatedResponse<Participant>>('/admin/participants', {
      params: { ...(search ? { search } : {}), per_page: perPage },
    })
    .then((r) => r.data.data ?? [])

// ── Chatbot logs (v2.0 F2) ────────────────────────────────────────────────────
export interface ChatbotConversation {
  phone: string
  message_count: number
  first_chat: string
  last_chat: string
  lead_id?: string | null
}
export interface ChatbotLog {
  id: string
  phone: string
  role: 'user' | 'assistant'
  message: string
  created_at: string
}

export const getChatbotConversations = (params: {
  phone?: string; from?: string; to?: string; page?: number; limit?: number
}) => api.get('/admin/chatbot-logs', { params }).then((r) => r.data)

export const getChatbotConversation = (phone: string): Promise<ChatbotLog[]> =>
  api.get(`/admin/chatbot-logs/${encodeURIComponent(phone)}`).then((r) => r.data.data ?? [])

export const createLeadFromChat = (
  phone: string,
  body: { name: string; package_id: string; pax?: number },
) => api.post(`/admin/chatbot-logs/${encodeURIComponent(phone)}/create-lead`, body).then((r) => r.data)

// ── Payment gateway (v2.0 F1) ─────────────────────────────────────────────────
export interface CreatePaymentResponse {
  snap_token: string
  client_key: string
}

export const createPortalPayment = (invoiceId: string): Promise<CreatePaymentResponse> =>
  portalApi.post(`/portal/invoices/${invoiceId}/create-payment`).then((r) => r.data.data)

// ConsultationPrefill is what FR-PORTAL-12 fills the public lead form with.
// Note what it does NOT contain: returning-customer status. The lead endpoint
// decides that from the phone number itself, so nothing sent from here can
// claim it.
export interface ConsultationPrefill {
  name: string
  phone: string
  email: string
  room_type: string
}

export const getConsultationPrefill = (): Promise<ConsultationPrefill> =>
  portalApi.get('/portal/consultation-prefill').then((r) => r.data.data)

// getTripItinerary returns the itinerary of one past trip (FR-PORTAL-10). The
// server checks the trip belongs to the caller's portal identity.
export const getTripItinerary = (participantId: string) =>
  portalApi.get(`/portal/my-trips/${participantId}/itinerary`).then((r) => r.data.data)

// ── Automation feature helpers (prompt §5.9) ──────────────────────────────────
// These wrap the actual backend routes (clean-arch adaptation of the spec).

export interface DashboardAnalytics {
  leads_summary: {
    total: number; baru: number; proses: number; deal: number
    expired: number; conversion_rate: number
  }
  revenue_summary: {
    total_invoiced: number; total_paid: number
    total_pending: number; total_overdue: number
  }
  batch_summary: {
    total_active_batches: number; total_participants: number
    nearest_departure: {
      batch_id: string; package_name: string; departure_date: string
      days_remaining: number; participant_count: number
    } | null
  }
  monthly_leads: { month: string; count: number }[]
}

export const getDashboardAnalytics = (): Promise<DashboardAnalytics> =>
  api.get('/admin/dashboard/analytics').then((r) => r.data.data)

// verifyPaymentProof reviews a payment proof; on 'disetujui' the backend derives
// the paid amount, settles the invoice, and activates the portal (§1.3/§1.4).
export const verifyPaymentProof = (
  invoiceId: string,
  proofId: string,
  status: 'disetujui' | 'ditolak',
  notes?: string,
) =>
  api
    .patch(`/admin/invoices/${invoiceId}/proofs/${proofId}/review`, { status, notes })
    .then((r) => r.data)

// PrivateFileKind names a resource the signer can unlock. The server resolves
// the bucket and path from the id; neither is sent from here, so a tampered
// request can only ask for a resource, never for a location.
export type PrivateFileKind = 'document' | 'payment_proof'

// openSignedFile asks the server for a short-lived signed URL to a private file
// and opens it in a new tab. Objects in the private buckets cannot be opened
// directly (§19.2). `path` is only inspected for the manual-URL fallback used
// when storage is disabled, where the stored value is already an absolute URL.
export const openSignedFile = async (kind: PrivateFileKind, id: string, path?: string) => {
  if (path && /^https?:\/\//.test(path)) {
    window.open(path, '_blank', 'noopener')
    return
  }
  if (!id) return
  const res = await api.get('/admin/signed-url', { params: { type: kind, id } })
  const url = res.data?.data?.url as string | undefined
  // Throw rather than no-op: callers show an error toast on rejection, and a
  // silent return leaves the user clicking a button that does nothing.
  if (!url) throw new Error('Server tidak mengembalikan URL berkas')
  window.open(url, '_blank', 'noopener')
}

export const approveDocument = (id: string) =>
  api.patch(`/admin/documents/${id}/review`, { status: 'disetujui' }).then((r) => r.data)

export const rejectDocument = (id: string, reason: string) =>
  api
    .patch(`/admin/documents/${id}/review`, { status: 'ditolak', rejection_reason: reason })
    .then((r) => r.data)

export type ReportType = 'leads' | 'participants' | 'invoices' | 'batches'
export type ReportFormat = 'excel' | 'pdf'

// exportReport downloads a report file (blob) and triggers a browser download.
export const exportReport = async (type: ReportType, format: ReportFormat) => {
  const res = await api.get('/admin/reports/export', {
    params: { type, format },
    responseType: 'blob',
  })
  const ext = format === 'excel' ? 'xlsx' : 'pdf'
  const stamp = new Date().toISOString().slice(0, 10).replace(/-/g, '')
  saveBlob(res.data as Blob, `pintour-${type}-${stamp}.${ext}`)
}
