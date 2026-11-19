import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

const read = name => readFile(new URL(`../static/${name}`, import.meta.url), 'utf8')

test('the boxed-card look is one stylesheet, driven by tokens, loaded by the app shell', async () => {
  const [css, shell] = await Promise.all([read('card-surface.css'), read('index.html')])
  for (const token of ['--card-pad', '--card-radius', '--card-border', '--card-bg', '--card-hover-border', '--card-lift']) assert.match(css, new RegExp(`${token}:`), `${token} is not defined`)
  assert.match(css, /padding: var\(--card-pad\)/)
  assert.match(shell, /card-surface\.css/)
})

test('every grid card it styles still exists in the code that draws it, so a renamed class cannot silently lose the look', async () => {
  const css = await read('card-surface.css')
  const sources = (await Promise.all(['book-cards.js', 'music-group-view.js', 'audio-item-view.js', 'audio-media-views.js', 'audio-library.js'].map(name => read(name).catch(() => '')))).join('\n')
  for (const name of ['music-group-card', 'podcast-card', 'audiobook-series-card']) {
    assert.match(css, new RegExp(`\\.${name}\\b`))
    assert.match(sources, new RegExp(name), `${name} is no longer drawn by any card module`)
  }
})
