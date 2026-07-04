import { expect, test } from '@playwright/test'

async function login(page: import('@playwright/test').Page, heading = '重要日期') {
  await page.getByLabel('用户名').fill('admin')
  await page.getByLabel('密码').fill('password')
  await page.getByRole('button', { name: '登录' }).click()
  await expect(page.getByRole('heading', { name: heading })).toBeVisible()
}

test('unauthenticated root shows login after redirect', async ({ page }) => {
  await page.goto('/')
  await expect(page).toHaveURL('/important-dates')
  await expect(page.getByRole('button', { name: '登录' })).toBeVisible()
})

test('three primary pages survive direct refresh', async ({ page }) => {
  await page.goto('/important-dates')
  await login(page)
  for (const [path, heading, navLabel] of [
    ['/important-dates', '重要日期', '日期'],
    ['/transactions', '账单流水', '账单'],
    ['/decisions', '决策记录', '决策']
  ] as const) {
    await page.goto(path)
    await page.reload()
    await expect(page.getByRole('heading', { name: heading })).toBeVisible()
    await expect(page.getByRole('link', { name: navLabel })).toHaveAttribute('aria-current', 'page')
  }
})

test('unknown api returns json 404', async ({ request }) => {
  const loginResponse = await request.post('/api/auth/login', {
    data: {
      username: 'admin',
      password: 'password',
      device_name: 'API test'
    }
  })
  expect(loginResponse.status()).toBe(200)
  const response = await request.get('/api/not-found')
  expect(response.status()).toBe(404)
  expect(response.headers()['content-type']).toContain('application/json')
  expect(response.headers()['access-control-allow-origin']).not.toBe('*')
})

test('unknown non-api route falls back to spa', async ({ page }) => {
  await page.goto('/somewhere')
  await expect(page.getByText('life-ledger')).toBeVisible()
})

test('login session can list devices and logout', async ({ page }) => {
  await page.goto('/important-dates')
  await login(page)
  await page.getByRole('button', { name: '设备' }).click()
  await expect(page.getByLabel('登录设备')).toContainText('当前设备')
  await page.getByRole('button', { name: '退出' }).click()
  await expect(page.getByRole('button', { name: '登录' })).toBeVisible()
})

test('important dates can be created and deleted', async ({ page }) => {
  await page.goto('/important-dates')
  await login(page)
  await page.getByLabel('标题').fill('护照到期自动化')
  await page.getByLabel('日期', { exact: true }).fill('2026-12-01')
  await page.getByLabel('类型').fill('证件')
  await page.getByLabel('标签').fill('证件,家庭')
  const createResponse = page.waitForResponse(
    (response) =>
      response.url().endsWith('/api/important-dates') && response.request().method() === 'POST'
  )
  await page.getByRole('button', { name: '保存日期' }).click()
  expect((await createResponse).status()).toBe(201)
  await expect(page.getByLabel('重要日期列表')).toContainText('护照到期自动化')
  await expect(page.getByLabel('重要日期列表')).toContainText('证件')
  await page.getByRole('button', { name: '删除' }).click()
  await expect(page.getByLabel('重要日期列表')).not.toContainText('护照到期自动化')
})

test('transactions update summary and budget usage', async ({ page }) => {
  await page.goto('/transactions')
  await login(page, '账单流水')
  await expect(page.getByRole('button', { name: '下载模板' })).toBeVisible()
  await expect(page.getByRole('button', { name: '导出账单' })).toBeVisible()
  await expect(page.getByRole('button', { name: '导入账单' })).toBeVisible()
  const billForm = page.getByLabel('账单表单')
  await billForm.getByLabel('日期', { exact: true }).fill('2026-07-04')
  await billForm.getByLabel('时间').fill('08:30')
  await billForm.getByLabel('金额').fill('25.50')
  await billForm.getByLabel('分类').fill('餐饮')
  await billForm.getByLabel('账户').fill('现金')
  await billForm.getByLabel('标签').fill('日常')
  await page.getByRole('button', { name: '保存账单' }).click()
  await expect(page.getByLabel('账单列表')).toContainText('餐饮')
  await expect(page.getByLabel('账单统计')).toContainText('支出 25.50')

  const budgetForm = page.getByLabel('预算表单')
  await budgetForm.getByLabel('月份').fill('2026-07')
  await budgetForm.getByLabel('分类').fill('餐饮')
  await budgetForm.getByLabel('预算').fill('100.00')
  const budgetResponse = page.waitForResponse(
    (response) => response.url().endsWith('/api/budgets') && response.request().method() === 'POST'
  )
  await page.getByRole('button', { name: '保存预算' }).click()
  expect((await budgetResponse).status()).toBe(200)
  await expect(page.getByLabel('预算列表')).toContainText('已用 25.50 / 100.00')

  const deleteTransactionResponse = page.waitForResponse(
    (response) =>
      response.url().includes('/api/transactions/') && response.request().method() === 'DELETE'
  )
  await page.getByLabel('账单列表').getByRole('button', { name: '删除' }).click()
  expect((await deleteTransactionResponse).status()).toBe(200)
  await expect(page.getByLabel('账单列表')).not.toContainText('餐饮')
})

test('decisions can be created and deleted', async ({ page }) => {
  await page.goto('/decisions')
  await login(page, '决策记录')
  const decisionForm = page.getByLabel('决策表单')
  await decisionForm.getByLabel('标题').fill('是否搬家自动化')
  await decisionForm.getByLabel('状态').selectOption('进行中')
  await decisionForm.getByLabel('复盘日期').fill('2020-01-01')
  await decisionForm.getByLabel('背景').fill('通勤时间过长')
  await decisionForm.getByLabel('最终选择').fill('搬近公司')
  await decisionForm.getByLabel('方案名称').fill('搬近公司')
  await decisionForm.getByLabel('优点').fill('节省通勤')
  await decisionForm.getByLabel('缺点').fill('租金更高')
  await decisionForm.getByLabel('标签').fill('生活')
  const createResponse = page.waitForResponse(
    (response) => response.url().endsWith('/api/decisions') && response.request().method() === 'POST'
  )
  await page.getByRole('button', { name: '保存决策' }).click()
  expect((await createResponse).status()).toBe(201)
  await expect(page.getByLabel('决策列表')).toContainText('是否搬家自动化')
  await expect(page.getByLabel('待复盘决策')).toContainText('是否搬家自动化')
  await expect(page.getByLabel('决策列表')).toContainText('搬近公司')
  await expect(page.getByLabel('决策列表')).toContainText('生活')
  const archiveResponse = page.waitForResponse(
    (response) => response.url().includes('/api/decisions/') && response.request().method() === 'PUT'
  )
  await page.getByLabel('待复盘决策').getByRole('button', { name: '复盘归档' }).click()
  expect((await archiveResponse).status()).toBe(200)
  await expect(page.getByLabel('已归档决策')).toContainText('是否搬家自动化')
  const deleteDecisionResponse = page.waitForResponse(
    (response) => response.url().includes('/api/decisions/') && response.request().method() === 'DELETE'
  )
  await page.getByLabel('已归档决策').getByRole('button', { name: '删除' }).click()
  expect((await deleteDecisionResponse).status()).toBe(200)
  await expect(page.getByLabel('决策列表')).not.toContainText('是否搬家自动化')
})
