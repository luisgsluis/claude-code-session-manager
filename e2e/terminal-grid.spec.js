// CCSM e2e for the "Modo terminal" multi-session grid,
// against the real tmux/claude stubs (same webServer as ccsm.spec.js).
//
// This spec exists because manual browser testing caught a real bug that unit
// tests and the JS syntax check could not: unguarded `grid.tiles[name].xxx`
// bindings in the tile template threw "Cannot read properties of undefined"
// whenever a tile's data hadn't settled yet — the fix was wrapping the tile
// in <template x-if="grid.tiles[name]">, mirroring the choice-picker guard
// already used elsewhere in index.html. The console-error assertions below
// are what would have caught that automatically.
const { test, expect } = require('@playwright/test');

async function createSession(page, name) {
  await page.getByRole('button', { name: /Nueva sesión avanzada/ }).click();
  await page.getByLabel('Nombre sesión tmux').fill(name);
  await page.getByRole('button', { name: 'Crear sesión' }).click();
  const liveModal = page.locator('div[x-show="live.open"]');
  await expect(liveModal).toBeVisible();
  await liveModal.click({ position: { x: 5, y: 5 } }); // close via backdrop
}

test('terminal grid: tiles, minimize/restore, zoom/unzoom, send — no console errors', async ({ page }) => {
  page.on('dialog', (d) => d.accept());
  const consoleErrors = [];
  page.on('pageerror', (e) => consoleErrors.push('PAGEERROR: ' + e.message));
  page.on('console', (m) => { if (m.type() === 'error') consoleErrors.push('CONSOLE: ' + m.text()); });

  await page.goto('/', { waitUntil: 'domcontentloaded' });

  await createSession(page, 'grid-alpha');
  await createSession(page, 'grid-beta');

  // Open the grid.
  await page.getByRole('button', { name: /Modo terminal/ }).click();
  const grid = page.locator('div[x-show="grid.open"]');
  await expect(grid).toBeVisible();

  // Both tiles render with their session name and pane content.
  await expect(page.getByText('grid-alpha', { exact: true })).toBeVisible();
  await expect(page.getByText('grid-beta', { exact: true })).toBeVisible();
  await page.waitForTimeout(1500); // let both /stream tiles deliver a first frame
  await expect(page.getByText(/pane-of-grid-alpha/)).toBeVisible();
  await expect(page.getByText(/pane-of-grid-beta/)).toBeVisible();

  // Minimize alpha: it becomes a rail chip, beta's tile keeps rendering.
  const alphaTile = page.locator('.tgrid-tile', { hasText: 'grid-alpha' });
  await alphaTile.getByTitle('Minimizar').click();
  await expect(page.locator('.tgrid', { hasText: 'grid-alpha' })).toHaveCount(0);
  const chip = page.getByRole('button', { name: /grid-alpha/ }).filter({ hasText: 'grid-alpha' });
  await expect(chip).toBeVisible();

  // Restore it.
  await chip.click();
  await expect(page.locator('.tgrid', { hasText: 'grid-alpha' })).toBeVisible();

  // Zoom beta, then un-zoom: alpha's tile must still be present in the DOM
  // throughout (its stream never tears down), just visually hidden.
  const betaTile = page.locator('.tgrid-tile', { hasText: 'grid-beta' });
  await betaTile.getByTitle('Pantalla completa').click();
  await expect(page.locator('.tgrid-zoomed', { hasText: 'grid-beta' })).toBeVisible();
  await expect(alphaTile).toHaveCount(1); // present (tgrid-hidden), not removed
  await betaTile.getByTitle('Salir de pantalla completa').click();
  await expect(page.locator('.tgrid-zoomed')).toHaveCount(0);

  // Send text from a tile's own prompt.
  const betaInput = betaTile.getByPlaceholder('Escribe y pulsa Enter…');
  await betaInput.fill('hello from tile');
  await betaInput.press('Enter');
  await expect(betaInput).toHaveValue('');

  // Close the grid, back to the normal session list.
  const gridOverlay = page.locator('div[x-show="grid.open"]');
  await gridOverlay.getByText('×', { exact: true }).click();
  await expect(grid).toBeHidden();
  await expect(page.locator('.group').filter({ hasText: 'sesión grid-alpha' })).toBeVisible();

  // The one class of bug this spec exists to catch.
  const relevant = consoleErrors.filter(e => !e.includes('Content Security Policy') && !e.includes('502'));
  expect(relevant, 'unexpected console/page errors while using the grid').toEqual([]);

  // Cleanup: close both sessions so the run ends in the empty state.
  await page.locator('.group').filter({ hasText: 'sesión grid-alpha' }).getByTitle('Cerrar sesión').click();
  await page.locator('.group').filter({ hasText: 'sesión grid-beta' }).getByTitle('Cerrar sesión').click();
  await expect(page.getByText('No hay sesiones activas')).toBeVisible();
});

