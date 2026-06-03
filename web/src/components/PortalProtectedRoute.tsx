import { Navigate, Outlet } from 'react-router-dom'

export default function PortalProtectedRoute() {
  const token = localStorage.getItem('portal_token')
  if (!token) return <Navigate to="/portal/login" replace />
  return <Outlet />
}
