import React from 'react'
import { Link } from 'react-router-dom'

function Footer() {
  return (
    <footer className="footer">
      <div className="container">
        <div className="footer-grid">
          <div>
            <div className="footer-brand">
              <span className="footer-brand-icon">⚡</span>
              Instadoh
            </div>
            <p className="footer-description">
              The leading Lightning Network payment platform. Send and receive
              Bitcoin instantly with near-zero fees.
            </p>
            <div className="footer-social">
              <a href="#twitter" className="footer-social-link" aria-label="Twitter">𝕏</a>
              <a href="#github" className="footer-social-link" aria-label="GitHub">GH</a>
              <a href="#linkedin" className="footer-social-link" aria-label="LinkedIn">in</a>
              <a href="#discord" className="footer-social-link" aria-label="Discord">DC</a>
            </div>
          </div>

          <div className="footer-column">
            <h4>Product</h4>
            <div className="footer-links">
              <Link to="/" className="footer-link" onClick={() => document.getElementById('features')?.scrollIntoView({ behavior: 'smooth' })}>Features</Link>
              <Link to="/" className="footer-link" onClick={() => document.getElementById('pricing')?.scrollIntoView({ behavior: 'smooth' })}>Pricing</Link>
              <a href="#api" className="footer-link">API</a>
              <a href="#changelog" className="footer-link">Changelog</a>
            </div>
          </div>

          <div className="footer-column">
            <h4>Resources</h4>
            <div className="footer-links">
              <a href="#docs" className="footer-link">Documentation</a>
              <a href="#guides" className="footer-link">Guides</a>
              <a href="#blog" className="footer-link">Blog</a>
              <a href="#support" className="footer-link">Support</a>
            </div>
          </div>

          <div className="footer-column">
            <h4>Company</h4>
            <div className="footer-links">
              <a href="#about" className="footer-link">About</a>
              <a href="#careers" className="footer-link">Careers</a>
              <a href="#contact" className="footer-link">Contact</a>
              <a href="#status" className="footer-link">Status</a>
            </div>
          </div>
        </div>

        <div className="footer-bottom">
          <p className="footer-copyright">
            &copy; {new Date().getFullYear()} Instadoh. All rights reserved.
          </p>
          <div className="footer-legal">
            <a href="#privacy">Privacy Policy</a>
            <a href="#terms">Terms of Service</a>
            <a href="#cookies">Cookie Policy</a>
          </div>
        </div>
      </div>
    </footer>
  )
}

export default Footer