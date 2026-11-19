import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

const source = await readFile(new URL('../static/audio-library.js', import.meta.url), 'utf8')

test('music groups and alphabet navigation share one visual commit', () => {
  assert.doesNotMatch(source, /void loadMusicNavigation\(\)/)
  assert.match(source, /Promise\.all\(\[loadMusicNavigation\(false\),loadMusicGroups\(true,requestedOffset,false\)\]\)/)
  assert.match(source, /if\(musicDiscovery\.isCurrent\(ticket\)&&\(navigationChanged\|\|groupsChanged\)\)render\(\)/)
})

test('track navigation settles before the track surface is committed', () => {
  assert.match(source, /pendingNavigation=loadMusicNavigation\(false\)/)
  assert.match(source, /if\(pendingNavigation\)await pendingNavigation;if\(request!==current\)return;render/)
})
