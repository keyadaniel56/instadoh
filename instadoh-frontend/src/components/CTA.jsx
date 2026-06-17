import React from 'react'
import { Link } from 'react-router-dom'

function CTA() {
  return (
    <section className="section cta">
      <div className="container">
        <h2>Ready to Transform Your<br />Payment Infrastructure?</h2>
        <p>
          Join thousands of businesses already accepting Lightning-fast Bitcoin
          payments. Get started in minutes, not days.
        </p>
        <div className="cta-actions">
          <Link to="/dashboard" className="btn btn-primary btn-lg">
            Get Started Free
            <span>→</span>
          </Link>
          <button className="btn btn-light btn-lg">
            Talk to Sales
          </button>
        </div>
      </div>
    </section>
  )
}

export default CTA