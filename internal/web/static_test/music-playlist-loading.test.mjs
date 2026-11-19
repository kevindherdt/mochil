import test from 'node:test'
import assert from 'node:assert/strict'
import {createMusicState} from '../static/music-state.js'

async function fixture() {
  const requests = []
  const state = createMusicState({key:item=>item.id, onChanged(){}, requestJSON:async(path, options={})=>{
    if (path.includes('favorites')) return {data:[]}
    if (path.includes('?view=summary')) return {data:{version:1, playlists:['A','B'].map(name=>({name,item_count:1,artwork_urls:[]}))}}
    if (options.method) return {data:null}
    return new Promise((resolve,reject)=>requests.push({path,resolve,reject}))
  }})
  await state.load()
  return {state,requests}
}
const detail = (name,id) => ({data:{name,items:[{id,library_id:'music'}]}})

test('concurrent playlist opens share one read per name, without serializing other playlists', async()=>{
  const {state,requests} = await fixture()
  const first = state.ensurePlaylist('A'), second = state.ensurePlaylist('A'), other = state.ensurePlaylist('B')
  assert.equal(requests.length,2)
  requests[0].resolve(detail('A','one')); requests[1].resolve(detail('B','two'))
  const results = await Promise.all([first,second,other])
  assert.equal(results[0],results[1])
  assert.equal(results[2].items[0].id,'two')
})

test('old read cleanup cannot clear a replacement read after refresh', async()=>{
  const {state,requests} = await fixture()
  const old = state.ensurePlaylist('A')
  await state.load({force:true})
  const fresh = state.ensurePlaylist('A')
  requests[0].resolve(detail('A','stale'))
  assert.equal(await old,undefined)
  const follower = state.ensurePlaylist('A')
  assert.equal(requests.length,2)
  requests[1].resolve(detail('A','fresh'))
  assert.equal((await fresh).items[0].id,'fresh')
  assert.equal(await follower,await fresh)
})

test('failed shared reads can be retried and successful edits invalidate pending details', async()=>{
  const {state,requests} = await fixture()
  const failed = state.ensurePlaylist('A'), follower = state.ensurePlaylist('A')
  requests[0].reject(new Error('Offline'))
  await Promise.all([failed,follower])
  assert.equal(state.playlists[0].detailError,'Offline')
  const retry = state.ensurePlaylist('A')
  assert.equal(requests.length,2)
  const saving = state.savePlaylist('A',[{id:'edited',library_id:'music'}])
  await new Promise(resolve=>setImmediate(resolve))
  assert.equal(requests.length,3, 'replacement obtains a fresh detail/revision')
  requests[2].resolve(detail('A','current'))
  await saving
  requests[1].resolve(detail('A','stale'))
  assert.equal(await retry,undefined)
  assert.equal((await state.ensurePlaylist('A')).items[0].id,'edited')
})
