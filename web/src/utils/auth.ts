import { LoginResponse } from '../types'

const USER_KEY = 'pintour_user'

// NOTE: the admin/staff JWT is NEVER stored in localStorage. Authentication is
// carried by the httpOnly session cookie set by the backend (§19.1), which is
// not readable by JavaScript and therefore safe against XSS token theft. We only
// persist non-sensitive user display info so the UI knows who is logged in.
export const authStorage = {
  setSession(data: LoginResponse) {
    localStorage.setItem(USER_KEY, JSON.stringify({ id: data.user_id, name: data.name, role: data.role }))
  },
  clearSession() {
    localStorage.removeItem(USER_KEY)
  },
  getUser(): { id: string; name: string; role: string } | null {
    const raw = localStorage.getItem(USER_KEY)
    if (!raw) return null
    try {
      return JSON.parse(raw)
    } catch {
      return null
    }
  },
  isLoggedIn(): boolean {
    return !!localStorage.getItem(USER_KEY)
  },
}
