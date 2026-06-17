import React from 'react'

const steps = [
  {
    number: '01',
    title: 'Create Your Account',
    description: 'Sign up in minutes with just your email. No lengthy KYC processes, no waiting for approvals.',
  },
  {
    number: '02',
    title: 'Connect Your Wallet',
    description: 'Link your Lightning Network wallet or let us create one for you. Supports LND, c-lightning, and Eclair.',
  },
  {
    number: '03',
    title: 'Start Transacting',
    description: 'Generate invoices, send payments, and manage transactions through our intuitive dashboard or API.',
  },
]

function HowItWorks() {
  return (
    <section className="section how-it-works" id="how-it-works">
      <div className="container">
        <div className="how-it-works-header">
          <span className="section-label">
            <span>✦</span>
            Simple Process
          </span>
          <h2 className="section-title">
            Get Started in{' '}
            <span className="text-gradient">3 Easy Steps</span>
          </h2>
          <p className="section-subtitle">
            Getting started with Lightning payments has never been easier.
            No technical expertise required.
          </p>
        </div>

        <div className="steps">
          {steps.map((step, index) => (
            <div className="step" key={index}>
              <div className="step-number">{step.number}</div>
              <h3>{step.title}</h3>
              <p>{step.description}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}

export default HowItWorks