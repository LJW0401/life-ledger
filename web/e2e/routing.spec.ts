import { expect, test } from '@playwright/test'

test('root redirects to the important dates page', async ({ page }) => {
  await page.goto('/')
  await expect(page).toHaveURL('/important-dates')
  await expect(page.getByRole('heading', { name: '重要日期' })).toBeVisible()
})

test('three primary pages survive direct refresh', async ({ page }) => {
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
  const response = await request.get('/api/not-found')
  expect(response.status()).toBe(404)
  expect(response.headers()['content-type']).toContain('application/json')
  expect(response.headers()['access-control-allow-origin']).not.toBe('*')
})

test('unknown non-api route falls back to spa', async ({ page }) => {
  await page.goto('/somewhere')
  await expect(page.getByText('life-ledger')).toBeVisible()
})

test.skip('protected routes show login after auth implementation', async () => {
  // Stage 2 introduces authentication. This pending test prevents pretending
  // that protected route behavior is already implemented in the stage 1 shell.
})
