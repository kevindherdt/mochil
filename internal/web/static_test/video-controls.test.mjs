import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

test('the video template carries every control the module drives', async () => {
  const { videoPlayerHTML } = await import('../static/video-player-template.js')
  for (const id of ['video-seek', 'video-play', 'video-centre-play', 'video-back', 'video-forward', 'video-elapsed', 'video-total', 'video-volume', 'video-mute', 'video-fullscreen']) {
    assert.match(videoPlayerHTML, new RegExp(`id="${id}"`))
  }
  assert.doesNotMatch(videoPlayerHTML, /<video[^>]* controls/)
})

test('the controls stylesheet is loaded with the video player', async () => {
  const controller = await readFile(new URL('../static/deferred-video-controller.js', import.meta.url), 'utf8')
  assert.match(controller, /video-controls\.css/)
})
