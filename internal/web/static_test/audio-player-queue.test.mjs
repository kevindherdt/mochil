import assert from 'node:assert/strict'
import test from 'node:test'
import { createAudioPlayerQueue, queuePage } from '../static/audio-player-queue.js'

const key = item => item.id
const tracks = Array.from({ length: 140 }, (_, index) => ({ id: String(index), title: `Track ${index}` }))

test('expanded player queue stays bounded and centers the active track', () => {
  const page = queuePage(tracks, tracks[100], key)
  assert.equal(page.items.length, 60)
  assert.equal(page.offset, 70)
  assert.equal(page.activeIndex, 100)
  assert.equal(page.items[30], tracks[100])
})

test('expanded player queue clamps explicit pages at both edges', () => {
  assert.deepEqual(queuePage(tracks, tracks[100], key, -500, false).items, tracks.slice(0, 60))
  const last = queuePage(tracks, tracks[100], key, 999, false)
  assert.equal(last.offset, 80)
  assert.deepEqual(last.items, tracks.slice(80))
  assert.equal(last.activeIndex, 100)
})

function fixture() {
  const document = { activeElement: null }
  const nodes = new Map()
  function element(dataset = {}) {
    return {
      dataset, hidden: false, ownerDocument: document,
      setAttribute() {},
      focus() { document.activeElement = this },
    }
  }
  const $ = selector => {
    if (!nodes.has(selector)) nodes.set(selector, element())
    return nodes.get(selector)
  }
  const list = $('#audio-queue-items')
  let rows = []
  Object.defineProperty(list, 'innerHTML', { set(html) {
    if (rows.includes(document.activeElement)) document.activeElement = null
    rows = [...html.matchAll(/data-queue-index="(\d+)"([^>]*)/g)].map(match => {
      const row = element({ queueIndex: match[1] })
      row.current = match[2].includes('aria-current')
      return row
    })
  } })
  list.contains = node => rows.includes(node)
  list.querySelector = selector => selector === '[aria-current="true"]'
    ? rows.find(row => row.current)
    : selector === '[data-queue-index]' ? rows[0]
      : rows.find(row => selector === `[data-queue-index="${row.dataset.queueIndex}"]`)
  const panel = $('#audio-player-queue')
  panel.hidden = true
  panel.querySelector = list.querySelector
  const queue = createAudioPlayerQueue({ $, key, escapeHTML: value => value, onSelect() {} })
  const current = { ...tracks[0], library_kind: 'music' }
  queue.sync(tracks, current)
  $('#audio-queue-toggle').onclick()
  return { $, queue, document, current, focusRow: index => list.querySelector(`[data-queue-index="${index}"]`).focus() }
}

test('queue refresh preserves keyboard focus by track identity after reordering', () => {
  const f = fixture()
  f.focusRow(4)
  const reordered = [tracks[4], ...tracks.filter(item => item !== tracks[4])]
  f.queue.sync(reordered, f.current)
  assert.equal(f.document.activeElement.dataset.queueIndex, '0')
})

test('queue removal focuses the nearby row, then close when the queue empties', () => {
  const f = fixture()
  f.focusRow(4)
  f.queue.sync(tracks.filter(item => item !== tracks[4]), f.current)
  assert.equal(f.document.activeElement.dataset.queueIndex, '4')
  f.queue.sync([], f.current)
  assert.equal(f.document.activeElement, f.$('#audio-queue-close'))
})

test('queue refresh does not steal focus from controls outside the track list', () => {
  const f = fixture()
  f.$('#audio-queue-later').focus()
  f.queue.sync(tracks, f.current)
  assert.equal(f.document.activeElement, f.$('#audio-queue-later'))
})

test('opening an empty queue focuses close and Escape restores the toggle', () => {
  const f = fixture()
  f.queue.close()
  f.queue.sync([], f.current)
  f.$('#audio-queue-toggle').onclick()
  assert.equal(f.document.activeElement, f.$('#audio-queue-close'))
  let prevented = false
  f.$('#audio-player-queue').onkeydown({ key: 'Escape', preventDefault() { prevented = true } })
  assert.equal(prevented, true)
  assert.equal(f.document.activeElement, f.$('#audio-queue-toggle'))
})
