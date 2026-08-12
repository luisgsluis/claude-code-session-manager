const { defineConfig } = require('@playwright/test');
const path = require('path');

const PORT = 8799;

module.exports = defineConfig({
  testDir: '.',
  testMatch: '**/*.spec.js',
  timeout: 30000,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  use: {
    baseURL: `http://127.0.0.1:${PORT}`,
    headless: true,
  },
  webServer: {
    command: 'bash e2e/run-e2e.sh',
    cwd: path.join(__dirname, '..'),
    url: `http://127.0.0.1:${PORT}/api/health`,
    reuseExistingServer: false,
    timeout: 90000,
    env: {
      CCSM_TMUX_STATE: path.join(__dirname, 'e2e', 'state', 'tmux'),
    },
  },
});
