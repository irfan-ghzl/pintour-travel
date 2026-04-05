import { Outlet, Link, useLocation, Navigate } from 'react-router-dom'
import {
  LayoutDashboard, Package, MessageSquare, FileText,
  LogOut, MapPin,
} from 'lucide-react'
import { authStorage } from '../utils/auth'

const sidebarLinks = [
  { to: '/admin', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/admin/packages', label: 'Paket Wisata', icon: Package },
  { to: '/admin/inquiries', label: 'Konsultasi', icon: MessageSquare },
  { to: '/admin/quotations', label: 'Penawaran', icon: FileText },
]

export default function AdminLayout() {
  const { pathname } = useLocation()
  const user = authStorage.getUser()

  if (!user) return <Navigate to="/login" replace />

  const handleLogout = () => {
    authStorage.clearSession()
    window.location.href = '/login'
  }

  return (
    <div className="min-h-screen flex bg-gray-100">
      {/* Sidebar */}
      <aside className="w-64 bg-gray-900 text-gray-300 flex flex-col">
        <div className="h-16 flex items-center gap-2 px-6 border-b border-gray-700">
          <MapPin className="w-5 h-5 text-accent-400" />
          <span className="font-bold text-white text-lg">Pintour Admin</span>
        </div>

        <nav className="flex-1 py-6 px-3 flex flex-col gap-1">
          {sidebarLinks.map(({ to, label, icon: Icon }) => {
            const active = pathname === to || (to !== '/admin' && pathname.startsWith(to))
            return (
              <Link
                key={to}
                to={to}
                className={`flex items-center gap-3 px-4 py-2.5 rounded-lg text-sm font-medium transition-colors ${
                  active
                    ? 'bg-primary-700 text-white'
                    : 'hover:bg-gray-800 text-gray-400 hover:text-white'
                }`}
              >
                <Icon className="w-4 h-4" />
                {label}
              </Link>
            )
          })}
        </nav>

        <div className="px-3 pb-6">
          <div className="px-4 py-3 rounded-lg bg-gray-800 text-xs text-gray-400 mb-2">
            {user.name} &bull; <span className="capitalize">{user.role}</span>
          </div>
          <button
            onClick={handleLogout}
            className="w-full flex items-center gap-3 px-4 py-2.5 rounded-lg text-sm
                       font-medium text-gray-400 hover:bg-gray-800 hover:text-red-400 transition-colors"
          >
            <LogOut className="w-4 h-4" />
            Keluar
          </button>
        </div>
      </aside>

      {/* Main content */}
      <div className="flex-1 flex flex-col min-w-0">
        <header className="h-16 bg-white border-b flex items-center px-6 shadow-sm">
          <h1 className="text-sm font-medium text-gray-500">
            {sidebarLinks.find((l) => l.to === pathname || (l.to !== '/admin' && pathname.startsWith(l.to)))?.label ?? 'Admin'}
          </h1>
        </header>
        <main className="flex-1 p-6 overflow-auto">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
