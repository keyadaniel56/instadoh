import React, { useState, useEffect } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'

function Navbar() {
  const [scrolled, setScrolled] = useState(false)
  const [mobileOpen, setMobileOpen] = useState(false)
  const { isAuthenticated, user, logout } = useAuth()
  const location = useLocation()
  const isLanding = location.pathname === '/'

  useEffect(() => {
    const handleScroll = () => {
      setScrolled(window.scrollY > 50)
    }
    window.addEventListener('scroll', handleScroll)
    return () => window.removeEventListener('scroll', handleScroll)
  }, [])

  useEffect(() => {
    document.body.style.overflow = mobileOpen ? 'hidden' : ''
    return () => { document.body.style.overflow = '' }
  }, [mobileOpen])

  const scrollToSection = (id) => {
    setMobileOpen(false)
    const el = document.getElementById(id)
    if (el) {
      el.scrollIntoView({ behavior: 'smooth', block: 'start' })
    }
  }

  return (
    <>
      <nav className={`navbar ${scrolled ? 'scrolled' : ''}`}>
        <div className="container">
          <Link to="/" className="navbar-brand">
            <span className="navbar-brand-icon">⚡</span>
            Instadoh
          </Link>

          {isLanding && (
            <div className="navbar-links">
              <button className="navbar-link" onClick={() => scrollToSection('features')}>
                Features
              </button>
              <button className="navbar-link" onClick={() => scrollToSection('how-it-works')}>
                How It Works
              </button>
              <button className="navbar-link" onClick={() => scrollToSection('pricing')}>
                Pricing
              </button>
            </div>
          )}

          <div className="navbar-actions">
            {isAuthenticated ? (
              <>
                <Link to="/dashboard" className="btn btn-sm btn-primary">
                  Dashboard
                </Link>
                <button className="btn btn-sm btn-secondary" onClick={logout}>
                  Sign Out
                </button>
              </>
            ) : (
              <>
                <Link to="/login" className={`btn btn-sm ${isLanding ? 'btn-light' : 'btn-secondary'}`}>
                  Sign In
                </Link>
                <Link to="/signup" className="btn btn-sm btn-primary">
                  Get Started
                </Link>
              </>
            )}
            <button
              className="navbar-toggle"
              onClick={() => setMobileOpen(true)}
              aria-label="Open menu"
            >
              <span></span>
              <span></span>
              <span></span>
            </button>
          </div>
        </div>
      </nav>

      {/* Mobile Navigation */}
      <div className={`mobile-nav ${mobileOpen ? 'open' : ''}`}>
        <div className="mobile-nav-content">
          <div className="mobile-nav-header">
            <Link to="/" className="navbar-brand" style={{ color: 'var(--color-dark)' }}>
              <span className="navbar-brand-icon">⚡</span>
              Instadoh
            </Link>
            <button
              className="mobile-nav-close"
              onClick={() => setMobileOpen(false)}
              aria-label="Close menu"
            >
              ✕
            </button>
          </div>

          <div className="mobile-nav-links">
            {isLanding && (
              <>
                <button className="mobile-nav-link" onClick={() => scrollToSection('features')}>
                  Features
                </button>
                <button className="mobile-nav-link" onClick={() => scrollToSection('how-it-works')}>
                  How It Works
                </button>
                <button className="mobile-nav-link" onClick={() => scrollToSection('pricing')}>
                  Pricing
                </button>
              </>
            )}
          </div>

          <div className="mobile-nav-actions">
            {isAuthenticated ? (
              <>
                <Link to="/dashboard" className="btn btn-primary" onClick={() => setMobileOpen(false)}>
                  Dashboard
                </Link>
                <button className="btn btn-secondary" onClick={() => { logout(); setMobileOpen(false) }}>
                  Sign Out
                </button>
              </>
            ) : (
              <>
                <Link to="/signup" className="btn btn-primary" onClick={() => setMobileOpen(false)}>
                  Get Started
                </Link>
                <Link to="/login" className="btn btn-secondary" onClick={() => setMobileOpen(false)}>
                  Sign In
                </Link>
              </>
            )}
          </div>
        </div>
      </div>
    </>
  )
}

export default Navbar
