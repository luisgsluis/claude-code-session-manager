// CCSM e2e for pane auto-resize: every grid tile must tell the backend how
// many columns/rows its raw pane actually renders, so Claude Code (a normal
// TUI that just wraps to the real terminal geometry) wraps its output to
// that width instead of tmux's default 80x24 — these sessions run detached
// and no real terminal client ever attaches to report a size on its own. See
// Host.sessionResize and requestPaneResize (app.js).
//
// The single-session Terminal tab does NOT do this: it renders the same
// transcript the Chat tab does (plain HTML text, termText/termHist), not a
// raw tmux pane capture, so it reflows to any width on its own — no resize
// needed, and the first test below asserts that.
const { test, expect } = require('@playwright/test');

async function createSession(page, name) {
  await page.getByRole('button', { name: /Nueva sesión avanzada/ }).click();
  await page.getByLabel('Nombre sesión tmux').fill(name);
  await page.getByRole('button', { name: 'Crear sesión' }).click();
}

test('live terminal tab does not resize the tmux window (plain-text transcript, not a raw pane)', async ({ page }) => {
  page.on('dialog', (d) => d.accept());
  await page.goto('/', { waitUntil: 'domcontentloaded' });

  await createSession(page, 'resize-tab');
  const liveModal = page.locator('div[x-show="live.open"]');
  await expect(liveModal).toBeVisible();

  let resizeSeen = false;
  page.on('requestfinished', (r) => { if (r.url().includes('/api/sessions/resize-tab/resize')) resizeSeen = true; });
  await page.getByRole('button', { name: 'Terminal', exact: true }).click(); // setLiveView('term')
  await page.waitForTimeout(1000); // resize.spec's own debounce is 300ms; give it a fair chance to (not) fire
  expect(resizeSeen, 'the single Terminal tab must never resize the tmux window').toBe(false);

  await liveModal.click({ position: { x: 5, y: 5 } }); // close via backdrop
  await page.locator('.group').filter({ hasText: 'sesión resize-tab' }).getByTitle('Archivar sesión').click();
});

test('a terminal grid tile resizes its own tmux window', async ({ page }) => {
  page.on('dialog', (d) => d.accept());
  await page.goto('/', { waitUntil: 'domcontentloaded' });

  await createSession(page, 'resize-tile');
  const liveModal = page.locator('div[x-show="live.open"]');
  await expect(liveModal).toBeVisible();
  await liveModal.click({ position: { x: 5, y: 5 } }); // close via backdrop

  const resizeReq = page.waitForResponse(
    (r) => r.url().includes('/api/sessions/resize-tile/resize') && r.request().method() === 'POST'
  );
  await page.getByRole('button', { name: /Modo terminal/ }).click();
  const resp = await resizeReq;
  expect(resp.status()).toBe(200);
  const body = JSON.parse(resp.request().postData());
  expect(body.cols).toBeGreaterThan(10);
  expect(body.rows).toBeGreaterThan(3);

  await page.locator('div[x-show="grid.open"]').getByText('×', { exact: true }).click();
  await page.locator('.group').filter({ hasText: 'sesión resize-tile' }).getByTitle('Archivar sesión').click();
});
