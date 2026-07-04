import { CalendarDays, CircleDollarSign, Scale } from 'lucide-react'
import { FormEvent, useEffect, useMemo, useState, useSyncExternalStore } from 'react'

type Route = '/important-dates' | '/transactions' | '/decisions'
type AuthStatus = 'checking' | 'anonymous' | 'authenticated'
type Device = {
  id: string
  device_name: string
  last_seen_ip: string
  expires_at: string
  revoked_at: string | null
  current: boolean
}
type ImportantDate = {
  id: string
  title: string
  date: string
  date_type: string
  repeat_rule: string
  note: string
  tags: string[]
}

const routes: Array<{
  path: Route
  label: string
  icon: typeof CalendarDays
  title: string
  summary: string
  rows: string[]
}> = [
  {
    path: '/important-dates',
    label: '日期',
    icon: CalendarDays,
    title: '重要日期',
    summary: '管理生日、纪念日、证件到期和缴费日。',
    rows: ['护照到期 · 2026-12-01 · 不重复', '家庭纪念日 · 2026-08-19 · 每年']
  },
  {
    path: '/transactions',
    label: '账单',
    icon: CircleDollarSign,
    title: '账单流水',
    summary: '记录收入支出，查看预算和基础统计。',
    rows: ['支出 · 餐饮 · 25.50', '收入 · 工资 · 10000.00']
  },
  {
    path: '/decisions',
    label: '决策',
    icon: Scale,
    title: '决策记录',
    summary: '保留关键决策的背景、候选方案和复盘。',
    rows: ['进行中 · 是否搬家', '待复盘 · 学习计划调整']
  }
]

export function App() {
  const currentPath = usePathname()
  const [authStatus, setAuthStatus] = useState<AuthStatus>('checking')
  const [csrfToken, setCsrfToken] = useState('')
  const [loginError, setLoginError] = useState('')
  const [devices, setDevices] = useState<Device[]>([])
  const [importantDates, setImportantDates] = useState<ImportantDate[]>([])
  const [dateError, setDateError] = useState('')
  const current = useMemo(() => {
    return routes.find((route) => route.path === currentPath) ?? routes[0]
  }, [currentPath])

  useEffect(() => {
    void refreshSession()
  }, [])

  async function refreshSession() {
    const response = await fetch('/api/session')
    if (response.status === 401) {
      setAuthStatus('anonymous')
      setCsrfToken('')
      return
    }
    if (!response.ok) {
      setAuthStatus('anonymous')
      return
    }
    const payload = (await response.json()) as { csrf_token: string }
    setCsrfToken(payload.csrf_token)
    setAuthStatus('authenticated')
    void loadImportantDates()
  }

  async function login(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setLoginError('')
    const form = new FormData(event.currentTarget)
    const response = await fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        username: form.get('username'),
        password: form.get('password'),
        device_name: navigator.userAgent.includes('Mobile') ? 'Mobile browser' : 'Desktop browser'
      })
    })
    if (!response.ok) {
      setLoginError('登录失败')
      return
    }
    const payload = (await response.json()) as { csrf_token: string }
    setCsrfToken(payload.csrf_token)
    setAuthStatus('authenticated')
    await loadImportantDates()
  }

  async function logout() {
    const response = await fetch('/api/auth/logout', {
      method: 'POST',
      headers: { 'X-CSRF-Token': csrfToken }
    })
    if (response.ok || response.status === 401) {
      setAuthStatus('anonymous')
      setCsrfToken('')
      setDevices([])
    }
  }

  async function loadDevices() {
    const response = await fetch('/api/devices')
    if (!response.ok) {
      return
    }
    const payload = (await response.json()) as { items: Device[] }
    setDevices(payload.items)
  }

  async function loadImportantDates() {
    const response = await fetch('/api/important-dates')
    if (!response.ok) {
      return
    }
    const payload = (await response.json()) as { items: ImportantDate[] }
    setImportantDates(payload.items ?? [])
  }

  async function createImportantDate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const formElement = event.currentTarget
    setDateError('')
    const form = new FormData(formElement)
    const tags = String(form.get('tags') ?? '')
      .split(',')
      .map((tag) => tag.trim())
      .filter(Boolean)
    const response = await fetch('/api/important-dates', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken
      },
      body: JSON.stringify({
        title: form.get('title'),
        date: form.get('date'),
        date_type: form.get('date_type'),
        repeat_rule: form.get('repeat_rule'),
        note: form.get('note'),
        tags
      })
    })
    if (!response.ok) {
      setDateError('保存失败')
      return
    }
    formElement.reset()
    await loadImportantDates()
  }

  async function deleteImportantDate(id: string) {
    const response = await fetch(`/api/important-dates/${id}`, {
      method: 'DELETE',
      headers: { 'X-CSRF-Token': csrfToken }
    })
    if (response.ok) {
      await loadImportantDates()
    }
  }

  if (authStatus === 'checking') {
    return <main className="login-screen" aria-label="加载中" />
  }

  if (authStatus === 'anonymous') {
    return (
      <main className="login-screen">
        <form className="login-form" onSubmit={login}>
          <div className="brand">life-ledger</div>
          <label>
            用户名
            <input name="username" autoComplete="username" />
          </label>
          <label>
            密码
            <input name="password" type="password" autoComplete="current-password" />
          </label>
          {loginError ? <p className="error">{loginError}</p> : null}
          <button type="submit">登录</button>
        </form>
      </main>
    )
  }

  return (
    <main className="shell">
      <aside className="sidebar" aria-label="一级导航">
        <div className="brand">life-ledger</div>
        <nav className="nav">
          {routes.map((route) => {
            const Icon = route.icon
            const active = route.path === current.path
            return (
              <a
                key={route.path}
                className={active ? 'nav-item active' : 'nav-item'}
                href={route.path}
                aria-current={active ? 'page' : undefined}
                onClick={(event) => {
                  event.preventDefault()
                  window.history.pushState(null, '', route.path)
                  window.dispatchEvent(new PopStateEvent('popstate'))
                }}
              >
                <Icon size={18} />
                <span>{route.label}</span>
              </a>
            )
          })}
        </nav>
      </aside>
      <section className="content">
        <header className="page-header">
          <div>
            <h1>{current.title}</h1>
            <p>{current.summary}</p>
          </div>
          <div className="actions">
            <button type="button" onClick={loadDevices}>
              设备
            </button>
            <button type="button" onClick={logout}>
              退出
            </button>
            <button type="button">新增</button>
          </div>
        </header>
        {devices.length > 0 ? (
          <section className="devices" aria-label="登录设备">
            {devices.map((device) => (
              <article className="device" key={device.id}>
                <strong>{device.device_name}</strong>
                <span>{device.current ? '当前设备' : device.revoked_at ? '已撤销' : '有效设备'}</span>
                <small>{device.last_seen_ip}</small>
              </article>
            ))}
          </section>
        ) : null}
        <div className="toolbar">
          <input aria-label="搜索" placeholder="搜索" />
          <button type="button">筛选</button>
        </div>
        {current.path === '/important-dates' ? (
          <ImportantDatesPage
            items={importantDates}
            error={dateError}
            onCreate={createImportantDate}
            onDelete={deleteImportantDate}
          />
        ) : (
          <section className="list" aria-label={`${current.title}列表`}>
            {current.rows.map((row) => (
              <article className="item" key={row}>
                <span>{row}</span>
                <button type="button">编辑</button>
              </article>
            ))}
          </section>
        )}
      </section>
    </main>
  )
}

