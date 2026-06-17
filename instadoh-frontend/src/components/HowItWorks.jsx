import React from 'react'
import { UserPlus, Wallet, Zap, ChevronRight } from 'lucide-react'

const steps = [
  {
    icon: <UserPlus size={24} />,
    title: 'Provision Account',
    description: 'Sign up in minutes. No legacy bank bureaucracy, just pure Lightning-fast access.',
  },
  {
    icon: <Wallet size={24} />,
    title: 'Initialize Node',
    description: 'Connect your sovereign LND node or utilize our high-availability hosted infrastructure.',
  },
  {
    icon: <Zap size={24} />,
    title: 'Execute Settlements',
    description: 'Deploy capital instantly. Generate invoices and settle global payments in milliseconds.',
  },
]

function HowItWorks() {
  return (
    <section className="section" id="how-it-works">
      <div className="container">
        <div style={{ textAlign: 'center', marginBottom: '80px' }}>
          <div className="section-label">The Process</div>
          <h2 className="section-title">Zero Friction <span className="text-gold">Onboarding</span></h2>
          <p className="section-subtitle" style={{ margin: '0 auto' }}>
            Integrate Bitcoin's payment layer into your stack with minimal engineering overhead.
          </p>
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '48px' }}>
          {steps.map((step, index) => (
            <div key={index} style={{ position: 'relative' }}>
              <div style={{ 
                width: '56px', height: '56px', background: 'rgba(255, 215, 0, 0.1)', 
                color: 'var(--color-primary)', borderRadius: 'var(--radius-md)', 
                display: 'flex', alignItems: 'center', justifyContent: 'center', marginBottom: '24px'
              }}>
                {step.icon}
              </div>
              <h3 style={{ fontSize: '1.25rem', marginBottom: '16px' }}>{step.title}</h3>
              <p style={{ color: 'var(--color-text-muted)', fontSize: '0.9375rem', lineHeight: '1.6' }}>{step.description}</p>
              
              {index < steps.length - 1 && (
                <div style={{ 
                  position: 'absolute', top: '28px', left: '100%', width: '40px', 
                  borderTop: '1px dashed var(--color-border)', transform: 'translateX(4px)',
                  display: 'none' // Hidden on smaller screens, we can add a media query if needed
                }}></div>
              )}
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}

export default HowItWorks
