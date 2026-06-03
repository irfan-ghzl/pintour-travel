import { Outlet, Link, useLocation } from 'react-router-dom'
import { MapPin, Menu, X, LogIn } from 'lucide-react'
import { useState } from 'react'

const navLinks = [
  { to: '/', label: 'Katalog' },
  { to: '/portal/login', label: 'Portal Peserta' },
]

export default function Layout() {
  const { pathname } = useLocation()
  const [menuOpen, setMenuOpen] = useState(false)

  return (
    <div className="min-h-screen flex flex-col bg-gray-50">
      {/* Navbar */}
      <header className="bg-white shadow-sm sticky top-0 z-30">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <Link to="/" className="flex items-center gap-2 font-bold text-emerald-700 text-xl">
              <MapPin className="w-6 h-6 text-emerald-500" />
              Pintour Travel
            </Link>

            {/* Desktop nav */}
            <nav className="hidden md:flex items-center gap-6">
              {navLinks.map((link) => (
                <Link
                  key={link.to}
                  to={link.to}
                  className={`text-sm font-medium transition-colors ${
                    pathname === link.to ? 'text-emerald-600' : 'text-gray-600 hover:text-emerald-600'
                  }`}
                >
                  {link.label}
                </Link>
              ))}
              <Link
                to="/portal/login"
                className="flex items-center gap-1.5 px-4 py-2 bg-emerald-600 text-white text-sm rounded-lg hover:bg-emerald-700 transition-colors"
              >
                <LogIn size={14} /> Masuk Portal
              </Link>
            </nav>

            <button className="md:hidden p-2 text-gray-600" onClick={() => setMenuOpen(!menuOpen)}>
              {menuOpen ? <X size={20} /> : <Menu size={20} />}
            </button>
          </div>
        </div>

        {menuOpen && (
          <div className="md:hidden border-t px-4 py-3 flex flex-col gap-3 bg-white">
            {navLinks.map((link) => (
              <Link key={link.to} to={link.to}
                className="text-sm font-medium text-gray-700"
                onClick={() => setMenuOpen(false)}>
                {link.label}
              </Link>
            ))}
          </div>
        )}
      </header>

      <main className="flex-1">
        <Outlet />
      </main>

      <footer className="bg-gray-900 text-gray-400 py-10">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex flex-col md:flex-row items-center justify-between gap-4">
            <div className="flex items-center gap-2 text-white font-bold text-lg">
              <MapPin className="w-5 h-5 text-emerald-400" />
              Pintour Travel
            </div>
            <div className="flex flex-wrap justify-center gap-4 text-xs">
              <Link to="/privacy-policy" className="hover:text-white transition-colors">
                Kebijakan Privasi
              </Link>
              <Link to="/portal/login" className="hover:text-white transition-colors">
                Portal Peserta
              </Link>
              <Link to="/login" className="hover:text-white transition-colors">
                Login Admin
              </Link>
            </div>
            <p className="text-xs text-center">
              &copy; {new Date().getFullYear()} Pintour Travel. Hak cipta dilindungi.
            </p>
          </div>
        </div>
      </footer>
    </div>
  )
}
