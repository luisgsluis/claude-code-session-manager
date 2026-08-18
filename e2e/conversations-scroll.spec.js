// CCSM e2e for the conversations list's infinite scroll (replaces the old
// tap-to-load-more button), against the real tmux/claude stubs.
const { test, expect } = require('@playwright/test');
const fs = require('fs');
const path = require('path');

test('conversations list: no load-more button, scrolling near the end loads the next page', async ({ page }) => {
  // The e2e stubs never produce real conversation history, so there's
  // nothing to naturally paginate — this injects a long fake list straight
  // into the Alpine store (conversations.hasMore driving the sentinel) to
  // exercise the real observer/fetch path (initConvInfiniteScroll in app.js).
  await page.goto('/', { waitUntil: 'domcontentloaded' });
  await page.waitForResponse((r) => r.url().includes('/api/conversations?')); // let the real initial load land first

  await expect(page.locator('button', { hasText: 'cargar más' })).toHaveCount(0);

  const convRequests = [];
  page.on('request', (r) => {
    if (r.url().includes('/api/conversations?')) convRequests.push(r.url());
  });

  await page.evaluate(() => {
    const data = window.Alpine.$data(document.body);
    data.conversations.items = Array.from({ length: 20 }, (_, i) => ({
      id: 'c' + i, date: '01/01/2026', origin: 'pc', title: 'conv ' + i,
      preview: 'preview ' + i, tags: [], is_alive: false, pinned: false,
    }));
    data.conversations.hasMore = true;
    data.conversations.page = 1;
  });

  const sentinel = page.locator('[x-ref="convSentinelList"]');
  await expect(sentinel).toBeVisible();
  await expect(sentinel).toContainText('desliza para ver más'); // the hidden card-view sentinel has the same text

  await sentinel.scrollIntoViewIfNeeded();
  await expect(async () => {
    expect(convRequests.some((u) => u.includes('page=2'))).toBe(true);
  }).toPass({ timeout: 2000 });
});

// The web version must load more sessions by scrolling just like the touch
// one. Where the test above drives the observer with an injected store, this
// one seeds >20 REAL transcripts so /api/conversations genuinely paginates and
// asserts the rendered list grows past the first page.
test('conversations list: real paginated data grows as you scroll (web)', async ({ page }) => {
  // The server reads the conversations dir per request, so seeding here is
  // picked up on reload (the stubs never produce conversation history).
  const convDir = path.join(__dirname, 'state', 'conversations');
  fs.mkdirSync(convDir, { recursive: true });
  const seeded = [];
  for (let i = 0; i < 25; i++) {
    const uuid = `20000000-0000-4000-8000-${String(i).padStart(12, '0')}`;
    seeded.push(uuid);
    const ts = '2026-08-' + String((i % 28) + 1).padStart(2, '0') + 'T10:00:00+02:00';
    fs.writeFileSync(
      path.join(convDir, uuid + '.jsonl'),
      JSON.stringify({ type: 'user', cwd: '/home/admin', timestamp: ts, message: { id: 'm' + i, content: 'sesión real ' + i } }) + '\n'
    );
  }
  try {
    const convRequests = [];
    page.on('request', (r) => { if (r.url().includes('/api/conversations?')) convRequests.push(r.url()); });

    await page.goto('/', { waitUntil: 'domcontentloaded' });
    await page.waitForResponse((r) => r.url().includes('/api/conversations?')); // initial page 1

    // First page of 20 rendered, sentinel ready to fetch the next one. The
    // count is scoped to the list view: other overlays (modals) share the
    // bg-bg-card class but sit outside it.
    const cards = page.locator('div[x-show="viewMode === \'list\'"] div.bg-bg-card');
    await expect(async () => { expect(await cards.count()).toBe(20); }).toPass({ timeout: 4000 });
    await expect(page.locator('[x-ref="convSentinelList"]')).toContainText('desliza para ver más');

    // Scrolling the page to the bottom intersects the sentinel → page 2 loads
    // and the list actually grows to all 25, then the sentinel disappears.
    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
    await expect(async () => {
      expect(convRequests.some((u) => u.includes('page=2'))).toBe(true);
    }).toPass({ timeout: 3000 });
    await expect(async () => { expect(await cards.count()).toBe(25); }).toPass({ timeout: 3000 });
    await expect(page.locator('[x-ref="convSentinelList"]')).not.toBeVisible();
  } finally {
    for (const uuid of seeded) fs.rmSync(path.join(convDir, uuid + '.jsonl'), { force: true });
  }
});
