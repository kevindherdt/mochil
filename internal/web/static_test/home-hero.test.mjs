import assert from 'node:assert/strict'
import test from 'node:test'
import { heroHTML, heroRemaining } from '../static/home-hero.js'

const escapeHTML = (value) => String(value).replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]))
const watchedLabel = (p) => (p?.position_seconds ? '12m in' : '')
const item = {
  id: 'abc',
  library_id: 'movies',
  title: 'When They Cry Higurashi (2006) - S01E05 - 005 - The Cotton Drifting [Bluray-576p][x265]-Okay-Subs',
  artwork_url: '/art/1.jpg',
  progress: { position_seconds: 1200, duration_seconds: 1500 },
}

test('the hero names the thing you are in the middle of, cleaned of its release tags', () => {
  const html = heroHTML(item, { escapeHTML, watchedLabel })
  assert.match(html, /Continue watching/)
  assert.match(html, /Higurashi \(2006\)/)
  assert.doesNotMatch(html, /Bluray|x265|Okay-Subs/)
})

test('the meta line says how far in and how long is left', () => {
  const html = heroHTML(item, { escapeHTML, watchedLabel })
  assert.match(html, /12m in · 5m left of 25m/)
})

test('Resume carries the item so the button can open the player', () => {
  const html = heroHTML(item, { escapeHTML, watchedLabel })
  assert.match(html, /data-item="abc"/)
  assert.match(html, /data-library="movies"/)
})

test('no item, no hero -- Home must not open with an empty card', () => {
  assert.equal(heroHTML(null, { escapeHTML, watchedLabel }), '')
  assert.equal(heroHTML({ title: '' }, { escapeHTML, watchedLabel }), '')
})

test('a length we do not know is left out rather than invented', () => {
  assert.equal(heroRemaining({ position_seconds: 10, duration_seconds: 0 }), '')
  assert.equal(heroRemaining({ position_seconds: 100, duration_seconds: 100 }), '')
  assert.equal(heroRemaining(undefined), '')
})

test('a nearly finished item still says one minute, not zero', () => {
  assert.equal(heroRemaining({ position_seconds: 1490, duration_seconds: 1500 }), '1m left of 25m')
})

test('barely started says the length instead of "1h 48m left of 1h 48m"', () => {
  assert.equal(heroRemaining({ position_seconds: 20, duration_seconds: 6480 }), '1h 48m')
})

test('a film longer than an hour reads as hours and minutes', () => {
  assert.equal(heroRemaining({ position_seconds: 600, duration_seconds: 6480 }), '1h 38m left of 1h 48m')
})
