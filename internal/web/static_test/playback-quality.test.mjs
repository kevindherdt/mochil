import test from 'node:test'
import assert from 'node:assert/strict'
import { worthConverting, effectivePlaybackQuality, loadPlaybackQualityPolicy, qualityHLSURL } from '../static/playback-quality.js'

const storage = values => ({ getItem: key => values[key] ?? null })

test('playback quality selects the configured network profile', () => {
  const values = storage({'bobobooks-playback-wifi-quality':'maximum','bobobooks-playback-mobile-quality':'low'})
  assert.equal(effectivePlaybackQuality({connection:{type:'wifi'}}, values), 'maximum')
  assert.equal(effectivePlaybackQuality({connection:{type:'cellular'}}, values), 'low')
  assert.equal(effectivePlaybackQuality({connection:{saveData:true}}, values), 'low')
})

test('quality HLS URLs retain encoded identities and query strings', () => {
  assert.equal(qualityHLSURL('/api/audio-hls/music/item/index.m3u8?v=one', 'medium'), '/api/audio-hls/music/item/medium/index.m3u8?v=one')
  assert.equal(qualityHLSURL('/ordinary/file.mp3', 'low'), '/ordinary/file.mp3')
})

test('server policy primes both browser network profiles', async () => {
  const values = new Map()
  await loadPlaybackQualityPolicy(async () => ({data:{wifi_quality:'high',mobile_quality:'medium'}}), {setItem:(key,value)=>values.set(key,value)})
  assert.equal(values.get('bobobooks-playback-wifi-quality'), 'high')
  assert.equal(values.get('bobobooks-playback-mobile-quality'), 'medium')
})

test('a file that is already small is not re-encoded, an unknown or larger one is', () => {
  assert.equal(worthConverting({ bitrate: 128000 }, 'high'), false, '128 kbps is already below the 192 kbps target')
  assert.equal(worthConverting({ bitrate: 200000 }, 'high'), false, 'within 10% of the target gains nothing')
  assert.equal(worthConverting({ bitrate: 320000 }, 'high'), true)
  assert.equal(worthConverting({ bitrate: 1411000 }, 'low'), true, 'lossless is worth converting')
  assert.equal(worthConverting({}, 'medium'), true, 'unknown bitrate is converted as before')
  assert.equal(worthConverting({ bitrate: 320000 }, 'maximum'), false, 'maximum is never converted')
})
