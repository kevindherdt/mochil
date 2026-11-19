import assert from 'node:assert/strict'
import test from 'node:test'
import { createAudioPlaybackSession } from '../static/audio-playback-session.js'

test('a newly opened source keeps autoplay intent through asynchronous preparation', async () => {
  const item = { id: 'next' }
  let plays = 0
  const audio = {
    autoplay: false,
    readyState: 1,
    duration: 180,
    currentTime: 0,
    play: async () => { plays++ },
  }
  const session = createAudioPlaybackSession({
    setStatus() {},
    isActive: value => value === item,
    updateTimeline() {},
    resumeGain() {},
  })

  session.open({ identity: 'music:next', item, audio })
  await Promise.resolve()

  assert.equal(audio.autoplay, true)
  assert.equal(plays, 1)
  session.close()
  assert.equal(audio.autoplay, false)
})

function fakeAudio(src) {
  return { autoplay: false, readyState: 0, duration: 200, currentTime: 42, play: async () => {}, load() { this.loads = (this.loads || 0) + 1 }, removeAttribute() {}, getAttribute(name) { return name === 'src' ? src : null } }
}

test('a converted stream that fails falls back to the original file, at the same position, instead of giving up', async () => {
  const item = { id: 'a', compatibility_url: '/api/audio-hls/music/a/index.m3u8', bitrate: 320000 }
  const audio = fakeAudio('/api/audio-stream/music/a/audio')
  const modes = [], statuses = []
  let onFatal, attaches = 0
  const session = createAudioPlaybackSession({
    setStatus: value => statuses.push(value), isActive: value => value === item, updateTimeline() {}, resumeGain() {}, onMode: (mode) => modes.push(mode),
    createCompatibility: () => ({ attach: async (options) => { attaches++; onFatal = options.onFatal; return true }, reset() {} }),
  })
  session.open({ identity: 'music:a', item, audio, quality: 'high' })
  await Promise.resolve()
  assert.equal(attaches, 1, 'on a mobile connection the converted stream is tried first')
  onFatal()
  await new Promise(resolve => setTimeout(resolve, 5))
  assert.equal(audio.src, '/api/audio-stream/music/a/audio', 'the original file takes over')
  assert.equal(modes.at(-1), 'original')
  assert.equal(attaches, 1, 'the converted stream is not retried over and over')
  session.close()
})

test('a file that is already small is played as it is, with no conversion attempted', async () => {
  const item = { id: 'b', compatibility_url: '/api/audio-hls/music/b/index.m3u8', bitrate: 128000 }
  const audio = fakeAudio('/api/audio-stream/music/b/audio')
  audio.readyState = 1
  let attaches = 0
  const session = createAudioPlaybackSession({ setStatus() {}, isActive: value => value === item, updateTimeline() {}, resumeGain() {}, createCompatibility: () => ({ attach: async () => { attaches++; return true }, reset() {} }) })
  session.open({ identity: 'music:b', item, audio, quality: 'high' })
  await Promise.resolve()
  assert.equal(attaches, 0)
  session.close()
})

test('the original file failing is still handled by the normal retries, not by the converted stream again', async () => {
  const item = { id: 'c', compatibility_url: '/api/audio-hls/music/c/index.m3u8', bitrate: 900000 }
  const audio = fakeAudio('/api/audio-stream/music/c/audio')
  let onFatal, attaches = 0
  const statuses = []
  const session = createAudioPlaybackSession({ setStatus: value => statuses.push(value), isActive: value => value === item, updateTimeline() {}, resumeGain() {}, createCompatibility: () => ({ attach: async (o) => { attaches++; onFatal = o.onFatal; return true }, reset() {} }) })
  session.open({ identity: 'music:c', item, audio, quality: 'medium' })
  await Promise.resolve()
  onFatal()
  await new Promise(resolve => setTimeout(resolve, 5))
  audio.onerror()
  assert.match(statuses.at(-1), /Reconnecting/)
  session.close()
})

test('after playback gives up a retry starts the same item again with a fresh set of attempts', async () => {
  const item = { id: 'd' }
  const audio = fakeAudio('/api/audio-stream/music/d/audio')
  const terminal = [], statuses = []
  const session = createAudioPlaybackSession({ setStatus: value => statuses.push(value), isActive: value => value === item, updateTimeline() {}, resumeGain() {}, onTerminal: failed => terminal.push(failed) })
  session.open({ identity: 'music:d', item, audio })
  for (let i = 0; i < 4; i++) audio.onerror()
  assert.equal(terminal.at(-1), true, 'the player is told it has given up, so it can offer Try again')
  assert.match(statuses.at(-1), /lost its connection/)
  session.retry()
  assert.equal(terminal.at(-1), false)
  audio.onerror()
  assert.match(statuses.at(-1), /Reconnecting… 1 of 3/, 'a retry gets the full set of attempts again')
  session.close()
})
