// e2e for voice dictation and prompt rewriting.
//
// The provider is stubbed (e2e/stubs/voice-provider.js) but everything else is
// real: a real MediaRecorder recording a real (fake) audio device, a real
// upload, the real Go handlers, and the real review panel. Only the model at
// the far end is fake, which is the minimum that can be faked.

const { test, expect, devices } = require('@playwright/test');

// openChat creates a session and lands in its live chat, which is where both
// voice buttons live. Creating one is the same flow ccsm.spec.js uses: the
// e2e server starts with no sessions.
async function openChat(page) {
  page.on('dialog', (d) => d.accept());
  await page.goto('/', { waitUntil: 'domcontentloaded' });
  await expect(page.getByRole('button', { name: /Nueva sesión rápida/ })).toBeVisible({ timeout: 15000 });
  await page.getByRole('button', { name: /Nueva sesión rápida/ }).click();
  await expect(page.locator('div[x-show="live.open"]')).toBeVisible({ timeout: 15000 });
  await expect(page.locator('textarea[x-model="live.input"]')).toBeVisible({ timeout: 15000 });
}

// openSettingsPanel opens the ☰ menu and then the settings modal.
async function openSettingsPanel(page) {
  await page.goto('/', { waitUntil: 'domcontentloaded' });
  await expect(page.getByRole('button', { name: /Nueva sesión rápida/ })).toBeVisible({ timeout: 15000 });
  await page.locator('button[title="Settings"]').click();
  await page.locator('button:has-text("Configuración")').first().click();
  await expect(page.locator('div[x-show="settings.open"]')).toBeVisible({ timeout: 10000 });
}

test.describe('voice buttons', () => {
  test('both buttons appear in the chat when voice is configured', async ({ page }) => {
    await openChat(page);
    await expect(page.locator('button[title*="Dictar"]')).toBeVisible();
    await expect(page.locator('button[title*="Reescribir"]')).toBeVisible();
  });

  // The regression for the header that silently disabled everything: with
  // microphone=() the browser denies getUserMedia with no error and no prompt.
  test('the server allows the microphone for its own origin', async ({ page }) => {
    const resp = await page.request.get('/api/health');
    const policy = resp.headers()['permissions-policy'] || '';
    expect(policy).toContain('microphone=(self)');
    expect(policy).toContain('camera=()');
  });

  // Dictation needs a secure context. Without this check the button would be
  // visible over plain http and simply do nothing when pressed.
  test('the mic button hides without a secure context', async ({ browser }) => {
    const ctx = await browser.newContext();
    await ctx.addInitScript(() => {
      Object.defineProperty(window, 'isSecureContext', { get: () => false });
    });
    const page = await ctx.newPage();
    await openChat(page);
    await expect(page.locator('button[title*="Dictar"]')).toBeHidden();
    // Rewriting needs no microphone, so it stays.
    await expect(page.locator('button[title*="Reescribir"]')).toBeVisible();
    await ctx.close();
  });
});

test.describe('rewrite (the sparkle button)', () => {
  test('rewrites the input and opens the review panel', async ({ page }) => {
    await openChat(page);
    await page.locator('textarea[x-model="live.input"]').fill('pues eh arregla el sonar ese');
    await page.locator('button[title*="Reescribir"]').click();

    const panel = page.locator('textarea.voice-compose-text');
    await expect(panel).toBeVisible({ timeout: 10000 });
    await expect(panel).toHaveValue(/PROMPT REESCRITO/);
    // The text that was in the input really reached the model.
    await expect(panel).toHaveValue(/arregla el sonar ese/);
  });

  test('refuses to rewrite an empty input', async ({ page }) => {
    await openChat(page);
    await page.locator('textarea[x-model="live.input"]').fill('   ');
    await page.locator('button[title*="Reescribir"]').click();
    await expect(page.locator('.voice-compose-panel')).toBeHidden();
  });

  // auto must show the model every role block; a forced role must show only
  // its own. The stub reports which blocks it received.
  test('auto sends every role block, a forced role only its own', async ({ page }) => {
    await openChat(page);
    await page.locator('textarea[x-model="live.input"]').fill('algo que hacer');
    await page.locator('button[title*="Reescribir"]').click();

    const panel = page.locator('textarea.voice-compose-text');
    await expect(panel).toBeVisible({ timeout: 10000 });
    await expect(panel).toHaveValue(/BLOQUES:todos/);

    // Force "docs" and retry from the same raw text.
    await page.locator('.voice-compose-panel select').first().selectOption('docs');
    await page.locator('.voice-compose-panel button:has-text("Reintentar")').click();
    await expect(panel).toHaveValue(/BLOQUES:docs/, { timeout: 10000 });
  });

  test('a provider failure surfaces as an error, without the credential', async ({ page }) => {
    await openChat(page);
    await page.locator('textarea[x-model="live.input"]').fill('esto provoca un FALLO');
    await page.locator('button[title*="Reescribir"]').click();

    // The toast shows the failure...
    await expect(page.locator('text=/provider returned 500/i')).toBeVisible({ timeout: 10000 });
    // ...and the panel never opens with a broken result.
    await expect(page.locator('.voice-compose-panel')).toBeHidden();
    // The stub echoes the Authorization header in its error body on purpose:
    // if that reached the browser, the API key would be on screen.
    await expect(page.locator('body')).not.toContainText('e2e-stub-key');
  });

  test('a model that wraps its JSON in prose is still parsed', async ({ page }) => {
    await openChat(page);
    await page.locator('textarea[x-model="live.input"]').fill('devuelve BASURA por favor');
    await page.locator('button[title*="Reescribir"]').click();
    await expect(page.locator('textarea.voice-compose-text')).toHaveValue('RESCATADO DE PROSA', { timeout: 10000 });
  });
});

