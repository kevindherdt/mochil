import assert from 'node:assert/strict'
import test from 'node:test'
import { createAudioProgress, startPosition } from '../static/audio-progress.js'
import { createPassiveProgressOwnership, shouldUploadPassiveProgress } from '../static/passive-progress-ownership.js'

const storage = () => {
  const values = new Map()
  return {
    getItem: key => values.get(key) || null,
    setItem: (key, value) => values.set(key, value),
  }
}

for (const conflict of [false, true]) {
  test(`a late ${conflict ? 'conflict lookup' : 'successful save'} preserves a newer same-item seek`, async () => {
    let release, signal
    const started = new Promise(resolve => { signal = resolve })
    const delayed = new Promise(resolve => { release = resolve })
    const writes = []
    const audio = {duration:200,currentTime:40,playbackRate:1,paused:false}
    const item = {library_id:'podcasts',id:'episode',progress:{revision:1}}
    const progress = createAudioProgress({
      requestJSON:async (_path, options = {}) => {
        if (!options.method) { signal(); return delayed }
        writes.push(JSON.parse(options.body))
        if (writes.length === 1) {
          if (conflict) throw Object.assign(new Error('conflict'), {status:409})
          signal()
          return delayed
        }
        return {data:{revision:9}}
      },
      getActive:() => item, getAudio:() => audio,
      storage:storage(), cryptoObject:{randomUUID:() => 'device'},
    })
    progress.reset(item)
    const previous = progress.save(true)
    await started
    audio.currentTime = 20
    progress.claim()
    release({data:{revision:8,position_seconds:200,finished:true}})
    assert.equal(await previous, !conflict)
    assert.equal(await progress.save(), true)
    assert.equal(writes.length, 2)
    assert.equal(writes[1].explicit_seek, true)
    assert.equal(writes[1].position_seconds, 20)
    assert.equal(writes[1].expected_revision, 8)
    assert.equal('claimGeneration' in writes[1], false)
    audio.currentTime = 30
    assert.equal(await progress.save(), true)
    assert.equal(writes[2].explicit_seek, false)
  })
}

test('audio progress resumes timed saves after a temporary offline failure', async () => {
  const writes = []
  const audio = {duration:200,currentTime:40,playbackRate:1,paused:false}
  const item = {library_id:'podcasts',id:'episode',progress:{revision:7}}
  const progress = createAudioProgress({
    requestJSON:async (_path, options) => {
      writes.push(JSON.parse(options.body))
      if (writes.length === 1) throw new Error('offline')
      return {data:{revision:8}}
    },
    getActive:() => item, getAudio:() => audio,
    storage:storage(), cryptoObject:{randomUUID:() => 'device-id'},
  })
  progress.reset(item)
  assert.equal(await progress.save(),false)
  audio.currentTime = 41
  assert.equal(await progress.save(),true)
  assert.equal(writes.length,2)
  assert.equal(writes[1].expected_revision,7)
  assert.equal(writes[1].position_seconds,41)
})

test('late conflict lookup cannot mark a newly selected item as finished or release its seek claim', async () => {
  let finishLookup, lookupStarted
  const started = new Promise(resolve => { lookupStarted = resolve })
  const response = new Promise(resolve => { finishLookup = resolve })
  let item = {library_id:'podcasts',id:'old',progress:{revision:1}}
  const audio = {duration:200,currentTime:40,playbackRate:1,paused:false}
  const writes = []
  const progress = createAudioProgress({
    requestJSON:async (path, options = {}) => {
      if (!options.method) { lookupStarted(); return response }
      writes.push({path,value:JSON.parse(options.body)})
      if (path.endsWith('/old')) throw Object.assign(new Error('conflict'),{status:409})
      return {data:{revision:8}}
    },
    getActive:() => item, getAudio:() => audio,
    storage:storage(), cryptoObject:{randomUUID:() => 'device-id'},
  })
  progress.reset(item)
  const oldSave = progress.save(true)
  await started
  item = {library_id:'podcasts',id:'new',progress:{revision:7}}
  progress.reset(item)
  progress.claim()
  finishLookup({data:{revision:99,position_seconds:200,finished:true}})
  assert.equal(await oldSave,false)
  assert.equal(await progress.save(true),true)
  assert.equal(writes.at(-1).path,'/api/audio-progress/podcasts/new')
  assert.equal(writes.at(-1).value.expected_revision,7)
  assert.equal(writes.at(-1).value.explicit_seek,true)
})

