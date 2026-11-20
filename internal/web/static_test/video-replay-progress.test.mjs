import assert from 'node:assert/strict'
import test from 'node:test'
import { createVideoProgress } from '../static/video-progress.js'

function setup(handler) {
  const video = {duration: 200, currentTime: 12, paused: false, ended: false}
  const item = {library_id: 'tv', id: 'episode', duration_seconds: 200, progress: {revision: 4, position_seconds: 200, finished: true}}
  const progress = createVideoProgress({requestJSON: handler, getActive: () => item, getVideo: () => video, deviceID: 'device'})
  progress.reset(item)
  return {video, item, progress}
}

test('opening and closing a finished item at zero does not clear watched status', async () => {
  const writes = []
  const {progress, video} = setup(async (_url, options) => {
    writes.push(JSON.parse(options.body))
    return {data: {revision: 5}}
  })
  video.currentTime = 0
  assert.equal(await progress.save(true), false)
  assert.equal(writes.length, 0)
  video.currentTime = 12
  assert.equal(await progress.save(true), true)
  assert.equal(writes[0].explicit_seek, true)
})

test('explicit seek to zero restarts a finished item', async () => {
  const writes = []
  const {progress, video} = setup(async (_url, options) => {
    writes.push(JSON.parse(options.body))
    return {data: {revision: 5}}
  })
  video.currentTime = 0
  progress.claim()
  assert.equal(await progress.save(true), true)
  assert.deepEqual(writes.map(w => [w.position_seconds, w.explicit_seek, w.expected_revision]), [[0, true, 4]])
})

test('replay writes a conditional restart and continues with passive checkpoints', async () => {
  const writes = []
  const {progress, video} = setup(async (_url, options) => {
    writes.push(JSON.parse(options.body))
    return {data: {revision: 4 + writes.length}}
  })
  assert.equal(await progress.save(), true)
  video.currentTime = 30
  assert.equal(await progress.save(), true)
  assert.deepEqual(writes.map(w => [w.expected_revision, w.explicit_seek]), [[4, true], [5, false]])
})

test('replay conflict never overwrites newer remote progress, including queued checkpoints', async () => {
  let release, started
  const gate = new Promise(resolve => { release = resolve })
  const began = new Promise(resolve => { started = resolve })
  const writes = []
  const {progress, video} = setup(async (_url, options = {}) => {
    if (!options.method) return {data: {revision: 5, position_seconds: 100, finished: false}}
    writes.push(JSON.parse(options.body))
    started()
    await gate
    throw Object.assign(new Error('conflict'), {status: 409})
  })
  const first = progress.save()
  await began
  video.currentTime = 20
  const second = progress.save(true)
  release()
  assert.equal(await first, false)
  assert.equal(await second, false)
  assert.equal(writes.length, 1)
  assert.equal(writes[0].expected_revision, 4)
})

test('continuous replay catches up to newer unfinished remote without rewinding it', async () => {
  const writes = []
  const {progress, video} = setup(async (_url, options = {}) => {
    if (!options.method) return {data: {revision: 5, position_seconds: 100, finished: false}}
    writes.push(JSON.parse(options.body))
    if (writes.length === 1) throw Object.assign(new Error('conflict'), {status: 409})
    return {data: {revision: 6}}
  })
  assert.equal(await progress.save(), false)
  video.currentTime = 99.9
  assert.equal(await progress.save(true), false)
  video.currentTime = 101
  assert.equal(await progress.save(), true)
  assert.deepEqual(writes.map(w => [w.position_seconds, w.expected_revision, w.explicit_seek]), [[12, 4, true], [101, 5, false]])
})

test('replay network failure retries the same conditional restart', async () => {
  const writes = []
  const {progress} = setup(async (_url, options) => {
    writes.push(JSON.parse(options.body))
    if (writes.length === 1) throw new Error('offline')
    return {data: {revision: 5}}
  })
  assert.equal(await progress.save(), false)
  assert.equal(await progress.save(), true)
  assert.deepEqual(writes.map(w => [w.expected_revision, w.explicit_seek]), [[4, true], [4, true]])
})

test('newer finished remote state is not overwritten by replay', async () => {
  let writes = 0
  const {progress, video} = setup(async (_url, options = {}) => {
    if (!options.method) return {data: {revision: 5, position_seconds: 200, finished: true}}
    writes++
    throw Object.assign(new Error('conflict'), {status: 409})
  })
  assert.equal(await progress.save(), false)
  progress.activate()
  video.currentTime = 40
  assert.equal(await progress.save(true), false)
  assert.equal(writes, 1)
})


