// e2e for voice dictation and prompt rewriting.
//
// The provider is stubbed (e2e/stubs/voice-provider.js) but everything else is
// real: a real MediaRecorder recording a real (fake) audio device, a real
// upload, the real Go handlers, and the real review panel. Only the model at
// the far end is fake, which is the minimum that can be faked.

const { test, expect } = require('@playwright/test');

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

  test('clarifying questions can be answered', async ({ page }) => {
    await rewriteInChat(page, 'esto es AMBIGUO del todo');

    const questions = page.locator('.voice-question');
    await expect(questions).toHaveCount(2);
    await expect(page.locator('textarea.voice-compose-text')).toHaveValue(/PROMPT PROVISIONAL/);

    await questions.first().locator('input').fill('el de la Pi');
    await page.locator('.voice-compose-panel button:has-text("Responder")').click();

    // The second pass folds the answer in and asks nothing further.
    await expect(page.locator('.voice-question')).toHaveCount(0, { timeout: 10000 });
    await expect(page.locator('textarea.voice-compose-text')).toHaveValue(/el de la Pi/);
  });

  test('clarifying questions can be skipped', async ({ page }) => {
    await rewriteInChat(page, 'esto es AMBIGUO del todo');
    await expect(page.locator('.voice-question')).toHaveCount(2);
    await page.locator('.voice-compose-panel button:has-text("Omitir")').click();
    await expect(page.locator('.voice-question')).toHaveCount(0);
    // The provisional prompt is still usable after skipping.
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

  test('opens with the shipped prompt and lists its sections', async ({ page }) => {
    await openEditor(page);
    const content = await page.locator('#prompt-editor-text').inputValue();
    expect(content).toContain('# Base');
    expect(content).toContain('# Role: devops');
    // Marked as the project original until it is edited.
    await expect(page.locator('.voice-editor-panel span:has-text("original del proyecto")')).toBeVisible();
  });

  // The validation that stops a role reaching the model with no instructions.
  test('rejects a prompt whose role has no section, and keeps the old one', async ({ page }) => {
    await openEditor(page);
    const editor = page.locator('#prompt-editor-text');
    await editor.fill('---\nroles:\n  - {id: devops, es: D, en: D}\n---\n# Base\n\nsin bloque\n');
    await page.locator('.voice-editor-panel button:has-text("Guardar")').click();

    // The error names exactly what is wrong, because the user has to fix it.
    await expect(page.locator('text=/# Role: devops/')).toBeVisible({ timeout: 10000 });

    // Reopening shows the previous prompt: nothing was written.
    await page.reload();
    await openEditor(page);
    expect(await page.locator('#prompt-editor-text').inputValue()).toContain('# Role: devops');
  });

  test('saves an edit, then restores the original', async ({ page }) => {
    await openEditor(page);
    const editor = page.locator('#prompt-editor-text');
    const original = await editor.inputValue();

    await editor.fill(original.replace('# Base', '# Base\n\nLINEA DE PRUEBA E2E.'));
    await page.locator('.voice-editor-panel button:has-text("Guardar")').click();
    await expect(page.locator('.voice-editor-panel span:has-text("modificado")')).toBeVisible({ timeout: 10000 });

    page.on('dialog', (d) => d.accept());
    await page.locator('.voice-editor-panel button:has-text("Restaurar original")').click();
    await expect(page.locator('.voice-editor-panel span:has-text("original del proyecto")')).toBeVisible({ timeout: 10000 });
    expect(await editor.inputValue()).not.toContain('LINEA DE PRUEBA E2E');
  });

  // Roles live in the prompt, so adding one there must reach the dropdown with
  // no rebuild.
  test('a role added to the prompt shows up in the panel dropdown', async ({ page }) => {
    await openEditor(page);
    const editor = page.locator('#prompt-editor-text');
    const original = await editor.inputValue();
    const withRole = original
      .replace('roles:', 'roles:\n  - {id: seguridad, es: Seguridad, en: Security}')
      .replace('# Role: software', '# Role: seguridad\n\nThreat modelling.\n\n# Role: software');

    await editor.fill(withRole);
    await page.locator('.voice-editor-panel button:has-text("Guardar")').click();
    await expect(page.locator('.voice-editor-panel span:has-text("modificado")')).toBeVisible({ timeout: 10000 });

    await page.reload();
    await openChat(page);
    await page.locator('textarea[x-model="live.input"]').fill('algo');
    await page.locator('button[title*="Reescribir"]').click();
    await expect(page.locator('.voice-compose-panel')).toBeVisible({ timeout: 10000 });

    const options = await page.locator('.voice-compose-panel select').first().locator('option').allTextContents();
    expect(options.join(',')).toContain('Seguridad');
  });
});
