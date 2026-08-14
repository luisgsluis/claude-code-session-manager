// CCSM e2e for the conversations list's infinite scroll (replaces the old
// tap-to-load-more button), against the real tmux/claude stubs.
const { test, expect } = require('@playwright/test');

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
