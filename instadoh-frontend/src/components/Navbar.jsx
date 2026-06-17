import React, { useState, useEffect } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { Zap, Menu, X, LayoutDashboard, LogOut, ChevronRight } from 'lucide-react'

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
        <div className="container" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: '100%' }}>
          <Link to="/" className="navbar-brand">
            <Zap size={24} className="text-gold" fill="currentColor" />
            <span>Instadoh</span>
          </Link>

          {isLanding && (
            <div className="navbar-links">
              <button className="navbar-link" onClick={() => scrollToSection('features')}>Features</button>
              <button className="navbar-link" onClick={() => scrollToSection('how-it-works')}>How It Works</button>
              <button className="navbar-link" onClick={() => scrollToSection('pricing')}>Pricing</button>
            </div>
          )}

          <div className="navbar-actions" style={{ display: 'flex', gap: '12px', alignItems: 'center' }}>
            {isAuthenticated ? (
              <>
                <Link to="/dashboard" className="btn btn-sm btn-primary">
                  <LayoutDashboard size={16} />
                  <span>Dashboard</span>
                </Link>
                <button className="btn btn-sm btn-secondary" onClick={logout} title="Sign Out">
                  <LogOut size={16} />
                </button>
              </>
            ) : (
              <>
                <Link to="/login" className="navbar-link" style={{ marginRight: '12px' }}>Sign In</Link>
                <Link to="/signup" className="btn btn-sm btn-primary">
                  <span>Get Started</span>
                  <ChevronRight size={16} />
                </Link>
              </>
            )}
            <button
              className="navbar-toggle"
              onClick={() => setMobileOpen(true)}
              aria-label="Open menu"
              style={{ padding: '8px', color: 'var(--color-text-main)' }}
            >
              <Menu size={24} />
            </button>
          </div>
        </div>
      </nav>

      {/* Mobile Navigation */}
      <div className={`mobile-nav ${mobileOpen ? 'open' : ''}`} style={{
        position: 'fixed',
        top: 0,
        right: 0,
        width: '100%',
        height: '100vh',
        background: 'var(--color-bg)',
        zIndex: 1100,
        padding: '32px',
        display: mobileOpen ? 'flex' : 'none',
        flexDirection: 'column',
        gap: '32px',
        transform: mobileOpen ? 'translateX(0)' : 'translateX(100%)',
        transition: 'transform 0.3s ease'
      }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Link to="/" className="navbar-brand" onClick={() => setMobileOpen(false)}>
            <Zap size={24} className="text-gold" fill="currentColor" />
            <span>Instadoh</span>
          </Link>
          <button onClick={() => setMobileOpen(false)} style={{ color: 'var(--color-text-main)' }}>
            <X size={32} />
          </button>
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
          {isLanding && (
            <>
              <button className="navbar-link" style={{ fontSize: '1.25rem', textAlign: 'left' }} onClick={() => scrollToSection('features')}>Features</button>
              <button className="navbar-link" style={{ fontSize: '1.25rem', textAlign: 'left' }} onClick={() => scrollToSection('how-it-works')}>How It Works</button>
              <button className="navbar-link" style={{ fontSize: '1.25rem', textAlign: 'left' }} onClick={() => scrollToSection('pricing')}>Pricing</button>
            </>
          )}
        </div>

        <div style={{ marginTop: 'auto', display: 'flex', flexDirection: 'column', gap: '16px' }}>
          {isAuthenticated ? (
            <Link to="/dashboard" className="btn btn-primary btn-lg" onClick={() => setMobileOpen(false)}>
              Dashboard
            </Link>
          ) : (
            <>
              <Link to="/signup" className="btn btn-primary btn-lg" onClick={() => setMobileOpen(false)}>
                Get Started
              </Link>
              <Link to="/login" className="btn btn-secondary btn-lg" onClick={() => setMobileOpen(false)}>
                Sign In
              </Link>
            </>
          )}
        </div>
      </div>
    </>
  )
}

export default Navbar
