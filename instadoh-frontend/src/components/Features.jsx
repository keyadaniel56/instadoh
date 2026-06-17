import React from 'react'
import { Zap, Shield, Cpu, Globe, BarChart3, Lock } from 'lucide-react'

function Features() {
  const features = [
    {
      icon: <Zap size={28} />,
      title: 'Real-time Settlement',
      desc: 'Payments settle in milliseconds, not days. Built on the global Bitcoin Lightning Network for instant liquidity.',
      size: 'large',
      image: 'https://images.unsplash.com/photo-1683322499436-f4383dd59f5a?auto=format&fit=crop&q=80&w=800'
    },
    {
      icon: <Shield size={28} />,
      title: 'Enterprise Security',
      desc: 'Multi-sig support and hardware wallet integration for maximum fund safety.',
      size: 'small'
    },
    {
      icon: <Cpu size={28} />,
      title: 'Scalable APIs',
      desc: 'Developer-first APIs designed to handle millions of transactions with ease.',
      size: 'small'
    },
    {
      icon: <Globe size={28} />,
      title: 'Borderless Rails',
      desc: 'Send and receive value anywhere in the world instantly with zero friction.',
      size: 'small'
    },
    {
      icon: <BarChart3 size={28} />,
      title: 'Deep Analytics',
      desc: 'Get real-time insights into your payment flows and liquidity management.',
      size: 'large'
    },
    {
      icon: <Lock size={28} />,
      title: 'Full Sovereignty',
      desc: 'Non-custodial by design. You hold the keys, you control the liquidity.',
      size: 'small'
    }
  ]

  return (
    <section id="features" className="section">
      <div className="container">
        <div style={{ textAlign: 'center', marginBottom: '80px' }}>
          <div className="section-label">Core Infrastructure</div>
          <h2 className="section-title">Engineered for the <span className="text-gold">Future of Finance</span></h2>
          <p className="section-subtitle" style={{ margin: '0 auto' }}>
            A powerful suite of tools designed to handle the most demanding enterprise payment requirements with Bitcoin's security.
          </p>
        </div>

        <div className="bento-grid" style={{ gridAutoRows: 'minmax(320px, auto)' }}>
          {features.map((f, i) => (
            <div 
              key={i} 
              className="bento-item"
              style={{ 
                gridColumn: f.size === 'large' ? 'span 2' : 'span 1',
                display: 'flex',
                flexDirection: 'column',
                justifyContent: 'flex-end',
                position: 'relative',
                overflow: 'hidden',
                zIndex: 1
              }}
            >
              {f.image && (
                <div style={{
                  position: 'absolute',
                  top: 0,
                  left: 0,
                  width: '100%',
                  height: '100%',
                  backgroundImage: `url(${f.image})`,
                  backgroundSize: 'cover',
                  backgroundPosition: 'center',
                  opacity: 0.15,
                  zIndex: -1
                }} />
              )}
              <div style={{ position: 'relative', zIndex: 2 }}>
                <div className="text-gold" style={{ marginBottom: '24px' }}>{f.icon}</div>
                <h3 style={{ fontSize: '1.75rem', marginBottom: '16px', fontWeight: 800 }}>{f.title}</h3>
                <p style={{ color: 'var(--color-text-muted)', fontSize: '1.0625rem', maxWidth: '480px' }}>{f.desc}</p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}

export default Features
