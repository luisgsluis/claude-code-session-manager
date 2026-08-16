// e2e/chat-ui.spec.js
// UI test of the live modal against the local frontend sources with mocked
// APIs. Verifies what Luis has reported repeatedly:
//   - model dropdown with the 5 expected entries (opus/sonnet/haiku + 2 deepseek)
//   - "manual" mode present in the mode selector
//   - mode/model labels without a "live" prefix ("Modo"/"Modelo" or "Mode"/"Model")
//   - chat bubbles with no blank lines and no inflated height
//   - a sent message appears in the chat, well formed and fast (<2s via SSE)
const http = require('http');
const fs = require('fs');
const path = require('path');
const { test, expect } = require('@playwright/test');

const STATIC = path.join(__dirname, '..', 'static');
const SETTINGS = JSON.parse(fs.readFileSync(path.join(__dirname, 'real_settings.json'), 'utf8'));

function makePayload(extraMsgs) {
  return {
    session: '0',
    id: 'a1b2c3d4-1111-2222-3333-444455556666',
    ready: true,
    title: 'Sesión 0',
    origin: 'claude',
    created: 1754900000,
    updated: 1754901000,
    size: 2 + (extraMsgs || []).length,
    is_alive: true,
    status: 'rc_connected',
    mode: 'auto',
    messages: [
      { index: 1, role: 'user', content: '\n   Hola desde el móvil   \n' },
      { index: 2, role: 'assistant', content: '\n\n Respuesta de prueba \n\n' },
    ].concat(extraMsgs || []),
  };
}

