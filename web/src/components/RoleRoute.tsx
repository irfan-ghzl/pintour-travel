import { Outlet } from 'react-router-dom'
import { authStorage } from '../utils/auth'
import ForbiddenPage from '../pages/ForbiddenPage'

// RoleRoute gates a group of routes to the given roles. It runs inside
// AdminLayout, so an unauthorized role sees the 403 page with the sidebar still
// rendered. This mirrors the backend RequireRole middleware — the sidebar hiding
// links is cosmetic; this (and the API) is what actually enforces access.
export default function RoleRoute({ allowedRoles }: { allowedRoles: string[] }) {
  const role = authStorage.getUser()?.role ?? ''
  if (!allowedRoles.includes(role)) {
    return <ForbiddenPage />
  }
  return <Outlet />
}
