// Config ligero para chat-ui.spec.js: sirve su propio mock, sin backend real.
const { defineConfig } = require('@playwright/test');
const fs = require('fs');

// On the rb we use the system Chromium, the only browser installed there. In CI
// it isn't there, so Playwright falls back to the one from `playwright install`.
const CHROMIUM = fs.existsSync('/usr/bin/chromium') ? '/usr/bin/chromium' : undefined;

module.exports = defineConfig({
  testDir: '.',
  testMatch: 'chat-ui.spec.js',
  timeout: 30000,
  workers: 1,
  use: { headless: true, launchOptions: { executablePath: CHROMIUM } },
});
