import React from 'react'
import { Link } from 'react-router-dom'
import { Check, ChevronRight } from 'lucide-react'

const plans = [
  {
    name: 'Starter',
    description: 'Perfect for individuals exploring the Lightning Network.',
    price: 'Free',
    period: 'forever',
    features: [
      'Up to 100 txs/month',
      'Basic analytics',
      'API access',
      'Single node wallet',
    ],
    cta: 'Get Started',
    featured: false,
  },
  {
    name: 'Business',
    description: 'For growing businesses requiring high-volume rails.',
    price: '$29',
    period: '/month',
    features: [
      '10,000 txs/month',
      'Advanced node metrics',
      'Priority settlement',
      'API + Webhooks',
      'Multi-node management',
      'Custom branding',
    ],
    cta: 'Start Free Trial',
    featured: true,
  },
  {
    name: 'Enterprise',
    description: 'Dedicated infrastructure for large scale operations.',
    price: '$99',
    period: '/month',
    features: [
      'Unlimited volume',
      'Real-time audit log',
      '24/7 dedicated support',
      'Advanced LND config',
      'SLA guarantee',
      'White-label solution',
    ],
    cta: 'Contact Sales',
    featured: false,
  },
]

function Pricing() {
  return (
    <section className="section" id="pricing" style={{ background: 'var(--color-surface)' }}>
      <div className="container">
        <div style={{ textAlign: 'center', marginBottom: '80px' }}>
          <div className="section-label">Pricing</div>
          <h2 className="section-title">Scale with <span className="text-gold">Confidence</span></h2>
          <p className="section-subtitle" style={{ margin: '0 auto' }}>
            Choose the settlement tier that matches your business throughput. All plans include non-custodial security.
          </p>
        </div>

        <div className="pricing-grid">
          {plans.map((plan, index) => (
            <div
              className={`pricing-card ${plan.featured ? 'featured' : ''}`}
              key={index}
            >
              {plan.featured && <span className="pricing-badge">Popular Choice</span>}
              <h3 style={{ fontSize: '1.25rem', marginBottom: '12px', color: plan.featured ? 'var(--color-primary)' : 'inherit' }}>{plan.name}</h3>
              <p style={{ color: 'var(--color-text-muted)', fontSize: '0.875rem', marginBottom: '32px' }}>{plan.description}</p>
              
              <div className="pricing-amount">
                {plan.price} <span>{plan.period}</span>
              </div>
              
              <div style={{ margin: '32px 0', flex: 1 }}>
                {plan.features.map((feature, i) => (
                  <div key={i} style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '16px', fontSize: '0.9375rem' }}>
                    <Check size={16} className="text-gold" />
                    <span style={{ color: 'var(--color-text-main)' }}>{feature}</span>
                  </div>
                ))}
              </div>

              <Link 
                to="/signup" 
                className={`btn ${plan.featured ? 'btn-primary' : 'btn-secondary'}`}
                style={{ width: '100%' }}
              >
                <span>{plan.cta}</span>
                <ChevronRight size={16} />
              </Link>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}

export default Pricing