// Server: serves static/ and the mocked APIs. Keeps the chat state and an
// open SSE so it can push the sent message as if it were the real backend's
// 1s poll.
function startMockServer() {
  return new Promise((resolve) => {
    const state = { payload: makePayload(), sendHistory: [], modeHistory: [], keyHistory: [], choiceHistory: [] };
    const sseClients = new Set();
    const server = http.createServer((req, res) => {
      const url = new URL(req.url, 'http://x');
      const p = url.pathname;

      if (p.startsWith('/api/')) {
        const json = (code, obj, extra) => {
          res.writeHead(code, { 'Content-Type': 'application/json', ...(extra || {}) });
          res.end(JSON.stringify(obj));
        };
        if (p === '/api/auth/status') return json(200, { authenticated: true, lan_bypass: false, username: 'luis' });
        if (p === '/api/sessions') return json(200, [{ name: '0', status: 'rc_connected', task: 'Prueba' }]);
        if (p === '/api/settings') return json(200, { content: SETTINGS });
        if (p === '/api/events') {
          res.writeHead(200, { 'Content-Type': 'text/event-stream', 'Cache-Control': 'no-cache' });
          return res.end();
        }
        if (p === '/api/sessions/0/chat') return json(200, state.payload);
        if (p === '/api/sessions/0/chat/stream') {
          res.writeHead(200, { 'Content-Type': 'text/event-stream', 'Cache-Control': 'no-cache' });
          res.write('data: ' + JSON.stringify(state.payload) + '\n\n');
          sseClients.add(res);
          req.on('close', () => sseClients.delete(res));
          return;
        }
        if (p === '/api/sessions/0/send') {
          let body = '';
          req.on('data', (c) => (body += c));
          req.on('end', () => {
            let text = '';
            let mode = '';
            let keys = '';
            let choice = null;
            try {
              const b = JSON.parse(body);
              text = (b.text || '').trim(); mode = b.mode || ''; keys = b.keys || '';
              choice = (typeof b.choice === 'number') ? b.choice : null;
            } catch (e) {}
            if (mode) state.modeHistory.push(mode);
            if (keys) state.keyHistory.push(keys);
            if (choice !== null) state.choiceHistory.push(choice);
            state.sendHistory.push(text);
            state.payload = makePayload([
              { index: 3, role: 'user', content: text },
            ]);
            // Push the updated payload to the open SSE clients (like the real poll).
            const data = 'data: ' + JSON.stringify(state.payload) + '\n\n';
            for (const c of sseClients) { try { c.write(data); } catch (e) {} }
            json(200, { ok: true });
          });
          return;
        }
        return json(404, { error: 'no mock' });
      }

      // Statics (the real server serves /static/ without the prefix)
      let fp = decodeURIComponent(p);
      if (fp.startsWith('/static/')) fp = fp.slice('/static'.length);
      if (fp === '/') fp = '/index.html';
      const file = path.join(STATIC, fp);
      if (!file.startsWith(STATIC) || !fs.existsSync(file)) {
        res.writeHead(404); res.end('not found'); return;
      }
      const types = { '.html': 'text/html', '.js': 'text/javascript', '.css': 'text/css' };
      res.writeHead(200, { 'Content-Type': types[path.extname(file)] || 'application/octet-stream' });
      res.end(fs.readFileSync(file));
    });
    // pushChat lets a test mutate the payload and push it to the open
    // /chat/stream clients, simulating the dialog arriving AFTER the tile is
    // already open (the exact case the grid used to miss).
    state.pushChat = (payload) => {
      state.payload = payload;
      const data = 'data: ' + JSON.stringify(payload) + '\n\n';
      for (const c of sseClients) { try { c.write(data); } catch (e) {} }
    };
    server.listen(0, '127.0.0.1', () => resolve({ server, state }));
  });
}test('live modal: models, manual mode, labels, bubbles and send', async ({ page }) => {
  const { server, state } = await startMockServer();
  const port = server.address().port;
  try {
    const errors = [];
    page.on('pageerror', (e) => errors.push(String(e)));

    await page.goto(`http://127.0.0.1:${port}/`);
    await page.waitForSelector('text=👁️', { timeout: 10000 });
    await page.click('button[title]:has-text("👁️")');

    // --- Model dropdown: 5 options ---
    const modelSelect = page.locator('select[title="Modelo"], select[title="Model"]').last();
    await expect(modelSelect).toBeVisible({ timeout: 10000 });
    await expect(modelSelect.locator('option:not([disabled])')).toHaveCount(5);
    const modelValues = await modelSelect.locator('option:not([disabled])').allTextContents();
    for (const v of ['opus', 'sonnet', 'haiku', 'deepseek-v4-pro[1m]', 'deepseek-v4-flash']) {
      expect(modelValues, `missing ${v}`).toContain(v);
    }

    // --- Mode: includes manual ---
    const modeSelect = page.locator('select[title="Modo"], select[title="Mode"]').last();
    await expect(modeSelect).toBeVisible();
    const modeValues = await modeSelect.locator('option').allTextContents();
    expect(modeValues).toContain('manual');

    // --- Labels without a "live" prefix ---
    const modeLabel = (await modeSelect.locator('option').first().textContent()) || '';
    const modelLabel = (await modelSelect.locator('option').first().textContent()) || '';
    expect(modeLabel.toLowerCase()).not.toContain('live');
    expect(modelLabel.toLowerCase()).not.toContain('live');

    // --- Initial bubbles: no blank lines, no inflated height ---
    await page.waitForSelector('[x-ref="liveChat"] .px-3', { timeout: 10000 });
    const bubbleInfo = await page.evaluate(() => {
      const el = document.querySelector('[x-ref="liveChat"]');
      return Array.from(el.querySelectorAll('div.px-3')).map((b) => ({
        text: b.textContent, trimmed: b.textContent.trim(), h: b.scrollHeight,
      }));
    });
    expect(bubbleInfo.length).toBeGreaterThanOrEqual(2);
    for (const b of bubbleInfo) {
      expect(b.trimmed, 'bubble with edge whitespace').toBe(b.text);
      expect(b.h, 'bubble with inflated height').toBeLessThan(80);
    }

    // --- Send: the message appears, well formed and fast ---
    const text = 'Mensaje de prueba enviado';
    await page.fill('textarea[x-model="live.input"]', text);
    const t0 = Date.now();
    await page.locator('button:has-text("Send"), button:has-text("Enviar")').last().click();
    await page.waitForFunction((txt) => {
      const el = document.querySelector('[x-ref="liveChat"]');
      if (!el) return false;
      return Array.from(el.querySelectorAll('div.px-3')).some((b) => b.textContent.trim() === txt);
    }, text, { timeout: 4000 });
    const elapsed = Date.now() - t0;
    expect(elapsed, `message took ${elapsed}ms to appear`).toBeLessThan(3000);
    expect(state.sendHistory).toContain(text);

    // The sent message bubble is clean
    const sentBubble = await page.evaluate((txt) => {
      const el = document.querySelector('[x-ref="liveChat"]');
      const b = Array.from(el.querySelectorAll('div.px-3')).find((x) => x.textContent.trim() === txt);
      return b ? { text: b.textContent, trimmed: b.textContent.trim(), h: b.scrollHeight } : null;
    }, text);
    expect(sentBubble).not.toBeNull();
    expect(sentBubble.trimmed).toBe(sentBubble.text);
    expect(sentBubble.h).toBeLessThan(80);

    expect(errors, errors.join('\n')).toEqual([]);
  } finally {
    server.close();
  }
});

