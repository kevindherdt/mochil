import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

const audioSource = await readFile(new URL('../static/audio-library.js', import.meta.url), 'utf8')
const audioContinueSource = await readFile(new URL('../static/audio-continue-rail.js', import.meta.url), 'utf8')
const videoSource = await readFile(new URL('../static/video-library.js', import.meta.url), 'utf8')
const videoContinueSource = await readFile(new URL('../static/video-continue-rail.js', import.meta.url), 'utf8')

test('explicit audio status actions patch only the requested field', () => {
  assert.match(audioSource, /method:'PATCH'.*JSON\.stringify\(\{finished:played\}\)/s)
  assert.match(audioContinueSource, /method: 'PATCH'.*JSON\.stringify\(\{ dismissed: true \}\)/s)
  assert.doesNotMatch(audioContinueSource, /JSON\.stringify\(\{\s*\.\.\.item\.progress,\s*dismissed:\s*true\s*\}\)/)
})

test('Home listening is reserved for long-form audio', () => {
  assert.match(audioContinueSource, /item\.library_kind !== ['"]music['"]/)
})

test('continue-watching dismissal does not resend stale playback position', () => {
  assert.match(videoContinueSource, /method:'PATCH'[\s\S]*body:JSON\.stringify\(\{ dismissed \}\)/)
  assert.doesNotMatch(videoContinueSource, /JSON\.stringify\(\{ \.\.\.item\.progress, dismissed \}\)/)
  assert.doesNotMatch(videoSource, /api\/video-progress/)
})
