// Login flow (password + TOTP) against the second e2e server (run-e2e-auth.sh,
// port 8798): no LAN bypass there, so the login form is really exercised. Every
// other spec runs on 8799, where 127.0.0.0/8 bypasses login on purpose.
const { test, expect } = require('@playwright/test');
const crypto = require('crypto');

const AUTH_URL = 'http://127.0.0.1:8798';

// Fields are addressed by id, not by label: the login form and the user modal
// both label a field "Usuario", and both live in the DOM at the same time.

// The user configured in run-e2e-auth.sh. The secret is the RFC 6238 test key,
// so codes can be computed here without shipping a TOTP library to the spec.
const USER = 'e2e';
const PASSWORD = 'test123';
const SECRET = 'GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ';

function base32Decode(s) {
  const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567';
  let bits = 0;
  let value = 0;
  const out = [];
  for (const c of s.toUpperCase().replace(/=+$/, '')) {
    value = (value << 5) | alphabet.indexOf(c);
    bits += 5;
    if (bits >= 8) {
      out.push((value >>> (bits - 8)) & 0xff);
      bits -= 8;
    }
  }
  return Buffer.from(out);
}

// RFC 6238 / RFC 4226, six digits, 30-second step.
function totp(secret, at = Date.now()) {
  const counter = Math.floor(at / 1000 / 30);
  const buf = Buffer.alloc(8);
  buf.writeUInt32BE(Math.floor(counter / 0x100000000), 0);
  buf.writeUInt32BE(counter >>> 0, 4);
  const hmac = crypto.createHmac('sha1', base32Decode(secret)).update(buf).digest();
  const off = hmac[hmac.length - 1] & 0x0f;
  const v = ((hmac[off] & 0x7f) << 24) | (hmac[off + 1] << 16) | (hmac[off + 2] << 8) | hmac[off + 3];
  return String(v % 1000000).padStart(6, '0');
}

// A code cannot be used twice (the server's replay guard), so two logins as
// the same user inside one 30-second window would see the second one rejected.
// This waits for a step the fixture user has not spent yet.
let lastLoginStep = -1;
async function freshLoginCode() {
  while (Math.floor(Date.now() / 1000 / 30) === lastLoginStep) {
    await new Promise((r) => setTimeout(r, 500));
  }
  lastLoginStep = Math.floor(Date.now() / 1000 / 30);
  return totp(SECRET);
}

test.use({ baseURL: AUTH_URL });

// Note: the server rate-limits failed attempts per IP (5 within 15 min). A
// green run spends two of them, on the two negative tests below. If you add
// more failing attempts, or a real failure makes CI retry them, the extra
// runs can trip the limiter and turn later tests into 429s — a cascade, not
// the original bug.

test('login asks for the TOTP code and only then opens the app', async ({ page }) => {
  await page.goto('/', { waitUntil: 'domcontentloaded' });

  // No LAN bypass here: the login form is what greets us.
  await expect(page.getByRole('button', { name: 'Entrar' })).toBeVisible();

  await page.locator('#login-user').fill(USER);
  await page.locator('#login-pass').fill(PASSWORD);
  await page.getByRole('button', { name: 'Entrar' }).click();

  // The right password is not enough: step two appears and the login overlay
  // stays up. (The app's own markup is always in the DOM behind it, so what
  // proves we are not in is the overlay, not the absence of app buttons.)
  const overlay = page.locator('div[x-show="showLogin"]');
  const code = page.locator('#login-totp');
  await expect(code).toBeVisible();
  await expect(overlay).toBeVisible();

  await code.fill(await freshLoginCode());
  await page.getByRole('button', { name: 'Verificar' }).click();

  await expect(overlay).toBeHidden();
  await expect(page.getByRole('button', { name: /Nueva sesión rápida/ })).toBeVisible();
});