function ImportantDatesPage({
  items,
  error,
  onCreate,
  onDelete
}: {
  items: ImportantDate[]
  error: string
  onCreate: (event: FormEvent<HTMLFormElement>) => void
  onDelete: (id: string) => void
}) {
  return (
    <section className="date-layout">
      <form className="date-form" onSubmit={onCreate}>
        <label>
          标题
          <input name="title" />
        </label>
        <label>
          日期
          <input name="date" type="date" />
        </label>
        <label>
          类型
          <input name="date_type" />
        </label>
        <label>
          重复
          <select name="repeat_rule" defaultValue="不重复">
            <option>不重复</option>
            <option>每年</option>
            <option>每月</option>
            <option>每周</option>
          </select>
        </label>
        <label>
          标签
          <input name="tags" />
        </label>
        <label>
          备注
          <input name="note" />
        </label>
        {error ? <p className="error">{error}</p> : null}
        <button type="submit">保存日期</button>
      </form>
      <section className="list" aria-label="重要日期列表">
        {items.length === 0 ? <p className="empty">暂无记录</p> : null}
        {items.map((item) => (
          <article className="item" key={item.id}>
            <span>
              {item.title} · {item.date} · {item.repeat_rule}
              {item.tags.length > 0 ? ` · ${item.tags.join(', ')}` : ''}
            </span>
            <button type="button" onClick={() => onDelete(item.id)}>
              删除
            </button>
          </article>
        ))}
      </section>
    </section>
  )
}

function usePathname(): string {
  return useSyncExternalStore(
    (onStoreChange) => {
      window.addEventListener('popstate', onStoreChange)
      return () => window.removeEventListener('popstate', onStoreChange)
    },
    () => window.location.pathname,
    () => '/important-dates'
  )
}
