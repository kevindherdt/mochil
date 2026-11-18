import assert from 'node:assert/strict'
import test from 'node:test'
import { mkdtemp, writeFile, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { spawnSync } from 'node:child_process'

const root = fileURLToPath(new URL('../../../', import.meta.url))

test('release gate rejects empty and truncated direct APK archives', async () => {
  const directory = await mkdtemp(join(tmpdir(), 'bobobooks-package-check-'))
  try {
    for (const [name, content] of [
      ['empty-direct.apk', Buffer.alloc(0)],
      ['truncated-direct.apk', Buffer.from('PK\x03\x04truncated')],
    ]) {
      const path = join(directory, name)
      await writeFile(path, content)
      const result = spawnSync('bash', ['scripts/check-store-readiness.sh', path], {
        cwd: root, encoding: 'utf8', timeout: 30000,
      })
      assert.ifError(result.error)
      assert.equal(result.status, 1)
      assert.match(result.stderr, /Package artifact is corrupt or unreadable:/)
    }
  } finally {
    await rm(directory, { recursive: true, force: true })
  }
})