test('audio progress owns device identity, revisions, and timed-save throttling', async () => {
  const writes = []
  const audio = { duration: 200, currentTime: 40, playbackRate: 1, paused: false }
  const item = { library_id: 'music one', id: 'track/two', progress: { revision: 7 } }
  const progress = createAudioProgress({
    requestJSON: async (path, options) => {
      writes.push({ path, value: JSON.parse(options.body) })
      return { data: { revision: 8 } }
    },
    getActive: () => item,
    getAudio: () => audio,
    storage: storage(),
    cryptoObject: { randomUUID: () => 'device-id' },
  })
  progress.reset(item)
  assert.equal(await progress.save(true), true)
  assert.equal(await progress.save(false), false)
  assert.equal(writes[0].path, '/api/audio-progress/music%20one/track%2Ftwo')
  assert.equal(writes[0].value.expected_revision, 7)
  assert.equal(writes[0].value.device_id, 'device-id')
  assert.equal(writes[0].value.explicit_seek, false)
})

test('an explicit browser seek is identified to the server once', async () => {
  const writes = []
  const audio = { duration: 200, currentTime: 40, playbackRate: 1, paused: false }
  const item = { library_id: 'music', id: 'track', progress: { revision: 2 } }
  const progress = createAudioProgress({
    requestJSON: async (_path, options) => {
      writes.push(JSON.parse(options.body))
      return { data: { revision: 3 + writes.length } }
    },
    getActive: () => item,
    getAudio: () => audio,
    storage: storage(),
    cryptoObject: { randomUUID: () => 'device-id' },
  })
  progress.reset(item)
  progress.claim()
  assert.equal(await progress.save(true), true)
  audio.currentTime = 50
  assert.equal(await progress.save(true), true)
  assert.equal(writes[0].explicit_seek, true)
  assert.equal(writes[1].explicit_seek, false)
})

test('an active player refreshes a stale revision and retries without interruption', async () => {
  const calls = []
  let putCount = 0
  const audio = { duration: 100, currentTime: 25, playbackRate: 1, paused: false }
  const item = { library_id: 'music', id: 'track', progress: { revision: 1 } }
  const progress = createAudioProgress({
    requestJSON: async (path, options = {}) => {
      calls.push({ path, value: options.body ? JSON.parse(options.body) : null })
      if (options.method === 'PUT' && putCount++ === 0) throw Object.assign(new Error('conflict'), { status: 409 })
      if (!options.method) return { data: { revision: 4, position_seconds: 20, finished: false } }
      return { data: { revision: 5 } }
    },
    getActive: () => item,
    getAudio: () => audio,
    storage: storage(),
    cryptoObject: { randomUUID: () => 'device-id' },
  })
  progress.reset(item)
  assert.equal(await progress.save(true), true)
  assert.equal(calls.length, 3)
  assert.equal(calls.at(-1).value.expected_revision, 4)
})

test('an active stale player yields until it catches newer remote progress', async () => {
  const calls = []
  let putCount = 0
  const audio = { duration: 200, currentTime: 25, playbackRate: 1, paused: false }
  const item = { library_id: 'music', id: 'track', progress: { revision: 1 } }
  const progress = createAudioProgress({
    requestJSON: async (path, options = {}) => {
      calls.push({ path, value: options.body ? JSON.parse(options.body) : null })
      if (options.method === 'PUT' && putCount++ === 0)
        throw Object.assign(new Error('conflict'), { status: 409 })
      if (!options.method) return { data: { revision: 8, position_seconds: 80, finished: false } }
      return { data: { revision: 9 } }
    },
    getActive: () => item,
    getAudio: () => audio,
    storage: storage(),
    cryptoObject: { randomUUID: () => 'device-id' },
  })
  progress.reset(item)
  assert.equal(await progress.save(true), false)
  assert.equal(calls.length, 2)
  audio.currentTime = 79.74
  assert.equal(await progress.save(true), false)
  assert.equal(calls.length, 2)
  audio.currentTime = 79.75
  assert.equal(await progress.save(true), true)
  assert.equal(calls.length, 3)
  assert.equal(calls.at(-1).value.expected_revision, 8)
})

test('finished remote audio cannot be reopened by a stale playing browser', async () => {
  let putCount = 0
  const audio = { duration: 100, currentTime: 99, playbackRate: 1, paused: false }
  const item = { library_id: 'podcasts', id: 'episode', progress: { revision: 2 } }
  const progress = createAudioProgress({
    requestJSON: async (_path, options = {}) => {
      if (options.method === 'PUT' && putCount++ === 0)
        throw Object.assign(new Error('conflict'), { status: 409 })
      if (!options.method) return { data: { revision: 7, position_seconds: 100, finished: true } }
      return { data: { revision: 8 } }
    },
    getActive: () => item,
    getAudio: () => audio,
    storage: storage(),
    cryptoObject: { randomUUID: () => 'device-id' },
  })
  progress.reset(item)
  assert.equal(await progress.save(true), false)
  assert.equal(await progress.save(true), false)
  assert.equal(putCount, 1)
})

