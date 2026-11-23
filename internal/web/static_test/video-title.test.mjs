import assert from 'node:assert/strict'
import test from 'node:test'
import { cleanVideoTitle } from '../static/video-title.js'

// Release names are file names: the technical groups are things the player already knows from the file.
test('a scene release name becomes the episode it is', () => {
  const raw = 'When They Cry Higurashi (2006) - S01E05 - 005 - The Cotton Drifting Chapter Part 1 Jealousy [Bluray-576p v2][Opus 2 0][JA][x265 10bit]-Okay-Subs'
  assert.equal(cleanVideoTitle(raw), 'When They Cry Higurashi (2006) - S1 E5 · The Cotton Drifting Chapter Part 1 Jealousy')
})

test('dot-separated names lose their technical tail', () => {
  assert.equal(cleanVideoTitle('Show.Name.S02E11.1080p.WEB-DL.x265-GROUP'), 'Show Name S2 E11')
})

test('a year in brackets is part of the title, a resolution is not', () => {
  assert.equal(cleanVideoTitle('Blade Runner 2049 (2017) [2160p] [HDR]'), 'Blade Runner 2049 (2017)')
})

test('an ordinary title is left alone', () => {
  assert.equal(cleanVideoTitle('Meet the Parents'), 'Meet the Parents')
  assert.equal(cleanVideoTitle('The Matrix'), 'The Matrix')
  assert.equal(cleanVideoTitle(''), '')
})

test('nothing to clean still returns the original rather than an empty card', () => {
  assert.equal(cleanVideoTitle('[1080p]'), '[1080p]')
})
