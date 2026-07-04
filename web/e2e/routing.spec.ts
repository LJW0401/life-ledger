import { expect, test } from '@playwright/test'

async function login(page: import('@playwright/test').Page) {
  await page.getByLabel('用户名').fill('admin')
  await page.getByLabel('密码').fill('password')
  await page.getByRole('button', { name: '登录' }).click()
  await expect(page.getByRole('heading', { name: '重要日期' })).toBeVisible()
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
