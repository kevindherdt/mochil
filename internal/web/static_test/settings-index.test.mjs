import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';

const read = (name) => readFileSync(new URL(`../static/${name}`, import.meta.url), 'utf8');
const index = read('index.html');
const css = read('settings-index.css');
const settings = read('settings.js');
const deferred = read('deferred-route-controllers.js');

// Settings used to be a horizontal strip of eight tabs over one very long page.
// Each area is now its own page reached from an index of cards, which is only
// useful if every tab carries the line that says what it is for.
test('every settings page has a card hint, so none lands on the index blank', () => {
  const tabs = [...index.matchAll(/data-settings-tab="([a-z]+)"([\s\S]{0,200}?)>/g)];
  assert.ok(tabs.length >= 8, `expected at least 8 settings tabs, found ${tabs.length}`);
  for (const [, name, attributes] of tabs) {
    assert.match(attributes, /data-hint="[^"]{10,}"/, `${name} has no data-hint`);
  }
});

test('settings opens on an index of cards, not a tab strip', () => {
  assert.match(index, /class="settings-nav settings-index"/);
  assert.match(index, /id="settings-back" hidden/, 'a way back must exist and start hidden');
  assert.match(css, /\.settings-index\s*\{[\s\S]*?grid-template-columns/);
  assert.match(css, /content:\s*attr\(data-hint\)/, 'the hint is what the card shows');
  assert.match(css, /data-settings-page\]:not\(\[data-settings-page="index"\]\)\s*\.settings-index/, 'an open page replaces the index');
});

test('the index and the back button drive the same page state as the tabs', () => {
  assert.match(settings, /settingsPage = "index"/);
  assert.match(settings, /settingsBack\.hidden = false/, 'opening a page reveals the way back');
  assert.match(settings, /settingsBack\.onclick = showSettingsIndex/);
  // Saving belongs to a page with fields, so the index hides the action bar.
  assert.match(settings, /showSettingsIndex[\s\S]*?settings-save"\)\.hidden = true/);
  assert.match(deferred, /\/assets\/settings-index\.css\?v=\d+/, 'the new stylesheet must be loaded on the route');
});
