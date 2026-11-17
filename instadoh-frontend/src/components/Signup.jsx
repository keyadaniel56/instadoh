import React, { useState, useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { authApi } from '../api/auth'
import { Zap, ChevronRight, AlertCircle, ArrowLeft } from 'lucide-react'

const fallbackCountries = [
  { Code: 'KE', Name: 'Kenya', Flag: '🇰🇪', Currency: 'KES', CurrencyName: 'Kenyan Shilling', IsActive: true },
  { Code: 'UG', Name: 'Uganda', Flag: '🇺🇬', Currency: 'UGX', CurrencyName: 'Ugandan Shilling', IsActive: true },
  { Code: 'US', Name: 'United States', Flag: '🇺🇸', Currency: 'USD', CurrencyName: 'US Dollar', IsActive: true },
  { Code: 'GB', Name: 'United Kingdom', Flag: '🇬🇧', Currency: 'GBP', CurrencyName: 'British Pound', IsActive: true },
  { Code: 'NG', Name: 'Nigeria', Flag: '🇳🇬', Currency: 'NGN', CurrencyName: 'Nigerian Naira', IsActive: true },
  { Code: 'ZA', Name: 'South Africa', Flag: '🇿🇦', Currency: 'ZAR', CurrencyName: 'South African Rand', IsActive: true },
  { Code: 'GH', Name: 'Ghana', Flag: '🇬🇭', Currency: 'GHS', CurrencyName: 'Ghanaian Cedi', IsActive: true },
  { Code: 'TZ', Name: 'Tanzania', Flag: '🇹🇿', Currency: 'TZS', CurrencyName: 'Tanzanian Shilling', IsActive: true },
  { Code: 'ET', Name: 'Ethiopia', Flag: '🇪🇹', Currency: 'ETB', CurrencyName: 'Ethiopian Birr', IsActive: true },
  { Code: 'IN', Name: 'India', Flag: '🇮🇳', Currency: 'INR', CurrencyName: 'Indian Rupee', IsActive: true },
  { Code: 'EU', Name: 'European Union', Flag: '🇪🇺', Currency: 'EUR', CurrencyName: 'Euro', IsActive: true },
  { Code: 'SG', Name: 'Singapore', Flag: '🇸🇬', Currency: 'SGD', CurrencyName: 'Singapore Dollar', IsActive: true },
]

function Signup() {
  const [form, setForm] = useState({
    email: '',
    phone: '',
    password: '',
    full_name: '',
    country_code: '',
    role: 'user',
  })
  const [countries, setCountries] = useState(fallbackCountries)
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const { register } = useAuth()
  const navigate = useNavigate()

  useEffect(() => {
    authApi.getCountries().then((res) => {
      const data = res?.data
      if (Array.isArray(data) && data.length > 0) {
        setCountries(data)
      }
    }).catch(() => {
      // Keep using fallbackCountries if API is unavailable
    })
  }, [])

  const handleChange = (e) => {
    const { name, value } = e.target
    setForm((prev) => ({ ...prev, [name]: value }))
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
      <Link to="/" style={{ position: 'absolute', top: '40px', left: '40px', zIndex: 20, display: 'flex', alignItems: 'center', gap: '8px', color: 'var(--color-text-muted)', fontSize: '0.875rem' }}>
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
              <label htmlFor="phone">Phone (Optional)</label>
              <input
                id="phone"
                name="phone"
                type="tel"
                placeholder="+1 234 567 8900 (for mobile money)"
                value={form.phone}
                onChange={handleChange}
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
