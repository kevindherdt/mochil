import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { chromium } from 'playwright';

const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_PATH || '/usr/bin/chromium', headless: true });
try {
  const page = await browser.newPage();
  await page.route('http://recovery.test/**', async route => {
    const path = new URL(route.request().url()).pathname;
    if (path === '/library-loader.js' || path === '/view-route.js') {
      await route.fulfill({ contentType: 'text/javascript', body: await readFile(new URL(`../internal/web/static${path}`, import.meta.url), 'utf8') });
    } else await route.fulfill({ contentType: 'text/html', body: '<!doctype html><title>Recovery test</title>' });
  });
  await page.clock.install();
  await page.goto('http://recovery.test/#/library/books/collections/abc');
  await page.evaluate(async () => {
    const { createLibraryLoader } = await import('/library-loader.js');
    const { createViewRoute } = await import('/view-route.js');
    window.result = { attempts: 0, loads: 0, place: null, selected: null };
    let view = 'dashboard', libraries = [];
    const route = createViewRoute({ window, getView: () => view, getLibrary: () => result.selected,
      hasLibrary: id => libraries.some(item => item.id === id), showHome: () => { view = 'dashboard'; },
      showSettings: () => { view = 'settings'; }, showLibrary: (id, place) => { result.selected = id; result.place = place; view = 'library'; result.loads++; } });
    window.loader = createLibraryLoader({
      requestJSON: async () => { if (++result.attempts < 5) throw new Error('offline'); return { data: [{ id: 'books' }] }; },
      readingDashboard: { load() {} }, videoLibrary: {}, audioLibrary: {},
      libraryState: { all: () => libraries, setAll: value => { libraries = value; }, setVisible() {}, initializeSelection() {}, ensureVisible() {} },
      libraryVisibility: { setLibraries: value => value }, renderToolbar() {}, renderAlphabet() {},
      isLibraryView: () => view === 'library', loadCurrentView: () => { result.loads++; }, setStatus() {}, onReady: () => route.restoreInitial(),
      retryDelays: [2000, 5000, 15000]
    });
    await loader.load();
  });
  for (const ms of [2000, 5000, 15000, 15000]) await page.clock.runFor(ms);
  const result = await page.evaluate(() => { loader.dispose(); return window.result; });
  assert.equal(result.attempts, 5);
  assert.equal(result.loads, 1);
  assert.equal(result.selected, 'books');
  assert.equal(result.place.collection, 'abc');
  assert.equal(new URL(page.url()).hash, '#/library/books/collections/abc');
  console.log('Chromium: recovery after four failures preserves route and loads the restored view once.');
} finally { await browser.close(); }
