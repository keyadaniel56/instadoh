import React from 'react'

const features = [
  {
    icon: '⚡',
    title: 'Instant Settlement',
    description: 'Transactions settle in milliseconds, not hours. No more waiting for blockchain confirmations.',
    color: '#2563eb',
    bg: 'rgba(37, 99, 235, 0.1)',
  },
  {
    icon: '🛡️',
    title: 'Bank-Grade Security',
    description: 'Enterprise security with multi-signature wallets, encryption at rest, and real-time threat monitoring.',
    color: '#7c3aed',
    bg: 'rgba(124, 58, 237, 0.1)',
  },
  {
    icon: '💰',
    title: 'Near-Zero Fees',
    description: 'Average transaction fee of 0.1%. Send any amount globally for pennies.',
    color: '#10b981',
    bg: 'rgba(16, 185, 129, 0.1)',
  },
  {
    icon: '🌐',
    title: 'Global Reach',
    description: 'Accept payments from anywhere in the world. No borders, no restrictions, no limits.',
    color: '#f59e0b',
    bg: 'rgba(245, 158, 11, 0.1)',
  },
  {
    icon: '📊',
    title: 'Real-Time Analytics',
    description: 'Comprehensive dashboard with live transaction tracking, volume metrics, and revenue reports.',
    color: '#ef4444',
    bg: 'rgba(239, 68, 68, 0.1)',
  },
  {
    icon: '🔗',
    title: 'API-First Platform',
    description: 'Integrate Lightning payments into your platform with our powerful REST API and webhooks.',
    color: '#06b6d4',
    bg: 'rgba(6, 182, 212, 0.1)',
  },
]

function Features() {
  return (
    <section className="section features" id="features">
      <div className="container">
        <div className="features-header">
          <span className="section-label">
            <span>✦</span>
            Platform Features
          </span>
          <h2 className="section-title">
            Everything You Need for<br />
            <span className="text-gradient">Bitcoin Payments</span>
          </h2>
          <p className="section-subtitle">
            From instant settlement to enterprise-grade security, we provide
            all the tools to manage your Lightning Network payments.
          </p>
        </div>

        <div className="features-grid">
          {features.map((feature, index) => (
            <div className="feature-card" key={index}>
              <div
                className="feature-icon"
                style={{ background: feature.bg, color: feature.color }}
              >
                {feature.icon}
              </div>
              <h3>{feature.title}</h3>
              <p>{feature.description}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}

export default Features