import { Routes, Route } from 'react-router-dom'
import Layout from './components/Layout'
import HomePage from './pages/HomePage'
import PackagesPage from './pages/PackagesPage'
import PackageDetailPage from './pages/PackageDetailPage'
import BuildMyTripPage from './pages/BuildMyTripPage'
import TestimonialsPage from './pages/TestimonialsPage'
import LoginPage from './pages/LoginPage'
import AdminLayout from './components/AdminLayout'
import AdminDashboardPage from './pages/admin/AdminDashboardPage'
import AdminPackagesPage from './pages/admin/AdminPackagesPage'
import AdminInquiriesPage from './pages/admin/AdminInquiriesPage'
import AdminQuotationsPage from './pages/admin/AdminQuotationsPage'
import AdminBookingsPage from './pages/admin/AdminBookingsPage'
import NotFoundPage from './pages/NotFoundPage'
import ProtectedRoute from './components/ProtectedRoute'

export default function App() {
  return (
    <Routes>
      {/* Public routes */}
      <Route element={<Layout />}>
        <Route path="/" element={<HomePage />} />
        <Route path="/packages" element={<PackagesPage />} />
        <Route path="/packages/:slug" element={<PackageDetailPage />} />
        <Route path="/build-my-trip" element={<BuildMyTripPage />} />
        <Route path="/testimonials" element={<TestimonialsPage />} />
      </Route>

      {/* Auth */}
      <Route path="/login" element={<LoginPage />} />

      {/* Admin routes */}
      <Route element={<ProtectedRoute />}>
        <Route element={<AdminLayout />}>
          <Route path="/admin" element={<AdminDashboardPage />} />
          <Route path="/admin/packages" element={<AdminPackagesPage />} />
          <Route path="/admin/inquiries" element={<AdminInquiriesPage />} />
          <Route path="/admin/quotations" element={<AdminQuotationsPage />} />
          <Route path="/admin/bookings" element={<AdminBookingsPage />} />
        </Route>
      </Route>

      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  )
}