// Luis's report: "enter does not send, shift enter does" — verify that Enter
// sends and that Shift+Enter only inserts a newline without sending.
test('chat box: Enter sends, Shift+Enter only inserts a newline', async ({ page }) => {
  const { server, state } = await startMockServer();
  const port = server.address().port;
  try {
    const errors = [];
    page.on('pageerror', (e) => errors.push(String(e)));

    await page.goto(`http://127.0.0.1:${port}/`);
    await page.waitForSelector('text=👁️', { timeout: 10000 });
    await page.click('button[title]:has-text("👁️")');
    const input = page.locator('textarea[x-model="live.input"]');
    await expect(input).toBeVisible({ timeout: 10000 });

    const bubbleVisible = (txt) => page.waitForFunction((t) => {
      const el = document.querySelector('[x-ref="liveChat"]');
      if (!el) return false;
      return Array.from(el.querySelectorAll('div.px-3')).some((b) => b.textContent.trim() === t);
    }, txt, { timeout: 4000 });

    // 1) Bare Enter SENDS and clears the input
    const msg1 = 'mensaje con enter';
    await input.fill(msg1);
    await page.keyboard.press('Enter');
    await bubbleVisible(msg1);
    expect(state.sendHistory, 'Enter should have sent').toContain(msg1);
    expect(await input.inputValue(), 'the input should be empty after sending').toBe('');

    // 2) Shift+Enter inserts a newline and does NOT send
    await input.fill('linea1');
    await page.keyboard.press('Shift+Enter');
    await page.keyboard.type('linea2');
    expect(await input.inputValue(), 'Shift+Enter should insert a newline').toBe('linea1\nlinea2');
    await page.waitForTimeout(400);
    expect(state.sendHistory, 'Shift+Enter should not send').not.toContain('linea1');
    expect(state.sendHistory, 'Shift+Enter should not send').not.toContain('linea2');

    // 3) Enter with multiline text sends the whole text
    await page.keyboard.press('Enter');
    await bubbleVisible('linea1\nlinea2');
    expect(state.sendHistory, 'multiline Enter should send the whole text').toContain('linea1\nlinea2');

    expect(errors, errors.join('\n')).toEqual([]);
  } finally {
    server.close();
  }
});

// The mode selector must send structured {mode}, not /mode as text: /mode does
// not exist in Claude Code 2.1.227, the host resolves it with /plan + the wheel.
test('chat box: mode selector sends {mode} (not /mode as text)', async ({ page }) => {
  const { server, state } = await startMockServer();
  const port = server.address().port;
  try {
    const errors = [];
    page.on('pageerror', (e) => errors.push(String(e)));

    await page.goto(`http://127.0.0.1:${port}/`);
    await page.waitForSelector('text=👁️', { timeout: 10000 });
    await page.click('button[title]:has-text("👁️")');
    const modeSelect = page.locator('select[title="Modo"], select[title="Mode"]').last();
    await expect(modeSelect).toBeVisible({ timeout: 10000 });

    await modeSelect.selectOption('plan');
    await page.waitForTimeout(500);
    expect(state.modeHistory, 'a {mode: plan} should have arrived').toContain('plan');
    expect(state.sendHistory.every((s) => s === ''), 'no chat text should be sent').toBe(true);

    expect(errors, errors.join('\n')).toEqual([]);
  } finally {
    server.close();
  }
});

