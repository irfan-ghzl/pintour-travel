import { Outlet, Link, useLocation } from 'react-router-dom'
import { MapPin, Menu, X } from 'lucide-react'
import { useState } from 'react'

const navLinks = [
  { to: '/', label: 'Beranda' },
  { to: '/packages', label: 'Paket Wisata' },
  { to: '/build-my-trip', label: 'Build My Trip' },
  { to: '/testimonials', label: 'Testimonial' },
]

export default function Layout() {
  const { pathname } = useLocation()
  const [menuOpen, setMenuOpen] = useState(false)

  return (
    <div className="min-h-screen flex flex-col bg-gray-50">
      {/* Navbar */}
      <header className="bg-white shadow-sm sticky top-0 z-30">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            {/* Logo */}
            <Link to="/" className="flex items-center gap-2 font-bold text-primary-700 text-xl">
              <MapPin className="w-6 h-6 text-accent-500" />
              Pintour Travel
            </Link>

            {/* Desktop nav */}
            <nav className="hidden md:flex items-center gap-6">
              {navLinks.map((link) => (
                <Link
                  key={link.to}
                  to={link.to}
                  className={`text-sm font-medium transition-colors ${
                    pathname === link.to
                      ? 'text-primary-600'
                      : 'text-gray-600 hover:text-primary-600'
                  }`}
                >
                  {link.label}
                </Link>
              ))}
              <Link to="/build-my-trip" className="btn-primary text-sm py-2 px-4">
                Konsultasi Sekarang
              </Link>
            </nav>

            {/* Mobile hamburger */}
            <button
              className="md:hidden p-2 text-gray-600"
              onClick={() => setMenuOpen(!menuOpen)}
              aria-label="Toggle menu"
            >
              {menuOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
            </button>
          </div>
        </div>

        {/* Mobile menu */}
        {menuOpen && (
          <div className="md:hidden border-t px-4 py-3 flex flex-col gap-3 bg-white">
            {navLinks.map((link) => (
              <Link
                key={link.to}
                to={link.to}
                className="text-sm font-medium text-gray-700 hover:text-primary-600"
                onClick={() => setMenuOpen(false)}
              >
                {link.label}
              </Link>
            ))}
          </div>
        )}
      </header>

      <main className="flex-1">
        <Outlet />
      </main>

      {/* Footer */}
      <footer className="bg-gray-900 text-gray-400 py-10">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex flex-col md:flex-row items-center justify-between gap-4">
            <div className="flex items-center gap-2 text-white font-bold text-lg">
              <MapPin className="w-5 h-5 text-accent-400" />
              Pintour Travel
            </div>
            <p className="text-sm text-center">
              &copy; {new Date().getFullYear()} Pintour Travel. Semua hak dilindungi.
            </p>
          </div>
        </div>
      </footer>
    </div>
  )
}
