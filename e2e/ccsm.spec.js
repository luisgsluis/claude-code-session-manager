// CCSM e2e against tmux/claude stubs. Verifies the full flow:
// LAN auto-login, create a named session and close it.
const { test, expect } = require('@playwright/test');

test('auto-login LAN, create session and close it', async ({ page }) => {
  page.on('dialog', (d) => {
    d.accept();
  });

  await page.goto('/', { waitUntil: 'domcontentloaded' });

  // LAN bypass: the main UI appears without going through login.
  const quickBtn = page.getByRole('button', { name: /Nueva sesión rápida/ });
  await expect(quickBtn).toBeVisible();

  // Create a named session from the advanced dropdown.
  await page.getByRole('button', { name: /Nueva sesión avanzada/ }).click();
  await page.getByLabel('Nombre sesión tmux').fill('e2e-ses');
  await page.getByRole('button', { name: 'Crear sesión' }).click();

  // Creation opens the live chat (same as the quick flow); close it to get
  // back to the list and check the card. The live modal's backdrop is the
  // only one with x-show="live.open".
  const liveModal = page.locator('div[x-show="live.open"]');
  await expect(liveModal).toBeVisible();
  await liveModal.click({ position: { x: 5, y: 5 } });

  // The session card appears with its name.
  const card = page.locator('.group').filter({ hasText: 'sesión e2e-ses' });
  await expect(card).toBeVisible();

  // Close it: the confirm() is accepted and the list becomes empty.
  await card.getByTitle('Cerrar sesión').click();
  await expect(page.getByText('No hay sesiones activas')).toBeVisible();
});

test('advanced session with project: starts in the project and shows the badge', async ({ page }) => {
  page.on('dialog', (d) => {
    d.accept();
  });

  await page.goto('/', { waitUntil: 'domcontentloaded' });

  // The dropdown offers the "principal" (home) entry plus the demo and alpha
  // projects, sorted alphabetically by their visible label.
  await page.getByRole('button', { name: /Nueva sesión avanzada/ }).click();
  const projSelect = page.getByLabel('Proyecto (CLAUDE.md)');
  const options = await projSelect.locator('option').allTextContents();
  expect(options.join(',')).toContain('Principal');
  expect(options.join(',')).toContain('demo');
  // The projects (excluding the fixed "principal" entry) are alphabetically sorted.
  const real = await projSelect.locator('option').evaluateAll(els =>
    els.filter(e => e.value !== 'principal').map(e => e.textContent));
  expect(real).toEqual([...real].sort());
  expect(real[0]).toBe('alpha');
  expect(real[1]).toBe('demo');

  // Picking a project shows its relative path below the dropdown.
  await projSelect.selectOption('projects/demo');
  await expect(page.getByText('projects/demo')).toBeVisible();

  // Pick the project, create the session and close the live modal.
  await page.getByLabel('Nombre sesión tmux').fill('e2e-proj');
  await page.getByRole('button', { name: 'Crear sesión' }).click();
  const liveModal = page.locator('div[x-show="live.open"]');
  await expect(liveModal).toBeVisible();
  await liveModal.click({ position: { x: 5, y: 5 } });

  // The card carries the badge with the project's base name.
  const card = page.locator('.group').filter({ hasText: 'sesión e2e-proj' });
  await expect(card).toBeVisible();
  await expect(card.getByText('demo')).toBeVisible();
  await expect(card.getByText('demo')).toHaveAttribute('title', 'projects/demo');

  // Close it and return to the empty state.
  await card.getByTitle('Cerrar sesión').click();
  await expect(page.getByText('No hay sesiones activas')).toBeVisible();
});
