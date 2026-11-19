import assert from 'node:assert/strict'
import test from 'node:test'
import { createAudioViewPreferences } from '../static/audio-view-preferences.js'

const element = () => ({ value: '', onchange: null })

test('audio view preferences restore defaults and persist the changed control', () => {
  const values = new Map([['mochil-music-layout', 'list']])
  const storage = { getItem: key => values.get(key) || null, setItem: (key, value) => values.set(key, value) }
  const viewMenu = { open: true }
  const controls = {
    sort: element(), podcastState: element(), podcastLayout: element(), musicLayout: element(),
    audiobookSeriesLayout: element(), audiobookItemLayout: element(),
  }
  let rendered = 0
  createAudioViewPreferences({
    storage, viewMenu, ...controls, activeKind: () => 'music', onSort() {}, onPodcastState() {},
    onPodcastLayout() {}, onMusicLayout: () => { rendered += 1 }, onAudiobookSeriesLayout() {}, onAudiobookItemLayout() {},
  })
  assert.equal(controls.musicLayout.value, 'list')
  assert.equal(controls.podcastState.value, 'all')
  controls.musicLayout.value = 'covers'
  controls.musicLayout.onchange()
  assert.equal(values.get('mochil-music-layout'), 'covers')
  assert.equal(viewMenu.open, false)
  assert.equal(rendered, 1)
})

test('audio sort keys remain isolated by active library kind', () => {
  const values = new Map()
  const storage = { getItem: key => values.get(key) || null, setItem: (key, value) => values.set(key, value) }
  const controls = { sort: element(), podcastState: element(), podcastLayout: element(), musicLayout: element(), audiobookSeriesLayout: element(), audiobookItemLayout: element() }
  const preferences = createAudioViewPreferences({ storage, viewMenu: null, ...controls, activeKind: () => 'podcasts', onSort() {}, onPodcastState() {}, onPodcastLayout() {}, onMusicLayout() {}, onAudiobookSeriesLayout() {}, onAudiobookItemLayout() {} })
  controls.sort.value = 'oldest'
  controls.sort.onchange()
  assert.equal(values.get('mochil-audio-sort-podcasts'), 'oldest')
  assert.equal(preferences.savedSort('podcasts'), 'oldest')
})
