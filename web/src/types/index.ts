// ─── Auth ─────────────────────────────────────���───────────────────────────────

export interface LoginRequest {
  email: string
  password: string
}

export interface LoginResponse {
  token: string
  user_id: string
  name: string
  role: string // super_admin | admin | konsultan | tour_leader
  expires_at: string
}

export interface PortalLoginRequest {
  phone: string
  password: string
}

export interface PortalLoginResponse {
  token: string
  participant_id: string
  name: string
  package_name: string
}

// ─── Packages ───────────���─────────────────────────────────────────────────────

export interface ItineraryDay {
  day: number
  title: string
  description: string
  activities: string[]
}

export interface Package {
  id: string
  name: string
  slug: string
  destination: string
  category: 'reguler' | 'umroh' | 'halal' | 'honeymoon'
  duration_days: number
  description: string
  itinerary: ItineraryDay[]
  requirements: string[]
  facilities: string[]          // FR-CAT-03 "fasilitas yang disertakan"
  terms_conditions: string      // FR-CAT-03 "syarat & ketentuan"
  visa_info: string             // FR-CAT-03 "informasi visa"
  base_price: number
  is_active: boolean
  thumbnail: string
  created_by: string
  created_at: string
  updated_at: string
}

export interface PackageImage {
  id: string
  package_id: string
  file_path: string
  alt_text: string
  sort_order: number
  is_thumbnail: boolean
  created_at: string
}

export interface PackageBatch {
  id: string
  package_id: string
  departure_date: string
  return_date: string
  quota: number
  price_single: number
  price_double: number
  price_triple: number
  status: 'tersedia' | 'penuh' | 'ditutup'
  tour_leader_id?: string
  created_at: string
  updated_at: string
  package_name: string
  tour_leader_name?: string
}

export interface PackageDetailResponse {
  package: Package
  batches: PackageBatch[]
  images: PackageImage[]
}

// ─── Leads / CRM ───────────────────��──────────────────────��───────────────────

export type LeadStatus = 'baru' | 'dihubungi' | 'konsultasi' | 'deal' | 'tidak_deal' | 'peserta'
export type LeadSource = 'meta_ads' | 'organic' | 'referral' | 'walk_in'

export interface Lead {
  id: string
  name: string
  phone: string
  email: string
  package_id: string
  batch_id?: string
  pax: number
  message: string
  source: LeadSource
  status: LeadStatus
  assigned_to?: string
  converted_at?: string
  portal_user_id?: string // v2.0 F4
  is_returning?: boolean  // v2.0 F4 — returning customer
  created_at: string
  updated_at: string
  package_name: string
  assignee_name?: string
}

export interface LeadNote {
  id: string
  lead_id: string
  user_id: string
  note: string
  created_at: string
  user_name: string
}

// LeadStatusChange is one recorded transition (FR-CRM-02). It is separate from
// LeadNote because a note is something a consultant wrote and a status change is
// something that happened — and only the latter knows what the previous status
// was. changed_by is empty when the nightly job made the change; changed_by_name
// then reads "Sistem".
export interface LeadStatusChange {
  id: string
  lead_id: string
  from_status: string
  to_status: string
  changed_by: string
  changed_by_name: string
  changed_at: string
}

// DocumentSummary counts a participant's documents across ALL of them, which is
// what the "N of M approved" figure on the review page has to describe — not the
// page or the status filter in front of it.
export interface DocumentSummary {
  total: number
  disetujui: number
  menunggu: number
  ditolak: number
}

export interface CreateLeadRequest {
  name: string
  phone: string
  email?: string
  package_id: string
  batch_id?: string
  pax: number
  message?: string
  source?: LeadSource
}

// ─── Participants ────────────────��───────────────────────��────────────────────

export interface Participant {
  id: string
  lead_id?: string
  batch_id: string
  name: string
  phone: string
  email: string
  room_type: 'single' | 'double' | 'triple'
  is_active: boolean
  briefing_viewed: boolean
  created_at: string
  updated_at: string
  batch_departure_date?: string
  package_name: string
}

export interface ConvertLeadRequest {
  lead_id: string
  batch_id: string
  room_type: 'single' | 'double' | 'triple'
}

export interface ConvertLeadResponse {
  participant: Participant
  temp_password: string
  reused_account: boolean // v2.0 F1 — true bila akun portal lama dipakai ulang
  message: string
}

// ─── Invoices ──��───────────────────���───────────────────────────────��──────────

