// Read-only compatibility gate. Does not build, sign, copy or publish artifacts.
// Usage: node scripts/verify-android-update-artifact.mjs CANDIDATE PREVIOUS VERSION CODE BUILD_TOOLS_DIR
import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import { statSync } from 'node:fs'
import { join, resolve } from 'node:path'

const [candidate, previous, version, code, buildTools] = process.argv.slice(2)
assert(candidate && previous && version && /^\d+$/.test(code || '') && buildTools,
  'Expected CANDIDATE PREVIOUS VERSION CODE BUILD_TOOLS_DIR')
assert.notEqual(resolve(candidate), resolve(previous), 'Candidate must not replace the previous archive')
assert(Number.isSafeInteger(Number(code)) && Number(code) > 0, 'Expected a positive safe version code')

function inspect(apk) {
  assert(statSync(apk).isFile(), `Not an APK file: ${apk}`)
  const options = { encoding: 'utf8', maxBuffer: 4 * 1024 * 1024, timeout: 60_000 }
  const badging = execFileSync(join(buildTools, 'aapt2'), ['dump', 'badging', apk], options)
  const line = badging.split('\n').find(value => value.startsWith('package:')) || ''
  const field = name => line.match(new RegExp(`\\b${name}='([^']*)'`))?.[1]
  const signatures = execFileSync(join(buildTools, 'apksigner'), ['verify', '--print-certs', apk], options)
  const certificates = [...signatures.matchAll(/certificate SHA-256 digest: ([a-f0-9]{64})/gi)]
    .map(match => match[1].toLowerCase())
  assert(certificates.length > 0, `No verified signing certificate: ${apk}`)
  const result = { package: field('name'), version: field('versionName'), code: Number(field('versionCode')),
    certificates: [...new Set(certificates)].sort() }
  assert(result.package && result.version && Number.isSafeInteger(result.code) && result.code > 0,
    `Unreadable APK identity: ${apk}`)
  return result
}

const old = inspect(previous)
const next = inspect(candidate)
assert.equal(next.package, old.package, 'Package identity changed')
assert.equal(next.version, version, 'Candidate version name differs from release metadata')
assert.equal(next.code, Number(code), 'Candidate version code differs from release metadata')
assert(next.code > old.code, 'Update version code must increase')
assert.deepEqual(next.certificates, old.certificates, 'Signing identity changed; do not publish as an in-place update')
console.log(`Update compatibility verified: ${next.package} ${old.version}/${old.code} -> ${next.version}/${next.code}`)
console.log(`Certificate SHA-256: ${next.certificates.join(', ')}`)
