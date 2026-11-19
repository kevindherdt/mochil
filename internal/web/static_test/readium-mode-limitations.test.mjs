import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

test('scroll mode is not presented as a continuous webtoon mode', async () => {
  const html = await readFile(new URL('../static/reader-readium.html', import.meta.url), 'utf8')
  assert.match(html, /Scroll is not a continuous webtoon mode/i)
  assert.match(html, /fixed-layout/i)
})
