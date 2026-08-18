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
  await page.locator('.group').filter({ hasText: 'sesión grid-alpha' }).getByTitle('Archivar sesión').click();
  await page.locator('.group').filter({ hasText: 'sesión grid-beta' }).getByTitle('Archivar sesión').click();
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
  const tileA = page.locator('.tgrid-tile', { hasText: 'grid-narrow-a' });
  await expect(tileA).toBeVisible();
  await expect(page.locator('.tgrid-tile')).toHaveCount(1);
  await expect(chipA).toBeHidden();

  // Zoom is meaningless when only one tile can ever be shown — hidden here,
  // unlike the scroll-up/down controls it's replaced by in the same row
  // (which the wide-viewport spec above already exercises on desktop).
  await expect(tileA.getByTitle('Pantalla completa')).toBeHidden();
  const upBtn = tileA.getByTitle('Subir');
  const endBtn = tileA.getByTitle('Ir al final');
  await expect(upBtn).toBeVisible();
  await expect(endBtn).toBeVisible();
  await endBtn.click();
  await upBtn.click(); // just confirms neither throws / is unreachable

  // The header row (name, mode/model selects, scroll/output/minimize
  // buttons) must fit without needing its own horizontal scroll — that's
  // the whole point of hiding zoom and tightening tgrid-tile-header's
  // spacing below 1024px (terminal-grid.css).
  const header = tileA.locator('.tgrid-tile-header');
  const overflow = await header.evaluate((el) => el.scrollWidth - el.clientWidth);
  expect(overflow, 'tile header should not overflow on a phone-width tile').toBeLessThanOrEqual(0);

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
  await page.locator('.group').filter({ hasText: 'sesión grid-narrow-a' }).getByTitle('Archivar sesión').click();
  await page.locator('.group').filter({ hasText: 'sesión grid-narrow-b' }).getByTitle('Archivar sesión').click();
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
  await page.locator('.group').filter({ hasText: 'sesión grid-scroll' }).getByTitle('Archivar sesión').click();
});

test('terminal grid: focused prompt takes the full row at 3 lines, controls stay one compact row; header selects show current mode/model', async ({ page }) => {
  page.on('dialog', (d) => d.accept());
  await page.setViewportSize({ width: 375, height: 700 });
  await page.goto('/', { waitUntil: 'domcontentloaded' });
  await createSession(page, 'grid-focus');
  await page.getByRole('button', { name: /Modo terminal/ }).click();
  const chip = page.getByRole('button', { name: /grid-focus/ });
  await expect(chip).toBeVisible(); // grid.tiles['grid-focus'] exists once its chip renders
  await chip.click();
  const tile = page.locator('.tgrid-tile', { hasText: 'grid-focus' });
  await expect(tile).toBeVisible();
  await expect(page.getByText(/pane-of-grid-focus/)).toBeVisible(); // first stream frame

  // Simulate a session that already calibrated its mode/model (what /meta and
  // the turn watcher deliver on a real one). Injected after the initial stream
  // burst settles: applyTileChat replaces the whole tile.meta on every /chat
  // payload, so injecting before that just gets overwritten.
  await page.waitForTimeout(1000);
  await page.evaluate(() => {
    const data = window.Alpine.$data(document.body);
    const t = data.grid.tiles['grid-focus'];
    t.meta = { ...(t.meta || {}), mode: 'auto', model: 'claude-sonnet-5' };
    if (!data.live.models.includes('claude-sonnet-5')) data.live.models = data.live.models.concat('claude-sonnet-5');
  });

  // The header selects display the active values, not the "Modo"/"Modelo"
  // placeholders (Alpine :selected on the matching option, x-if placeholder).
  const selects = tile.locator('.tgrid-tile-header select');
  await expect(selects.nth(0)).toHaveValue('auto');
  await expect(selects.nth(1)).toHaveValue('claude-sonnet-5');

  const input = tile.getByPlaceholder('Escribe y pulsa Enter…');
  const bar = tile.locator('.tgrid-prompt');
  const lineH = parseFloat(await input.evaluate((el) => getComputedStyle(el).lineHeight));

  // Unfocused: the input is its single line.
  let box = await input.boundingBox();
  expect(box.height).toBeLessThan(lineH * 2);

  // Focus: expands to at least three lines and takes the whole bar width on
  // its own row (nothing else shares it); the controls drop below as one
  // compact row — never wrapping onto several lines.
  await input.focus();
  box = await input.boundingBox();
  expect(box.height).toBeGreaterThanOrEqual(lineH * 3 - 1);
  const barBox = await bar.boundingBox();
  expect(box.width).toBeGreaterThanOrEqual(barBox.width - 32);
  const actions = tile.locator('.tgrid-prompt-actions');
  const actionsBox = await actions.boundingBox();
  expect(actionsBox.y).toBeGreaterThanOrEqual(box.y + box.height - 1);
  const btnH = await actions.locator('button:visible').first().evaluate((el) => el.getBoundingClientRect().height);
  expect(actionsBox.height).toBeLessThan(btnH * 2);

  // Blur collapses the input back to one line (a long prompt must not leave
  // the bar permanently tall).
  await input.blur();
  box = await input.boundingBox();
  expect(box.height).toBeLessThan(lineH * 2);

  await page.locator('div[x-show="grid.open"]').getByText('×', { exact: true }).click();
  await page.locator('.group').filter({ hasText: 'sesión grid-focus' }).getByTitle('Archivar sesión').click();
});
