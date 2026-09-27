import { defineConfig } from '@playwright/test';

// Runs against a live Lighthouse (docker compose up), at BASE_URL.
export default defineConfig({
  testDir: 'e2e',
  timeout: 120_000,
  retries: 0,
  use: {
    baseURL: process.env.BASE_URL ?? 'http://localhost:8080',
    // A regular browser user agent: headless Chromium's own would be skipped as a bot.
    userAgent: 'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0 Safari/537.36',
    trace: 'retain-on-failure',
  },
});
