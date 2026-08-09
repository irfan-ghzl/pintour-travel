import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { FileText, LayoutDashboard, List, BookOpen, LogOut, Receipt, Shield, User, History } from 'lucide-react'

const navItems = [
  { to: '/portal', label: 'Dashboard', icon: LayoutDashboard, exact: true },
  { to: '/portal/trips', label: 'Riwayat', icon: History },
  { to: '/portal/invoices', label: 'Invoice', icon: Receipt },
  { to: '/portal/documents', label: 'Dokumen', icon: FileText },
  { to: '/portal/itinerary', label: 'Itinerary', icon: List },
  { to: '/portal/briefing', label: 'Briefing', icon: BookOpen },
  { to: '/portal/insurance', label: 'Asuransi', icon: Shield },
  { to: '/portal/profile', label: 'Profil', icon: User },
]

export default function PortalLayout() {
  const location = useLocation()
  const navigate = useNavigate()

  const participant = JSON.parse(localStorage.getItem('portal_participant') || '{}')

  function logout() {
    localStorage.removeItem('portal_token')
    localStorage.removeItem('portal_participant')
    navigate('/portal/login')
  }

  function isActive(to: string, exact?: boolean) {
    if (exact) return location.pathname === to
    return location.pathname.startsWith(to)
  }

  return (
    <div className="min-h-screen bg-gray-50 flex flex-col">
      {/* Header */}
      <header className="bg-emerald-700 text-white px-4 py-3 flex items-center justify-between shadow">
        <div>
          <h1 className="text-sm font-semibold">Portal Peserta</h1>
          <p className="text-xs text-emerald-200">{participant.name || '—'} · {participant.package_name || '—'}</p>
        </div>
        <button onClick={logout} className="flex items-center gap-1 text-sm text-emerald-200 hover:text-white">
          <LogOut size={16} />
          Keluar
        </button>
      </header>

      <div className="flex flex-1">
        {/* Sidebar (desktop) */}
        <nav className="hidden md:flex flex-col w-56 bg-white border-r p-4 gap-0.5">
          {navItems.map((item) => (
            <Link
              key={item.to}
              to={item.to}
              className={`flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                isActive(item.to, item.exact)
                  ? 'bg-emerald-50 text-emerald-700'
                  : 'text-gray-600 hover:bg-gray-50'
              }`}
            >
              <item.icon size={15} />
              {item.label}
            </Link>
          ))}
        </nav>

        {/* Main content */}
        <main className="flex-1 p-4 md:p-6 max-w-4xl mx-auto w-full pb-24 md:pb-6">
          <Outlet />
        </main>
      </div>

      {/* Bottom nav (mobile) — show only 5 most important */}
      <nav className="md:hidden fixed bottom-0 left-0 right-0 bg-white border-t flex">
        {navItems.slice(0, 5).map((item) => (
          <Link
            key={item.to}
            to={item.to}
            className={`flex-1 flex flex-col items-center py-2 text-xs gap-0.5 ${
              isActive(item.to, item.exact) ? 'text-emerald-700' : 'text-gray-500'
            }`}
          >
            <item.icon size={20} />
            {item.label}
          </Link>
        ))}
      </nav>
    </div>
  )
}