test('closed session does not retry a replay conflict even beyond remote position', async () => {
  let release, started, active = true
  const gate = new Promise(resolve => { release = resolve })
  const began = new Promise(resolve => { started = resolve })
  const writes = []
  const video = {duration: 200, currentTime: 12, paused: false, ended: false}
  const item = {library_id: 'tv', id: 'episode', progress: {revision: 4, finished: true}}
  const progress = createVideoProgress({getActive: () => active ? item : null, getVideo: () => video, deviceID: 'device', requestJSON: async (_url, options = {}) => {
    if (!options.method) return {data: {revision: 5, position_seconds: 30, finished: false}}
    writes.push(JSON.parse(options.body))
    started(); await gate
    throw Object.assign(new Error('conflict'), {status: 409})
  }})
  progress.reset(item)
  const first = progress.save(); await began
  video.currentTime = 100
  const final = progress.save(true)
  active = false
  release()
  assert.equal(await first, false)
  assert.equal(await final, false)
  assert.equal(writes.length, 1)
})

test('forced close checkpoint survives an in-flight replay', async () => {
  let release, started
  const gate = new Promise(resolve => { release = resolve })
  const began = new Promise(resolve => { started = resolve })
  const writes = []
  let active = true
  const video = {duration: 200, currentTime: 12, paused: false, ended: false}
  const item = {library_id: 'tv', id: 'episode', progress: {revision: 4, finished: true}}
  const progress = createVideoProgress({getActive: () => active ? item : null, getVideo: () => video, deviceID: 'device', requestJSON: async (_url, options) => {
    writes.push(JSON.parse(options.body))
    if (writes.length === 1) { started(); await gate }
    return {data: {revision: 4 + writes.length}}
  }})
  progress.reset(item)
  const first = progress.save()
  await began
  video.currentTime = 60
  const final = progress.save(true)
  active = false
  video.currentTime = 0
  video.duration = NaN
  release()
  assert.equal(await first, true)
  assert.equal(await final, true)
  assert.deepEqual(writes.map(w => [w.position_seconds, w.duration_seconds, w.expected_revision, w.explicit_seek]), [[12, 200, 4, true], [60, 200, 5, false]])
})


test('ended checkpoint retains finished while replay is delayed', async () => {
  let release, started
  const gate = new Promise(resolve => { release = resolve })
  const began = new Promise(resolve => { started = resolve })
  const writes = []
  const {progress, video} = setup(async (_url, options) => {
    writes.push(JSON.parse(options.body))
    if (writes.length === 1) { started(); await gate }
    return {data: {revision: 4 + writes.length}}
  })
  const first = progress.save()
  await began
  video.currentTime = 200
  video.ended = true
  const final = progress.save(true)
  video.ended = false
  release()
  assert.equal(await first, true)
  assert.equal(await final, true)
  assert.deepEqual(writes.map(w => [w.position_seconds, w.finished, w.expected_revision]), [[12, false, 4], [200, true, 5]])
})

test('delayed failed replay retries the final snapshot against the original revision', async () => {
  let release, started
  const gate = new Promise(resolve => { release = resolve })
  const began = new Promise(resolve => { started = resolve })
  const writes = []
  const {progress, video} = setup(async (_url, options) => {
    writes.push(JSON.parse(options.body))
    if (writes.length === 1) { started(); await gate; throw new Error('offline') }
    return {data: {revision: 5}}
  })
  const first = progress.save()
  await began
  video.currentTime = 60
  const final = progress.save(true)
  release()
  assert.equal(await first, false)
  assert.equal(await final, true)
  assert.deepEqual(writes.map(w => [w.position_seconds, w.expected_revision, w.explicit_seek]), [[12, 4, true], [60, 4, true]])
})

test('queued final conflict does not overwrite newer remote progress', async () => {
  let release, started
  const gate = new Promise(resolve => { release = resolve })
  const began = new Promise(resolve => { started = resolve })
  const writes = []
  const {progress, video} = setup(async (_url, options = {}) => {
    if (!options.method) return {data: {revision: 6, position_seconds: 100, finished: true}}
    writes.push(JSON.parse(options.body))
    if (writes.length === 1) { started(); await gate; return {data: {revision: 5}} }
    throw Object.assign(new Error('conflict'), {status: 409})
  })
  const first = progress.save()
  await began
  video.currentTime = 60
  video.paused = true
  const final = progress.save(true)
  release()
  assert.equal(await first, true)
  assert.equal(await final, false)
  assert.deepEqual(writes.map(w => w.expected_revision), [4, 5])
})


