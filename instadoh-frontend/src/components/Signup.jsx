import React, { useState, useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { authApi } from '../api/auth'
import { Zap, ChevronRight, AlertCircle, ArrowLeft } from 'lucide-react'
const countryPhoneCodes = {
  US: '+1',
  GB: '+44',
  EU: '+32',
  KE: '+254',
  NG: '+234',
  ZA: '+27',
  GH: '+233',
  IN: '+91',
  BR: '+55',
  MX: '+52',
  JP: '+81',
  CN: '+86',
  AU: '+61',
  CA: '+1',
  CH: '+41',
  SG: '+65',
  PH: '+63',
  TZ: '+255',
  UG: '+256',
  ET: '+251',
}

function Signup() {
  const [form, setForm] = useState({
    email: '',
    phone: '',
    password: '',
    full_name: '',
    country_code: '',
    role: 'user',
  })
  const [countries, setCountries] = useState([])
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const { register } = useAuth()
  const navigate = useNavigate()

  useEffect(() => {
    authApi.getCountries().then((res) => {
      setCountries(res.data)
    }).catch(() => {})
  }, [])

  const handleChange = (e) => {
    const { name, value } = e.target
    setForm((prev) => {
      const next = { ...prev, [name]: value }
      
      // If country changed, update phone prefix if phone is empty or contains only a prefix
      if (name === 'country_code' && countryPhoneCodes[value]) {
        const currentPhone = prev.phone.trim()
        const isOnlyPrefix = Object.values(countryPhoneCodes).some(code => currentPhone === code)
        if (!currentPhone || isOnlyPrefix) {
          next.phone = countryPhoneCodes[value]
        }
      }
      
      return next
    })
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    setError('')
    setSubmitting(true)
    try {
      await register(form)
      navigate('/dashboard')
    } catch (err) {
      const msg = err.response?.data?.message || 'Identity creation failed. Please try again.'
      setError(msg)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="auth-page">
      <Link to="/" style={{ position: 'absolute', top: '40px', left: '40px', display: 'flex', alignItems: 'center', gap: '8px', color: 'var(--color-text-muted)', fontSize: '0.875rem' }}>
        <ArrowLeft size={16} />
        <span>Back to Terminal</span>
      </Link>

      <div className="auth-card" style={{ maxWidth: '560px' }}>
        <div className="auth-header">
          <div style={{ display: 'inline-flex', marginBottom: '24px' }}>
            <Zap size={32} className="text-gold" fill="currentColor" />
          </div>
          <h1>Create Identity</h1>
          <p>Initialize your enterprise Lightning vault</p>
        </div>

        {error && (
          <div style={{ 
            background: 'rgba(239, 68, 68, 0.1)', border: '1px solid rgba(239, 68, 68, 0.2)', 
            color: 'var(--color-danger)', padding: '12px', borderRadius: 'var(--radius-md)', 
            marginBottom: '24px', display: 'flex', alignItems: 'center', gap: '10px', fontSize: '0.875rem' 
          }}>
            <AlertCircle size={16} />
            <span>{error}</span>
          </div>
        )}

        <form onSubmit={handleSubmit}>
          <div className="form-row">
            <div className="form-group">
              <label htmlFor="full_name">Legal Name</label>
              <input
                id="full_name"
                name="full_name"
                type="text"
                placeholder="John Doe"
                value={form.full_name}
                onChange={handleChange}
                required
              />
            </div>

            <div className="form-group">
              <label htmlFor="phone">Phone System</label>
              <input
                id="phone"
                name="phone"
                type="tel"
                placeholder={form.country_code ? `${countryPhoneCodes[form.country_code]} ...` : "+1 234 567 8900"}
                value={form.phone}
                onChange={handleChange}
                required
              />
            </div>
          </div>

          <div className="form-group">
            <label htmlFor="email">Email Identity</label>
            <input
              id="email"
              name="email"
              type="email"
              placeholder="operator@vault.io"
              value={form.email}
              onChange={handleChange}
              required
            />
          </div>

          <div className="form-group">
            <label htmlFor="country_code">Jurisdiction</label>
            <select
              id="country_code"
              name="country_code"
              value={form.country_code}
              onChange={handleChange}
              required
            >
              <option value="">Select Region</option>
              {countries.map((c) => (
                <option key={c.Code} value={c.Code}>{c.Flag} {c.Name}</option>
              ))}
            </select>
          </div>

          <div className="form-group">
            <label htmlFor="password">Access Key</label>
            <input
              id="password"
              name="password"
              type="password"
              placeholder="Minimum 8 characters"
              value={form.password}
              onChange={handleChange}
              required
            />
          </div>

          <button
            type="submit"
            className="btn btn-primary btn-lg"
            style={{ width: '100%', marginTop: '8px' }}
            disabled={submitting}
          >
            {submitting ? 'Initializing...' : 'Create Vault Identity'}
            {!submitting && <ChevronRight size={20} />}
          </button>
        </form>

        <div style={{ marginTop: '32px', textAlign: 'center', fontSize: '0.875rem', color: 'var(--color-text-muted)' }}>
          Already have an identity?{' '}
          <Link to="/login" style={{ color: 'var(--color-primary)', fontWeight: 600 }}>Authenticate</Link>
        </div>
      </div>
    </div>
  )
}

export default Signup
