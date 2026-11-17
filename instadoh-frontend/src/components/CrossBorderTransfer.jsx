import React, { useState, useEffect } from 'react'
import { useAuth } from '../context/AuthContext'
import { crossBorderApi, mpesaApi, ugandaMobileApi } from '../api/crossBorder'
import { paymentsApi } from '../api/payments'
import {
  RefreshCw, Send, TrendingUp, DollarSign, Mail, CheckCircle2,
  Zap, Copy, Check, Wallet, Globe,
} from 'lucide-react'

function CrossBorderTransfer() {
  const { user } = useAuth()
  const [activeTab, setActiveTab] = useState('send') // send, deposit, withdraw
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [config, setConfig] = useState({
    lightning_enabled: true,
    ln_url_domain: 'instadoh.com',
    mpesa_enabled: false,
    uganda_mobile_enabled: false,
  })
  const [copied, setCopied] = useState('')

  const lnAddress = user?.ln_address || ''
  const mpesaAvailable = config.mpesa_enabled && user?.country_code === 'KE'
  const ugandaMobileAvailable = config.uganda_mobile_enabled && user?.country_code === 'UG'

  // Send (global — any country, any currency, identified by email)
  const [sendForm, setSendForm] = useState({
    recipient_email: '',
    amount: '',
    currency: user?.currency || 'USD',
    description: '',
  })
  const [quote, setQuote] = useState(null)
  const [quoteLoading, setQuoteLoading] = useState(false)
  const [recipient, setRecipient] = useState(null)
  const [recipientLoading, setRecipientLoading] = useState(false)

  // Lightning deposit (fund wallet)
  const [depositForm, setDepositForm] = useState({
    amount: '',
    currency: user?.currency || 'USD',
  })
  const [invoice, setInvoice] = useState(null)
  const [invoiceLoading, setInvoiceLoading] = useState(false)

  // Lightning withdraw (cash out globally)
  const [withdrawForm, setWithdrawForm] = useState({
    destination: '',
    amount: '',
    currency: user?.currency || 'USD',
    description: '',
  })

  // Mobile money (optional, region-specific)
  const [mobileForm, setMobileForm] = useState({
    phone_number: '',
    amount: '',
    provider: 'mtn',
  })

  // Transactions
  const [transactions, setTransactions] = useState([])
  const [txsLoading, setTxsLoading] = useState(false)

  useEffect(() => {
    fetchTransactions()
    paymentsApi.getConfig()
      .then((res) => setConfig(res.data))
      .catch(() => {})
  }, [])

  // Resolve recipient by email (debounced) — email is the universal recipient
  // identifier, so the recipient's country/currency come from their account.
  useEffect(() => {
    const trimmed = sendForm.recipient_email.trim()
    if (!trimmed || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(trimmed)) {
      setRecipient(null)
      return
    }

    const timer = setTimeout(async () => {
      setRecipientLoading(true)
      setError('')
      try {
        const res = await paymentsApi.resolveRecipient(trimmed)
        setRecipient(res.data)
      } catch (err) {
        setRecipient(null)
      } finally {
        setRecipientLoading(false)
      }
    }, 500)

    return () => clearTimeout(timer)
  }, [sendForm.recipient_email])

  // Fetch quote when amount/recipient changes
  useEffect(() => {
    if (!sendForm.amount || parseFloat(sendForm.amount) <= 0) {
      setQuote(null)
      return
    }

    const amount = parseFloat(sendForm.amount)
    const toCurrency = recipient?.currency

    // Without a resolved recipient we don't know the target currency yet.
    if (!toCurrency) {
      setQuote(null)
      return
    }

    const timer = setTimeout(async () => {
      setQuoteLoading(true)
      setError('')
      try {
        if (toCurrency === sendForm.currency) {
          const fee = Math.round(amount * 0.02 * 100) / 100
          setQuote({
            send_amount: amount,
            receive_amount: amount,
            from_currency: sendForm.currency,
            to_currency: toCurrency,
            exchange_rate: 1,
            fee,
            total_in_fiat: amount + fee,
            valid_until: '',
          })
        } else {
          const res = await crossBorderApi.getQuote(sendForm.currency, toCurrency, amount)
          setQuote(res.data)
        }
      } catch (err) {
        setQuote(null)
      } finally {
        setQuoteLoading(false)
      }
    }, 500)

    return () => clearTimeout(timer)
  }, [sendForm.amount, sendForm.currency, recipient?.currency])

  const fetchTransactions = async () => {
    setTxsLoading(true)
    try {
      const res = await paymentsApi.listTransactions(1, 10)
      const data = res?.data?.data ?? res?.data ?? []
      setTransactions(Array.isArray(data) ? data : [])
    } catch (err) {
      setTransactions([])
    } finally {
      setTxsLoading(false)
    }
  }

  const copyText = async (text) => {
    try {
      await navigator.clipboard.writeText(text)
      setCopied(text)
      setTimeout(() => setCopied(''), 1500)
    } catch (err) {
      // clipboard unavailable
    }
  }

  const handleSend = async (e) => {
    e.preventDefault()
    setLoading(true)
    setError('')
    setSuccess('')

    try {
      await paymentsApi.sendByEmail({
        recipient_email: sendForm.recipient_email,
        amount: parseFloat(sendForm.amount),
        currency: sendForm.currency,
        description: sendForm.description,
      })
      setSuccess(`Successfully sent ${sendForm.amount} ${sendForm.currency} to ${sendForm.recipient_email}!`)
      setSendForm({ ...sendForm, recipient_email: '', amount: '', description: '' })
      setQuote(null)
      setRecipient(null)
      fetchTransactions()
    } catch (err) {
      setError(err.response?.data?.message || 'Failed to send payment')
    } finally {
      setLoading(false)
    }
  }

  const handleCreateInvoice = async (e) => {
    e.preventDefault()
    setInvoiceLoading(true)
    setError('')
    setSuccess('')

    try {
      const res = await paymentsApi.createInvoice({
        amount: parseFloat(depositForm.amount),
        currency: depositForm.currency,
        description: 'Instadoh wallet deposit',
        expiry_seconds: 3600,
      })
      setInvoice(res.data)
    } catch (err) {
      setError(err.response?.data?.message || 'Failed to create invoice')
    } finally {
      setInvoiceLoading(false)
    }
  }

  const handleWithdrawLightning = async (e) => {
    e.preventDefault()
    setLoading(true)
    setError('')
    setSuccess('')

    try {
      const res = await paymentsApi.withdraw({
        destination: withdrawForm.destination,
        amount: parseFloat(withdrawForm.amount),
        currency: withdrawForm.currency,
        description: withdrawForm.description,
      })
      setSuccess(`Withdrawn ${res.data.amount} ${res.data.currency} to ${withdrawForm.destination}`)
      setWithdrawForm({ ...withdrawForm, destination: '', amount: '', description: '' })
      fetchTransactions()
    } catch (err) {
      setError(err.response?.data?.message || 'Withdrawal failed')
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
      await mpesaApi.deposit({
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
      await mpesaApi.withdraw({
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
      await ugandaMobileApi.deposit({
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
      await ugandaMobileApi.withdraw({
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
            <span>{tab === 'send' ? 'Send Money' : tab}</span>
          </button>
        ))}
      </div>

      {/* ————— Send Money (global) ————— */}
      {activeTab === 'send' && (
        <div className="bento-item">
          <h3 style={{ fontSize: '1.125rem', marginBottom: '8px', display: 'flex', alignItems: 'center', gap: '8px' }}>
            <Send size={18} className="text-gold" />
            Send Money — Any Country
          </h3>
          <p style={{ fontSize: '0.875rem', color: 'var(--color-text-muted)', marginBottom: '24px' }}>
            Send to any Instadoh user worldwide using just their email. The recipient receives funds in their own currency. No phone number, no mobile money required.
          </p>

          <form onSubmit={handleSend} style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
            <div>
              <label className="form-label">Recipient Email</label>
              <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                <Mail size={16} style={{ color: 'var(--color-text-muted)' }} />
                <input
                  className="form-input"
                  type="email"
                  placeholder="bob@example.com"
                  value={sendForm.recipient_email}
                  onChange={(e) => setSendForm({ ...sendForm, recipient_email: e.target.value })}
                  required
                />
              </div>

              {recipientLoading && (
                <div style={{ fontSize: '0.8125rem', color: 'var(--color-text-muted)', marginTop: '6px' }}>
                  Looking up recipient...
                </div>
              )}

              {recipient && (
                <div className="bento-item" style={{ background: 'var(--color-bg-secondary)', padding: '12px', marginTop: '10px', display: 'flex', alignItems: 'center', gap: '10px' }}>
                  <CheckCircle2 size={16} style={{ color: 'var(--color-success)', flexShrink: 0 }} />
                  <div style={{ fontSize: '0.875rem' }}>
                    <strong>{recipient.full_name}</strong>
                    <span style={{ color: 'var(--color-text-muted)' }}>
                      {' '}— receives {recipient.currency} in {recipient.country_code}
                    </span>
                  </div>
                </div>
              )}

              {!recipient && !recipientLoading && sendForm.recipient_email.trim() && (
                <div style={{ fontSize: '0.8125rem', color: 'var(--color-danger)', marginTop: '6px' }}>
                  No Instadoh account found for this email
                </div>
              )}
            </div>

            <div>
              <label className="form-label">Your Currency</label>
              <select
                className="form-input"
                value={sendForm.currency}
                onChange={(e) => setSendForm({ ...sendForm, currency: e.target.value })}
                required
              >
                <option value="USD">USD</option>
                <option value="KES">KES - Kenyan Shilling</option>
                <option value="UGX">UGX - Ugandan Shilling</option>
                <option value="GBP">GBP - British Pound</option>
                <option value="EUR">EUR - Euro</option>
                <option value="NGN">NGN - Nigerian Naira</option>
                <option value="ZAR">ZAR - South African Rand</option>
                <option value="TZS">TZS - Tanzanian Shilling</option>
                <option value="ETB">ETB - Ethiopian Birr</option>
              </select>
              <div style={{ fontSize: '0.75rem', color: 'var(--color-text-muted)', marginTop: '4px' }}>
                Your current balance is in {user?.currency || 'USD'}
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
                  placeholder="100"
                  value={sendForm.amount}
                  onChange={(e) => setSendForm({ ...sendForm, amount: e.target.value })}
                  required
                />
              </div>
            </div>

            {quoteLoading && (
              <div style={{ fontSize: '0.875rem', color: 'var(--color-text-muted)' }}>
                Converting currencies...
              </div>
            )}

            {quote && !quoteLoading && (
              <div className="bento-item" style={{ background: 'var(--color-bg-secondary)', padding: '16px' }}>
                <h4 style={{ fontSize: '0.875rem', color: 'var(--color-text-muted)', marginBottom: '12px' }}>Summary</h4>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.875rem' }}>
                    <span style={{ color: 'var(--color-text-muted)' }}>You Send</span>
                    <span style={{ fontWeight: 700 }}>{quote.send_amount.toLocaleString()} {quote.from_currency}</span>
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.875rem' }}>
                    <span style={{ color: 'var(--color-text-muted)' }}>Recipient Gets</span>
                    <span style={{ fontWeight: 700, color: 'var(--color-success)' }}>{quote.receive_amount.toLocaleString()} {quote.to_currency}</span>
                  </div>
                  {quote.from_currency !== quote.to_currency && (
                    <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.875rem' }}>
                      <span style={{ color: 'var(--color-text-muted)' }}>Exchange Rate</span>
                      <span>1 {quote.from_currency} = {quote.exchange_rate} {quote.to_currency}</span>
                    </div>
                  )}
                  <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.875rem' }}>
                    <span style={{ color: 'var(--color-text-muted)' }}>Fee (2%)</span>
                    <span style={{ color: 'var(--color-danger)' }}>{quote.fee.toLocaleString()} {quote.from_currency}</span>
                  </div>
                  <div style={{ borderTop: '1px solid var(--color-border)', paddingTop: '8px', display: 'flex', justifyContent: 'space-between', fontSize: '0.875rem' }}>
                    <span style={{ color: 'var(--color-text-muted)' }}>Total Deducted</span>
                    <span style={{ fontWeight: 700 }}>{quote.total_in_fiat.toLocaleString()} {quote.from_currency}</span>
                  </div>
                </div>
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

            <button type="submit" className="btn btn-primary" disabled={loading || !quote || !recipient} style={{ width: '100%' }}>
              {loading ? 'Sending...' : !recipient ? 'Enter a valid recipient email to continue' : 'Send Money'}
            </button>
          </form>
        </div>
      )}

      {/* ————— Deposit (fund wallet) ————— */}
      {activeTab === 'deposit' && (
        <div className="bento-item">
          <h3 style={{ fontSize: '1.125rem', marginBottom: '8px', display: 'flex', alignItems: 'center', gap: '8px' }}>
            <TrendingUp size={18} style={{ color: 'var(--color-success)' }} />
            Deposit Funds
          </h3>
          <p style={{ fontSize: '0.875rem', color: 'var(--color-text-muted)', marginBottom: '24px' }}>
            Fund your wallet from anywhere in the world over the Lightning Network. Anyone can pay your Lightning Address or scan your invoice.
          </p>

          {/* Lightning Address — global receive */}
          <div className="bento-item" style={{ background: 'var(--color-bg-secondary)', padding: '16px', marginBottom: '20px' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px' }}>
              <Globe size={16} className="text-gold" />
              <h4 style={{ fontSize: '0.875rem', fontWeight: 700 }}>Your Lightning Address (worldwide)</h4>
            </div>
            {lnAddress ? (
              <>
                <div style={{ display: 'flex', gap: '8px', alignItems: 'center', fontFamily: 'monospace', fontSize: '1rem' }}>
                  <span style={{ fontWeight: 700 }}>{lnAddress}</span>
                  <button type="button" className="btn btn-sm btn-secondary" onClick={() => copyText(lnAddress)}>
                    {copied === lnAddress ? <Check size={14} /> : <Copy size={14} />}
                    <span>Copy</span>
                  </button>
                </div>
                <p style={{ fontSize: '0.8125rem', color: 'var(--color-text-muted)', marginTop: '8px' }}>
                  Share this address — any Lightning wallet can pay it and your balance is credited automatically.
                </p>
              </>
            ) : (
              <p style={{ fontSize: '0.8125rem', color: 'var(--color-text-muted)' }}>
                Lightning Address not assigned to this account.
              </p>
            )}
          </div>

          {/* Create invoice to fund wallet */}
          <form onSubmit={handleCreateInvoice} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
              <div>
                <label className="form-label">Amount</label>
                <input
                  className="form-input"
                  type="number"
                  min="1"
                  step="0.01"
                  placeholder={user?.currency || '100'}
                  value={depositForm.amount}
                  onChange={(e) => setDepositForm({ ...depositForm, amount: e.target.value })}
                  required
                />
              </div>
              <div>
                <label className="form-label">Currency</label>
                <select
                  className="form-input"
                  value={depositForm.currency}
                  onChange={(e) => setDepositForm({ ...depositForm, currency: e.target.value })}
                >
                  <option value="USD">USD</option>
                  <option value="KES">KES</option>
                  <option value="UGX">UGX</option>
                  <option value="GBP">GBP</option>
                  <option value="EUR">EUR</option>
                  <option value="NGN">NGN</option>
                </select>
              </div>
            </div>

            <button type="submit" className="btn btn-primary" disabled={invoiceLoading || !config.lightning_enabled} style={{ width: '100%' }}>
              {invoiceLoading ? 'Creating invoice...' : 'Create Payment Invoice'}
            </button>
          </form>

          {invoice && (
            <div className="bento-item" style={{ background: 'var(--color-success)10', border: '1px solid var(--color-success)30', padding: '16px', marginTop: '20px' }}>
              <h4 style={{ fontSize: '0.875rem', fontWeight: 700, marginBottom: '8px' }}>
                Pay this invoice to credit your wallet ({invoice.amount} {invoice.currency})
              </h4>
              <div style={{ display: 'flex', gap: '8px', alignItems: 'flex-start' }}>
                <code style={{ flex: 1, fontSize: '0.8125rem', wordBreak: 'break-all', fontFamily: 'monospace', padding: '10px', background: 'var(--color-surface)', borderRadius: 'var(--radius-sm)' }}>
                  {invoice.payment_request}
                </code>
                <button type="button" className="btn btn-sm btn-secondary" onClick={() => copyText(invoice.payment_request)}>
                  {copied === invoice.payment_request ? <Check size={14} /> : <Copy size={14} />}
                  <span>Copy</span>
                </button>
              </div>
            </div>
          )}

          {/* Mobile money — optional regional on-ramp */}
          {(mpesaAvailable || ugandaMobileAvailable) && (
            <div style={{ marginTop: '32px', paddingTop: '24px', borderTop: '1px solid var(--color-border)' }}>
              <h4 style={{ fontSize: '0.875rem', fontWeight: 700, marginBottom: '12px', display: 'flex', alignItems: 'center', gap: '6px' }}>
                <Zap size={14} className="text-gold" />
                Mobile Money Deposit (region-specific)
              </h4>

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
              ) : null}
            </div>
          )}
        </div>
      )}

      {/* ————— Withdraw (cash out globally) ————— */}
      {activeTab === 'withdraw' && (
        <div className="bento-item">
          <h3 style={{ fontSize: '1.125rem', marginBottom: '8px', display: 'flex', alignItems: 'center', gap: '8px' }}>
            <DollarSign size={18} style={{ color: 'var(--color-danger)' }} />
            Withdraw to Lightning
          </h3>
          <p style={{ fontSize: '0.875rem', color: 'var(--color-text-muted)', marginBottom: '24px' }}>
            Cash out to any Lightning wallet in the world by pasting a BOLT11 invoice or a Lightning Address (alice@example.com). No mobile money needed.
          </p>

          <form onSubmit={handleWithdrawLightning} style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
            <div>
              <label className="form-label">Destination — Invoice or Lightning Address</label>
              <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                <Wallet size={16} style={{ color: 'var(--color-text-muted)' }} />
                <input
                  className="form-input"
                  type="text"
                  placeholder="lnbc1... or alice@getalby.com"
                  value={withdrawForm.destination}
                  onChange={(e) => setWithdrawForm({ ...withdrawForm, destination: e.target.value })}
                  required
                />
              </div>
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
              <div>
                <label className="form-label">Amount</label>
                <input
                  className="form-input"
                  type="number"
                  min="1"
                  step="0.01"
                  placeholder={user?.currency || '100'}
                  value={withdrawForm.amount}
                  onChange={(e) => setWithdrawForm({ ...withdrawForm, amount: e.target.value })}
                />
                <div style={{ fontSize: '0.75rem', color: 'var(--color-text-muted)', marginTop: '4px' }}>
                  Optional for fixed-amount invoices
                </div>
              </div>
              <div>
                <label className="form-label">Currency</label>
                <select
                  className="form-input"
                  value={withdrawForm.currency}
                  onChange={(e) => setWithdrawForm({ ...withdrawForm, currency: e.target.value })}
                >
                  <option value="USD">USD</option>
                  <option value="KES">KES</option>
                  <option value="UGX">UGX</option>
                  <option value="GBP">GBP</option>
                  <option value="EUR">EUR</option>
                  <option value="NGN">NGN</option>
                </select>
              </div>
            </div>

            <div>
              <label className="form-label">Description (Optional)</label>
              <input
                className="form-input"
                type="text"
                placeholder="e.g., Cash out"
                value={withdrawForm.description}
                onChange={(e) => setWithdrawForm({ ...withdrawForm, description: e.target.value })}
              />
            </div>

            <button type="submit" className="btn btn-primary" disabled={loading || !config.lightning_enabled} style={{ width: '100%' }}>
              {loading ? 'Withdrawing...' : 'Withdraw via Lightning'}
            </button>
          </form>

          {/* Mobile money — optional regional off-ramp */}
          {(mpesaAvailable || ugandaMobileAvailable) && (
            <div style={{ marginTop: '32px', paddingTop: '24px', borderTop: '1px solid var(--color-border)' }}>
              <h4 style={{ fontSize: '0.875rem', fontWeight: 700, marginBottom: '12px' }}>
                <DollarSign size={14} style={{ color: 'var(--color-danger)' }} />
                {' '}Withdraw to Mobile Money (region-specific)
              </h4>

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
              ) : null}
            </div>
          )}
        </div>
      )}

      {/* Transactions */}
      <div className="bento-item" style={{ padding: '0', overflow: 'hidden' }}>
        <div className="dashboard-panel-header">
          <h3 style={{ fontSize: '1.125rem' }}>Recent Payments</h3>
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
            No payments yet.
          </div>
        ) : (
          <table className="dashboard-table">
            <thead>
              <tr>
                <th>Date</th>
                <th>Counterparty</th>
                <th>Amount</th>
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
                      <Mail size={14} />
                      {tx.counterparty || (tx.direction === 'incoming' ? 'Received' : 'Sent')}
                    </span>
                  </td>
                  <td style={{ fontWeight: 700, color: tx.direction === 'incoming' ? 'var(--color-success)' : 'var(--color-text-main)' }}>
                    {tx.direction === 'incoming' ? '+' : '-'}{Number(tx.amount || 0).toLocaleString()} {tx.currency}
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