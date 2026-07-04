import { defineConfig } from '@playwright/test'
import { fileURLToPath } from 'node:url'
import path from 'node:path'

const configDir = path.dirname(fileURLToPath(import.meta.url))

export default defineConfig({
  testDir: './e2e',
  webServer: {
    command: 'sh scripts/e2e-server.sh',
    cwd: path.resolve(configDir, '..'),
    url: 'http://127.0.0.1:18080/important-dates',
    reuseExistingServer: !process.env.CI,
    timeout: 120_000
  },
  use: {
    baseURL: 'http://127.0.0.1:18080'
  }
})
