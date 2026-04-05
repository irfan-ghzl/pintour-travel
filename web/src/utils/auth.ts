import { LoginResponse } from '../types'

const TOKEN_KEY = 'pintour_token'
const USER_KEY = 'pintour_user'

export const authStorage = {
  setSession(data: LoginResponse) {
    localStorage.setItem(TOKEN_KEY, data.token)
    localStorage.setItem(USER_KEY, JSON.stringify({ id: data.user_id, name: data.name, role: data.role }))
  },
  clearSession() {
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USER_KEY)
  },
  getToken(): string | null {
    return localStorage.getItem(TOKEN_KEY)
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
    return !!localStorage.getItem(TOKEN_KEY)
  },
}
