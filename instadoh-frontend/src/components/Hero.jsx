import React from 'react'
import { Link } from 'react-router-dom'
import { ChevronRight, Zap, Shield, Globe } from 'lucide-react'

function Hero() {
  return (
    <section className="hero mesh-bg section" style={{ minHeight: '90vh', display: 'flex', alignItems: 'center' }}>
      <div className="container">
        <div className="hero-content animate-fade-in-up">
          <div className="section-label">
            <Zap size={14} fill="currentColor" />
            <span>Lightning Network Powered</span>
          </div>
          <h1 className="section-title">
            Instant Settlement. <br />
            <span className="text-gradient">Ultimate Sovereignty.</span>
          </h1>
          <p className="section-subtitle">
            The next generation of Bitcoin payments for modern businesses. 
            Send, receive, and manage assets on the Lightning Network with enterprise-grade security and millisecond finality.
          </p>
          
          <div className="hero-actions glass">
            <Link to="/signup" className="btn btn-lg btn-primary">
              <span>Get Started Now</span>
              <ChevronRight size={20} />
            </Link>
            <Link to="/login" className="btn btn-lg btn-secondary">
              Explore Demo
            </Link>
          </div>
          
          <div className="hero-stats">
            <div className="hero-stat-item">
              <Shield size={18} className="text-gold" />
              <span className="hero-stat-label">Non-Custodial</span>
            </div>
            <div className="hero-stat-item">
              <Zap size={18} className="text-gold" />
              <span className="hero-stat-label">Millisecond Finality</span>
            </div>
            <div className="hero-stat-item">
              <Globe size={18} className="text-gold" />
              <span className="hero-stat-label">Global Payment Rails</span>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}

export default Hero
