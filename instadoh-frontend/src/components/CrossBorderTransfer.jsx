import React, { useState, useEffect } from 'react'
import { useAuth } from '../context/AuthContext'
import { crossBorderApi, mpesaApi, ugandaMobileApi } from '../api/crossBorder'
import { ArrowLeftRight, Zap, RefreshCw, Send, TrendingUp, DollarSign, Phone } from 'lucide-react'

function CrossBorderTransfer() {
  const { user } = useAuth()
  const [activeTab, setActiveTab] = useState('send') // send, deposit, withdraw
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  // Send cross-border
  const [sendForm, setSendForm] = useState({
    recipient_phone: '',
    recipient_country: 'UG',
    amount: '',
    currency: user?.currency || 'KES',
    description: '',
  })
  const [quote, setQuote] = useState(null)
  const [quoteLoading, setQuoteLoading] = useState(false)

  // Deposit/Withdraw (Mobile Money)
  const [mobileForm, setMobileForm] = useState({
    phone_number: '',
    amount: '',
    provider: 'mtn', // for Uganda: mtn or airtel
  })

  // Transactions
  const [transactions, setTransactions] = useState([])
  const [txsLoading, setTxsLoading] = useState(false)

  useEffect(() => {
    fetchTransactions()
  }, [])

  // Fetch quote when amount changes
  useEffect(() => {
    if (!sendForm.amount || parseFloat(sendForm.amount) <= 0) {
      setQuote(null)
      return
    }

    const timer = setTimeout(async () => {
      setQuoteLoading(true)
      setError('')
      try {
        const toCurrency = sendForm.recipient_country === 'UG' ? 'UGX' : 'KES'
        const res = await crossBorderApi.getQuote(sendForm.currency, toCurrency, parseFloat(sendForm.amount))
        setQuote(res.data)
      } catch (err) {
        setQuote(null)
      } finally {
        setQuoteLoading(false)
      }
    }, 500)

    return () => clearTimeout(timer)
  }, [sendForm.amount, sendForm.currency, sendForm.recipient_country])

  const fetchTransactions = async () => {
    setTxsLoading(true)
    try {
      const res = await crossBorderApi.listTransactions(1, 10)
      setTransactions(res.data.data || [])
    } catch (err) {
      // silently fail
    } finally {
      setTxsLoading(false)
    }
  }

  const handleSend = async (e) => {
    e.preventDefault()
    setLoading(true)
    setError('')
    setSuccess('')

    try {
      const res = await crossBorderApi.sendCrossBorder({
        recipient_phone: sendForm.recipient_phone,
        recipient_country: sendForm.recipient_country,
        amount: parseFloat(sendForm.amount),
        currency: sendForm.currency,
        description: sendForm.description,
      })
      setSuccess(`Successfully sent ${sendForm.amount} ${sendForm.currency}! Recipient will receive ${quote?.receive_amount} ${quote?.to_currency}`)
      setSendForm({
        ...sendForm,
        recipient_phone: '',
        amount: '',
        description: '',
      })
      setQuote(null)
      fetchTransactions()
    } catch (err) {
      setError(err.response?.data?.message || 'Failed to send cross-border payment')
    } finally {
      setLoading(false)
    }
  }

  const handleMpesaDeposit = async (e) => {
    e.preventDefault()
    setLoading(true)
    setError('')
    setSuccess('')

    try {
      const res = await mpesaApi.deposit({
        phone_number: mobileForm.phone_number,
        amount: parseFloat(mobileForm.amount),
      })
      setSuccess(`M-Pesa STK Push sent to ${mobileForm.phone_number}. Check your phone and enter PIN to complete.`)
      setMobileForm({ ...mobileForm, phone_number: '', amount: '' })
    } catch (err) {
      setError(err.response?.data?.message || 'M-Pesa deposit failed')
    } finally {
      setLoading(false)
    }
  }

  const handleMpesaWithdraw = async (e) => {
    e.preventDefault()
    setLoading(true)
    setError('')
    setSuccess('')

    try {
      const res = await mpesaApi.withdraw({
        phone_number: mobileForm.phone_number,
        amount: parseFloat(mobileForm.amount),
      })
      setSuccess(`Withdrawal of KES ${mobileForm.amount} sent to ${mobileForm.phone_number}.`)
      setMobileForm({ ...mobileForm, phone_number: '', amount: '' })
    } catch (err) {
      setError(err.response?.data?.message || 'M-Pesa withdrawal failed')
    } finally {
      setLoading(false)
    }
  }

  const handleUgandaDeposit = async (e) => {
    e.preventDefault()
    setLoading(true)
    setError('')
    setSuccess('')

    try {
      const res = await ugandaMobileApi.deposit({
        phone_number: mobileForm.phone_number,
        amount: parseFloat(mobileForm.amount),
        provider: mobileForm.provider,
      })
      setSuccess(`Deposit initiated! Check your phone to authorize.`)
      setMobileForm({ ...mobileForm, phone_number: '', amount: '' })
    } catch (err) {
      setError(err.response?.data?.message || 'Deposit failed')
    } finally {
      setLoading(false)
    }
  }

  const handleUgandaWithdraw = async (e) => {
    e.preventDefault()
    setLoading(true)
    setError('')
    setSuccess('')

    try {
      const res = await ugandaMobileApi.withdraw({
        phone_number: mobileForm.phone_number,
        amount: parseFloat(mobileForm.amount),
        provider: mobileForm.provider,
      })
      setSuccess(`Withdrawal initiated! Funds sent to ${mobileForm.phone_number}.`)
      setMobileForm({ ...mobileForm, phone_number: '', amount: '' })
    } catch (err) {
      setError(err.response?.data?.message || 'Withdrawal failed')
    } finally {
      setLoading(false)
    }
  }

  const formatDate = (dateStr) => {
    const d = new Date(dateStr)
    return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      {error && (
        <div className="bento-item" style={{ borderLeft: '3px solid var(--color-danger)', background: 'var(--color-danger)10' }}>
          <p style={{ color: 'var(--color-danger)', fontSize: '0.875rem' }}>{error}</p>
        </div>
      )}
      {success && (
        <div className="bento-item" style={{ borderLeft: '3px solid var(--color-success)', background: 'var(--color-success)10' }}>
          <p style={{ color: 'var(--color-success)', fontSize: '0.875rem' }}>{success}</p>
        </div>
      )}

      {/* Tab Navigation */}
      <div style={{ display: 'flex', gap: '8px', borderBottom: '1px solid var(--color-border)', paddingBottom: '12px' }}>
        {['send', 'deposit', 'withdraw'].map((tab) => (
          <button
            key={tab}
            onClick={() => { setActiveTab(tab); setError(''); setSuccess('') }}
            className={`btn btn-sm ${activeTab === tab ? 'btn-primary' : 'btn-secondary'}`}
            style={{ textTransform: 'capitalize' }}
          >
            {tab === 'send' && <Send size={14} />}
            {tab === 'deposit' && <TrendingUp size={14} />}
            {tab === 'withdraw' && <DollarSign size={14} />}
            <span>{tab}</span>
          </button>
        ))}
      </div>

      {/* Cross-Border Send Form */}
      {activeTab === 'send' && (
        <div className="bento-item">
          <h3 style={{ fontSize: '1.125rem', marginBottom: '24px', display: 'flex', alignItems: 'center', gap: '8px' }}>
            <Send size={18} className="text-gold" />
            Send Cross-Border
          </h3>

          <form onSubmit={handleSend} style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
              <div>
                <label className="form-label">Recipient Country</label>
                <select
                  className="form-input"
                  value={sendForm.recipient_country}
                  onChange={(e) => setSendForm({ ...sendForm, recipient_country: e.target.value })}
                  required
                >
                  <option value="UG">Uganda (UGX)</option>
                  <option value="KE">Kenya (KES)</option>
                </select>
              </div>
              <div>
                <label className="form-label">Your Currency</label>
                <select
                  className="form-input"
                  value={sendForm.currency}
                  onChange={(e) => setSendForm({ ...sendForm, currency: e.target.value })}
                  required
                >
                  <option value="KES">KES - Kenyan Shilling</option>
                  <option value="UGX">UGX - Ugandan Shilling</option>
                </select>
              </div>
            </div>

            <div>
              <label className="form-label">Recipient Phone Number</label>
              <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                <Phone size={16} style={{ color: 'var(--color-text-muted)' }} />
                <input
                  className="form-input"
                  type="tel"
                  placeholder={sendForm.recipient_country === 'UG' ? '+256 77X XXX XXX' : '+254 7XX XXX XXX'}
                  value={sendForm.recipient_phone}
                  onChange={(e) => setSendForm({ ...sendForm, recipient_phone: e.target.value })}
                  required
                />
              </div>
            </div>

            <div>
              <label className="form-label">Amount to Send</label>
              <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                <DollarSign size={16} style={{ color: 'var(--color-text-muted)' }} />
                <input
                  className="form-input"
                  type="number"
                  min="1"
                  step="0.01"
                  placeholder="1000"
                  value={sendForm.amount}
                  onChange={(e) => setSendForm({ ...sendForm, amount: e.target.value })}
                  required
                />
              </div>
            </div>

            {quoteLoading && (
              <div style={{ fontSize: '0.875rem', color: 'var(--color-text-muted)' }}>
                Calculating exchange rate...
              </div>
            )}

            {quote && !quoteLoading && (
              <div className="bento-item" style={{ background: 'var(--color-bg-secondary)', padding: '16px' }}>
                <h4 style={{ fontSize: '0.875rem', color: 'var(--color-text-muted)', marginBottom: '12px' }}>Quote Summary</h4>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.875rem' }}>
                    <span style={{ color: 'var(--color-text-muted)' }}>You Send</span>
                    <span style={{ fontWeight: 700 }}>{quote.send_amount.toLocaleString()} {quote.from_currency}</span>
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.875rem' }}>
                    <span style={{ color: 'var(--color-text-muted)' }}>Recipient Gets</span>
                    <span style={{ fontWeight: 700, color: 'var(--color-success)' }}>{quote.receive_amount.toLocaleString()} {quote.to_currency}</span>
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.875rem' }}>
                    <span style={{ color: 'var(--color-text-muted)' }}>Exchange Rate</span>
                    <span>1 {quote.from_currency} = {quote.exchange_rate} {quote.to_currency}</span>
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.875rem' }}>
                    <span style={{ color: 'var(--color-text-muted)' }}>Fee (2%)</span>
                    <span style={{ color: 'var(--color-danger)' }}>{quote.fee.toLocaleString()} {quote.from_currency}</span>
                  </div>
                  <div style={{ borderTop: '1px solid var(--color-border)', paddingTop: '8px', display: 'flex', justifyContent: 'space-between', fontSize: '0.875rem' }}>
                    <span style={{ color: 'var(--color-text-muted)' }}>Total Deducted</span>
                    <span style={{ fontWeight: 700 }}>{quote.total_in_fiat.toLocaleString()} {quote.from_currency}</span>
                  </div>
                </div>
                <p style={{ fontSize: '0.75rem', color: 'var(--color-text-muted)', marginTop: '8px' }}>
                  Quote valid until {quote.valid_until}
                </p>
              </div>
            )}

            <div>
              <label className="form-label">Description (Optional)</label>
              <input
                className="form-input"
                type="text"
                placeholder="e.g., Payment for goods"
                value={sendForm.description}
                onChange={(e) => setSendForm({ ...sendForm, description: e.target.value })}
              />
            </div>

            <button type="submit" className="btn btn-primary" disabled={loading || !quote} style={{ width: '100%' }}>
              {loading ? 'Processing...' : 'Send Cross-Border Payment'}
            </button>
          </form>
        </div>
      )}

      {/* Deposit Form */}
      {activeTab === 'deposit' && (
        <div className="bento-item">
          <h3 style={{ fontSize: '1.125rem', marginBottom: '24px', display: 'flex', alignItems: 'center', gap: '8px' }}>
            <TrendingUp size={18} style={{ color: 'var(--color-success)' }} />
            Deposit via Mobile Money
          </h3>

          {user?.country_code === 'KE' ? (
            <form onSubmit={handleMpesaDeposit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
              <p style={{ fontSize: '0.875rem', color: 'var(--color-text-muted)' }}>
                Fund your wallet using M-Pesa. An STK Push prompt will be sent to your phone.
              </p>
              <div>
                <label className="form-label">M-Pesa Phone Number</label>
                <input
                  className="form-input"
                  type="tel"
                  placeholder="+254 7XX XXX XXX"
                  value={mobileForm.phone_number}
                  onChange={(e) => setMobileForm({ ...mobileForm, phone_number: e.target.value })}
                  required
                />
              </div>
              <div>
                <label className="form-label">Amount (KES)</label>
                <input
                  className="form-input"
                  type="number"
                  min="10"
                  step="1"
                  placeholder="100"
                  value={mobileForm.amount}
                  onChange={(e) => setMobileForm({ ...mobileForm, amount: e.target.value })}
                  required
                />
              </div>
              <button type="submit" className="btn btn-primary" disabled={loading} style={{ width: '100%' }}>
                {loading ? 'Processing...' : 'Deposit via M-Pesa'}
              </button>
            </form>
          ) : user?.country_code === 'UG' ? (
            <form onSubmit={handleUgandaDeposit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
              <p style={{ fontSize: '0.875rem', color: 'var(--color-text-muted)' }}>
                Fund your wallet using Mobile Money.
              </p>
              <div>
                <label className="form-label">Provider</label>
                <select
                  className="form-input"
                  value={mobileForm.provider}
                  onChange={(e) => setMobileForm({ ...mobileForm, provider: e.target.value })}
                  required
                >
                  <option value="mtn">MTN Mobile Money</option>
                  <option value="airtel">Airtel Money</option>
                </select>
              </div>
              <div>
                <label className="form-label">Phone Number</label>
                <input
                  className="form-input"
                  type="tel"
                  placeholder="+256 77X XXX XXX"
                  value={mobileForm.phone_number}
                  onChange={(e) => setMobileForm({ ...mobileForm, phone_number: e.target.value })}
                  required
                />
              </div>
              <div>
                <label className="form-label">Amount (UGX)</label>
                <input
                  className="form-input"
                  type="number"
                  min="500"
                  step="1"
                  placeholder="5000"
                  value={mobileForm.amount}
                  onChange={(e) => setMobileForm({ ...mobileForm, amount: e.target.value })}
                  required
                />
              </div>
              <button type="submit" className="btn btn-primary" disabled={loading} style={{ width: '100%' }}>
                {loading ? 'Processing...' : 'Deposit via Mobile Money'}
              </button>
            </form>
          ) : (
            <p style={{ color: 'var(--color-text-muted)', textAlign: 'center', padding: '32px' }}>
              Mobile money deposit is currently available for Kenya (M-Pesa) and Uganda (MTN/Airtel).
            </p>
          )}
        </div>
      )}

      {/* Withdraw Form */}
      {activeTab === 'withdraw' && (
        <div className="bento-item">
          <h3 style={{ fontSize: '1.125rem', marginBottom: '24px', display: 'flex', alignItems: 'center', gap: '8px' }}>
            <DollarSign size={18} style={{ color: 'var(--color-danger)' }} />
            Withdraw to Mobile Money
          </h3>

          {user?.country_code === 'KE' ? (
            <form onSubmit={handleMpesaWithdraw} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
              <p style={{ fontSize: '0.875rem', color: 'var(--color-text-muted)' }}>
                Withdraw funds from your wallet to your M-Pesa account.
              </p>
              <div>
                <label className="form-label">M-Pesa Phone Number</label>
                <input
                  className="form-input"
                  type="tel"
                  placeholder="+254 7XX XXX XXX"
                  value={mobileForm.phone_number}
                  onChange={(e) => setMobileForm({ ...mobileForm, phone_number: e.target.value })}
                  required
                />
              </div>
              <div>
                <label className="form-label">Amount (KES)</label>
                <input
                  className="form-input"
                  type="number"
                  min="10"
                  step="1"
                  placeholder="100"
                  value={mobileForm.amount}
                  onChange={(e) => setMobileForm({ ...mobileForm, amount: e.target.value })}
                  required
                />
              </div>
              <button type="submit" className="btn btn-primary" disabled={loading} style={{ width: '100%' }}>
                {loading ? 'Processing...' : 'Withdraw to M-Pesa'}
              </button>
            </form>
          ) : user?.country_code === 'UG' ? (
            <form onSubmit={handleUgandaWithdraw} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
              <p style={{ fontSize: '0.875rem', color: 'var(--color-text-muted)' }}>
                Withdraw funds to your Mobile Money account.
              </p>
              <div>
                <label className="form-label">Provider</label>
                <select
                  className="form-input"
                  value={mobileForm.provider}
                  onChange={(e) => setMobileForm({ ...mobileForm, provider: e.target.value })}
                  required
                >
                  <option value="mtn">MTN Mobile Money</option>
                  <option value="airtel">Airtel Money</option>
                </select>
              </div>
              <div>
                <label className="form-label">Phone Number</label>
                <input
                  className="form-input"
                  type="tel"
                  placeholder="+256 77X XXX XXX"
                  value={mobileForm.phone_number}
                  onChange={(e) => setMobileForm({ ...mobileForm, phone_number: e.target.value })}
                  required
                />
              </div>
              <div>
                <label className="form-label">Amount (UGX)</label>
                <input
                  className="form-input"
                  type="number"
                  min="500"
                  step="1"
                  placeholder="5000"
                  value={mobileForm.amount}
                  onChange={(e) => setMobileForm({ ...mobileForm, amount: e.target.value })}
                  required
                />
              </div>
              <button type="submit" className="btn btn-primary" disabled={loading} style={{ width: '100%' }}>
                {loading ? 'Processing...' : 'Withdraw to Mobile Money'}
              </button>
            </form>
          ) : (
            <p style={{ color: 'var(--color-text-muted)', textAlign: 'center', padding: '32px' }}>
              Mobile money withdrawal is currently available for Kenya (M-Pesa) and Uganda (MTN/Airtel).
            </p>
          )}
        </div>
      )}

      {/* Transactions */}
      <div className="bento-item" style={{ padding: '0', overflow: 'hidden' }}>
        <div className="dashboard-panel-header">
          <h3 style={{ fontSize: '1.125rem' }}>Cross-Border Transactions</h3>
          <button className="btn btn-sm btn-secondary" onClick={fetchTransactions}>
            <RefreshCw size={14} />
            <span>Refresh</span>
          </button>
        </div>
        {txsLoading ? (
          <div style={{ padding: '48px', textAlign: 'center', color: 'var(--color-text-muted)' }}>
            Loading...
          </div>
        ) : transactions.length === 0 ? (
          <div style={{ padding: '48px', textAlign: 'center', color: 'var(--color-text-muted)' }}>
            No cross-border transactions yet.
          </div>
        ) : (
          <table className="dashboard-table">
            <thead>
              <tr>
                <th>Date</th>
                <th>Direction</th>
                <th>Sent</th>
                <th>Received</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {transactions.map((tx) => (
                <tr key={tx.id}>
                  <td style={{ fontSize: '0.8125rem', color: 'var(--color-text-muted)' }}>
                    {formatDate(tx.created_at)}
                  </td>
                  <td>
                    <span style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '0.875rem' }}>
                      <ArrowLeftRight size={14} />
                      {tx.send_currency} → {tx.receive_currency}
                    </span>
                  </td>
                  <td style={{ fontWeight: 700 }}>
                    {tx.send_amount.toLocaleString()} {tx.send_currency}
                  </td>
                  <td style={{ fontWeight: 700, color: 'var(--color-success)' }}>
                    {tx.receive_amount.toLocaleString()} {tx.receive_currency}
                  </td>
                  <td>
                    <div className={`status-pill ${tx.status === 'completed' ? 'success' : tx.status === 'pending' ? 'warning' : 'error'}`}>
                      {tx.status}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}

export default CrossBorderTransfer