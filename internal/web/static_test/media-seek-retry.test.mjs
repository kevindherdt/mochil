import assert from 'node:assert/strict'
import test from 'node:test'
import { createAudioProgress } from '../static/audio-progress.js'
import { createVideoProgress } from '../static/video-progress.js'

for (const kind of ['audio', 'video']) {
  test(`${kind} preserves an explicit rewind when the conflict retry fails offline`, async () => {
    const item = {library_id:'library',id:'item',progress:{revision:1}}
    const element = {currentTime:20,duration:200,playbackRate:1,paused:false,ended:false}
    const writes = []
    const requestJSON = async (_path, options={}) => {
      if (!options.method) return {data:{revision:8,position_seconds:100}}
      const payload = JSON.parse(options.body)
      writes.push(payload)
      if (writes.length === 1) throw Object.assign(new Error('conflict'), {status:409})
      if (writes.length === 2) throw new Error('network unavailable')
      return {data:{revision:9}}
    }
    const common = {requestJSON,getActive:() => item}
    const progress = kind === 'video'
      ? createVideoProgress({...common,getVideo:() => element,deviceID:'device'})
      : createAudioProgress({...common,getAudio:() => element,
        storage:{getItem:() => 'device',setItem:() => {}},cryptoObject:{}})
    progress.reset(item)
    progress.claim()
    assert.equal(await progress.save(),false)
    assert.equal(writes.length,2)
    assert.equal(await progress.save(),true)
    assert.equal(writes[2].expected_revision,8)
    assert.equal(writes[2].position_seconds,20)
    assert.equal(writes[2].explicit_seek,true)
    element.currentTime=30
    assert.equal(await progress.save(),true)
    assert.equal(writes[3].explicit_seek,false)
  })
}
