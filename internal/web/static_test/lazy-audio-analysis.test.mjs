import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import { createLazyAudioAnalysis } from '../static/lazy-audio-analysis.js'

const element = () => ({ hidden: true, disabled: false, textContent: '', addEventListener(type, handler) { this[type] = handler } })

test('ordinary shelf navigation exposes the tool without requesting analysis data', () => {
  const container = element(), button = element(), status = element()
  let requests = 0
  const analysis = createLazyAudioAnalysis({ container, button, status, requestJSON: async () => { requests += 1 } })
  analysis.setContext('Music', 'music')
  assert.equal(container.hidden, false)
  assert.equal(button.textContent, 'Check loudness')
  assert.equal(requests, 0)
  analysis.setContext('Books', 'books')
  assert.equal(container.hidden, true)
  assert.equal(requests, 0)
})

test('the ordinary audio graph has no static analysis-controller dependency', async () => {
  const source = await readFile(new URL('../static/audio-library.js', import.meta.url), 'utf8')
  const lazySource = await readFile(new URL('../static/lazy-audio-analysis.js', import.meta.url), 'utf8')
  assert.doesNotMatch(source, /from '.\/audio-analysis-controller\.js/)
  assert.match(source, /createLazyAudioAnalysis/)
  assert.match(lazySource, /import\('\.\/audio-analysis-controller\.js/)
})
