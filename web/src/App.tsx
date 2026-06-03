import { Routes, Route } from 'react-router-dom'
import Layout from './components/Layout'
import AdminLayout from './components/AdminLayout'
import PortalLayout from './components/PortalLayout'
import ProtectedRoute from './components/ProtectedRoute'
import PortalProtectedRoute from './components/PortalProtectedRoute'

// Public pages
import CatalogPage from './pages/catalog/CatalogPage'
import PackageDetailPage from './pages/catalog/PackageDetailPage'
import LoginPage from './pages/LoginPage'
import ForgotPasswordPage from './pages/ForgotPasswordPage'
import ResetPasswordPage from './pages/ResetPasswordPage'
import PrivacyPolicyPage from './pages/PrivacyPolicyPage'
import NotFoundPage from './pages/NotFoundPage'

// Portal pages
import PortalLoginPage from './pages/portal/PortalLoginPage'
import PortalDashboardPage from './pages/portal/PortalDashboardPage'
import PortalInvoicePage from './pages/portal/PortalInvoicePage'
import PortalDocumentsPage from './pages/portal/PortalDocumentsPage'
import PortalItineraryPage from './pages/portal/PortalItineraryPage'
import PortalBriefingPage from './pages/portal/PortalBriefingPage'
import PortalInsurancePage from './pages/portal/PortalInsurancePage'
import PortalProfilePage from './pages/portal/PortalProfilePage'

// Admin pages
import AdminDashboardPage from './pages/admin/AdminDashboardPage'
import AdminPackagesPage from './pages/admin/AdminPackagesPage'
import AdminLeadsPage from './pages/admin/AdminLeadsPage'
import AdminParticipantsPage from './pages/admin/AdminParticipantsPage'
import AdminInvoicesPage from './pages/admin/AdminInvoicesPage'
import AdminDocumentsPage from './pages/admin/AdminDocumentsPage'
import AdminAirportPage from './pages/admin/AdminAirportPage'
import AdminUsersPage from './pages/admin/AdminUsersPage'
import AdminTourLeadersPage from './pages/admin/AdminTourLeadersPage'
import AdminCountryRequirementsPage from './pages/admin/AdminCountryRequirementsPage'

export default function App() {
  return (
    <Routes>
      {/* Public — e-catalog */}
      <Route element={<Layout />}>
        <Route path="/" element={<CatalogPage />} />
        <Route path="/packages/:slug" element={<PackageDetailPage />} />
      </Route>

      {/* Auth + static */}
      <Route path="/login" element={<LoginPage />} />
      <Route path="/forgot-password" element={<ForgotPasswordPage />} />
      <Route path="/reset-password" element={<ResetPasswordPage />} />
      <Route path="/privacy-policy" element={<PrivacyPolicyPage />} />
      <Route path="/portal/login" element={<PortalLoginPage />} />

      {/* Participant portal */}
      <Route element={<PortalProtectedRoute />}>
        <Route element={<PortalLayout />}>
          <Route path="/portal" element={<PortalDashboardPage />} />
          <Route path="/portal/invoices" element={<PortalInvoicePage />} />
          <Route path="/portal/documents" element={<PortalDocumentsPage />} />
          <Route path="/portal/itinerary" element={<PortalItineraryPage />} />
          <Route path="/portal/briefing" element={<PortalBriefingPage />} />
          <Route path="/portal/insurance" element={<PortalInsurancePage />} />
          <Route path="/portal/profile" element={<PortalProfilePage />} />
        </Route>
      </Route>

      {/* Admin */}
      <Route element={<ProtectedRoute />}>
        <Route element={<AdminLayout />}>
          <Route path="/admin" element={<AdminDashboardPage />} />
          <Route path="/admin/packages" element={<AdminPackagesPage />} />
          <Route path="/admin/leads" element={<AdminLeadsPage />} />
          <Route path="/admin/participants" element={<AdminParticipantsPage />} />
          <Route path="/admin/invoices" element={<AdminInvoicesPage />} />
          <Route path="/admin/documents" element={<AdminDocumentsPage />} />
          <Route path="/admin/airport" element={<AdminAirportPage />} />
          <Route path="/admin/users" element={<AdminUsersPage />} />
          <Route path="/admin/tour-leaders" element={<AdminTourLeadersPage />} />
          <Route path="/admin/country-requirements" element={<AdminCountryRequirementsPage />} />
        </Route>
      </Route>

      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  )
}
