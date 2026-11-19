import assert from 'node:assert/strict'
import test from 'node:test'
import { createVideoProgress } from '../static/video-progress.js'

function fixture(requestJSON) {
  const video = {duration:200,currentTime:40,paused:false,ended:false}
  const item = {library_id:'tv',id:'episode',duration_seconds:210,progress:{revision:1}}
  const progress = createVideoProgress({requestJSON,getActive:() => item,getVideo:() => video,deviceID:'device'})
  progress.reset(item)
  return {video,item,progress}
}

test('video retries failed explicit seeks without losing intent or throttling the retry', async () => {
  const writes = []
  const {progress} = fixture(async (_path, options) => {
    writes.push(JSON.parse(options.body))
    assert.equal(options.keepalive,true)
    if (writes.length === 1) throw new Error('offline')
    return {data:{revision:2}}
  })
  progress.claim()
  assert.equal(await progress.save(),false)
  assert.equal(await progress.save(),true)
  assert.equal(writes[1].explicit_seek,true)
  assert.equal(writes[1].expected_revision,1)
  assert.equal(writes[1].duration_seconds,210)
  assert.equal('claimGeneration' in writes[1],false)
  assert.equal(await progress.save(true),true)
  assert.equal(writes[2].explicit_seek,false)
})

for (const conflict of [false,true]) {
  test(`video preserves newer seek during delayed ${conflict ? 'conflict' : 'success'}`, async () => {
    let release, notify
    const delayed = new Promise(resolve => {release=resolve})
    const started = new Promise(resolve => {notify=resolve})
    const writes = []
    const {progress,video} = fixture(async (_path, options={}) => {
      if (!options.method) {notify();return delayed}
      writes.push(JSON.parse(options.body))
      if (writes.length === 1) {
        if (conflict) throw Object.assign(new Error('conflict'),{status:409})
        notify();return delayed
      }
      return {data:{revision:9}}
    })
    const previous = progress.save()
    await started
    video.currentTime=10
    progress.claim()
    release({data:{revision:8,position_seconds:210,finished:true}})
    assert.equal(await previous,!conflict)
    assert.equal(await progress.save(),true)
    assert.equal(writes.length,2)
    assert.equal(writes[1].position_seconds,10)
    assert.equal(writes[1].explicit_seek,true)
    assert.equal(writes[1].expected_revision,8)
  })
}

test('passive video cannot rewind a remotely finished item, but an explicit seek can', async () => {
  let writes=0
  const {progress,video} = fixture(async (_path, options={}) => {
    if (!options.method) return {data:{revision:8,position_seconds:210,finished:true}}
    if (++writes === 1) throw Object.assign(new Error('conflict'),{status:409})
    return {data:{revision:9}}
  })
  assert.equal(await progress.save(),false)
  video.currentTime=60
  assert.equal(await progress.save(),false)
  assert.equal(writes,1)
  video.currentTime=10
  progress.claim()
  assert.equal(await progress.save(),true)
  assert.equal(writes,2)
})
