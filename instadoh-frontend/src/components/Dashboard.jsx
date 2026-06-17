import React, { useState, useEffect } from 'react'
import { useAuth } from '../context/AuthContext'
import { paymentsApi } from '../api/payments'
import { 
  LayoutDashboard, 
  ArrowLeftRight, 
  Settings, 
  User, 
  RefreshCw, 
  LogOut, 
  Bitcoin, 
  Zap, 
  DollarSign, 
  History,
  TrendingUp,
  TrendingDown,
  ChevronRight
} from 'lucide-react'

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
      <div className="dashboard-page" style={{ justifyContent: 'center', alignItems: 'center' }}>
        <div style={{ textAlign: 'center' }}>
          <Zap size={48} className="text-gold" fill="currentColor" style={{ marginBottom: '24px', opacity: 0.5 }} />
          <p style={{ color: 'var(--color-text-muted)', fontSize: '1.125rem' }}>Syncing with Lightning Network...</p>
        </div>
      </div>
    )
  }

  const statCards = [
    {
      icon: <Bitcoin size={20} />,
      label: 'Bitcoin Balance',
      value: balance?.balance?.toFixed(8) || '0.00000000',
      unit: 'BTC',
      change: stats?.balance_change || 0,
      color: 'var(--color-primary)',
    },
    {
      icon: <DollarSign size={20} />,
      label: 'USD Equivalent',
      value: balance?.usd_value ? `$${balance.usd_value.toLocaleString()}` : '$0.00',
      unit: '',
      change: stats?.usd_change || 0,
      color: '#10b981',
    },
    {
      icon: <Zap size={20} />,
      label: 'Active Invoices',
      value: stats?.active_invoices?.toString() || '0',
      unit: '',
      change: stats?.invoice_change || 0,
      color: '#f59e0b',
    },
    {
      icon: <History size={20} />,
      label: 'Total Volume',
      value: stats?.total_transactions?.toString() || '0',
      unit: 'TXs',
      change: stats?.tx_change || 0,
      color: 'var(--color-secondary)',
    },
  ]

  const formatDate = (dateStr) => {
    const d = new Date(dateStr)
    return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
  }

  return (
    <div className="dashboard-page">
      {/* Sidebar */}
      <aside className="dashboard-sidebar">
        <div className="dashboard-sidebar-brand">
          <div className="navbar-brand">
            <Zap size={24} className="text-gold" fill="currentColor" />
            <span>Instadoh</span>
          </div>
        </div>

        <nav className="dashboard-sidebar-nav">
          <button className="dashboard-sidebar-link active">
            <LayoutDashboard size={20} />
            <span>Overview</span>
          </button>
          <button className="dashboard-sidebar-link">
            <ArrowLeftRight size={20} />
            <span>Payments</span>
          </button>
          <button className="dashboard-sidebar-link">
            <Settings size={20} />
            <span>Node Config</span>
          </button>
          <button className="dashboard-sidebar-link">
            <User size={20} />
            <span>Account</span>
          </button>
        </nav>

        <div className="dashboard-sidebar-footer">
          <button onClick={logout} className="dashboard-sidebar-link" style={{ color: 'var(--color-danger)', width: '100%', border: 'none' }}>
            <LogOut size={20} />
            <span>Sign Out</span>
          </button>
        </div>
      </aside>

      {/* Main Content */}
      <main className="dashboard-main">
        <header className="dashboard-header">
          <div>
            <h1 className="dashboard-header-title">Vault Overview</h1>
            <p className="dashboard-header-subtitle">Welcome back, {user?.full_name || 'Operator'}</p>
          </div>
          <button className="btn btn-secondary btn-sm" onClick={fetchData}>
            <RefreshCw size={14} />
            <span>Sync Rails</span>
          </button>
        </header>

        <div className="dashboard-grid">
          {statCards.map((stat, i) => (
            <div key={i} className="dashboard-card">
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
                <div style={{ color: stat.color, background: `${stat.color}15`, padding: '8px', borderRadius: 'var(--radius-sm)' }}>
                  {stat.icon}
                </div>
                <div style={{ 
                  display: 'flex', alignItems: 'center', gap: '4px', fontSize: '0.75rem', fontWeight: 700,
                  color: stat.change >= 0 ? 'var(--color-success)' : 'var(--color-danger)'
                }}>
                  {stat.change >= 0 ? <TrendingUp size={14} /> : <TrendingDown size={14} />}
                  <span>{Math.abs(stat.change)}%</span>
                </div>
              </div>
              <div style={{ color: 'var(--color-text-muted)', fontSize: '0.75rem', fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: '4px' }}>{stat.label}</div>
              <div style={{ fontSize: '1.75rem', fontWeight: 800, fontFamily: 'var(--font-secondary)', color: 'var(--color-text-main)' }}>
                {stat.value}
                {stat.unit && <span style={{ fontSize: '0.875rem', color: 'var(--color-text-light)', marginLeft: '4px', fontWeight: 500 }}>{stat.unit}</span>}
              </div>
            </div>
          ))}
        </div>

        <div className="dashboard-content-grid">
          {/* Transactions Panel */}
          <div className="bento-item" style={{ padding: '0', overflow: 'hidden' }}>
            <div className="dashboard-panel-header">
              <h3 style={{ fontSize: '1.125rem' }}>Settled Payments</h3>
              <button style={{ color: 'var(--color-primary)', fontSize: '0.8125rem', fontWeight: 600, display: 'flex', alignItems: 'center', gap: '4px' }}>
                View Audit Log <ChevronRight size={14} />
              </button>
            </div>
            {transactions.length === 0 ? (
              <div style={{ padding: '64px', textAlign: 'center', color: 'var(--color-text-muted)' }}>
                No settled payments found in this epoch.
              </div>
            ) : (
              <table className="dashboard-table">
                <thead>
                  <tr>
                    <th>Invoice ID</th>
                    <th>Timestamp</th>
                    <th>Volume</th>
                    <th>Status</th>
                  </tr>
                </thead>
                <tbody>
                  {transactions.map((tx) => (
                    <tr key={tx.id}>
                      <td style={{ fontFamily: 'monospace', color: 'var(--color-text-muted)', fontSize: '0.8125rem' }}>
                        {tx.id.substring(0, 12)}...
                      </td>
                      <td style={{ fontSize: '0.875rem' }}>{formatDate(tx.created_at)}</td>
                      <td style={{ fontWeight: 700, color: tx.direction === 'incoming' ? 'var(--color-success)' : 'var(--color-text-main)' }}>
                        {tx.direction === 'incoming' ? '+' : '-'}{tx.amount.toLocaleString()} {tx.currency || 'sats'}
                      </td>
                      <td>
                        <div className="status-pill success">
                          <div style={{ width: '6px', height: '6px', borderRadius: '50%', background: 'currentColor' }}></div>
                          Settled
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>

          {/* Node Health / Quick Actions */}
          <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
            <div className="bento-item">
              <h3 style={{ fontSize: '1.125rem', marginBottom: '24px' }}>Lightning Node</h3>
              <div style={{ marginBottom: '24px' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '8px' }}>
                  <span style={{ fontSize: '0.8125rem', color: 'var(--color-text-muted)' }}>Channel Liquidity</span>
                  <span style={{ fontSize: '0.8125rem', fontWeight: 700 }}>84%</span>
                </div>
                <div style={{ height: '6px', background: 'rgba(255, 255, 255, 0.05)', borderRadius: '100px', overflow: 'hidden' }}>
                  <div style={{ width: '84%', height: '100%', background: 'var(--color-primary)' }}></div>
                </div>
              </div>
              
              <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                <button className="btn btn-primary" style={{ width: '100%' }}>Create Invoice</button>
                <button className="btn btn-secondary" style={{ width: '100%' }}>Manage Channels</button>
              </div>
              
              <div style={{ marginTop: '24px', paddingTop: '24px', borderTop: '1px solid var(--color-border)' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '10px', fontSize: '0.8125rem' }}>
                  <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: 'var(--color-success)' }}></div>
                  <span style={{ color: 'var(--color-text-muted)' }}>LND Node Online (v0.17.4)</span>
                </div>
              </div>
            </div>

            <div className="bento-item" style={{ background: 'linear-gradient(135deg, var(--color-accent-mauve) 0%, var(--color-bg) 100%)', border: '1px solid var(--color-accent-mauve)' }}>
              <h3 style={{ fontSize: '1.125rem', marginBottom: '12px' }}>Enterprise Support</h3>
              <p style={{ fontSize: '0.875rem', color: 'var(--color-text-muted)', marginBottom: '20px' }}>
                Need help with node optimization or liquidity management? Talk to our specialists.
              </p>
              <button className="btn btn-secondary btn-sm" style={{ background: 'white', color: 'black', border: 'none' }}>
                Open Ticket
              </button>
            </div>
          </div>
        </div>
      </main>
    </div>
  )
}

export default Dashboard
