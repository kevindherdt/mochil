// Run only against a disposable candidate server with synthetic credentials.
// Usage: node scripts/android-update-smoke.mjs URL APK VERSION CODE
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { createReadStream } from 'node:fs';
import { stat } from 'node:fs/promises';

const [address, apk, version, code] = process.argv.slice(2);
assert(address && apk && version && /^\d+$/.test(code || ''), 'Expected URL APK VERSION CODE');
const base = new URL(address);
assert(base.protocol === 'http:' && base.hostname === '127.0.0.1' && base.port,
  'Use an explicit loopback HTTP port for a disposable fixture');
assert(base.port !== '8099', 'Never target the personal server');
assert(!base.username && !base.password && base.pathname === '/' && !base.search && !base.hash,
  'Expected a bare fixture origin');
const info = await stat(apk);
assert(info.isFile() && info.size > 0, 'Expected a nonempty APK file');
const localHash = createHash('sha256');
for await (const chunk of createReadStream(apk)) localHash.update(chunk);
const expectedHash = localHash.digest('hex');

async function request(path, options = {}) {
  return fetch(new URL(path, base), {
    ...options, redirect: 'error', signal: AbortSignal.timeout(30_000),
  });
}

// These are intentionally public fixture credentials, never live credentials.
const basic = Buffer.from('release-smoke:temporary-release-fixture-only').toString('base64');
const login = await request('/api/android/v1/login', {
  method: 'POST',
  headers: { Authorization: `Basic ${basic}`, 'Content-Type': 'application/json' },
  body: JSON.stringify({ device_name: 'Release verification fixture' }),
});
assert.equal(login.status, 201, 'Fixture device login failed');
const session = await login.json();
assert.equal(typeof session.device_token, 'string');
assert(session.device_token.length > 0);
const headers = { Authorization: `Bearer ${session.device_token}` };
const metadataResponse = await request('/api/android/v1/app', { headers });
assert.equal(metadataResponse.status, 200);
const metadata = await metadataResponse.json();
assert.equal(metadata.version_name, version);
assert.equal(metadata.version_code, Number(code));
assert.equal(metadata.download_url, '/api/android/v1/app.apk');
for (const category of ['fixes', 'performance', 'improvements']) {
  assert(Array.isArray(metadata.release_notes?.[category]));
  assert(metadata.release_notes[category].every(note => typeof note === 'string' && note.length > 0));
}
const download = await request(metadata.download_url, { headers });
assert.equal(download.status, 200);
assert.equal(download.headers.get('X-BoboBooks-App-Version'), version);
assert.equal(download.headers.get('Content-Type'), 'application/vnd.android.package-archive');
const receivedHash = createHash('sha256');
let receivedBytes = 0;
for await (const chunk of download.body) {
  receivedBytes += chunk.length;
  assert(receivedBytes <= info.size, 'Download exceeds the candidate APK size');
  receivedHash.update(chunk);
}
assert.equal(receivedBytes, info.size);
assert.equal(receivedHash.digest('hex'), expectedHash);
const head = await request(metadata.download_url, { method: 'HEAD', headers });
assert.equal(head.status, 200);
assert.equal(Number(head.headers.get('Content-Length')), info.size);
assert.equal((await head.arrayBuffer()).byteLength, 0);
assert.equal(head.headers.get('ETag'), download.headers.get('ETag'));
// Update clients must be able to resume the same published file without receiving
// a login document or an entire second APK in place of the requested suffix.
const rangeStart = Math.max(0, info.size - 256);
const range = await request(metadata.download_url, {
  headers: { ...headers, Range: `bytes=${rangeStart}-` },
});
assert.equal(range.status, 206);
assert.equal(range.headers.get('Content-Range'), `bytes ${rangeStart}-${info.size - 1}/${info.size}`);
const suffix = [];
for await (const chunk of createReadStream(apk, { start: rangeStart })) suffix.push(chunk);
assert.deepEqual(Buffer.from(await range.arrayBuffer()), Buffer.concat(suffix));
console.log(`Authenticated update verified: ${version}/${code}, ${receivedBytes} bytes, SHA-256 ${expectedHash}`);