test('audio conflict ownership is monotonic and tolerates small timing drift', () => {
  const local = { position_seconds: 30, duration_seconds: 120 }
  assert.equal(shouldUploadPassiveProgress({ position_seconds: 29.8, finished: false }, local), false)
  assert.equal(shouldUploadPassiveProgress({ position_seconds: 29.7, finished: false }, local), true)
  assert.equal(shouldUploadPassiveProgress({ position_seconds: 30.2, finished: false }, local), false)
  assert.equal(shouldUploadPassiveProgress({ position_seconds: 30.3, finished: false }, local), false)
  assert.equal(shouldUploadPassiveProgress({ position_seconds: 35, finished: false }, local), false)
  assert.equal(shouldUploadPassiveProgress({ position_seconds: 30, finished: true }, local), false)
})

test('equivalent stale audio samples adopt the remote revision without a retry write', async () => {
  const calls = []
  let putCount = 0
  const audio = { duration: 100, currentTime: 30, playbackRate: 1, paused: false }
  const item = { library_id: 'music', id: 'track', progress: { revision: 1 } }
  const progress = createAudioProgress({
    requestJSON: async (path, options = {}) => {
      calls.push({ path, method: options.method || 'GET' })
      if (options.method === 'PUT' && putCount++ === 0)
        throw Object.assign(new Error('conflict'), { status: 409 })
      if (!options.method) return { data: { revision: 5, position_seconds: 30.1, finished: false } }
      return { data: { revision: 6 } }
    },
    getActive: () => item,
    getAudio: () => audio,
    storage: storage(),
    cryptoObject: { randomUUID: () => 'device-id' },
  })
  progress.reset(item)
  assert.equal(await progress.save(true), false)
  assert.deepEqual(calls.map(call => call.method), ['PUT', 'GET'])
})

test('shared passive ownership yields, catches up, and protects finished media', () => {
  const ownership = createPassiveProgressOwnership()
  assert.equal(ownership.resolve({ position_seconds: 80 }, { position_seconds: 25, duration_seconds: 200 }, true), false)
  assert.equal(ownership.canSave(79.7), false)
  assert.equal(ownership.canSave(79.75), true)
  ownership.reset()
  assert.equal(ownership.resolve({ position_seconds: 200, finished: true }, { position_seconds: 199, duration_seconds: 200 }, true), false)
  assert.equal(ownership.canSave(200), false)
  ownership.activate()
  assert.equal(ownership.canSave(200), false)
  ownership.claim()
  ownership.activate()
  assert.equal(ownership.canSave(200), true)
})

test('an explicit seek owns one conflict retry without weakening later passive saves', () => {
  const ownership = createPassiveProgressOwnership()
  ownership.claim()
  ownership.activate()
  assert.equal(ownership.resolve({ position_seconds: 80 }, { position_seconds: 25, duration_seconds: 200 }, true), true)
  assert.equal(ownership.resolve({ position_seconds: 90 }, { position_seconds: 30, duration_seconds: 200 }, true), false)
  assert.equal(ownership.canSave(89), false)
})

test('an inactive stale player must catch up after playback is activated', async () => {
  const calls = []
  let putCount = 0
  const audio = { duration: 100, currentTime: 20, playbackRate: 1, paused: true }
  const item = { library_id: 'podcasts', id: 'episode', progress: { revision: 1 } }
  const progress = createAudioProgress({
    requestJSON: async (path, options = {}) => {
      calls.push({ path, value: options.body ? JSON.parse(options.body) : null })
      if (options.method === 'PUT' && putCount++ === 0)
        throw Object.assign(new Error('conflict'), { status: 409 })
      if (!options.method) return { data: { revision: 5, position_seconds: 70 } }
      return { data: { revision: 6 } }
    },
    getActive: () => item,
    getAudio: () => audio,
    storage: storage(),
    cryptoObject: { randomUUID: () => 'device-id' },
  })
  progress.reset(item)
  assert.equal(await progress.save(true), false)
  assert.equal(await progress.save(true), false)
  assert.equal(calls.length, 2)

  audio.paused = false
  progress.activate()
  assert.equal(await progress.save(true), false)
  assert.equal(calls.length, 2)

  audio.currentTime = 69.8
  assert.equal(await progress.save(true), true)
  assert.equal(calls.length, 3)
  assert.equal(calls.at(-1).value.expected_revision, 5)
})

test('a song always starts from the beginning; audiobooks and podcasts continue where you stopped', () => {
  const progress = { position_seconds: 95.5 }
  assert.equal(startPosition({ library_kind: 'music', progress }), 0, 'a saved song position must not be resumed')
  assert.equal(startPosition({ library_kind: 'audiobooks', progress }), 95.5)
  assert.equal(startPosition({ library_kind: 'podcasts', progress }), 95.5)
})

test('with no usable saved position playback starts at zero', () => {
  assert.equal(startPosition({ library_kind: 'audiobooks' }), 0)
  assert.equal(startPosition({ library_kind: 'audiobooks', progress: { position_seconds: 'x' } }), 0)
  assert.equal(startPosition({ library_kind: 'audiobooks', progress: { position_seconds: -4 } }), 0)
  assert.equal(startPosition(null), 0)
})
