import { Outlet, Link, useLocation } from 'react-router-dom'
import {
  LayoutDashboard, Package, Users, FileText, ClipboardList,
  Plane, FolderCheck, LogOut, MapPin, Shield, UserCheck, Globe,
} from 'lucide-react'
import { authStorage } from '../utils/auth'

const sidebarLinks = [
  { to: '/admin', label: 'Dashboard', icon: LayoutDashboard, exact: true, roles: [] },
  { to: '/admin/packages', label: 'Paket Wisata', icon: Package, roles: [] },
  { to: '/admin/leads', label: 'CRM Leads', icon: Users, roles: ['super_admin', 'admin', 'konsultan'] },
  { to: '/admin/participants', label: 'Peserta', icon: ClipboardList, roles: ['super_admin', 'admin'] },
  { to: '/admin/invoices', label: 'Invoice', icon: FileText, roles: ['super_admin', 'admin'] },
  { to: '/admin/documents', label: 'Review Dokumen', icon: FolderCheck, roles: ['super_admin', 'admin'] },
  { to: '/admin/airport', label: 'Airport Handling', icon: Plane, roles: [] },
  { to: '/admin/tour-leaders', label: 'Profil Tour Leader', icon: UserCheck, roles: ['super_admin', 'admin'] },
  { to: '/admin/country-requirements', label: 'Persyaratan Dokumen', icon: Globe, roles: ['super_admin', 'admin'] },
  { to: '/admin/users', label: 'Manajemen User', icon: Shield, roles: ['super_admin'] },
]

export default function AdminLayout() {
  const { pathname } = useLocation()
  const user = authStorage.getUser()
  const role = user?.role ?? ''

  const visibleLinks = sidebarLinks.filter(l =>
    l.roles.length === 0 || l.roles.includes(role)
  )

  function isActive(to: string, exact?: boolean) {
    if (exact) return pathname === to
    return pathname === to || pathname.startsWith(to + '/')
  }

  function handleLogout() {
    authStorage.clearSession()
    window.location.href = '/login'
  }

  return (
    <div className="min-h-screen flex bg-gray-100">
      {/* Sidebar */}
      <aside className="w-60 bg-gray-900 text-gray-300 flex flex-col shrink-0">
        <div className="h-14 flex items-center gap-2 px-5 border-b border-gray-700">
          <MapPin className="w-5 h-5 text-emerald-400" />
          <span className="font-bold text-white text-sm">Pintour Admin</span>
        </div>

        <nav className="flex-1 py-4 px-3 flex flex-col gap-0.5 overflow-y-auto">
          {visibleLinks.map(({ to, label, icon: Icon, exact }) => (
            <Link
              key={to}
              to={to}
              className={`flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                isActive(to, exact)
                  ? 'bg-emerald-700 text-white'
                  : 'text-gray-400 hover:bg-gray-800 hover:text-white'
              }`}
            >
              <Icon className="w-4 h-4 shrink-0" />
              {label}
            </Link>
          ))}
        </nav>

        <div className="px-3 pb-4">
          <div className="px-3 py-2.5 rounded-lg bg-gray-800 text-xs text-gray-400 mb-2">
            <p className="font-medium text-gray-200 truncate">{user?.name ?? '—'}</p>
            <p className="capitalize">{role?.replace('_', ' ') ?? '—'}</p>
          </div>
          <button
            onClick={handleLogout}
            className="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm
                       font-medium text-gray-400 hover:bg-gray-800 hover:text-red-400 transition-colors"
          >
            <LogOut className="w-4 h-4" />
            Keluar
          </button>
        </div>
      </aside>

      {/* Main */}
      <div className="flex-1 flex flex-col min-w-0">
        <header className="h-13 bg-white border-b flex items-center px-6 shadow-sm shrink-0 py-3.5">
          <h1 className="text-sm font-semibold text-gray-600">
            {visibleLinks.find(l => isActive(l.to, l.exact))?.label ?? 'Admin'}
          </h1>
        </header>
        <main className="flex-1 p-6 overflow-auto">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
