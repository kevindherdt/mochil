import test from 'node:test'
import assert from 'node:assert/strict'
import {createMusicState} from '../static/music-state.js'

test('visible edits carry original identities while explicit replacement stays separate',async()=>{
  const tracks=['one','two'].map(id=>({id,library_id:'music'}))
  const writes=[]
  const state=createMusicState({key:x=>x.id,onChanged(){},requestJSON:async(path,options={})=>{
    if(options.method){writes.push({path,...options});return {data:null,response:{headers:new Headers({ETag:`"r${writes.length}"`})}}}
    if(path.includes('view=summary'))return {data:{version:1,playlists:[{name:'List',item_count:3,artwork_urls:[]}]}}
    if(path.endsWith('/List'))return {data:{name:'List',items:tracks},response:{headers:new Headers({ETag:'"r0"','X-Mochil-Playlist-Edit':'visible-v1'})}}
    return {data:[]}
  }})
  await state.load()
  await state.editPlaylist('List',items=>items.toReversed())
  const refs=ids=>ids.map(item_id=>({library_id:'music',item_id}))
  assert.equal(writes[0].path,'/api/audio-playlists/List?edit=visible')
  assert.deepEqual(JSON.parse(writes[0].body),{visible_entries:refs(['one','two']),entries:refs(['two','one'])})
  await state.editPlaylist('List',items=>items.slice(1))
  assert.equal(writes[1].headers['If-Match'],'"r1"')
  assert.deepEqual(JSON.parse(writes[1].body),{visible_entries:refs(['two','one']),entries:refs(['one'])})
  await state.savePlaylist('List',[])
  assert.equal(writes[2].path,'/api/audio-playlists/List')
  assert.deepEqual(JSON.parse(writes[2].body),[])
  await state.editPlaylist('List',()=>tracks)
  assert.equal(writes[3].path,'/api/audio-playlists/List?edit=visible')
})

test('advertised visible edit without revision cannot fall back to destructive replacement',async()=>{
  let writes=0
  const state=createMusicState({key:x=>x.id,onChanged(){},requestJSON:async(path,options={})=>{
    if(options.method){writes++;return {data:null}}
    if(path.includes('view=summary'))return {data:{version:1,playlists:[{name:'List',item_count:1,artwork_urls:[]}]}}
    if(path.endsWith('/List'))return {data:{name:'List',items:[]},response:{headers:new Headers({'X-Mochil-Playlist-Edit':'visible-v1'})}}
    return {data:[]}
  }})
  await state.load()
  await assert.rejects(state.editPlaylist('List',()=>[]),/No changes were sent/)
  assert.equal(writes,0)
})
