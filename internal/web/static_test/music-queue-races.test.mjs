import test from 'node:test'
import assert from 'node:assert/strict'
import {createMusicState} from '../static/music-state.js'

const one={library_id:'music',id:'one'}, two={library_id:'music',id:'two'}
const tick=()=>new Promise(resolve=>setTimeout(resolve,0))

for (const action of ['addToQueue','playNext']) {
  for (const replacement of [[],[one]]) {
    test(`${action} waiting on initial load cannot undo a newer queue replacement (${replacement.length} tracks)`,async()=>{
      let resolve
      const writes=[]
      const state=createMusicState({key:item=>item.id,onChanged(){},requestJSON:(path,options={})=>{
        if(options.method){writes.push(JSON.parse(options.body));return Promise.resolve({data:null})}
        return new Promise(done=>{resolve=done})
      }})
      const adding=state[action](two,one)
      if(replacement.length)state.setQueue(replacement)
      else state.clearQueue()
      resolve({data:[one]})
      await adding;await tick()
      assert.deepEqual(state.queue,replacement)
      assert.equal(writes.length,1,'superseded action must not send a second save')
      await state.addToQueue(two)
      assert.deepEqual(state.queue,[...replacement,two], 'a new action after replacement still works')
    })
  }
}

test('failed initial queue read cannot turn an append into a replacement',async()=>{
  let fail=true
  const writes=[]
  const state=createMusicState({key:item=>item.id,onChanged(){},requestJSON:async(path,options={})=>{
    if(options.method){writes.push(JSON.parse(options.body));return{data:null}}
    if(fail)throw new Error('offline')
    return{data:[one]}
  }})
  await state.addToQueue(two)
  await state.playNext(two,one)
  assert.deepEqual(writes,[])
  assert.match(state.loadErrors.queue,/No changes were sent/)
  fail=false
  await state.retryQueueSave()
  assert.deepEqual(writes,[], 'retry of a failed read must not PUT an empty queue')
  assert.deepEqual(state.queue,[one])
  await state.addToQueue(two)
  await tick()
  assert.deepEqual(writes,[[{library_id:'music',item_id:'one'},{library_id:'music',item_id:'two'}]])
})

test('malformed queue response is not treated as a successful empty queue',async()=>{
  const methods=[]
  const state=createMusicState({key:item=>item.id,onChanged(){},requestJSON:async(path,options={})=>{
    methods.push(options.method || 'GET')
    return{data:{error:'sign in'}}
  }})
  await state.addToQueue(two)
  assert.deepEqual(methods,['GET'])
  assert.match(state.loadErrors.queue,/Could not load/)
})

test('late initial queue read does not replace an explicit newer queue',async()=>{
  let resolve
  const state=createMusicState({key:item=>item.id,onChanged(){},requestJSON:async(path,options={})=>options.method?{data:null}:new Promise(done=>{resolve=done})})
  const loading=state.ensureQueue()
  state.setQueue([two])
  resolve({data:[one]})
  await loading
  assert.deepEqual(state.queue,[two])
})

test('failed older save does not fetch stale server state or erase newer edits',async()=>{
  let reject
  const methods=[]
  const state=createMusicState({key:item=>item.id,onChanged(){},requestJSON:async(path,options={})=>{
    methods.push(options.method)
    if(methods.length===1)return new Promise((_,fail)=>{reject=fail})
    return{data:null}
  }})
  state.setQueue([one])
  state.setQueue([two,one])
  reject(new Error('offline'))
  await tick()
  assert.deepEqual(methods,['PUT','PUT'])
  assert.deepEqual(state.queue,[two,one])
  assert.equal(state.loadErrors.queue,undefined)
})

test('latest failed save retains edits and explicit retry sends that queue',async()=>{
  let fail=true
  const writes=[]
  const state=createMusicState({key:item=>item.id,onChanged(){},requestJSON:async(path,options)=>{
    writes.push(JSON.parse(options.body))
    if(fail)throw new Error('offline')
    return{data:null}
  }})
  state.setQueue([two])
  await tick()
  assert.deepEqual(state.queue,[two])
  assert.match(state.loadErrors.queue,/could not be saved/)
  fail=false
  state.retryQueueSave()
  await tick()
  assert.deepEqual(writes,[[{library_id:'music',item_id:'two'}],[{library_id:'music',item_id:'two'}]])
  assert.equal(state.loadErrors.queue,undefined)
})
