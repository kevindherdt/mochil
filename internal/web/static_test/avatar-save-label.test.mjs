import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';

test('avatar save identifies its independent action', () => {
  const source = readFileSync(new URL('../static/avatar-settings.js', import.meta.url), 'utf8');
  assert.ok(/id="avatar-save"[^>]*>Save avatar<\/button>/.test(source));
});