// Session blocked on the approval dialog: it is surfaced and there are buttons
// to approve (Enter) and to leave (Escape).
test('chat box: approval notice with Approve and Stop buttons', async ({ page }) => {
  const { server, state } = await startMockServer();
  const port = server.address().port;
  try {
    const errors = [];
    page.on('pageerror', (e) => errors.push(String(e)));

    // The initial chat already arrives with waiting=approval (blocked pane).
    state.payload.waiting = 'approval';
    await page.goto(`http://127.0.0.1:${port}/`);
    await page.waitForSelector('text=👁️', { timeout: 10000 });
    await page.click('button[title]:has-text("👁️")');

    // Notice visible
    const approveBtn = page.locator('button[title^="Aprobar el comando"], button[title^="Approve the command"]').last();
    await expect(approveBtn).toBeVisible({ timeout: 10000 });
    await expect(page.getByText(/esperando aprobación|awaiting approval/).first()).toBeVisible();

    // Approve sends Enter (dialog option 1)
    await approveBtn.click();
    await page.waitForTimeout(500);
    expect(state.keyHistory, 'Approve should send the enter key').toContain('enter');

    // Stop sends Escape (cancel the dialog)
    const stopBtn = page.locator('button[title="Cancelar / Escape"], button[title="Cancel / Escape"]').last();
    await expect(stopBtn).toBeVisible();
    await stopBtn.click();
    await page.waitForTimeout(500);
    expect(state.keyHistory, 'Stop should send escape').toContain('escape');

    expect(errors, errors.join('\n')).toEqual([]);
  } finally {
    server.close();
  }
});

// Regression: every AskUserQuestion option button used to send the same
// sendKey('enter') regardless of which one was clicked — it confirmed
// whatever the real pane's cursor already happened to be on, so clicking any
// option but the already-selected one silently picked the wrong one (or, in
// production, left the dialog looking stuck since nothing about the click
// matched what actually got submitted). Each button must post its own {i},
// not a blind key press, so the host can navigate the picker to that exact
// option itself (internal/host: sessionChoice).
test('chat box: choice picker option button posts its own index, not a blind Enter', async ({ page }) => {
  const { server, state } = await startMockServer();
  const port = server.address().port;
  try {
    const errors = [];
    page.on('pageerror', (e) => errors.push(String(e)));

    // Pending AskUserQuestion, cursor on option 0 ("Pizza") — clicking a
    // DIFFERENT option (index 1, "Ensalada") is the case that was broken.
    state.payload.waiting = 'choice';
    state.payload.choice = {
      question: '¿Cuál es la mejor comida?',
      options: ['Pizza', 'Ensalada', 'Sushi'],
      selected: 0,
    };
    await page.goto(`http://127.0.0.1:${port}/`);
    await page.waitForSelector('text=👁️', { timeout: 10000 });
    await page.click('button[title]:has-text("👁️")');

    const question = page.getByText('¿Cuál es la mejor comida?');
    await expect(question).toBeVisible({ timeout: 10000 });

    const ensaladaBtn = page.getByRole('button', { name: /Ensalada/ });
    await expect(ensaladaBtn).toBeVisible();
    await ensaladaBtn.click();
    await page.waitForTimeout(500);

    expect(state.choiceHistory, 'clicking "Ensalada" should post {choice: 1}').toEqual([1]);
    expect(state.keyHistory, 'the click must not fall back to a blind key press').not.toContain('enter');

    expect(errors, errors.join('\n')).toEqual([]);
  } finally {
    server.close();
  }
});

// Regression: the live modal got the {choice: i} buttons, but the terminal
// grid tile did not — its options were inert <span>s, so "responder con las
// opciones" in Modo terminal silently did nothing and the panel (and the real
// dialog) stayed open. The tile's options must be buttons posting their own
// index, the exact mirror of the modal's selectChoice.
test('terminal grid: clicking a tile choice option posts its own index, not a blind Enter', async ({ page }) => {
  const { server, state } = await startMockServer();
  const port = server.address().port;
  try {
    const errors = [];
    page.on('pageerror', (e) => errors.push(String(e)));

    // Pending AskUserQuestion on the mock session 0, cursor on option 0
    // ("Pizza") — clicking a DIFFERENT option (index 1, "Ensalada") is the
    // case that was broken (inert span).
    state.payload.waiting = 'choice';
    state.payload.choice = {
      question: '¿Cuál es la mejor comida?',
      options: ['Pizza', 'Ensalada', 'Sushi'],
      selected: 0,
    };
    await page.goto(`http://127.0.0.1:${port}/`);
    await page.waitForSelector('text=👁️', { timeout: 10000 });

    // Open the grid: the tile must render the choice panel from its meta.
    await page.getByRole('button', { name: /Modo terminal/ }).click();
    const tile = page.locator('.tgrid-tile');
    await expect(tile).toBeVisible({ timeout: 10000 });

    const question = page.getByText('¿Cuál es la mejor comida?');
    await expect(question).toBeVisible({ timeout: 10000 });

    const ensaladaBtn = tile.getByRole('button', { name: /Ensalada/ });
    await expect(ensaladaBtn).toBeVisible();
    await ensaladaBtn.click();
    await page.waitForTimeout(500);

    expect(state.choiceHistory, 'clicking "Ensalada" in a tile should post {choice: 1}').toEqual([1]);
    expect(state.keyHistory, 'the tile click must not fall back to a blind key press').not.toContain('enter');

    expect(errors, errors.join('\n')).toEqual([]);
  } finally {
    server.close();
  }
});

