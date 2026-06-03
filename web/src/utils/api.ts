import axios from 'axios'

// Admin/staff API — uses httpOnly session cookie (PRD §19.1).
// Backend masih support Bearer header sebagai fallback (untuk Swagger UI / CLI).
const api = axios.create({
  baseURL: '/api/v1',
  headers: { 'Content-Type': 'application/json' },
  withCredentials: true, // §19.1: kirim cookie httpOnly otomatis
})

// Authorization header tidak lagi di-set manual — cookie httpOnly menanganinya.
// Bearer header tetap di-attach kalau token disimpan di localStorage (compat dev).
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('pintour_token')
  if (token && !config.headers.Authorization) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (res) => res,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('pintour_token')
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