test('a wrong TOTP code keeps the app closed', async ({ page }) => {
  await page.goto('/', { waitUntil: 'domcontentloaded' });

  await page.locator('#login-user').fill(USER);
  await page.locator('#login-pass').fill(PASSWORD);
  await page.getByRole('button', { name: 'Entrar' }).click();

  await page.locator('#login-totp').fill('000000');
  await page.getByRole('button', { name: 'Verificar' }).click();

  await expect(page.getByText('Código incorrecto o caducado')).toBeVisible();
  await expect(page.locator('div[x-show="showLogin"]')).toBeVisible();
});

// Covers the only part of the 2FA feature no Go test can reach: the QR is
// actually drawn in a browser (the encoder is lazy-loaded, and its SVG output
// is the one variant the CSP allows). Enrolls a THROWAWAY user so the fixture
// user's secret — which the login tests above depend on — is never replaced.
test('2FA enrollment draws a QR and only enables on a valid code', async ({ page }) => {
  // freshLoginCode may sit out the rest of a 30 s window before it can log in.
  test.setTimeout(90000);
  page.on('dialog', (d) => d.accept());

  await page.goto('/', { waitUntil: 'domcontentloaded' });
  await page.locator('#login-user').fill(USER);
  await page.locator('#login-pass').fill(PASSWORD);
  await page.getByRole('button', { name: 'Entrar' }).click();
  await page.locator('#login-totp').fill(await freshLoginCode());
  await page.getByRole('button', { name: 'Verificar' }).click();
  await expect(page.locator('div[x-show="showLogin"]')).toBeHidden();

  await page.getByTitle('Settings').click();
  await page.getByRole('button', { name: /Configuración/ }).click();

  // A throwaway user to enroll.
  await page.getByRole('button', { name: /Añadir usuario/ }).click();
  await page.locator('#user-modal-name').fill('enroll');
  await page.locator('#user-modal-pass').fill('enroll-password');
  await page.getByRole('button', { name: 'Añadir usuario', exact: true }).click();

  // exact: getByTitle is a case-insensitive substring match, so a plain
  // 'Activar 2FA' also matches the fixture user's 'Desactivar 2FA' button.
  await page.getByTitle('Activar 2FA', { exact: true }).click();

  // The QR is rendered inline as SVG (createImgTag's data: URL would be blocked
  // by the CSP), and the secret is shown as a fallback for manual entry.
  await expect(page.locator('div[x-show="totpModal.open"] svg')).toBeVisible();
  const secret = (await page.locator('div[x-show="totpModal.open"] button.font-mono').innerText()).trim();
  expect(secret).toMatch(/^[A-Z2-7]{32}$/);

  // A wrong code changes nothing.
  await page.locator('#enroll-totp').fill('000000');
  await page.getByRole('button', { name: 'Confirmar y activar' }).click();
  await expect(page.getByText('invalid code')).toBeVisible();

  // The right one enables it, and the list shows the badge.
  await page.locator('#enroll-totp').fill(totp(secret));
  await page.getByRole('button', { name: 'Confirmar y activar' }).click();
  await expect(page.getByText('2FA activado para enroll.')).toBeVisible();
  // Both users have it now, so the list offers two "disable" buttons and no
  // "enable" one — the state really made it back into the panel.
  await expect(page.getByTitle('Desactivar 2FA', { exact: true })).toHaveCount(2);
  await expect(page.getByTitle('Activar 2FA', { exact: true })).toHaveCount(0);

  // Clean up so the fixture user is the only one left for any retry.
  await page.getByTitle('Eliminar', { exact: true }).last().click();
  await expect(page.getByText('Usuario enroll eliminado.')).toBeVisible();
});

test('a wrong password never reaches the code step', async ({ page }) => {
  await page.goto('/', { waitUntil: 'domcontentloaded' });

  await page.locator('#login-user').fill(USER);
  await page.locator('#login-pass').fill('not-the-password');
  await page.getByRole('button', { name: 'Entrar' }).click();

  // x-show only toggles display, so the field stays in the DOM: assert it is
  // hidden, not absent.
  await expect(page.locator('#login-totp')).toBeHidden();
  await expect(page.getByRole('button', { name: 'Entrar' })).toBeVisible();
});
