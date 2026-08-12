// Config ligero para chat-ui.spec.js: sirve su propio mock, sin backend real.
const { defineConfig } = require('@playwright/test');

module.exports = defineConfig({
  testDir: '.',
  testMatch: 'chat-ui.spec.js',
  timeout: 30000,
  workers: 1,
  use: { headless: true },
});
