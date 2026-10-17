// auth: proxy against the third e2e server (run-e2e-proxy.sh, port 8796), where
// loopback is the trusted proxy. The reverse proxy owns login, logout and
// users, so none of them may show up in CCSM's own UI.
const { test, expect } = require('@playwright/test');

test.use({
  baseURL: 'http://127.0.0.1:8796',
  // What the authenticating proxy (Tinyauth, Authelia…) adds after a login.
  extraHTTPHeaders: { 'Remote-User': 'ana' },
});

test('opens straight into the app, as the proxy user, without logout', async ({ page }) => {
  await page.goto('/', { waitUntil: 'domcontentloaded' });

  await expect(page.getByRole('button', { name: /Nueva sesión rápida/ })).toBeVisible();
  await expect(page.locator('#login-overlay')).toBeHidden();
  await expect(page.locator('input[type="password"]')).toHaveCount(0);
  await expect(page.getByText('ana', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Salir' })).toBeHidden();
});

test('settings show the proxy in charge, with no users to manage', async ({ page }) => {
  await page.goto('/', { waitUntil: 'domcontentloaded' });
  await page.getByTitle('Settings').click();
  await page.getByRole('button', { name: /Configuración/ }).click();

  await expect(page.getByText('la hace el proxy inverso')).toBeVisible();
  await expect(page.getByRole('button', { name: /Añadir usuario/ })).toBeHidden();
  await expect(page.getByText('redes LAN')).toHaveCount(0);
});
