import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';

// The tab strip used to scroll the chosen tab into view. Settings is an index of
// cards now, so a page replaces the index and the screen has to start at the top
// of the view rather than where the old page was scrolled to.
test('opening a settings page starts the view at its top', async () => {
  const source = await readFile(new URL('../static/settings.js', import.meta.url), 'utf8');
  assert.match(source, /settingsView\.scrollIntoView\?\.\(\{\s*block:\s*["']start["']\s*\}\)/);
});

test('the index itself is what Settings shows without a remembered page', async () => {
  const source = await readFile(new URL('../static/settings.js', import.meta.url), 'utf8');
  assert.match(source, /else showSettingsIndex\(\)/);
  assert.doesNotMatch(source, /inline:\s*["']start["']/, 'nothing scrolls sideways any more');
});