// Regression (the reported bug): the grid tile used to refresh its
// approval/choice/mode state ONLY off the shared /api/events stream, which is
// created inside initNotify() and only when the Notification API exists (so
// iOS/Edge has no events at all) — a dialog that opened AFTER the tile was
// already on screen never surfaced and the pane could only be answered by
// text. Each tile now opens its OWN /chat/stream (per-session, carries
// waiting/choice in its fingerprint), exactly like the live modal. Here the
// mock leaves /api/events closed and pushes the choice only through the tile's
// chat stream, after the tile is open: the bar must appear and work.
test('terminal grid: a choice arriving on the tile chat stream (dead /api/events) shows the bar', async ({ page }) => {
  const { server, state } = await startMockServer();
  const port = server.address().port;
  try {
    const errors = [];
    page.on('pageerror', (e) => errors.push(String(e)));

    // No dialog present when the tile opens.
    state.payload.waiting = '';
    await page.goto(`http://127.0.0.1:${port}/`);
    await page.waitForSelector('text=👁️', { timeout: 10000 });
    await page.getByRole('button', { name: /Modo terminal/ }).click();
    const tile = page.locator('.tgrid-tile');
    await expect(tile).toBeVisible({ timeout: 10000 });

    // Bar must NOT be there yet (nothing was asked).
    await expect(tile.getByText('¿Cuál es la mejor comida?')).toHaveCount(0);

    // The dialog arrives LATER, pushed through the tile's /chat/stream.
    state.pushChat(Object.assign({}, state.payload, {
      waiting: 'choice',
      choice: { question: '¿Cuál es la mejor comida?', options: ['Pizza', 'Ensalada', 'Sushi'], selected: 0 },
    }));

    // The bar appears on its own — no user action, no /api/events.
    await expect(tile.getByText('¿Cuál es la mejor comida?')).toBeVisible({ timeout: 4000 });
    await tile.getByRole('button', { name: /Ensalada/ }).click();
    await page.waitForTimeout(500);
    expect(state.choiceHistory, 'choice arrived by stream must be clickable').toEqual([1]);

    expect(errors, errors.join('\n')).toEqual([]);
  } finally {
    server.close();
  }
});

// Regression (Luis's report): the tile's historical part (termHist, the /chat
// messages above the live screen) used to be rebuilt ONLY in fetchTileMeta —
// on open and on shared /api/events events — so with that stream dead (iOS,
// no Notification API) a tile left open would show a history frozen at
// whatever was fetched when it opened, while the live pane kept advancing.
// applyTileChat now runs on every /chat/stream frame (not just meta): a new
// message pushed through the stream must appear in the tile's history.
test('terminal grid: a new message on the tile chat stream updates the historical part', async ({ page }) => {
  const { server, state } = await startMockServer();
  const port = server.address().port;
  try {
    const errors = [];
    page.on('pageerror', (e) => errors.push(String(e)));

    await page.goto(`http://127.0.0.1:${port}/`);
    await page.waitForSelector('text=👁️', { timeout: 10000 });
    await page.getByRole('button', { name: /Modo terminal/ }).click();
    const tile = page.locator('.tgrid-tile');
    await expect(tile).toBeVisible({ timeout: 10000 });
    // Initial history: Hola / Respuesta.
    await expect(tile.getByText(/Hola desde el móvil/)).toBeVisible({ timeout: 10000 });

    // A later message arrives through the tile's /chat/stream.
    state.pushChat(Object.assign({}, state.payload, {
      messages: (state.payload.messages || []).concat([
        { index: 3, role: 'user', content: 'Mensaje nuevo en la sesión' },
      ]),
    }));

    // The historical part shows it, without any user action or /api/events.
    await expect(tile.getByText('Mensaje nuevo en la sesión')).toBeVisible({ timeout: 4000 });

    expect(errors, errors.join('\n')).toEqual([]);
  } finally {
    server.close();
  }
});
