import React from 'react'
import { Link } from 'react-router-dom'
import { ChevronRight, Zap } from 'lucide-react'

function CTA() {
  return (
    <section className="section" style={{ background: 'var(--color-bg)', position: 'relative', overflow: 'hidden' }}>
      <div className="container">
        <div className="glass" style={{ 
          padding: '80px 40px', 
          borderRadius: 'var(--radius-xl)', 
          textAlign: 'center',
          position: 'relative',
          overflow: 'hidden'
        }}>
          {/* Subtle background glow */}
          <div style={{ 
            position: 'absolute', top: '50%', left: '50%', width: '400px', height: '400px', 
            background: 'var(--color-primary)', filter: 'blur(100px)', opacity: 0.1, 
            transform: 'translate(-50%, -50%)', zIndex: -1 
          }}></div>

          <h2 className="section-title" style={{ fontSize: '3rem', marginBottom: '24px' }}>
            Ready to Upgrade Your <br />
            <span className="text-gold">Payment Infrastructure?</span>
          </h2>
          <p className="section-subtitle" style={{ margin: '0 auto 48px', maxWidth: '600px' }}>
            Join the global movement of businesses settling value on the Lightning Network. 
            Deploy your node today and experience the speed of sound finance.
          </p>
          <div style={{ display: 'flex', gap: '16px', justifyContent: 'center', flexWrap: 'wrap' }}>
            <Link to="/signup" className="btn btn-lg btn-primary">
              <span>Initialize Vault</span>
              <ChevronRight size={20} />
            </Link>
            <button className="btn btn-lg btn-secondary">
              Talk to an Architect
            </button>
          </div>
        </div>
      </div>
    </section>
  )
}

export default CTA
