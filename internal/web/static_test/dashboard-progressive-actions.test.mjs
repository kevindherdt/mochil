import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

const css = await readFile(new URL('../static/dashboard.css', import.meta.url), 'utf8')

test('dashboard removal actions use progressive disclosure only for fine pointers', () => {
  assert.match(css, /@media\(hover:hover\) and \(pointer:fine\)/)
  for (const selector of ['.reading-dismiss', '.video-reading-dismiss', '.audio-dismiss']) {
    assert.match(css, new RegExp(selector.replace('.', '\\.') + '[^}]*\\)|' + selector.replace('.', '\\.')))
  }
  assert.match(css, /:is\(:hover,:focus-within\)/)
  assert.match(css, /opacity:0/)
  assert.match(css, /opacity:1/)
})

test('dashboard removal disclosure respects reduced motion', () => {
  assert.match(css, /@media\(prefers-reduced-motion:reduce\)/)
  assert.match(css, /transition:none/)
})
