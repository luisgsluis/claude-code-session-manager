const { defineConfig } = require('@playwright/test');
const path = require('path');

const PORT = 8799;
// Second server (auth.spec.js): no LAN bypass and a user with 2FA, so the
// login form is actually reachable. The main one bypasses login on purpose.
const AUTH_PORT = 8798;

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
  webServer: [
    {
      command: 'bash e2e/run-e2e.sh',
      cwd: path.join(__dirname, '..'),
      url: `http://127.0.0.1:${PORT}/api/health`,
      reuseExistingServer: false,
      timeout: 90000,
      env: {
        CCSM_TMUX_STATE: path.join(__dirname, 'e2e', 'state', 'tmux'),
      },
    },
    {
      command: 'bash e2e/run-e2e-auth.sh',
      cwd: path.join(__dirname, '..'),
      url: `http://127.0.0.1:${AUTH_PORT}/api/health`,
      reuseExistingServer: false,
      timeout: 90000,
    },
  ],
});