test.describe('review panel', () => {
  async function rewriteInChat(page, text) {
    await openChat(page);
    await page.locator('textarea[x-model="live.input"]').fill(text);
    await page.locator('button[title*="Reescribir"]').click();
    await expect(page.locator('.voice-compose-panel')).toBeVisible({ timeout: 10000 });
  }

  // The stub (AMBIGUO) walks two rounds: an options question, then a
  // free-text one, then stops — proving the panel actually loops (asks
  // again after an answer) rather than only handling a single round.
  test('clarifying questions are asked one at a time, in sequence', async ({ page }) => {
    await rewriteInChat(page, 'esto es AMBIGUO del todo');
    await expect(page.locator('textarea.voice-compose-text')).toHaveValue(/PROMPT PROVISIONAL/);

    // Round 1: an options question.
    await expect(page.getByText('¿El sonarr de la Pi o el del NAS?')).toBeVisible();
    await page.locator('.voice-compose-panel button:has-text("el de la Pi")').click();

    // Round 2: a free-text question, no options.
    await expect(page.getByText('¿Qué episodio concreto?')).toBeVisible({ timeout: 10000 });
    await page.locator('.voice-question input[type="text"]').fill('el 4x08');
    await page.locator('.voice-compose-panel button:has-text("Responder")').click();

    // Round 3: nothing further to ask, and both answers reached the model.
    await expect(page.locator('.voice-question')).toBeHidden({ timeout: 10000 });
    await expect(page.locator('textarea.voice-compose-text')).toHaveValue(/PROMPT FINAL/);
    await expect(page.locator('textarea.voice-compose-text')).toHaveValue(/el de la Pi/);
  });

  test('a clarifying question can be skipped without asking the model again', async ({ page }) => {
    const calls = [];
    page.on('request', (r) => { if (r.url().includes('/api/voice/rewrite')) calls.push(r); });

    await rewriteInChat(page, 'esto es AMBIGUO del todo');
    await expect(page.getByText('¿El sonarr de la Pi o el del NAS?')).toBeVisible();
    const before = calls.length;

    await page.locator('.voice-compose-panel button:has-text("Omitir")').click();
    await expect(page.locator('.voice-question')).toBeHidden();
    // Skipping does not call the rewriter again: the provisional prompt
    // already shown is the model's best attempt.
    expect(calls.length).toBe(before);
    await expect(page.locator('textarea.voice-compose-text')).toHaveValue(/PROMPT PROVISIONAL/);
  });

  test('shows the raw transcription on demand', async ({ page }) => {
    await rewriteInChat(page, 'texto original tal cual lo dije');
    await page.locator('.voice-compose-panel button:has-text("Ver lo que dicté")').click();
    await expect(page.locator('.voice-compose-panel')).toContainText('texto original tal cual lo dije');
    await page.locator('.voice-compose-panel button:has-text("Ver el prompt")').click();
    await expect(page.locator('textarea.voice-compose-text')).toBeVisible();
  });

  test('blocks sending over the character limit', async ({ page }) => {
    await rewriteInChat(page, 'dame algo LARGO');
    const send = page.locator('.voice-compose-panel button:has-text("Enviar")');
    await expect(send).toBeDisabled();
    await expect(page.locator('.voice-count-over')).toBeVisible();
  });

  test('sends to the session and closes', async ({ page }) => {
    const posts = [];
    page.on('request', (r) => {
      if (r.method() === 'POST' && /\/api\/sessions\/[^/]+\/send$/.test(r.url())) {
        posts.push(r.postData());
      }
    });

    await rewriteInChat(page, 'manda esto a la sesion');
    await page.locator('.voice-compose-panel button:has-text("Enviar")').click();

    // ~10s per send here: the tmux stub has no list-panes, so ensurePaneReady
    // burns its full timeout before every send. Real sends are immediate.
    await expect(page.locator('.voice-compose-panel')).toBeHidden({ timeout: 25000 });
    expect(posts.length).toBe(1);
    expect(posts[0]).toContain('PROMPT REESCRITO');
  });

  test('"to input" drops the text in the chat input and closes', async ({ page }) => {
    await rewriteInChat(page, 'esto va al input');
    await page.locator('.voice-compose-panel button:has-text("Al input")').click();
    await expect(page.locator('.voice-compose-panel')).toBeHidden();
    await expect(page.locator('textarea[x-model="live.input"]')).toHaveValue(/PROMPT REESCRITO/);
  });

  test('discard closes without sending', async ({ page }) => {
    const posts = [];
    page.on('request', (r) => {
      if (r.method() === 'POST' && /\/send$/.test(r.url())) posts.push(r.url());
    });
    await rewriteInChat(page, 'esto se descarta');
    await page.locator('.voice-compose-panel button:has-text("Descartar")').click();
    await expect(page.locator('.voice-compose-panel')).toBeHidden();
    expect(posts.length).toBe(0);
  });

  test('the edited text is what gets sent', async ({ page }) => {
    const posts = [];
    page.on('request', (r) => {
      if (r.method() === 'POST' && /\/send$/.test(r.url())) posts.push(r.postData());
    });
    await rewriteInChat(page, 'texto base');
    await page.locator('textarea.voice-compose-text').fill('LO QUE YO EDITE A MANO');
    await page.locator('.voice-compose-panel button:has-text("Enviar")').click();
    await expect(page.locator('.voice-compose-panel')).toBeHidden({ timeout: 25000 });
    expect(posts[0]).toContain('LO QUE YO EDITE A MANO');
  });
});

