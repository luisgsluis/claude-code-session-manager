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
    // A fake audio device plus auto-granted permission is what makes the
    // dictation path testable end to end: MediaRecorder produces a real webm
    // blob, which is really uploaded to the stub provider. Without these the
    // mic tests could only check that a button exists.
    permissions: ['microphone'],
    launchOptions: {
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
  ],
});
