import test from 'node:test'
import assert from 'node:assert/strict'
import {createMusicState, playlistItemsAfterAction} from '../static/music-state.js'

const tracks = ['one','two','three'].map(id => ({id,library_id:'music'}))

test('rapid playlist removals compose against preceding confirmed edits', async () => {
  const writes = []
  let release, started
  const barrier = new Promise(resolve => {release=resolve})
  const began = new Promise(resolve => {started=resolve})
  const state = createMusicState({key:item=>item.id,onChanged(){},requestJSON:async(path,options={})=>{
    if (options.method === 'PUT') {
      writes.push(JSON.parse(options.body).map(item=>item.item_id))
      if (writes.length === 1) {started();await barrier}
      return {data:null}
    }
    return {data:path.includes('playlists')?[{name:'List',items:tracks}]:[]}
  }})
  await state.load()
  const remove = item => state.editPlaylist('List', items=>playlistItemsAfterAction(items,item,'playlist-remove',value=>value.id))
  const first = remove(tracks[0])
  await began
  const second = remove(tracks[1])
  release()
  await Promise.all([first,second])
  assert.deepEqual(writes,[['two','three'],['three']])
  assert.deepEqual(state.playlists[0].items,[tracks[2]])
})

test('editing an unloaded playlist requires successful detail loading before writing', async () => {
  let writes=0
  const state=createMusicState({key:item=>item.id,onChanged(){},requestJSON:async(path,options={})=>{
    if(options.method){writes++;return {data:null}}
    if(path.includes('view=summary'))return {data:{version:1,playlists:[{name:'List',item_count:3,artwork_urls:[]}]}}
    if(path.endsWith('/List'))throw new Error('offline')
    return {data:[]}
  }})
  await state.load()
  await assert.rejects(state.editPlaylist('List',()=>[]),/No changes were sent/)
  assert.equal(writes,0)
})
