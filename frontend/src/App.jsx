import React, { useEffect, useMemo, useState } from 'react'
import { createRoot } from 'react-dom/client'
import {
  Activity, Globe, KeyRound, LogOut, Mic, Menu, Waves,
  ShieldCheck, Settings, Volume2, X
} from 'lucide-react'
import { api } from './lib/api.js'
import Home from './pages/Home.jsx'
import Admin from './pages/Admin.jsx'
import './style.css'

function getToken() { return localStorage.getItem('adminToken') || '' }

const NAV_ITEMS = [
  { id: 'tts',      label: '语音合成',  icon: Volume2,    section: '核心功能' },
  { id: 'voices',   label: '音色管理',  icon: Mic,        section: '核心功能' },
  { id: 'mimo',     label: 'MiMo Keys', icon: KeyRound,   section: '管理' },
  { id: 'channels', label: '上游渠道',  icon: Globe,      section: '管理' },
  { id: 'ext',      label: 'API Keys',  icon: KeyRound,   section: '管理' },
  { id: 'calllogs', label: '调用记录',  icon: Activity,   section: '监控' },
  { id: 'status',   label: '系统状态',  icon: ShieldCheck, section: '监控' },
  { id: 'settings', label: '系统设置',  icon: Settings, section: '系统' },
]

const PAGE_TITLES = {
  tts:      '语音合成',
  voices:   '音色管理',
  mimo:     'MiMo API Keys',
  channels: '上游渠道',
  ext:      '外部 API Keys',
  calllogs: '调用记录',
  status:   '系统状态',
  settings: '系统设置',
}

function App() {
  const [token, setToken] = useState(getToken())
  const [authed, setAuthed] = useState(false)
  const [checking, setChecking] = useState(!!getToken())
  const [loginToken, setLoginToken] = useState('')
  const [loginErr, setLoginErr] = useState('')
  const [page, setPage] = useState(() => {
    const h = location.hash.replace('#', '')
    const validPages = NAV_ITEMS.map(n => n.id)
    return validPages.includes(h) ? h : 'tts'
  })
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const [healthOk, setHealthOk] = useState(null) // null = unknown

  const auth = useMemo(() => ({
    Authorization: `Bearer ${token}`,
    'Content-Type': 'application/json'
  }), [token])

  async function verify(nextToken) {
    setLoginErr('')
    try {
      await api('/api/admin/status', {
        headers: { Authorization: `Bearer ${nextToken}`, 'Content-Type': 'application/json' }
      })
      localStorage.setItem('adminToken', nextToken)
      setToken(nextToken)
      setAuthed(true)
    } catch (e) {
      localStorage.removeItem('adminToken')
      setAuthed(false)
      setLoginErr('登录失败：' + e.message)
    } finally {
      setChecking(false)
    }
  }

  // Check health on load
  useEffect(() => {
    let cancelled = false
    async function checkHealth() {
      try {
        await api('/api/health')
        if (!cancelled) setHealthOk(true)
      } catch {
        if (!cancelled) setHealthOk(false)
      }
    }
    if (authed) checkHealth()
    return () => { cancelled = true }
  }, [authed])

  useEffect(() => {
    const saved = getToken()
    if (saved) verify(saved)
    else setChecking(false)
  }, [])

  // Sync hash navigation
  useEffect(() => {
    const onHash = () => {
      const h = location.hash.replace('#', '')
      const validPages = NAV_ITEMS.map(n => n.id)
      if (validPages.includes(h)) setPage(h)
    }
    window.addEventListener('hashchange', onHash)
    return () => window.removeEventListener('hashchange', onHash)
  }, [])

  function nav(p) {
    setPage(p)
    location.hash = p === 'tts' ? '' : p
    setSidebarOpen(false)
  }

  function logout() {
    localStorage.removeItem('adminToken')
    setAuthed(false)
    setToken('')
    location.hash = ''
  }

  // ---- Loading screen ----
  if (checking) {
    return (
      <div className="login-page">
        <div className="login-card" style={{ textAlign: 'center' }}>
          <div className="flex items-center gap-2" style={{ justifyContent: 'center', marginBottom: 12 }}>
            <Waves size={28} style={{ color: 'var(--primary)' }} />
            <span style={{ fontSize: 22, fontWeight: 700 }}>MimoTTS</span>
          </div>
          <p className="text-muted">正在校验管理员 Token...</p>
        </div>
      </div>
    )
  }

  // ---- Login screen ----
  if (!authed) {
    return (
      <div className="login-page">
        <div className="login-card">
          <div className="flex items-center gap-2 mb-4">
            <Waves size={28} style={{ color: 'var(--primary)' }} />
            <div>
              <h1 style={{ margin: 0, fontSize: 22 }}>MimoTTS</h1>
              <p className="text-muted text-sm" style={{ marginTop: 2 }}>语音合成管理控制台</p>
            </div>
          </div>
          <p className="text-muted text-sm mb-4">输入管理员 Token 登录后台</p>
          <input
            value={loginToken}
            onChange={e => setLoginToken(e.target.value)}
            placeholder="请输入 ADMIN_TOKEN"
            type="password"
            onKeyDown={e => e.key === 'Enter' && verify(loginToken)}
          />
          <button className="primary" onClick={() => verify(loginToken)}>登录</button>
          {loginErr && <p className="error mt-4" style={{ textAlign: 'center' }}>{loginErr}</p>}
        </div>
      </div>
    )
  }

  // ---- Main App Shell ----
  const groupedSections = [...new Set(NAV_ITEMS.map(n => n.section))]

  return (
    <div className="app-shell">
      {/* Sidebar */}
      <aside className={`sidebar ${sidebarOpen ? 'open' : ''}`}>
        <div className="sidebar-brand">
          <Waves size={22} />
          <span>MimoTTS</span>
        </div>
        <nav className="sidebar-nav">
          {groupedSections.map(section => (
            <React.Fragment key={section}>
              <div className="sidebar-section">{section}</div>
              {NAV_ITEMS.filter(n => n.section === section).map(item => {
                const Icon = item.icon
                return (
                  <button
                    key={item.id}
                    className={page === item.id ? 'active' : ''}
                    onClick={() => nav(item.id)}
                  >
                    <Icon size={16} />
                    {item.label}
                  </button>
                )
              })}
            </React.Fragment>
          ))}
        </nav>
        <div className="sidebar-footer">
          <button onClick={logout}>
            <LogOut size={16} />
            退出登录
          </button>
        </div>
      </aside>

      {/* Main Area */}
      <div className="main-area">
        {/* Topbar */}
        <header className="topbar">
          <div className="flex items-center gap-2">
            <button
              className="smallbtn"
              onClick={() => setSidebarOpen(!sidebarOpen)}
              style={{ display: 'none' }}
            >
              {sidebarOpen ? <X size={18} /> : <Menu size={18} />}
            </button>
            <span className="topbar-title">{PAGE_TITLES[page] || 'MimoTTS'}</span>
          </div>
          <div className="topbar-right">
            <div className="topbar-status">
              <span className={`status-dot ${healthOk === false ? 'offline' : ''}`} />
              {healthOk === false ? '服务异常' : '服务正常'}
            </div>
          </div>
        </header>

        {/* Content */}
        <main className="content">
          {page === 'tts' && <Home auth={auth} />}
          {page !== 'tts' && <Admin page={page} auth={auth} />}
        </main>
      </div>
    </div>
  )
}

createRoot(document.getElementById('root')).render(<App />)