test.describe('dictation (real MediaRecorder against a fake device)', () => {
  test('records, uploads, transcribes, rewrites and opens the panel', async ({ page, context }) => {
    await context.grantPermissions(['microphone']);

    const uploads = [];
    page.on('request', (r) => {
      if (r.url().includes('/api/voice/transcribe')) uploads.push(r);
    });

    await openChat(page);
    const mic = page.locator('button[title*="Dictar"]');
    await mic.click();

    // Recording state: the button turns into a stop control.
    await expect(page.locator('button[title*="Parar"]')).toBeVisible({ timeout: 10000 });
    await page.waitForTimeout(1200); // let the fake device produce some audio
    await page.locator('button[title*="Parar"]').click();

    // The panel opens with the stub's transcription already rewritten — the
    // mic button does BOTH stages, there is no transcribe-only mode.
    const panel = page.locator('textarea.voice-compose-text');
    await expect(panel).toBeVisible({ timeout: 20000 });
    await expect(panel).toHaveValue(/PROMPT REESCRITO/);
    await expect(panel).toHaveValue(/sonarr/);

    // A real recording was uploaded, with a real audio content type.
    expect(uploads.length).toBe(1);
    expect(uploads[0].headers()['content-type']).toMatch(/^audio\//);
  });

  test('the raw transcription is kept and viewable after dictating', async ({ page, context }) => {
    await context.grantPermissions(['microphone']);
    await openChat(page);
    await page.locator('button[title*="Dictar"]').click();
    await expect(page.locator('button[title*="Parar"]')).toBeVisible({ timeout: 10000 });
    await page.waitForTimeout(1200);
    await page.locator('button[title*="Parar"]').click();
    await expect(page.locator('textarea.voice-compose-text')).toBeVisible({ timeout: 20000 });

    await page.locator('.voice-compose-panel button:has-text("Ver lo que dicté")').click();
    // The stub's fixed transcript, including the word the vocabulary hint is
    // meant to protect.
    await expect(page.locator('.voice-compose-panel')).toContainText('sonarr');
  });
});

test.describe('meta-prompt editor', () => {
  async function openEditor(page) {
    await openSettingsPanel(page);
    await page.locator('button:has-text("Editar meta-prompt")').click();
    await expect(page.locator('#prompt-editor-text')).toBeVisible({ timeout: 10000 });
  }

  const versionSelect = (page) => page.locator('.voice-editor-panel select').nth(1);
  const activeBadge = (page) => page.locator('.voice-editor-panel span:has-text("Activa:")');

  test('opens with the shipped original active, and lists its sections', async ({ page }) => {
    await openEditor(page);
    const content = await page.locator('#prompt-editor-text').inputValue();
    expect(content).toContain('# Base');
    expect(content).toContain('# Role: devops');
    // The original is active by default, and it is the only entry.
    await expect(activeBadge(page)).toHaveText('Activa: Original');
    expect(await versionSelect(page).locator('option').count()).toBe(1);
    // Overwriting the original is not offered — only saving as new is.
    // Exact name match: "Guardar" is a prefix of "Guardar como…" too.
    await expect(page.locator('.voice-editor-panel').getByRole('button', { name: 'Guardar', exact: true })).toBeHidden();
    await expect(page.locator('.voice-editor-panel button:has-text("Guardar como")')).toBeVisible();
  });

  // The validation that stops a role reaching the model with no instructions.
  test('rejects a prompt whose role has no section, and saves nothing', async ({ page }) => {
    await openEditor(page);
    const editor = page.locator('#prompt-editor-text');
    await editor.fill('---\nroles:\n  - {id: devops, es: D, en: D}\n---\n# Base\n\nsin bloque\n');

    page.on('dialog', (d) => d.accept('rota'));
    await page.locator('.voice-editor-panel button:has-text("Guardar como")').click();

    // The error names exactly what is wrong, because the user has to fix it.
    await expect(page.locator('text=/# Role: devops/')).toBeVisible({ timeout: 10000 });

    // Reopening shows only the original: nothing was written.
    await page.reload();
    await openEditor(page);
    expect(await page.locator('#prompt-editor-text').inputValue()).toContain('# Role: devops');
    expect(await versionSelect(page).locator('option').count()).toBe(1);
  });

  test('saves an edit as a new named version without activating it, then applies it', async ({ page }) => {
    await openEditor(page);
    const editor = page.locator('#prompt-editor-text');
    const original = await editor.inputValue();

    await editor.fill(original.replace('# Base', '# Base\n\nLINEA DE PRUEBA E2E.'));
    page.on('dialog', (d) => d.accept('Versión E2E'));
    await page.locator('.voice-editor-panel button:has-text("Guardar como")').click();

    // Saving alone must not activate it: the original is still in effect.
    await expect(page.getByText('Versión guardada')).toBeVisible({ timeout: 10000 });
    await expect(activeBadge(page)).toHaveText('Activa: Original');
    await expect(versionSelect(page)).toHaveText(/Versión E2E/);

    // Now apply it: the badge and the active check mark both move.
    await page.locator('.voice-editor-panel button:has-text("Aplicar esta versión")').click();
    await expect(activeBadge(page)).toHaveText('Activa: Versión E2E', { timeout: 10000 });
    await expect(page.locator('.voice-editor-panel button:has-text("Aplicar esta versión")')).toBeDisabled();

    // Reloading proves it stuck server-side, and the saved version was not
    // discarded by activating something else earlier in the same session.
    await page.reload();
    await openEditor(page);
    expect(await editor.inputValue()).toContain('LINEA DE PRUEBA E2E');
    await expect(activeBadge(page)).toHaveText('Activa: Versión E2E');

    // Applying is non-destructive: switching back to the original keeps the
    // saved version around, still selectable.
    await versionSelect(page).selectOption({ label: 'Original' });
    await page.locator('.voice-editor-panel button:has-text("Aplicar esta versión")').click();
    await expect(activeBadge(page)).toHaveText('Activa: Original', { timeout: 10000 });
    expect(await versionSelect(page).locator('option').count()).toBe(2);
  });

  // Roles live in the prompt, so adding one there and applying it must reach
  // the rewrite panel's dropdown with no rebuild.
  test('a role added to the prompt and applied shows up in the panel dropdown', async ({ page }) => {
    await openEditor(page);
    const editor = page.locator('#prompt-editor-text');
    const original = await editor.inputValue();
    const withRole = original
      .replace('roles:', 'roles:\n  - {id: seguridad, es: Seguridad, en: Security}')
      .replace('# Role: software', '# Role: seguridad\n\nThreat modelling.\n\n# Role: software');

    await editor.fill(withRole);
    page.on('dialog', (d) => d.accept('con seguridad'));
    await page.locator('.voice-editor-panel button:has-text("Guardar como")').click();
    await expect(page.getByText('Versión guardada')).toBeVisible({ timeout: 10000 });
    await page.locator('.voice-editor-panel button:has-text("Aplicar esta versión")').click();
    await expect(activeBadge(page)).toHaveText('Activa: con seguridad', { timeout: 10000 });

    await page.reload();
    await openChat(page);
    await page.locator('textarea[x-model="live.input"]').fill('algo');
    await page.locator('button[title*="Reescribir"]').click();
    await expect(page.locator('.voice-compose-panel')).toBeVisible({ timeout: 10000 });

    const options = await page.locator('.voice-compose-panel select').first().locator('option').allTextContents();
    expect(options.join(',')).toContain('Seguridad');
  });

  // This runs after the earlier tests in the file have already saved their
  // own versions into the same server-side prompts directory, so it asserts
  // on the CHANGE in option count rather than an absolute one.
  test('editing an existing (non-original) version can overwrite it in place', async ({ page }) => {
    await openEditor(page);
    const editor = page.locator('#prompt-editor-text');
    const original = await editor.inputValue();
    const countBefore = await versionSelect(page).locator('option').count();

    page.on('dialog', (d) => d.accept('borrador'));
    await editor.fill(original.replace('# Base', '# Base\n\nBORRADOR V1.'));
    await page.locator('.voice-editor-panel button:has-text("Guardar como")').click();
    await expect(page.getByText('Versión guardada')).toBeVisible({ timeout: 10000 });
    expect(await versionSelect(page).locator('option').count()).toBe(countBefore + 1);

    // Now viewing "borrador" (savePromptAsNew switches to it): overwrite is
    // offered, and editing it again keeps the same name and id rather than
    // creating yet another version.
    const saveBtn = page.locator('.voice-editor-panel').getByRole('button', { name: 'Guardar', exact: true });
    await expect(saveBtn).toBeVisible();
    await editor.fill((await editor.inputValue()).replace('BORRADOR V1', 'BORRADOR V2'));
    await saveBtn.click();
    await expect(page.getByText('Versión guardada')).toBeVisible({ timeout: 10000 });
    expect(await versionSelect(page).locator('option').count()).toBe(countBefore + 1);
    await expect(versionSelect(page)).toHaveText(/borrador/);
    expect(await editor.inputValue()).toContain('BORRADOR V2');
  });

  // Rename/delete live in the bottom bar next to Guardar (not in the toolbar
  // with the version picker — they were moved there after review), and act
  // on whichever version is currently loaded in the editor.
  test('renames a saved version and the dropdown reflects it immediately', async ({ page }) => {
    await openEditor(page);
    const editor = page.locator('#prompt-editor-text');
    const original = await editor.inputValue();

    page.once('dialog', (d) => d.accept('para renombrar'));
    await editor.fill(original.replace('# Base', '# Base\n\nPARA RENOMBRAR.'));
    await page.locator('.voice-editor-panel button:has-text("Guardar como")').click();
    await expect(page.getByText('Versión guardada')).toBeVisible({ timeout: 10000 });
    await expect(versionSelect(page)).toHaveText(/para renombrar/);

    const bottomBar = page.locator('.voice-editor-panel > div:last-child');
    await expect(bottomBar.locator('button:has-text("Renombrar")')).toBeVisible();
    await expect(bottomBar.locator('button:has-text("Borrar")')).toBeVisible();

    page.once('dialog', (d) => d.accept('nuevo nombre'));
    await bottomBar.locator('button:has-text("Renombrar")').click();
    await expect(page.getByText('Versión renombrada')).toBeVisible({ timeout: 10000 });
    await expect(versionSelect(page)).toHaveText(/nuevo nombre/);
  });

  test('deletes a saved version, removing it from the dropdown and disk', async ({ page }) => {
    await openEditor(page);
    const editor = page.locator('#prompt-editor-text');
    const original = await editor.inputValue();
    const countBefore = await versionSelect(page).locator('option').count();

    page.once('dialog', (d) => d.accept('para borrar'));
    await editor.fill(original.replace('# Base', '# Base\n\nPARA BORRAR.'));
    await page.locator('.voice-editor-panel button:has-text("Guardar como")').click();
    await expect(page.getByText('Versión guardada')).toBeVisible({ timeout: 10000 });
    expect(await versionSelect(page).locator('option').count()).toBe(countBefore + 1);

    page.once('dialog', (d) => d.accept());
    await page.locator('.voice-editor-panel button:has-text("Borrar")').click();
    await expect(page.getByText('Versión borrada')).toBeVisible({ timeout: 10000 });
    expect(await versionSelect(page).locator('option').count()).toBe(countBefore);
  });

  // The important one: PromptStore.Delete falls back the active pointer to
  // the original server-side, and the editor must show that, not a version
  // that no longer exists.
  test('deleting the active version falls back to the original', async ({ page }) => {
    await openEditor(page);
    const editor = page.locator('#prompt-editor-text');
    const original = await editor.inputValue();

    page.once('dialog', (d) => d.accept('activa a borrar'));
    await editor.fill(original.replace('# Base', '# Base\n\nACTIVA A BORRAR.'));
    await page.locator('.voice-editor-panel button:has-text("Guardar como")').click();
    await expect(page.getByText('Versión guardada')).toBeVisible({ timeout: 10000 });
    await page.locator('.voice-editor-panel button:has-text("Aplicar esta versión")').click();
    await expect(activeBadge(page)).toHaveText('Activa: activa a borrar', { timeout: 10000 });

    page.once('dialog', (d) => d.accept());
    await page.locator('.voice-editor-panel button:has-text("Borrar")').click();
    await expect(page.getByText('Versión borrada')).toBeVisible({ timeout: 10000 });
    await expect(activeBadge(page)).toHaveText('Activa: Original', { timeout: 10000 });
  });

  // Regression guard for a bug that shipped and went unnoticed: the badge
  // used a Tailwind arbitrary-value class (text-[0.65rem]) that was never
  // actually compiled into app.css (this project's build is a committed
  // artifact, not regenerated from source — see voice.css's own header
  // comment), so it silently rendered at the 16px browser default instead.
  // Assert the real computed size, not just that the class name is present.
  test('the active-version badge renders at the app\'s small text size', async ({ page }) => {
    await openEditor(page);
    const fontSize = await activeBadge(page).evaluate(el => parseFloat(getComputedStyle(el).fontSize));
    // text-xs is 12px; anything near 16px means the class silently no-oped.
    expect(fontSize).toBeLessThanOrEqual(13);
  });

  // Regression guard: the textarea used white-space:pre, so a long line
  // forced horizontal scrolling instead of wrapping to the panel's width.
  test('a long line wraps to the panel width instead of scrolling', async ({ page }) => {
    await openEditor(page);
    const editor = page.locator('#prompt-editor-text');
    await editor.fill('# Base\n\n' + 'x'.repeat(400) + '\n');
    const overflow = await editor.evaluate(el => el.scrollWidth - el.clientWidth);
    expect(overflow).toBeLessThanOrEqual(1);
  });

  // Regression guard, touch devices only: a size-on-rest / size-on-focus
  // split was tried (smaller while reading, 16px while typing, to dodge
  // iOS's auto-zoom threshold) and reverted for being a jarring resize on
  // every tap. The textarea's font size must never change on focus.
  // Regression guard: app.css forces every textarea to 16px on touch devices
  // (iOS auto-zoom prevention), which twice made this element visibly bigger
  // than the rest of the app on an actual phone — first always, then only
  // while focused. voice.css now overrides it back to text-xs unconditionally
  // (with !important, since that is what it takes to beat the other rule's
  // own !important); assert the real computed value, not just "unchanged",
  // so a dropped !important is caught here instead of by a phone screenshot.
  test('the prompt textarea stays at the small text-xs size on a touch device, focused or not', async ({ browser }) => {
    const ctx = await browser.newContext({ ...devices['iPhone 13'] });
    const page = await ctx.newPage();
    await openEditor(page);
    const editor = page.locator('#prompt-editor-text');
    const before = await editor.evaluate(el => getComputedStyle(el).fontSize);
    expect(before).toBe('12px');
    await editor.click();
    const after = await editor.evaluate(el => getComputedStyle(el).fontSize);
    expect(after).toBe('12px');
    await ctx.close();
  });
});
