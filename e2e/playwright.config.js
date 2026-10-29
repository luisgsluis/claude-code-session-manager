const { defineConfig } = require('@playwright/test');
const fs = require('fs');
const path = require('path');

const PORT = 8799;
// Second server (auth.spec.js): no LAN bypass and a user with 2FA, so the
// login form is actually reachable. The main one bypasses login on purpose.
const AUTH_PORT = 8798;
// Third server (proxy-auth.spec.js): auth: proxy, loopback as the trusted proxy.
const PROXY_PORT = 8796;

// On the rb we use the system Chromium, the only browser installed there. In CI
// it isn't there, so Playwright falls back to the one from `playwright install`.
const CHROMIUM = fs.existsSync('/usr/bin/chromium') ? '/usr/bin/chromium' : undefined;

module.exports = defineConfig({
  testDir: '.',
  testMatch: '**/*.spec.js',
  timeout: 30000,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  use: {
    baseURL: `http://127.0.0.1:${PORT}`,
    headless: true,
    // A fake audio device plus auto-granted permission is what makes the
    // dictation path testable end to end: MediaRecorder produces a real webm
    // blob, which is really uploaded to the stub provider. Without these the
    // mic tests could only check that a button exists.
    permissions: ['microphone'],
    launchOptions: {
      executablePath: CHROMIUM,
      args: [
        '--use-fake-device-for-media-stream',
        '--use-fake-ui-for-media-stream',
        '--autoplay-policy=no-user-gesture-required',
      ],
    },
  },
  webServer: [
    {
      command: 'bash e2e/run-e2e.sh',
      cwd: path.join(__dirname, '..'),
      url: `http://127.0.0.1:${PORT}/api/health`,
      reuseExistingServer: false,
      timeout: 90000,
      env: {
        // __dirname is already e2e/, so the old path.join(__dirname, 'e2e', …)
        // resolved to e2e/e2e/state/tmux — a directory run-e2e.sh never wiped
        // (it cleans e2e/state). Sessions therefore accumulated across runs and
        // specs that create a named session started failing with "name already
        // in use" once an earlier run had used that name. Keep this in step
        // with STATE in run-e2e.sh.
        CCSM_TMUX_STATE: path.join(__dirname, 'state', 'tmux'),
      },
    },
    {
      // Stub of an OpenAI-compatible provider (Groq/DeepSeek shape). Managed by
      // Playwright so it cannot outlive the run.
      command: 'node e2e/stubs/voice-provider.js',
      cwd: path.join(__dirname, '..'),
      url: 'http://127.0.0.1:8797/health',
      ignoreHTTPSErrors: true,
      reuseExistingServer: false,
      timeout: 30000,
    },
    {
      command: 'bash e2e/run-e2e-auth.sh',
      cwd: path.join(__dirname, '..'),
      url: `http://127.0.0.1:${AUTH_PORT}/api/health`,
      reuseExistingServer: false,
      timeout: 90000,
    },
    {
      command: 'bash e2e/run-e2e-proxy.sh',
      cwd: path.join(__dirname, '..'),
      url: `http://127.0.0.1:${PROXY_PORT}/api/health`,
      reuseExistingServer: false,
      timeout: 90000,
    },
  ],
});
