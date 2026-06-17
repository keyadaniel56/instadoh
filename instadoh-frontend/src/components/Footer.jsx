import React from 'react'
import { Link } from 'react-router-dom'
import { Zap, Globe, Github, Twitter, Linkedin } from 'lucide-react'

function Footer() {
  return (
    <footer className="footer">
      <div className="container">
        <div className="footer-grid">
          <div style={{ maxWidth: '320px' }}>
            <div className="navbar-brand" style={{ marginBottom: '24px' }}>
              <Zap size={24} className="text-gold" fill="currentColor" />
              <span>Instadoh</span>
            </div>
            <p style={{ color: 'var(--color-text-muted)', fontSize: '0.9375rem', marginBottom: '32px', lineHeight: '1.6' }}>
              The leading non-custodial Lightning Network payment platform. 
              Built for high-trust financial applications requiring instant finality and sovereign control.
            </p>
            <div style={{ display: 'flex', gap: '16px' }}>
              <a href="#" className="btn btn-sm btn-secondary" style={{ padding: '8px', border: 'none' }} aria-label="Twitter">
                <Twitter size={20} />
              </a>
              <a href="#" className="btn btn-sm btn-secondary" style={{ padding: '8px', border: 'none' }} aria-label="GitHub">
                <Github size={20} />
              </a>
              <a href="#" className="btn btn-sm btn-secondary" style={{ padding: '8px', border: 'none' }} aria-label="LinkedIn">
                <Linkedin size={20} />
              </a>
            </div>
          </div>

          <div className="footer-column">
            <h4 style={{ marginBottom: '24px', fontSize: '1rem' }}>Infrastructure</h4>
            <div className="footer-links">
              <Link to="/" className="footer-link">Features</Link>
              <Link to="/" className="footer-link">Pricing</Link>
              <a href="#" className="footer-link">API Reference</a>
              <a href="#" className="footer-link">Node Monitoring</a>
            </div>
          </div>

          <div className="footer-column">
            <h4 style={{ marginBottom: '24px', fontSize: '1rem' }}>Resources</h4>
            <div className="footer-links">
              <a href="#" className="footer-link">Developer Docs</a>
              <a href="#" className="footer-link">Security Audit</a>
              <a href="#" className="footer-link">LND Guides</a>
              <a href="#" className="footer-link">Network Status</a>
            </div>
          </div>

          <div className="footer-column">
            <h4 style={{ marginBottom: '24px', fontSize: '1rem' }}>Company</h4>
            <div className="footer-links">
              <a href="#" className="footer-link">About</a>
              <a href="#" className="footer-link">Careers</a>
              <a href="#" className="footer-link">Privacy</a>
              <a href="#" className="footer-link">Terms</a>
            </div>
          </div>
        </div>

        <div style={{ 
          display: 'flex', 
          justifyContent: 'space-between', 
          alignItems: 'center', 
          paddingTop: '40px', 
          borderTop: '1px solid var(--color-border)',
          color: 'var(--color-text-light)',
          fontSize: '0.8125rem'
        }}>
          <p>&copy; {new Date().getFullYear()} Instadoh. Secure Bitcoin Infrastructure.</p>
          <div style={{ display: 'flex', gap: '24px', alignItems: 'center' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              <Globe size={14} />
              <span>Mainnet Enabled</span>
            </div>
            <a href="#" style={{ color: 'inherit' }}>Cookie Policy</a>
          </div>
        </div>
      </div>
    </footer>
  )
}

export default Footer
