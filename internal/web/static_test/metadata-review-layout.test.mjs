import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

const css = await readFile(new URL('../static/metadata-review.css', import.meta.url), 'utf8')
const controller = await readFile(new URL('../static/metadata-review.js', import.meta.url), 'utf8')

test('desktop metadata review uses an adaptive two-column canvas', () => {
  assert.match(css, /max-width:\s*1180px/)
  assert.match(css, /grid-template-columns:\s*repeat\(2,\s*minmax\(0,\s*1fr\)\)/)
  assert.match(css, /article:has\(details\[open\]\)/)
})

test('switching metadata modes removes publication-only batch actions', () => {
  const mode = controller.slice(controller.indexOf('function updateMetadataMode()'), controller.indexOf('function bind()'))
  assert.match(mode, /querySelector\('#metadata-batch'\)\?\.remove\(\)/)
})
