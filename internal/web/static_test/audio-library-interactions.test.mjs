import test from 'node:test'
import assert from 'node:assert/strict'
import { bindAudioLibraryInteractions, discoveryGroup } from '../static/audio-library-interactions.js'

function harness(overrides = {}) {
  const capture = []
  const container = {
    onclick: null,
    addEventListener(_type, handler, enabled) { if (enabled) capture.push(handler) },
  }
  const calls = []
  bindAudioLibraryInteractions({
    container,
    playlistController: { handleClick: async () => false, handleItemAction: async () => {}, setSelected: value => calls.push(['playlist', value]) },
    musicCatalogue: { findGroup: () => null }, podcastCatalogue: { closeShow() {} },
    getItems: () => [], findItem: () => null, groupLabel: () => '', getActive: () => null,
    getKind: () => 'music', getBrowse: () => 'queue', getDiscovery: () => ({}),
    openMusicGroup: async () => {}, closeMusicGroup: async () => {}, loadPodcastShow: async () => {},
    loadAudiobookGroup: async () => {}, loadAudiobookRoot: async () => {}, markPodcastPlayed: async () => {},
    open: () => {}, setQueue: () => {}, setShuffle: () => {}, orderedQueue: () => [], playNext: async () => {},
    addToQueue: async () => {}, toggleFavorite: async () => {}, openTrackInfo: () => {}, clearQueue: () => calls.push(['clear']),
    removeFromQueue: item => calls.push(['remove', item.id]), moveInQueue: () => {}, render: () => calls.push(['render']),
    confirmClear: () => true, selectBrowse: value => calls.push(['browse', value]), selectSort: value => calls.push(['sort', value]), selectPlaylist: value => calls.push(['playlist', value]),
    reload: async () => {}, findDiscoveryGroup: discoveryGroup,
    ...overrides,
  })
  return { container, capture, calls }
}

function eventFor(matches) {
  return {
    stopped: false,
    target: { closest: selector => matches[selector] || null },
    stopImmediatePropagation() { this.stopped = true },
  }
}

test('queue clear is captured once before ordinary audio actions', () => {
  const { capture, calls } = harness()
  const event = eventFor({ '[data-clear-queue]': {} })
  capture[0](event)
  assert.equal(event.stopped, true)
  assert.deepEqual(calls, [['clear'], ['render']])
})

test('queue removal uses the matched item and rerenders', () => {
  const item = { id: 'track-7' }
  const action = { dataset: { action: 'queue-remove' } }
  const { capture, calls } = harness({ findItem: () => item })
  const event = eventFor({ '[data-action]': action })
  capture[0](event)
  assert.deepEqual(calls, [['remove', 'track-7'], ['render']])
})

test('discovery group lookup remains null safe', () => {
  const album = { name: 'After Laughter' }
  assert.equal(discoveryGroup({ albums: [album] }, 'albums', album.name), album)
  assert.equal(discoveryGroup({}, 'albums', album.name), null)
})

test('recently added opens the complete track shelf in recent order', async () => {
  const { capture, calls } = harness()
  const destination = { dataset: { discoveryDestination: 'tracks', discoverySort: 'recent' } }
  const event = eventFor({ '[data-discovery-destination]': destination })

  await capture[1](event)

  assert.equal(event.stopped, true)
  assert.deepEqual(calls, [['browse', 'tracks'], ['sort', 'recent']])
})
