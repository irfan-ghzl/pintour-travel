// ─── Tour Packages ───────────────────────────────────────────────────────────

export interface Destination {
  id: string
  name: string
  country: string
  description?: string
  image_url?: string
}

export interface ItineraryItem {
  id: string
  tour_package_id: string
  day_number: number
  title: string
  description?: string
  location?: string
  start_time?: string
  end_time?: string
  activity_type?: string
  sort_order: number
}

export interface TourPackage {
  id: string
  destination_id?: string
  destination_name?: string
  destination_country?: string
  title: string
  slug: string
  description?: string
  price: number
  price_label?: string
  duration_days: number
  max_participants?: number
  min_participants: number
  package_type: string
  cover_image_url?: string
  is_active: boolean
  itinerary?: ItineraryItem[]
  created_at: string
}

export interface PackagesResponse {
  data: TourPackage[]
  total: number
  page: number
  per_page: number
}

// ─── Inquiries ────────────────────────────────────────────────────────────────

export interface CreateInquiryRequest {
  full_name: string
  email?: string
  phone?: string
  destination?: string
  tour_package_id?: string
  num_people: number
  budget?: number
  duration_days?: number
  departure_date?: string
  notes?: string
}

export interface CreateInquiryResponse {
  id: string
  wa_link: string
  created_at: string
  message: string
}

export interface Inquiry {
  id: string
  full_name: string
  email?: string
  phone?: string
  destination?: string
  num_people: number
  budget?: number
  duration_days?: number
  departure_date?: string
  status: string
  wa_link?: string
  package_title?: string
  created_at: string
}

export interface InquiriesResponse {
  data: Inquiry[]
  total: number
  page: number
  per_page: number
}

// ─── Quotations ───────────────────────────────────────────────────────────────

export interface QuotationItem {
  id: string
  description: string
  category?: string
  quantity: number
  unit_price: number
  total_price: number
}

export interface Quotation {
  id: string
  title: string
  customer_name: string
  customer_email?: string
  customer_phone?: string
  valid_until?: string
  total_price: number
  notes?: string
  status: string
  pdf_url?: string
  items?: QuotationItem[]
  created_by_name?: string
  created_at: string
}

export interface QuotationsResponse {
  data: Quotation[]
  total: number
  page: number
  per_page: number
}

// ─── Auth ─────────────────────────────────────────────────────────────────────

export interface LoginRequest {
  email: string
  password: string
}

export interface LoginResponse {
  token: string
  user_id: string
  name: string
  role: string
  expires_at: string
}

// ─── Testimonials ─────────────────────────────────────────────────────────────

export interface Testimonial {
  id: string
  customer_name: string
  content: string
  rating: number
  photo_url?: string
  created_at: string
}

// ─── Bookings ─────────────────────────────────────────────────────────────────

export interface BookingParticipant {
  id: string
  full_name: string
  id_type: string
  id_number: string
  date_of_birth?: string
  phone?: string
}

export interface Booking {
  id: string
  booking_code: string
  customer_name: string
  customer_email?: string
  customer_phone?: string
  package_title?: string
  package_id?: string
  departure_date: string
  num_people: number
  total_price: number
  payment_status: string
  booking_status: string
  notes?: string
  participants?: BookingParticipant[]
  created_at: string
}

export interface BookingsResponse {
  data: Booking[]
  total: number
  page: number
  per_page: number
}

