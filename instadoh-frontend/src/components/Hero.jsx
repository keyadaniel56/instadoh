import React from 'react'
import { Link } from 'react-router-dom'

function Hero() {
  return (
    <section className="hero" id="hero">
      <div className="hero-bg-grid"></div>
      <div className="hero-bg-glow"></div>
      <div className="hero-bg-glow-2"></div>

      <div className="container">
        <div className="hero-content">
          <div className="hero-badge">
            <span className="hero-badge-dot"></span>
            Lightning Network Ready
          </div>

          <h1 className="hero-title">
            Lightning-Fast<br />
            <span className="text-gradient">Bitcoin Payments</span>
          </h1>

          <p className="hero-description">
            Send and receive Bitcoin instantly with near-zero fees. 
            Powered by the Lightning Network for merchants, creators, 
            and businesses worldwide.
          </p>

          <div className="hero-actions">
            <Link to="/dashboard" className="btn btn-primary btn-lg">
              Start Accepting Payments
              <span>→</span>
            </Link>
            <button className="btn btn-light btn-lg" onClick={() => {
              document.getElementById('how-it-works')?.scrollIntoView({ behavior: 'smooth' })
            }}>
              How It Works
            </button>
          </div>

          <div className="hero-stats">
            <div>
              <div className="hero-stat-value">$2.4B+</div>
              <div className="hero-stat-label">Total Volume Processed</div>
            </div>
            <div>
              <div className="hero-stat-value">50K+</div>
              <div className="hero-stat-label">Active Merchants</div>
            </div>
            <div>
              <div className="hero-stat-value">0.1%</div>
              <div className="hero-stat-label">Average Fee</div>
            </div>
          </div>
        </div>

        <div className="hero-visual">
          <div className="hero-card">
            <div className="hero-card-header">
              <div>
                <div className="hero-card-amount">
                  0.005 <span>BTC</span>
                </div>
                <div style={{ color: 'rgba(255,255,255,0.4)', fontSize: '0.813rem', marginTop: 4 }}>
                  ≈ $350.00 USD
                </div>
              </div>
              <span className="hero-card-badge">✓ Confirmed</span>
            </div>

            <div className="hero-card-row">
              <span className="hero-card-label">From</span>
              <span className="hero-card-value">alice@instadoh.io</span>
            </div>
            <div className="hero-card-row">
              <span className="hero-card-label">To</span>
              <span className="hero-card-value">merchant@instadoh.io</span>
            </div>
            <div className="hero-card-row">
              <span className="hero-card-label">Network Fee</span>
              <span className="hero-card-value" style={{ color: '#10b981' }}>0.000001 BTC</span>
            </div>
            <div className="hero-card-row">
              <span className="hero-card-label">Settled</span>
              <span className="hero-card-value">1.2 seconds</span>
            </div>

            <div className="hero-card-progress">
              <span style={{ color: 'rgba(255,255,255,0.5)', fontSize: '0.813rem' }}>
                Network Capacity
              </span>
              <div className="hero-card-progress-bar">
                <div className="hero-card-progress-fill"></div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}

export default Hero