test('terminal grid: narrow viewport starts every tile minimized, one at a time', async ({ page }) => {
  page.on('dialog', (d) => d.accept());
  await page.setViewportSize({ width: 375, height: 700 });
  await page.goto('/', { waitUntil: 'domcontentloaded' });
  await createSession(page, 'grid-narrow-a');
  await createSession(page, 'grid-narrow-b');

  await page.getByRole('button', { name: /Modo terminal/ }).click();

  // Nothing open yet: both sessions show as header chips, and the pick-one
  // hint fills the body instead of a mosaic (which would be unusable here).
  await expect(page.getByText('Elige una sesión de arriba')).toBeVisible();
  await expect(page.locator('.tgrid-tile')).toHaveCount(0);
  const chipA = page.getByRole('button', { name: /grid-narrow-a/ });
  const chipB = page.getByRole('button', { name: /grid-narrow-b/ });
  await expect(chipA).toBeVisible();
  await expect(chipB).toBeVisible();

  // Opening one tile shows only that one — no mosaic of two.
  await chipA.click();
  await expect(page.locator('.tgrid-tile', { hasText: 'grid-narrow-a' })).toBeVisible();
  await expect(page.locator('.tgrid-tile')).toHaveCount(1);
  await expect(chipA).toBeHidden();

  // Opening the other tile replaces it, rather than tiling both.
  await chipB.click();
  await expect(page.locator('.tgrid-tile', { hasText: 'grid-narrow-b' })).toBeVisible();
  await expect(page.locator('.tgrid-tile')).toHaveCount(1);
  await expect(chipA).toBeVisible(); // a is back to being a chip

  // Minimizing the open tile returns to the pick-one hint.
  await page.locator('.tgrid-tile', { hasText: 'grid-narrow-b' }).getByTitle('Minimizar').click();
  await expect(page.getByText('Elige una sesión de arriba')).toBeVisible();
  await expect(page.locator('.tgrid-tile')).toHaveCount(0);

  await page.locator('div[x-show="grid.open"]').getByText('×', { exact: true }).click();
  await page.locator('.group').filter({ hasText: 'sesión grid-narrow-a' }).getByTitle('Cerrar sesión').click();
  await page.locator('.group').filter({ hasText: 'sesión grid-narrow-b' }).getByTitle('Cerrar sesión').click();
});

test('terminal grid: restoring a tile opens scrolled to the bottom, not the top', async ({ page }) => {
  // A minimized tile isn't mounted in the DOM at all, so it can accumulate
  // pane content in the background with nothing to scroll — restoreTile()
  // must force it to the bottom once it mounts. The tmux/claude stubs only
  // ever produce one short line of pane content, so this injects a long one
  // straight into the Alpine store (the same content shape a real session's
  // /stream would deliver) to actually overflow the pane and make the bug
  // observable.
  page.on('dialog', (d) => d.accept());
  await page.setViewportSize({ width: 375, height: 700 });
  await page.goto('/', { waitUntil: 'domcontentloaded' });
  await createSession(page, 'grid-scroll');

  await page.getByRole('button', { name: /Modo terminal/ }).click();
  const chip = page.getByRole('button', { name: /grid-scroll/ });
  await expect(chip).toBeVisible(); // grid.tiles['grid-scroll'] exists once its chip renders

  await page.evaluate((name) => {
    const data = window.Alpine.$data(document.body);
    data.grid.tiles[name].content = Array.from({ length: 300 }, (_, i) => 'line ' + i).join('\n');
  }, 'grid-scroll');

  await chip.click(); // restoreTile()
  await expect(page.locator('.tgrid-tile', { hasText: 'grid-scroll' })).toBeVisible();
  await expect(async () => {
    const atBottom = await page.evaluate(() => {
      const el = document.getElementById('tile-pane-grid-scroll');
      return el.scrollHeight - el.scrollTop - el.clientHeight < 8;
    });
    expect(atBottom).toBe(true);
  }).toPass({ timeout: 2000 });

  await page.locator('div[x-show="grid.open"]').getByText('×', { exact: true }).click();
  await page.locator('.group').filter({ hasText: 'sesión grid-scroll' }).getByTitle('Cerrar sesión').click();
});
