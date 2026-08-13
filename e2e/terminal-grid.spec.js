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
  const grid = page.locator('div.tgrid-wide-only');
  await expect(grid).toBeVisible();

  // Both tiles render with their session name and pane content.
  await expect(page.getByText('grid-alpha', { exact: true })).toBeVisible();
  await expect(page.getByText('grid-beta', { exact: true })).toBeVisible();
  await page.waitForTimeout(1500); // let both /stream tiles deliver a first frame
  await expect(page.getByText(/pane-of-grid-alpha/)).toBeVisible();
  await expect(page.getByText(/pane-of-grid-beta/)).toBeVisible();

  // Minimize alpha: it becomes a rail chip, beta's tile keeps rendering.
  const alphaTile = page.locator('.tgrid > div', { hasText: 'grid-alpha' });
  await alphaTile.getByTitle('Minimizar').click();
  await expect(page.locator('.tgrid', { hasText: 'grid-alpha' })).toHaveCount(0);
  const chip = page.getByRole('button', { name: /grid-alpha/ }).filter({ hasText: 'grid-alpha' });
  await expect(chip).toBeVisible();

  // Restore it.
  await chip.click();
  await expect(page.locator('.tgrid', { hasText: 'grid-alpha' })).toBeVisible();

  // Zoom beta, then un-zoom: alpha's tile must still be present in the DOM
  // throughout (its stream never tears down), just visually hidden.
  const betaTile = page.locator('.tgrid > div', { hasText: 'grid-beta' });
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

test('terminal grid: narrow viewport shows the fallback notice, not the mosaic', async ({ page }) => {
  page.on('dialog', (d) => d.accept());
  await page.setViewportSize({ width: 375, height: 700 });
  await page.goto('/', { waitUntil: 'domcontentloaded' });
  await createSession(page, 'grid-narrow');

  await page.getByRole('button', { name: /Modo terminal/ }).click();
  await expect(page.getByText('La vista en mosaico necesita')).toBeVisible();
  await expect(page.locator('.tgrid')).toBeHidden();

  await page.locator('div[x-show="grid.open"]').getByText('×', { exact: true }).click();
  await page.locator('.group').filter({ hasText: 'sesión grid-narrow' }).getByTitle('Cerrar sesión').click();
});