// Mirrors the invoices_status_check constraint as amended by
// db/migrations/006_v2_features.sql. menunggu_konfirmasi_gateway was missing
// here while the backend already set it, so a participant whose payment was
// awaiting the gateway hit an undefined lookup and the page failed to render.
export type InvoiceStatus =
  | 'diterbitkan'
  | 'menunggu_bayar'
  | 'menunggu_konfirmasi_gateway'
  | 'dibayar'
  | 'lunas'

export interface Invoice {
  id: string
  invoice_number: string
  participant_id: string
  batch_id: string
  amount: number
  due_date: string
  status: InvoiceStatus
  pdf_path: string
  notes: string
  issued_by: string
  confirmed_by?: string
  confirmed_at?: string
  created_at: string
  updated_at: string
  participant_name: string
  participant_phone: string
  package_name: string
  issued_by_name: string
}

export interface PaymentProof {
  id: string
  invoice_id: string
  file_path: string
  amount_claimed: number
  notes: string
  status: 'menunggu' | 'disetujui' | 'ditolak'
  reviewed_by?: string
  review_notes: string
  uploaded_at: string
  reviewed_at?: string
}

export interface CreateInvoiceRequest {
  participant_id: string
  batch_id: string
  amount: number
  due_date: string
  notes?: string
}

export interface UploadProofRequest {
  file_path: string
  amount_claimed: number
  notes?: string
}

// ─── Documents ─────────────────���──────────────────────────────��───────────────

export type DocumentType = 'passport' | 'ktp' | 'rekening_koran' | 'visa_support' | 'lainnya'
export type DocumentStatus = 'menunggu' | 'disetujui' | 'ditolak'

export interface Document {
  id: string
  participant_id: string
  document_type: DocumentType
  file_path: string
  file_name: string
  status: DocumentStatus
  rejection_reason: string
  reviewed_by?: string
  uploaded_at: string
  reviewed_at?: string
  participant_name: string
}

export interface CountryRequirement {
  id: string
  country_code: string
  country_name: string
  document_type: string
  is_required: boolean
  description: string
  created_at: string
}

export interface UploadDocumentRequest {
  document_type: DocumentType
  file_path: string
  file_name: string
}

// ─── Airport Handling ───────────���─────────────────────────────────────────────

export interface AirportChecklist {
  id: string
  participant_id: string
  batch_id: string
  baggage_checked: boolean
  baggage_checked_at?: string
  ticket_distributed: boolean
  ticket_distributed_at?: string
  passport_returned: boolean
  passport_returned_at?: string
  handled_by?: string
  notes: string
  updated_at: string
  participant_name: string
  participant_phone: string
  handled_by_name?: string
}

export interface BatchProgress {
  batch_id: string
  total_pax: number
  done_count: number
  pending_count: number
}

export interface AirportChecklistResponse {
  checklists: AirportChecklist[]
  progress: BatchProgress
}

// ─── Users ────────────���───────────────────────────────────���───────────────────

export type UserRole = 'super_admin' | 'admin' | 'konsultan' | 'tour_leader'

export interface User {
  id: string
  name: string
  email: string
  role: UserRole
  phone: string
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface TourLeader {
  id: string
  user_id: string
  bio: string
  photo_path: string
  experience_years: number
  specialization: string
  emergency_phone: string
  created_at: string
  name: string
  email: string
  phone: string
}

// ─── API meta ─────────────��─────────────────────────────��─────────────────────

export interface PaginatedResponse<T> {
  success: boolean
  data: T[]
  meta: {
    page: number
    limit: number
    total: number
    total_pages: number
  }
}

export interface ApiResponse<T> {
  success: boolean
  data: T
  message?: string
}

export interface ApiError {
  success: false
  error: string
  message: string
  details?: Record<string, string>
}

// ─── WA Notifications ────────────────────────────────────────────────────────

export interface WANotification {
  id: string
  recipient_phone: string
  recipient_name: string
  message_type: string
  message_content: string
  reference_id?: string
  reference_type?: string
  status: 'pending' | 'sent' | 'failed'
  fonnte_response?: unknown
  sent_at?: string
  created_at: string
}

// ─── Portal ───────────────────────────────────────────────────────────────────

export interface PortalMeResponse {
  participant: Participant
  days_to_depart?: number
  briefing_active: boolean
  passport_expiry?: string         // v2.0 FR-PORTAL-11 (YYYY-MM-DD)
  passport_expiring_soon?: boolean // expired or within 6 months
}
