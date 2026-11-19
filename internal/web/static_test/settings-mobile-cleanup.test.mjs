import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
const css = readFileSync(new URL('../static/settings-cards.css', import.meta.url), 'utf8');
const html = readFileSync(new URL('../static/index.html', import.meta.url), 'utf8');

test('editor menu card follows actual availability without removing its toggle', () => {
  assert.match(html, /class="system-card editor-menu-card"/);
  assert.match(html, /id="editor-menu-toggle" hidden/);
  assert.match(css, /\.editor-menu-card:not\(:has\(#editor-menu-toggle:not\(\[hidden\]\)\)\)\s*\{display:none\}/);
  assert.match(css, /#editor-menu-toggle\[hidden\]\s*\{display:none\}/);
});

test('empty plugin status does not reserve space and module options are not nested cards', () => {
  assert.match(css, /#plugin-status:empty\s*\{display:none\}/);
  assert.match(css, /#settings-view \.module-row\.module-sub\s*\{[^}]*border:0/);
  assert.match(css, /#settings-view \.module-row\.module-sub\s*\{[^}]*background:transparent/);
});

test('phone voice filters remain wrapped and compact without hiding controls', () => {
  assert.match(css, /@media\s*\(max-width:700px\)[\s\S]*?\.nv-chips\s*\{[^}]*gap:4px/);
});
