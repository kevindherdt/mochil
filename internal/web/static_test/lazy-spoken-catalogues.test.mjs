import assert from 'node:assert/strict'
import test from 'node:test'
import { createLazyAudiobookCatalogue, createLazyPodcastCatalogue } from '../static/lazy-spoken-catalogues.js'

test('spoken catalogues remain inert until their first media-specific request', () => {
  const podcast = createLazyPodcastCatalogue(() => { throw new Error('not called') })
  const audiobook = createLazyAudiobookCatalogue(() => { throw new Error('not called') }, 120)
  assert.deepEqual({ show: podcast.show, episodes: podcast.episodes, loading: podcast.loading }, { show: '', episodes: [], loading: false })
  assert.deepEqual({ group: audiobook.group, books: audiobook.books, loading: audiobook.loading }, { group: '', books: [], loading: false })
  podcast.reset()
  audiobook.abort()
})

test('resetting a lazy catalogue invalidates a request waiting for its module', async () => {
  const requests = []
  const catalogue = createLazyPodcastCatalogue(value => requests.push(value))
  const pending = catalogue.loadShow('Example', { library: 'podcasts' })
  catalogue.reset()
  assert.equal(await pending, false)
  assert.deepEqual(requests, [])
})
