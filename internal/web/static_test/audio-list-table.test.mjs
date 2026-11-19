import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

test('music rows carry the table columns and the stylesheet is loaded with the audio player', async () => {
  const [view, controller] = await Promise.all(['audio-item-view.js', 'deferred-audio-controller.js'].map(name => readFile(new URL(`../static/${name}`, import.meta.url), 'utf8')))
  for (const column of ['audio-col-artist', 'audio-col-album', 'audio-col-time']) assert.match(view, new RegExp(column))
  assert.match(controller, /audio-table\.css/)
})
