import React, { useState, useEffect } from 'react'
import { useAuth } from '../context/AuthContext'
import { paymentsApi } from '../api/payments'

function Dashboard() {
  const { user, logout } = useAuth()
  const [balance, setBalance] = useState(null)
  const [stats, setStats] = useState(null)
  const [transactions, setTransactions] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const fetchData = async () => {
    setLoading(true)
    setError('')
    try {
      const [balRes, statsRes, txsRes] = await Promise.all([
        paymentsApi.getBalance(),
        paymentsApi.getStats(),
        paymentsApi.listTransactions(1, 10),
      ])
      setBalance(balRes.data)
      setStats(statsRes.data)
      setTransactions(txsRes.data.data || [])
    } catch (err) {
      setError(err.response?.data?.message || 'Failed to load dashboard data')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [])

  if (loading) {
    return (
      <div className="dashboard-page">
        <div className="dashboard-loading">
          <div className="spinner"></div>
          <p>Loading your dashboard...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="dashboard-page">
        <div className="dashboard-error">
          <span className="dashboard-error-icon">⚠</span>
          <h3>Something went wrong</h3>
          <p>{error}</p>
          <button className="btn btn-primary" onClick={fetchData}>Try Again</button>
        </div>
      </div>
    )
  }

  const statCards = [
    {
      icon: '₿',
      label: 'Bitcoin Balance',
      value: balance?.balance?.toFixed(4) || '0.0000',
      unit: 'BTC',
      change: stats?.balance_change ? `${stats.balance_change > 0 ? '+' : ''}${stats.balance_change}%` : '0%',
      positive: (stats?.balance_change || 0) >= 0,
      color: '#2563eb',
      bg: 'rgba(37, 99, 235, 0.1)',
    },
    {
      icon: '$',
      label: 'USD Value',
      value: balance?.usd_value ? `$${balance.usd_value.toLocaleString()}` : '$0',
      unit: '',
      change: stats?.usd_change ? `${stats.usd_change > 0 ? '+' : ''}${stats.usd_change}%` : '0%',
      positive: (stats?.usd_change || 0) >= 0,
      color: '#10b981',
      bg: 'rgba(16, 185, 129, 0.1)',
    },
    {
      icon: '📨',
      label: 'Active Invoices',
      value: stats?.active_invoices?.toString() || '0',
      unit: '',
      change: stats?.invoice_change ? `${stats.invoice_change > 0 ? '+' : ''}${stats.invoice_change}` : '0',
      positive: (stats?.invoice_change || 0) >= 0,
      color: '#f59e0b',
      bg: 'rgba(245, 158, 11, 0.1)',
    },
    {
      icon: '⚡',
      label: 'Total Transactions',
      value: stats?.total_transactions?.toString() || '0',
      unit: '',
      change: stats?.tx_change ? `${stats.tx_change > 0 ? '+' : ''}${stats.tx_change}%` : '0%',
      positive: (stats?.tx_change || 0) >= 0,
      color: '#7c3aed',
      bg: 'rgba(124, 58, 237, 0.1)',
    },
  ]

  const formatDate = (dateStr) => {
    const d = new Date(dateStr)
    const now = new Date()
    const diff = now - d
    const mins = Math.floor(diff / 60000)
    if (mins < 1) return 'Just now'
    if (mins < 60) return `${mins} minute${mins > 1 ? 's' : ''} ago`
    const hours = Math.floor(mins / 60)
    if (hours < 24) return `${hours} hour${hours > 1 ? 's' : ''} ago`
    const days = Math.floor(hours / 24)
    return `${days} day${days > 1 ? 's' : ''} ago`
  }

  return (
    <div className="dashboard-page">
      <div className="dashboard-header">
        <div className="container">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '16px' }}>
            <div>
              <h1 className="dashboard-title">Dashboard</h1>
              <p className="dashboard-subtitle">
                Welcome back{user?.full_name ? `, ${user.full_name}` : ''}! Here's your payment overview.
              </p>
            </div>
            <div style={{ display: 'flex', gap: '12px', alignItems: 'center' }}>
              <button className="btn btn-secondary btn-sm" onClick={fetchData}>
                <span>↻</span>
                Refresh
              </button>
              <button className="btn btn-secondary btn-sm" onClick={logout}>
                Sign Out
              </button>
            </div>
          </div>
        </div>
      </div>

      <div className="dashboard-content">
        <div className="container">
          <div className="dashboard-grid">
            {statCards.map((stat, index) => (
              <div className="dashboard-card" key={index}>
                <div className="dashboard-card-icon" style={{ background: stat.bg, color: stat.color }}>
                  {stat.icon}
                </div>
                <div className="dashboard-card-label">{stat.label}</div>
                <div className="dashboard-card-value">
                  {stat.value}
                  {stat.unit && <span style={{ fontSize: '1rem', color: 'var(--color-text-muted)', fontWeight: 500 }}> {stat.unit}</span>}
                </div>
                <div className={`dashboard-card-change ${stat.positive ? 'positive' : 'negative'}`}>
                  {stat.positive ? '↑' : '↓'} {stat.change}
                </div>
              </div>
            ))}
          </div>

          <div className="dashboard-main">
            <div className="dashboard-panel">
              <div className="dashboard-panel-header">
                <h3 className="dashboard-panel-title">Recent Transactions</h3>
                <button className="dashboard-panel-action">View All</button>
              </div>
              {transactions.length === 0 ? (
                <div className="dashboard-empty">
                  <p>No transactions yet. Create your first invoice to get started.</p>
                </div>
              ) : (
                <div className="transaction-list">
                  {transactions.map((tx) => (
                    <div className="transaction-item" key={tx.id}>
                      <div className="transaction-info">
                        <div className={`transaction-icon ${tx.direction}`}>
                          {tx.direction === 'incoming' ? '↓' : '↑'}
                        </div>
                        <div>
                          <div className="transaction-name">
                            {tx.description || (tx.direction === 'incoming' ? 'Payment received' : 'Payment sent')}
                          </div>
                          <div className="transaction-date">{formatDate(tx.created_at)}</div>
                        </div>
                      </div>
                      <div className={`transaction-amount ${tx.direction === 'incoming' ? 'positive' : 'negative'}`}>
                        {tx.direction === 'incoming' ? '+' : '-'}{tx.amount} {tx.currency}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>

            <div className="dashboard-panel">
              <div className="dashboard-panel-header">
                <h3 className="dashboard-panel-title">Account Overview</h3>
                <button className="dashboard-panel-action">Profile</button>
              </div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                <div className="dashboard-info-row">
                  <div>
                    <div className="info-label">Email</div>
                    <div className="info-value">{user?.email || '—'}</div>
                  </div>
                </div>
                <div className="dashboard-info-row">
                  <div>
                    <div className="info-label">Phone</div>
                    <div className="info-value">{user?.phone || '—'}</div>
                  </div>
                </div>
                <div className="dashboard-info-row">
                  <div>
                    <div className="info-label">Country</div>
                    <div className="info-value">{user?.country_code || '—'}</div>
                  </div>
                </div>
                <div className="dashboard-info-row">
                  <div>
                    <div className="info-label">Currency</div>
                    <div className="info-value">{user?.currency || '—'}</div>
                  </div>
                </div>
                <div className="dashboard-info-row">
                  <div>
                    <div className="info-label">Account Type</div>
                    <div className="info-value" style={{ textTransform: 'capitalize' }}>{user?.role || '—'}</div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

export default Dashboard