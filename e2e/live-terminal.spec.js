// CCSM e2e for the single-session "Terminal" tab (live.view === 'term'),
// against the real tmux/claude stubs (same webServer as ccsm.spec.js).
const { test, expect } = require('@playwright/test');

test('live terminal tab opens scrolled to the bottom, not the top', async ({ page }) => {
  // The pane mounts fresh each time the Terminal tab is selected (x-if in
  // index.html), with the conversation history (termHist) already in
  // termText — setLiveView() must force it to the bottom once it mounts.
  // The tmux/claude stubs never produce a real conversation, so this
  // injects a long termHist straight into the Alpine store to actually
  // overflow the pane and make the bug observable.
  page.on('dialog', (d) => d.accept());
  await page.goto('/', { waitUntil: 'domcontentloaded' });

  await page.getByRole('button', { name: /Nueva sesión avanzada/ }).click();
  await page.getByLabel('Nombre sesión tmux').fill('term-scroll');
  await page.getByRole('button', { name: 'Crear sesión' }).click();
  const liveModal = page.locator('div[x-show="live.open"]');
  await expect(liveModal).toBeVisible();

  await page.evaluate(() => {
    const data = window.Alpine.$data(document.body);
    data.live.termHist = Array.from({ length: 300 }, (_, i) => 'line ' + i).join('\n');
  });

  await page.getByRole('button', { name: 'Terminal', exact: true }).click(); // setLiveView('term')
  await expect(async () => {
    const atBottom = await page.evaluate(() => {
      const el = document.querySelector('[x-ref="livePane"]');
      return el.scrollHeight - el.scrollTop - el.clientHeight < 8;
    });
    expect(atBottom).toBe(true);
  }).toPass({ timeout: 2000 });

  await liveModal.click({ position: { x: 5, y: 5 } }); // close via backdrop
  await page.locator('.group').filter({ hasText: 'sesión term-scroll' }).getByTitle('Cerrar sesión').click();
});
