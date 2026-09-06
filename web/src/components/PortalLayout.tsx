import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { FileText, LayoutDashboard, List, BookOpen, LogOut, Receipt, Shield, User, History, Lock } from 'lucide-react'

import { usePaymentSettled, PAYMENT_LOCKED_TITLE } from '../hooks/usePortalMe'

// needsPayment marks the menus whose contents the API withholds until the
// payment is confirmed. They stay on the list in a locked state rather than
// disappearing: a participant who cannot find "Itinerary" concludes the portal
// does not have one, while a greyed-out "Itinerary" says it is waiting on them.
const navItems = [
  { to: '/portal', label: 'Dashboard', icon: LayoutDashboard, exact: true },
  { to: '/portal/trips', label: 'Riwayat', icon: History },
  { to: '/portal/invoices', label: 'Invoice', icon: Receipt },
  { to: '/portal/documents', label: 'Dokumen', icon: FileText },
  { to: '/portal/itinerary', label: 'Itinerary', icon: List, needsPayment: true },
  { to: '/portal/briefing', label: 'Briefing', icon: BookOpen, needsPayment: true },
  { to: '/portal/insurance', label: 'Asuransi', icon: Shield },
  { to: '/portal/profile', label: 'Profil', icon: User },
]

// NavEntry renders one menu item, as a link or as the locked stand-in. Both navs
// go through it: written out twice, the desktop sidebar and the mobile bar drift
// — one of them gets the lock and the other keeps handing out a live link.
function NavEntry({
  item,
  locked,
  active,
  className,
  children,
}: {
  item: (typeof navItems)[number]
  locked: boolean
  active: boolean
  className: (state: { locked: boolean; active: boolean }) => string
  children: React.ReactNode
}) {
  const cls = className({ locked, active })
  if (locked) {
    return (
      <span className={cls} title={PAYMENT_LOCKED_TITLE} aria-disabled="true">
        {children}
      </span>
    )
  }
  return (
    <Link to={item.to} className={cls}>
      {children}
    </Link>
  )
}

export default function PortalLayout() {
  const location = useLocation()
  const navigate = useNavigate()

  const participant = JSON.parse(localStorage.getItem('portal_participant') || '{}')
  // Read from the server, not from what login stored: the menus have to open the
  // moment a payment is confirmed, without the participant logging in again.
  const settled = usePaymentSettled()

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
          {navItems.map((item) => {
            const locked = Boolean(item.needsPayment) && !settled
            return (
              <NavEntry
                key={item.to}
                item={item}
                locked={locked}
                active={isActive(item.to, item.exact)}
                className={({ locked, active }) =>
                  `flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                    locked
                      ? 'text-gray-300 cursor-not-allowed'
                      : active
                        ? 'bg-emerald-50 text-emerald-700'
                        : 'text-gray-600 hover:bg-gray-50'
                  }`
                }
              >
                <item.icon size={15} />
                {item.label}
                {locked && <Lock size={12} className="ml-auto" />}
              </NavEntry>
            )
          })}
          {!settled && (
            <p className="mt-3 px-3 text-[11px] leading-relaxed text-gray-400">
              <Lock size={11} className="inline mr-1 -mt-0.5" />
              {PAYMENT_LOCKED_TITLE}. Anda tetap bisa membayar, mengunggah bukti
              transfer, dan melengkapi dokumen.
            </p>
          )}
        </nav>

        {/* Main content */}
        <main className="flex-1 p-4 md:p-6 max-w-4xl mx-auto w-full pb-24 md:pb-6">
          <Outlet />
        </main>
      </div>

      {/* Bottom nav (mobile) — show only 5 most important */}
      <nav className="md:hidden fixed bottom-0 left-0 right-0 bg-white border-t flex">
        {navItems.slice(0, 5).map((item) => {
          const locked = Boolean(item.needsPayment) && !settled
          return (
            <NavEntry
              key={item.to}
              item={item}
              locked={locked}
              active={isActive(item.to, item.exact)}
              className={({ locked, active }) =>
                `flex-1 flex flex-col items-center py-2 text-xs gap-0.5 ${
                  locked ? 'text-gray-300' : active ? 'text-emerald-700' : 'text-gray-500'
                }`
              }
            >
              <span className="relative">
                <item.icon size={20} />
                {locked && (
                  <Lock size={10} className="absolute -right-1.5 -top-1 bg-white rounded-full" />
                )}
              </span>
              {item.label}
            </NavEntry>
          )
        })}
      </nav>
    </div>
  )
}
