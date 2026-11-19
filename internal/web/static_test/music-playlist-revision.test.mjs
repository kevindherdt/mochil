import test from 'node:test'
import assert from 'node:assert/strict'
import {createMusicState} from '../static/music-state.js'

for (const method of ['PATCH','DELETE']) test(`playlist ${method} rejects stale state and never replays automatically`, async () => {
  let revision='"first"'
  const writes=[]
  const state=createMusicState({key:x=>x.id,onChanged(){},requestJSON:async(path,options={})=>{
    if(options.method) {
      writes.push(options)
      if(options.headers['If-Match']!==revision) throw Object.assign(new Error('Playlist changed'),{status:412})
      return {data:null}
    }
    if(path.includes('view=summary'))return {data:{version:1,playlists:[{name:'List',item_count:0,artwork_urls:[]}]}}
    if(path.endsWith('/List'))return {data:{name:'List',items:[]},response:{headers:new Headers({ETag:revision})}}
    return {data:[]}
  }})
  await state.load()
  await state.ensurePlaylist('List')
  revision='"newer"'
  const mutate=()=>method==='PATCH'?state.renamePlaylist('List','Renamed'):state.deletePlaylist('List')
  await assert.rejects(mutate(),/changed/)
  assert.equal(writes.length,1)
  assert.equal(writes[0].method,method)
  assert.equal(state.playlists[0].name,'List')
  assert.equal(state.playlists[0].revision,revision)
  await mutate()
  assert.equal(writes[1].headers['If-Match'],revision)
  assert.equal(state.playlists.length,method==='PATCH'?1:0)
  if(method==='PATCH')assert.equal(state.playlists[0].name,'Renamed')
})

test('snapshot replacement cannot bypass a failed detail read or create a missing playlist', async () => {
  let writes=0
  const state=createMusicState({key:x=>x.id,onChanged(){},requestJSON:async(path,options={})=>{
    if(options.method){writes++;return {data:null}}
    if(path.includes('view=summary'))return {data:{version:1,playlists:[{name:'List',item_count:2,artwork_urls:[]}]}}
    if(path.endsWith('/List'))throw new Error('offline')
    return {data:[]}
  }})
  await state.load()
  await assert.rejects(state.savePlaylist('List',[]),/No changes were sent/)
  await assert.rejects(state.savePlaylist('Missing',[]),/No changes were sent/)
  await assert.rejects(state.renamePlaylist('List','New'),/No changes were sent/)
  await assert.rejects(state.deletePlaylist('List'),/No changes were sent/)
  assert.equal(writes,0)
})

test('playlist edits send current revisions and reload conflicts without replaying stale contents', async () => {
  const track=id=>({id,library_id:'music'})
  let revision='"first"', items=[track('one'),track('two')]
  const writes=[]
  const state=createMusicState({key:x=>x.id,onChanged(){},requestJSON:async(path,options={})=>{
    if(options.method==='PUT') {
      writes.push(options)
      if(options.headers['If-Match']!==revision) throw Object.assign(new Error('Playlist changed'),{status:412})
      items=JSON.parse(options.body).map(x=>track(x.item_id))
      revision='"next"'
      return {data:null,response:{headers:new Headers({ETag:revision})}}
    }
    if(path.includes('view=summary')) return {data:{version:1,playlists:[{name:'List',item_count:items.length,artwork_urls:[]}]}}
    if(path.endsWith('/List'))return {data:{name:'List',items},response:{headers:new Headers({ETag:revision})}}
    return {data:[]}
  }})
  await state.load()
  await state.ensurePlaylist('List')
  revision='"other-device"'
  items=[...items,track('three')]
  await assert.rejects(state.editPlaylist('List', list=>list.slice(1)),/changed/)
  assert.equal(writes.length,1)
  assert.equal(writes[0].headers['If-Match'],'"first"')
  assert.deepEqual(state.playlists[0].items.map(x=>x.id),['one','two','three'])
  await state.editPlaylist('List', list=>list.slice(1))
  assert.equal(writes[1].headers['If-Match'],'"other-device"')
  await state.editPlaylist('List', list=>list.slice(1))
  assert.equal(writes[2].headers['If-Match'],'"next"')
  assert.deepEqual(items.map(x=>x.id),['three'])
})
