import test from 'node:test';
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';

test('release update smoke rejects the personal port before touching files or network', () => {
  const script = new URL('../../../scripts/android-update-smoke.mjs', import.meta.url);
  const result = spawnSync(process.execPath, [script.pathname, 'http://127.0.0.1:8099',
    '/nonexistent-release-fixture.apk', '0.16.83', '244'], { encoding: 'utf8', timeout: 5000 });
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /Never target the personal server/);
  assert.doesNotMatch(result.stderr, /ENOENT|fetch failed/);
});
