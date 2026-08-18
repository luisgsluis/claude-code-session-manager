// CCSM e2e for the active-profile indicator: the ✓ in the change-profile
// dropdown and the advanced-session profile select must agree with settings.json.
// The e2e stubs ship an empty profiles dir and no settings.json, so the tests
// seed the catalog + settings and let the page reload pick them up.
const { test, expect } = require('@playwright/test');
const fs = require('fs');
const path = require('path');

const PROFILES_DIR = path.join(__dirname, 'state', 'profiles');
const SETTINGS = path.join(__dirname, 'state', 'settings.json');

function seedProfiles(active) {
  fs.mkdirSync(PROFILES_DIR, { recursive: true });
  fs.writeFileSync(path.join(PROFILES_DIR, 'estandar.json'), '{"model":"sonnet"}');
  fs.writeFileSync(path.join(PROFILES_DIR, 'deepseek.json'), '{"model":"deepseek"}');
  fs.writeFileSync(SETTINGS, active === 'estandar' ? '{"model":"sonnet"}' : '{"model":"deepseek"}');
}

// The change-profile dropdown rows (apply button + view button per profile).
const profileRows = (page) => page.locator('div[class*="hover:bg-bg-hover"]');

// The one row that currently shows the visible active tick.
const activeRow = (page) =>
  profileRows(page).filter({ has: page.locator('span.text-success:visible') });

async function openChangeProfile(page) {
  const btn = page.getByRole('button', { name: /Cambiar perfil/ });
  await expect(btn).toBeEnabled();
  await btn.click();
}

test('active profile indicator matches settings.json in both views', async ({ page }) => {
  seedProfiles('deepseek'); // deepseek.json == settings.json
  await page.goto('/', { waitUntil: 'domcontentloaded' });
  await page.waitForResponse((r) => r.url().includes('/api/profiles'));

  // Advanced-session select: the active profile carries the ✓, the default
  // option names it, the inactive one carries nothing.
  await page.getByRole('button', { name: /Nueva sesión avanzada/ }).click();
  const select = page.locator('select[id="adv-profile"]');
  await expect(select).toBeVisible();
  const opts = await select.locator('option').allTextContents();
  expect(opts.join('|')).toContain('✓ deepseek');
  expect(opts.join('|')).toContain('Perfil activo (por defecto) · deepseek');
  expect(opts.join('|')).not.toContain('✓ estandar');
  await page.keyboard.press('Escape');

  // Change-profile dropdown: exactly one visible ✓, on the active profile.
  await openChangeProfile(page);
  await expect(page.locator('span.text-success:visible')).toHaveCount(1);
  await expect(activeRow(page)).toContainText('deepseek');
});

test('applying a profile moves the tick, synced with settings.json', async ({ page }) => {
  seedProfiles('deepseek');
  await page.goto('/', { waitUntil: 'domcontentloaded' });
  await page.waitForResponse((r) => r.url().includes('/api/profiles'));

  await openChangeProfile(page);
  await expect(page.locator('span.text-success:visible')).toHaveCount(1);
  await expect(activeRow(page)).toContainText('deepseek');

  // Apply estandar from the dropdown.
  await profileRows(page).filter({ hasText: 'estandar' }).first().locator('button').first().click();

  // The tick moves to estandar live (applyProfile reloads /api/profiles)…
  await openChangeProfile(page);
  await expect(async () => {
    await expect(page.locator('span.text-success:visible')).toHaveCount(1);
    await expect(activeRow(page)).toContainText('estandar');
  }).toPass({ timeout: 5000 });

  // …and survives a reload because settings.json now holds estandar's content.
  await page.reload();
  await page.waitForResponse((r) => r.url().includes('/api/profiles'));
  await openChangeProfile(page);
  await expect(page.locator('span.text-success:visible')).toHaveCount(1);
  await expect(activeRow(page)).toContainText('estandar');
});

test.afterAll(async () => {
  // Restore the empty baseline the other specs were designed against.
  fs.rmSync(path.join(PROFILES_DIR, 'estandar.json'), { force: true });
  fs.rmSync(path.join(PROFILES_DIR, 'deepseek.json'), { force: true });
  fs.rmSync(SETTINGS, { force: true });
});
