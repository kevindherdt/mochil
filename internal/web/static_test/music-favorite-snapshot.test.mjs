import test from 'node:test'
import assert from 'node:assert/strict'
import {createMusicFavoritesState} from '../static/music-favorites-state.js'

const key = item => `${item.library_id}:${item.id}`
const tick = () => new Promise(resolve => setImmediate(resolve))

test('favorite selection is captured before the request and remains stable after completion', async()=>{
  const writes=[]
  const state=createMusicFavoritesState({key,onChanged(){},requestJSON:async(path,options)=>{
    writes.push(JSON.parse(options.body));return {data:null}
  }})
  const selected={library_id:'music',id:'one',title:'Original'}
  const pending=state.toggle(selected)
  selected.id='two';selected.title='Changed'
  assert.equal(await pending,true)
  assert.deepEqual(writes,[{library_id:'music',item_id:'one',favorite:true}])
  assert.deepEqual(state.items,[{library_id:'music',id:'one',title:'Original'}])
  selected.library_id='other'
  assert.equal(state.isFavorite({library_id:'music',id:'one'}),true)
})

test('queued favorite toggles retain selected identity while another request is pending', async()=>{
  const writes=[]
  const state=createMusicFavoritesState({key,onChanged(){},requestJSON:(path,options)=>new Promise(resolve=>{
    writes.push({body:JSON.parse(options.body),resolve})
  })})
  const selected={library_id:'music',id:'one'}
  const first=state.toggle(selected)
  await tick()
  const second=state.toggle(selected)
  selected.library_id='other';selected.id='two'
  assert.equal(writes.length,1)
  writes[0].resolve({data:null});await first;await tick()
  assert.deepEqual(writes.map(write=>write.body),[
    {library_id:'music',item_id:'one',favorite:true},
    {library_id:'music',item_id:'one',favorite:false},
  ])
  writes[1].resolve({data:null});await second
  assert.deepEqual(state.items,[])
  assert.equal(state.isFavorite({library_id:'music',id:'one'}),false)
})

test('a failed favorite write leaves state intact and does not block another track or retry', async()=>{
  const writes=[]
  const state=createMusicFavoritesState({key,onChanged(){},requestJSON:(path,options)=>new Promise((resolve,reject)=>{
    writes.push({body:JSON.parse(options.body),resolve,reject})
  })})
  const one={library_id:'music',id:'one'},two={library_id:'music',id:'two'}
  const failed=state.toggle(one)
  const rejection=assert.rejects(failed,/Offline/)
  const retry=state.toggle(one),other=state.toggle(two)
  await tick()
  assert.equal(writes.length,2)
  writes[1].resolve({data:null});await other
  assert.deepEqual(state.items,[two])
  writes[0].reject(new Error('Offline'));await rejection;await tick()
  assert.equal(writes.length,3)
  assert.deepEqual(writes[2].body,{library_id:'music',item_id:'one',favorite:true})
  writes[2].resolve({data:null});await retry
  assert.equal(state.isFavorite(one),true)
  assert.equal(state.isFavorite(two),true)
})
