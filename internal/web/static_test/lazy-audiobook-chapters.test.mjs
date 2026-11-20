import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import { createLazyAudiobookChapters } from '../static/lazy-audiobook-chapters.js'

test('Music keeps audiobook chapter DOM bindings behind a lazy facade', async () => {
  const [library, facade] = await Promise.all([
    readFile(new URL('../static/audio-library.js', import.meta.url), 'utf8'),
    readFile(new URL('../static/lazy-audiobook-chapters.js', import.meta.url), 'utf8'),
  ])
  assert.doesNotMatch(library, /from '\.\/audiobook-chapters\.js/)
  assert.match(library, /createLazyAudiobookChapters/)
  assert.match(facade, /import\('\.\/audiobook-chapters\.js\?v=2'\)/)
  assert.match(facade, /item\?\.library_kind !== 'audiobooks'/)
  assert.match(facade, /requestedGeneration === generation/)
})

test('the first chaptered audiobook mounts and renders the deferred controller', async () => {
  const elements = {
    '#audio-chapters': { hidden: true, contains: () => false, addEventListener() {} },
    '#audio-chapters-toggle': { hidden: true, onclick: null, setAttribute() {} },
    '#audio-chapters-close': { onclick: null },
    '#audio-chapter-list': {
      innerHTML: '',
      onclick: null,
      querySelectorAll: () => [],
      querySelector: () => null,
    },
  }
  const chapters = createLazyAudiobookChapters({
    $: selector => elements[selector],
    audio: { currentTime: 0, play: async () => {} },
    player: { ownerDocument: {activeElement:null}, querySelector: () => ({ open: true }) },
    getActive: () => null,
    clock: value => `${value}:00`,
    escapeHTML: value => value,
    expandPlayer: () => {},
    updateTimeline: () => {},
  })

  chapters.openItem({ library_kind: 'audiobooks', chapters: [{ title: 'Opening', start_seconds: 0, end_seconds: 60 }] })
  for (let attempt = 0; attempt < 20 && elements['#audio-chapters-toggle'].hidden; attempt++) {
    await new Promise(resolve => setTimeout(resolve, 1))
  }
  assert.equal(elements['#audio-chapters-toggle'].hidden, false)
  assert.match(elements['#audio-chapter-list'].innerHTML, /Opening/)
})
