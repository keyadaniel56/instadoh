import React from 'react'
import { Link } from 'react-router-dom'

const plans = [
  {
    name: 'Starter',
    description: 'Perfect for individuals and small businesses getting started with Lightning.',
    price: 'Free',
    period: 'forever',
    features: [
      'Up to 100 transactions/month',
      'Basic dashboard',
      'Email support',
      'API access',
      'Single wallet',
    ],
    cta: 'Get Started Free',
    featured: false,
  },
  {
    name: 'Business',
    description: 'For growing businesses that need more power and flexibility.',
    price: '$29',
    period: '/month',
    features: [
      'Up to 10,000 transactions/month',
      'Advanced analytics',
      'Priority support',
      'API + Webhooks',
      'Multi-wallet support',
      'Custom invoices',
      'Team access (5 users)',
    ],
    cta: 'Start Free Trial',
    featured: true,
  },
  {
    name: 'Enterprise',
    description: 'For large organizations requiring dedicated infrastructure and support.',
    price: '$99',
    period: '/month',
    features: [
      'Unlimited transactions',
      'Real-time analytics',
      'Dedicated support',
      'Advanced API features',
      'Unlimited wallets',
      'Custom integrations',
      'Unlimited team members',
      'SLA guarantee',
    ],
    cta: 'Contact Sales',
    featured: false,
  },
]

function Pricing() {
  return (
    <section className="section pricing" id="pricing">
      <div className="container">
        <div className="pricing-header">
          <span className="section-label">
            <span>✦</span>
            Pricing
          </span>
          <h2 className="section-title">
            Simple, Transparent{' '}
            <span className="text-gradient">Pricing</span>
          </h2>
          <p className="section-subtitle">
            No hidden fees, no surprise charges. Choose the plan that fits your needs.
          </p>
        </div>

        <div className="pricing-grid">
          {plans.map((plan, index) => (
            <div
              className={`pricing-card ${plan.featured ? 'featured' : ''}`}
              key={index}
            >
              {plan.featured && <span className="pricing-badge">Most Popular</span>}
              <h3 className="pricing-name">{plan.name}</h3>
              <p className="pricing-description">{plan.description}</p>
              <div className="pricing-amount">
                {plan.price} <span>{plan.period}</span>
              </div>
              <div className="pricing-period">
                {plan.price === 'Free' ? 'No credit card required' : ''}
              </div>

              <div className="pricing-features">
                {plan.features.map((feature, i) => (
                  <div className="pricing-feature" key={i}>
                    <span className="pricing-feature-icon">✓</span>
                    {feature}
                  </div>
                ))}
              </div>

              <Link to="/dashboard" className="btn btn-primary">
                {plan.cta}
              </Link>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}

export default Pricing