test('queued final does not retry against newer remote even while still playing', async () => {
  let release, started
  const gate = new Promise(resolve => { release = resolve })
  const began = new Promise(resolve => { started = resolve })
  const writes = []
  const {progress, video} = setup(async (_url, options = {}) => {
    if (!options.method) return {data: {revision: 6, position_seconds: 30, finished: false}}
    writes.push(JSON.parse(options.body))
    if (writes.length === 1) { started(); await gate; return {data: {revision: 5}} }
    if (writes.length === 2) throw Object.assign(new Error('conflict'), {status: 409})
    return {data: {revision: 7}}
  })
  const first = progress.save()
  await began
  video.currentTime = 60
  const final = progress.save(true)
  release()
  assert.equal(await first, true)
  assert.equal(await final, false)
  assert.equal(writes.length, 2)
})


test('switching items preserves queued final checkpoint and new item revision', async () => {
  let release, started
  const gate = new Promise(resolve => { release = resolve })
  const began = new Promise(resolve => { started = resolve })
  const writes = []
  const video = {duration: 200, currentTime: 12, paused: false, ended: false}
  let item = {library_id: 'tv', id: 'old', progress: {revision: 4, finished: true}}
  const progress = createVideoProgress({getActive: () => item, getVideo: () => video, deviceID: 'device', requestJSON: async (url, options) => {
    const body = JSON.parse(options.body); writes.push({url, ...body})
    if (url.endsWith('/old') && body.position_seconds === 12) { started(); await gate }
    return {data: {revision: body.expected_revision + 1}}
  }})
  progress.reset(item)
  const first = progress.save(); await began
  video.currentTime = 60; const final = progress.save(true)
  item = {library_id: 'tv', id: 'new', progress: {revision: 20}}
  progress.reset(item); video.currentTime = 35
  assert.equal(await progress.save(true), true)
  release(); await first
  assert.equal(await final, true)
  video.currentTime = 50; assert.equal(await progress.save(true), true)
  assert.deepEqual(writes.map(w => [w.url.split('/').at(-1), w.position_seconds, w.expected_revision]), [['old',12,4], ['new',35,20], ['old',60,5], ['new',50,21]])
})


for (const scenario of ['restart-conflict', 'final-conflict', 'restart-failure']) {
  test(`item switch isolates ${scenario} from the new player`, async () => {
    let release, started
    const gate = new Promise(resolve => { release = resolve })
    const began = new Promise(resolve => { started = resolve })
    const writes = []
    const video = {duration: 200, currentTime: 12, paused: false, ended: false}
    let item = {library_id: 'tv', id: 'old', progress: {revision: 4, finished: true}}
    const progress = createVideoProgress({getActive: () => item, getVideo: () => video, deviceID: 'device', requestJSON: async (url, options = {}) => {
      if (!options.method) return {data: {revision: 8, position_seconds: 150, finished: true}}
      const body = JSON.parse(options.body); writes.push({url, ...body})
      if (url.endsWith('/old') && body.position_seconds === 12) {
        started(); await gate
        if (scenario === 'restart-conflict') throw Object.assign(new Error('conflict'), {status: 409})
        if (scenario === 'restart-failure') throw new Error('offline')
      }
      if (url.endsWith('/old') && body.position_seconds === 60 && scenario === 'final-conflict') throw Object.assign(new Error('conflict'), {status: 409})
      return {data: {revision: body.expected_revision + 1}}
    }})
    progress.reset(item)
    const first = progress.save(); await began
    video.currentTime = 60; const final = progress.save(true)
    item = {library_id: 'tv', id: 'new', progress: {revision: 20}}
    progress.reset(item); video.currentTime = 35
    assert.equal(await progress.save(true), true)
    release()
    assert.equal(await first, scenario === 'final-conflict')
    assert.equal(await final, scenario === 'restart-failure')
    video.currentTime = 50; assert.equal(await progress.save(true), true)
    const old = writes.filter(w => w.url.endsWith('/old'))
    assert.deepEqual(old.map(w => [w.position_seconds, w.expected_revision, w.explicit_seek]), scenario === 'restart-conflict' ? [[12,4,true]] : scenario === 'final-conflict' ? [[12,4,true],[60,5,false]] : [[12,4,true],[60,4,true]])
    assert.deepEqual(writes.filter(w => w.url.endsWith('/new')).map(w => w.expected_revision), [20,21])
  })
}
