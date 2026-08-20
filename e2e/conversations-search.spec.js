// CCSM e2e for the conversations search boxes, against the real conversations
// dir (not mocked): title search must also match tags, full-conversation
// search must also match notes, and both boxes must start empty (no
// leftover value, no placeholder ghost text) — see internal/host/host.go
// conversationsList and static/index.html's search inputs.
const { test, expect } = require('@playwright/test');
const fs = require('fs');
const path = require('path');

test('conversations search: title box also matches tags, text box also matches notes, both start empty', async ({ page }) => {
  const convDir = path.join(__dirname, 'state', 'conversations');
  fs.mkdirSync(convDir, { recursive: true });

  // Tagged conversation: neither the transcript text nor the title contains
  // "kodi" — only the tag does. A title search for "kodi" must still surface
  // it, or the "título y tags" box is lying about what it searches.
  const tagged = '20000000-0000-4000-8000-000000000a01';
  fs.writeFileSync(
    path.join(convDir, tagged + '.jsonl'),
    JSON.stringify({
      type: 'user', cwd: '/home/admin', timestamp: '2026-08-01T10:00:00+02:00',
      message: { id: 'm-tagged', content: 'configuración de la tele del salón' },
    }) + '\n'
  );
  fs.writeFileSync(
    path.join(convDir, tagged + '.meta.json'),
    JSON.stringify({ tags: ['kodi'], notes: '' })
  );

  // Noted conversation: neither the transcript text nor the title contains
  // "sonarr" — only the note does. A full-conversation search for "sonarr"
  // must still surface it, or the "conversación y notas" box is lying too.
  const noted = '20000000-0000-4000-8000-000000000a02';
  fs.writeFileSync(
    path.join(convDir, noted + '.jsonl'),
    JSON.stringify({
      type: 'user', cwd: '/home/admin', timestamp: '2026-08-02T10:00:00+02:00',
      message: { id: 'm-noted', content: 'plan de mantenimiento del router' },
    }) + '\n'
  );
  fs.writeFileSync(
    path.join(convDir, noted + '.meta.json'),
    JSON.stringify({ tags: [], notes: 'pendiente revisar el emparejador de sonarr' })
  );

  try {
    await page.goto('/', { waitUntil: 'domcontentloaded' });
    await page.waitForResponse((r) => r.url().includes('/api/conversations?'));

    const titleBox = page.locator('input[type="search"]').nth(0);
    const textBox = page.locator('input[type="search"]').nth(1);

    // Both boxes start empty — no leftover value, no placeholder ghost text
    // standing in for one.
    await expect(titleBox).toHaveValue('');
    await expect(textBox).toHaveValue('');
    await expect(titleBox).not.toHaveAttribute('placeholder');
    await expect(textBox).not.toHaveAttribute('placeholder');

    // Title search for a tag-only word.
    await titleBox.fill('kodi');
    const reqTitle = page.waitForResponse((r) => r.url().includes('/api/conversations?') && r.url().includes('q=kodi'));
    await reqTitle;
    await expect(page.getByText('configuración de la tele del salón').first()).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('plan de mantenimiento del router')).toHaveCount(0);
    await titleBox.fill('');
    await page.waitForResponse((r) => r.url().includes('/api/conversations?') && !r.url().includes('q=kodi'));

    // Full-conversation search for a note-only word.
    await textBox.fill('sonarr');
    const reqText = page.waitForResponse((r) => r.url().includes('/api/conversations?') && r.url().includes('q_text=sonarr'));
    await reqText;
    await expect(page.getByText('plan de mantenimiento del router').first()).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('configuración de la tele del salón')).toHaveCount(0);
  } finally {
    for (const uuid of [tagged, noted]) {
      fs.rmSync(path.join(convDir, uuid + '.jsonl'), { force: true });
      fs.rmSync(path.join(convDir, uuid + '.meta.json'), { force: true });
    }
  }
});
