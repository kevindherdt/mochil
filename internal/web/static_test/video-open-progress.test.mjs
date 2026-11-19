import test from 'node:test'
import assert from 'node:assert/strict'
import * as player from '../static/video-player.js'
import { createVideoProgress } from '../static/video-progress.js'

test('saved replay reopens at its checkpoint using the same stale library item', async () => {
  let remote = {revision: 4, finished: true, position_seconds: 200}
  const item = {library_id:'tv', id:'episode', duration_seconds:200, progress:{...remote}}
  const video = {duration:200, currentTime:60, paused:false, ended:false}
  const request = async (_url, options = {}) => {
    if (!options.method) return {data:{...remote}}
    const body = JSON.parse(options.body)
    assert.equal(body.expected_revision, remote.revision)
    remote = {...body, revision:remote.revision + 1}
    return {data:{...remote}}
  }
  const progress = createVideoProgress({requestJSON:request, getActive:() => item, getVideo:() => video, deviceID:'device'})
  const coordinator = player.createVideoOpenProgress(request)
  progress.reset(item)
  assert.equal(await coordinator.track(item, progress.save(true)), true)
  const reopened = await coordinator.open(item)
  assert.equal(player.videoResumePosition(200, reopened.progress.position_seconds, reopened.progress.finished), 60)
  assert.equal(reopened.progress.revision, 5)
})

test('reopening a cached finished item waits for its final save then reads fresh progress', async () => {
  let release
  let remote = {revision: 4, finished: true, position_seconds: 200}
  const gate = new Promise(resolve => { release = resolve })
  const requests = []
  const coordinator = player.createVideoOpenProgress(async (url, options) => {
    requests.push([url, options]); return {data: remote}
  })
  const item = {library_id: 'tv', id: 'ep', progress: {...remote}}
  coordinator.track(item, gate.then(() => { remote = {revision: 5, finished: false, position_seconds: 60} }))
  const opening = coordinator.open(item)
  await Promise.resolve()
  assert.equal(requests.length, 0)
  release()
  const refreshed = await opening
  assert.deepEqual(refreshed.progress, remote)
  assert.equal(item.progress.finished, true)
  assert.equal(requests[0][1].cache, 'no-store')
})

test('a slower earlier open cannot replace a newer open or a closed player', async () => {
  let release
  const gate = new Promise(resolve => { release = resolve })
  const coordinator = player.createVideoOpenProgress(async url => {
    if (url.endsWith('/old')) await gate
    return {data: {position_seconds: url.endsWith('/old') ? 10 : 30}}
  })
  const old = coordinator.open({library_id: 'tv', id: 'old'})
  const current = await coordinator.open({library_id: 'tv', id: 'new'})
  release()
  assert.equal(current.progress.position_seconds, 30)
  assert.equal(await old, null)
  let finish
  const waiting = new Promise(resolve => { finish = resolve })
  coordinator.track({library_id:'tv', id:'new'}, waiting)
  const cancelled = coordinator.open({library_id:'tv', id:'new'})
  coordinator.cancel(); finish()
  assert.equal(await cancelled, null)
})

test('failed progress refresh falls back to the original snapshot without mutation', async () => {
  const coordinator = player.createVideoOpenProgress(async () => { throw new Error('offline') })
  const item = {library_id:'tv', id:'ep', progress:{revision:2, position_seconds:20}}
  assert.deepEqual((await coordinator.open(item)).progress, item.progress)
})


test('cancellation promptly ends an open waiting for a save', async () => {
  const coordinator = player.createVideoOpenProgress(async () => ({data:{}}), {timeoutMs:30})
  const item = {library_id:'tv', id:'ep'}
  coordinator.track(item, new Promise(() => {}))
  const opening = coordinator.open(item)
  coordinator.cancel()
  assert.equal(await opening, null)
})

test('stalled save and refresh are bounded and do not open stale state', async () => {
  const item = {library_id:'tv', id:'ep'}
  const save = player.createVideoOpenProgress(async () => ({data:{}}), {timeoutMs:10})
  save.track(item, new Promise(() => {}))
  await assert.rejects(save.open(item), /progress/i)
  const refresh = player.createVideoOpenProgress(() => new Promise(() => {}), {timeoutMs:10})
  await assert.rejects(refresh.open(item), /progress/i)
})

test('failed final save is reported on reopen instead of silently forgotten', async () => {
  const item = {library_id:'tv', id:'ep'}
  const coordinator = player.createVideoOpenProgress(async () => ({data:{revision:8}}), {timeoutMs:10})
  const failed = Promise.reject(new Error('offline'))
  coordinator.track(item, failed)
  await failed.catch(() => {})
  await assert.rejects(coordinator.open(item), /progress/i)
})


test('failed forced checkpoint is surfaced by the real progress session', async () => {
  const item = {library_id:'tv', id:'ep', duration_seconds:100, progress:{revision:2}}
  const video = {duration:100, currentTime:30, paused:true, ended:false}
  const request = async (_url, opts = {}) => {
    if (opts.method === 'PUT') throw new Error('offline')
    return {data:{revision:3, position_seconds:80}}
  }
  const progress = createVideoProgress({requestJSON:request, getActive:() => item, getVideo:() => video, deviceID:'device'})
  const coordinator = player.createVideoOpenProgress(request)
  progress.reset(item)
  await coordinator.track(item, progress.save(true)).catch(() => {})
  await assert.rejects(coordinator.open(item), /progress/i)
})


test('navigation cancellation aborts an in-flight progress fetch', async () => {
  let signal
  const coordinator = player.createVideoOpenProgress((_url, options) => { signal = options.signal; return new Promise(() => {}) }, {timeoutMs:100})
  const opening = coordinator.open({library_id:'tv', id:'ep'})
  coordinator.cancel()
  assert.equal(signal.aborted, true)
  assert.equal(await opening, null)
})
