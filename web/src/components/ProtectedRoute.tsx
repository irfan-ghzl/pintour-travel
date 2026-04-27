import { Navigate, Outlet } from 'react-router-dom'
import { authStorage } from '../utils/auth'

export default function ProtectedRoute() {
  if (!authStorage.isLoggedIn()) {
    return <Navigate to="/login" replace />
  }
  return <Outlet />
}
