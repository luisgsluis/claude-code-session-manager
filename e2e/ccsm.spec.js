// e2e de CCSM contra stubs de tmux/claude. Verifica el flujo completo:
// auto-login por LAN, crear sesión con nombre y cerrarla.
const { test, expect } = require('@playwright/test');

test('auto-login LAN, crear sesión y cerrarla', async ({ page }) => {
  page.on('dialog', (d) => {
    d.accept();
  });

  await page.goto('/', { waitUntil: 'domcontentloaded' });

  // LAN bypass: la UI principal aparece sin pasar por el login.
  const quickBtn = page.getByRole('button', { name: /Nueva sesión rápida/ });
  await expect(quickBtn).toBeVisible();

  // Crear una sesión con nombre desde el desplegable avanzado.
  await page.getByRole('button', { name: /Nueva sesión avanzada/ }).click();
  await page.getByLabel('Nombre sesión tmux').fill('e2e-ses');
  await page.getByRole('button', { name: 'Crear sesión' }).click();

  // La creación abre el chat en vivo (igual que la rápida); lo cerramos para
  // volver a la lista y comprobar la tarjeta. El backdrop del modal live es el
  // único con x-show="live.open".
  const liveModal = page.locator('div[x-show="live.open"]');
  await expect(liveModal).toBeVisible();
  await liveModal.click({ position: { x: 5, y: 5 } });

  // La tarjeta de la sesión aparece con su nombre.
  const card = page.locator('.group').filter({ hasText: 'sesión e2e-ses' });
  await expect(card).toBeVisible();

  // Cerrarla: el confirm() se acepta y la lista queda vacía.
  await card.getByTitle('Cerrar sesión').click();
  await expect(page.getByText('No hay sesiones activas')).toBeVisible();
});
