import { CalendarDays, CircleDollarSign, Scale } from 'lucide-react'
import { FormEvent, useEffect, useMemo, useRef, useState, useSyncExternalStore } from 'react'

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
type Transaction = {
  id: string
  date: string
  time: string
  type: string
  amount: string
  category: string
  include_income: boolean
  include_budget: boolean
  ledger: string
  account: string
  tags: string[]
}
type Summary = {
  income: string
  expense: string
  balance: string
}
type Budget = {
  id: string
  month: string
  category: string
  amount: string
  used: string
  remaining: string
  overspent: boolean
}
type Decision = {
  id: string
  title: string
  background: string
  final_choice: string
  status: string
  review_date: string
  review_note: string
  tags: string[]
  options: Array<{ name: string; pros: string; cons: string; note: string }>
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
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [summary, setSummary] = useState<Summary>({ income: '0.00', expense: '0.00', balance: '0.00' })
  const [budgets, setBudgets] = useState<Budget[]>([])
  const [transactionError, setTransactionError] = useState('')
  const transactionLoadID = useRef(0)
  const [decisions, setDecisions] = useState<Decision[]>([])
  const [decisionError, setDecisionError] = useState('')
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
    void loadTransactions()
    void loadDecisions()
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
    await loadTransactions()
    await loadDecisions()
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

  async function loadTransactions() {
    const loadID = transactionLoadID.current + 1
    transactionLoadID.current = loadID
    const [transactionsResponse, summaryResponse, budgetsResponse] = await Promise.all([
      fetch('/api/transactions'),
      fetch('/api/transactions/summary'),
      fetch('/api/budgets')
    ])
    let nextTransactions: Transaction[] | null = null
    let nextSummary: Summary | null = null
    let nextBudgets: Budget[] | null = null
    if (transactionsResponse.ok) {
      const payload = (await transactionsResponse.json()) as { items: Transaction[] }
      nextTransactions = payload.items ?? []
    }
    if (summaryResponse.ok) {
      nextSummary = (await summaryResponse.json()) as Summary
    }
    if (budgetsResponse.ok) {
      const payload = (await budgetsResponse.json()) as { items: Budget[] }
      nextBudgets = payload.items ?? []
    }
    if (loadID !== transactionLoadID.current) {
      return
    }
    if (nextTransactions !== null) {
      setTransactions(nextTransactions)
    }
    if (nextSummary !== null) {
      setSummary(nextSummary)
    }
    if (nextBudgets !== null) {
      setBudgets(nextBudgets)
    }
  }

  async function createTransaction(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const formElement = event.currentTarget
    const form = new FormData(formElement)
    setTransactionError('')
    const tags = String(form.get('tags') ?? '')
      .split(',')
      .map((tag) => tag.trim())
      .filter(Boolean)
    const response = await fetch('/api/transactions', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken
      },
      body: JSON.stringify({
        date: form.get('date'),
        time: form.get('time'),
        type: form.get('type'),
        amount: form.get('amount'),
        category: form.get('category'),
        include_income: form.get('include_income') === 'on',
        include_budget: form.get('include_budget') === 'on',
        ledger: form.get('ledger'),
        account: form.get('account'),
        tags
      })
    })
    if (!response.ok) {
      setTransactionError('保存失败')
      return
    }
    formElement.reset()
    await loadTransactions()
  }

  async function deleteTransaction(id: string) {
    const response = await fetch(`/api/transactions/${id}`, {
      method: 'DELETE',
      headers: { 'X-CSRF-Token': csrfToken }
    })
    if (response.ok) {
      await loadTransactions()
    }
  }

  async function saveBudget(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const formElement = event.currentTarget
    const form = new FormData(formElement)
    const response = await fetch('/api/budgets', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken
      },
      body: JSON.stringify({
        month: form.get('month'),
        category: form.get('category'),
        amount: form.get('amount')
      })
    })
    if (response.ok) {
      formElement.reset()
      await loadTransactions()
    }
  }

  async function downloadExcel(path: string, filename: string) {
    const response = await fetch(path)
    if (!response.ok) {
      setTransactionError('下载失败')
      return
    }
    const blob = await response.blob()
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = filename
    anchor.click()
    URL.revokeObjectURL(url)
  }

  async function importTransactions(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const formElement = event.currentTarget
    const fileInput = formElement.elements.namedItem('file') as HTMLInputElement | null
    const file = fileInput?.files?.[0]
    if (!file) {
      setTransactionError('请选择文件')
      return
    }
    const form = new FormData()
    form.append('file', file)
    const response = await fetch('/api/transactions/import.xlsx', {
      method: 'POST',
      headers: { 'X-CSRF-Token': csrfToken },
      body: form
    })
    if (!response.ok) {
      setTransactionError('导入失败')
      return
    }
    formElement.reset()
    await loadTransactions()
  }

  async function loadDecisions() {
    const response = await fetch('/api/decisions')
    if (!response.ok) {
      return
    }
    const payload = (await response.json()) as { items: Decision[] }
    setDecisions(payload.items ?? [])
  }

  async function createDecision(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const formElement = event.currentTarget
    const form = new FormData(formElement)
    setDecisionError('')
    const tags = String(form.get('tags') ?? '')
      .split(',')
      .map((tag) => tag.trim())
      .filter(Boolean)
    const response = await fetch('/api/decisions', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken
      },
      body: JSON.stringify({
        title: form.get('title'),
        background: form.get('background'),
        final_choice: form.get('final_choice'),
        status: form.get('status'),
        review_date: form.get('review_date'),
        review_note: form.get('review_note'),
        options: [
          {
            name: form.get('option_name'),
            pros: form.get('option_pros'),
            cons: form.get('option_cons'),
            note: ''
          }
        ].filter((option) => String(option.name ?? '').trim() !== ''),
        tags
      })
    })
    if (!response.ok) {
      setDecisionError('保存失败')
      return
    }
    formElement.reset()
    await loadDecisions()
  }

  async function archiveDecision(item: Decision) {
    setDecisionError('')
    const response = await fetch(`/api/decisions/${item.id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken
      },
      body: JSON.stringify({
        title: item.title,
        background: item.background,
        final_choice: item.final_choice,
        status: '已归档',
        review_date: item.review_date,
        review_note: item.review_note || '已完成复盘',
        options: item.options,
        tags: item.tags
      })
    })
    if (!response.ok) {
      setDecisionError('归档失败')
      return
    }
    await loadDecisions()
  }

  async function deleteDecision(id: string) {
    const response = await fetch(`/api/decisions/${id}`, {
      method: 'DELETE',
      headers: { 'X-CSRF-Token': csrfToken }
    })
    if (response.ok) {
      await loadDecisions()
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
        ) : current.path === '/transactions' ? (
          <TransactionsPage
            items={transactions}
            summary={summary}
            budgets={budgets}
            error={transactionError}
            onCreate={createTransaction}
            onDelete={deleteTransaction}
            onSaveBudget={saveBudget}
            onTemplate={() =>
              void downloadExcel('/api/transactions/template.xlsx', 'life-ledger-transactions-template.xlsx')
            }
            onExport={() =>
              void downloadExcel('/api/transactions/export.xlsx', 'life-ledger-transactions.xlsx')
            }
            onImport={importTransactions}
          />
        ) : current.path === '/decisions' ? (
          <DecisionsPage
            items={decisions}
            error={decisionError}
            onCreate={createDecision}
            onArchive={archiveDecision}
            onDelete={deleteDecision}
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

function DecisionsPage({
  items,
  error,
  onCreate,
  onArchive,
  onDelete
}: {
  items: Decision[]
  error: string
  onCreate: (event: FormEvent<HTMLFormElement>) => void
  onArchive: (item: Decision) => void
  onDelete: (id: string) => void
}) {
  const groups = ['进行中', '待复盘', '已归档'].map((status) => ({
    status,
    items: items.filter((item) => item.status === status)
  }))
  return (
    <section className="date-layout">
      <form className="date-form" aria-label="决策表单" onSubmit={onCreate}>
        <label>
          标题
          <input name="title" />
        </label>
        <label>
          状态
          <select name="status" defaultValue="进行中">
            <option>进行中</option>
            <option>待复盘</option>
            <option>已归档</option>
          </select>
        </label>
        <label>
          复盘日期
          <input name="review_date" type="date" />
        </label>
        <label>
          背景
          <input name="background" />
        </label>
        <label>
          最终选择
          <input name="final_choice" />
        </label>
        <label>
          复盘内容
          <input name="review_note" />
        </label>
        <label>
          方案名称
          <input name="option_name" />
        </label>
        <label>
          优点
          <input name="option_pros" />
        </label>
        <label>
          缺点
          <input name="option_cons" />
        </label>
        <label>
          标签
          <input name="tags" />
        </label>
        {error ? <p className="error">{error}</p> : null}
        <button type="submit">保存决策</button>
      </form>
      <section className="list" aria-label="决策列表">
        {items.length === 0 ? <p className="empty">暂无记录</p> : null}
        {groups.map((group) => (
          <section key={group.status} aria-label={`${group.status}决策`}>
            <h3>{group.status}</h3>
            {group.items.map((item) => (
              <article className="item" key={item.id}>
                <span>
                  {item.title}
                  {item.options.length > 0 ? ` · ${item.options[0].name}` : ''}
                  {item.tags.length > 0 ? ` · ${item.tags.join(', ')}` : ''}
                </span>
                {item.status !== '已归档' ? (
                  <button type="button" onClick={() => onArchive(item)}>
                    复盘归档
                  </button>
                ) : null}
                <button type="button" onClick={() => onDelete(item.id)}>
                  删除
                </button>
              </article>
            ))}
          </section>
        ))}
      </section>
    </section>
  )
}

function TransactionsPage({
  items,
  summary,
  budgets,
  error,
  onCreate,
  onDelete,
  onSaveBudget,
  onTemplate,
  onExport,
  onImport
}: {
  items: Transaction[]
  summary: Summary
  budgets: Budget[]
  error: string
  onCreate: (event: FormEvent<HTMLFormElement>) => void
  onDelete: (id: string) => void
  onSaveBudget: (event: FormEvent<HTMLFormElement>) => void
  onTemplate: () => void
  onExport: () => void
  onImport: (event: FormEvent<HTMLFormElement>) => void
}) {
  return (
    <section className="transaction-layout">
      <section className="stats" aria-label="账单统计">
        <strong>收入 {summary.income}</strong>
        <strong>支出 {summary.expense}</strong>
        <strong>余额 {summary.balance}</strong>
      </section>
      <div className="excel-actions">
        <button type="button" onClick={onTemplate}>
          下载模板
        </button>
        <button type="button" onClick={onExport}>
          导出账单
        </button>
        <form aria-label="Excel 导入" onSubmit={onImport}>
          <input name="file" type="file" accept=".xlsx" />
          <button type="submit">导入账单</button>
        </form>
      </div>
      <form className="date-form" aria-label="账单表单" onSubmit={onCreate}>
        <label>
          日期
          <input name="date" type="date" />
        </label>
        <label>
          时间
          <input name="time" type="time" />
        </label>
        <label>
          类型
          <select name="type" defaultValue="支出">
            <option>支出</option>
            <option>收入</option>
          </select>
        </label>
        <label>
          金额
          <input name="amount" inputMode="decimal" />
        </label>
        <label>
          分类
          <input name="category" />
        </label>
        <label>
          所属账本
          <input name="ledger" defaultValue="默认账本" />
        </label>
        <label>
          账户
          <input name="account" />
        </label>
        <label>
          标签
          <input name="tags" />
        </label>
        <label className="check">
          <input name="include_income" type="checkbox" defaultChecked />
          计入收支
        </label>
        <label className="check">
          <input name="include_budget" type="checkbox" defaultChecked />
          计入预算
        </label>
        {error ? <p className="error">{error}</p> : null}
        <button type="submit">保存账单</button>
      </form>
      <form className="budget-form" aria-label="预算表单" onSubmit={onSaveBudget}>
        <label>
          月份
          <input name="month" placeholder="2026-07" />
        </label>
        <label>
          分类
          <input name="category" />
        </label>
        <label>
          预算
          <input name="amount" inputMode="decimal" />
        </label>
        <button type="submit">保存预算</button>
      </form>
      <section className="list" aria-label="预算列表">
        {budgets.map((budget) => (
          <article className="item" key={budget.id}>
            <span>
              {budget.month} · {budget.category} · 已用 {budget.used} / {budget.amount}
              {budget.overspent ? ' · 超支' : ''}
            </span>
          </article>
        ))}
      </section>
      <section className="list" aria-label="账单列表">
        {items.length === 0 ? <p className="empty">暂无记录</p> : null}
        {items.map((item) => (
          <article className="item" key={item.id}>
            <span>
              {item.type} · {item.category} · {item.amount}
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
