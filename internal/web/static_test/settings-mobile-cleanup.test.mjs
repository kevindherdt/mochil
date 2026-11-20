import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
const css = readFileSync(new URL('../static/settings-cards.css', import.meta.url), 'utf8');
const html = readFileSync(new URL('../static/index.html', import.meta.url), 'utf8');

test('the editor menu card and its switch are gone', () => {
  // It held one per-device switch. The editor is a normal page in the menu now, so there is no card, no
  // switch and no rule hiding an empty card.
  assert.doesNotMatch(html, /editor-menu-card/);
  assert.doesNotMatch(html, /editor-menu-toggle/);
  assert.doesNotMatch(css, /editor-menu-card/);
});

test('empty plugin status does not reserve space and module options are not nested cards', () => {
  assert.match(css, /#plugin-status:empty\s*\{display:none\}/);
  assert.match(css, /#settings-view \.module-row\.module-sub\s*\{[^}]*border:0/);
  assert.match(css, /#settings-view \.module-row\.module-sub\s*\{[^}]*background:transparent/);
});

test('phone voice filters remain wrapped and compact without hiding controls', () => {
  assert.match(css, /@media\s*\(max-width:700px\)[\s\S]*?\.nv-chips\s*\{[^}]*gap:4px/);
});
