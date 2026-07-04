import { CalendarDays, CircleDollarSign, Scale } from 'lucide-react'
import { useMemo, useSyncExternalStore } from 'react'

type Route = '/important-dates' | '/transactions' | '/decisions'

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
  const current = useMemo(() => {
    return routes.find((route) => route.path === currentPath) ?? routes[0]
  }, [currentPath])

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
          <button type="button">新增</button>
        </header>
        <div className="toolbar">
          <input aria-label="搜索" placeholder="搜索" />
          <button type="button">筛选</button>
        </div>
        <section className="list" aria-label={`${current.title}列表`}>
          {current.rows.map((row) => (
            <article className="item" key={row}>
              <span>{row}</span>
              <button type="button">编辑</button>
            </article>
          ))}
        </section>
      </section>
    </main>
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
