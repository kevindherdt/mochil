import test from 'node:test'
import assert from 'node:assert/strict'
import {createMusicState} from '../static/music-state.js'

test('queued playlist saves snapshot track identity and order before waiting',async()=>{
  let release
  const barrier=new Promise(resolve=>{release=resolve})
  const writes=[]
  const state=createMusicState({key:item=>item.id,onChanged(){},requestJSON:async(path,options={})=>{
    writes.push({path,...options})
    if(options.method==='DELETE')await barrier
    if(path.includes('view=summary'))return{data:[{name:'Saved',items:[]},{name:'Old',items:[]}]}
    return{data:null}
  }})
  await state.load()
  const first=state.deletePlaylist('Old')
  const item={library_id:'music',id:'one',title:'Original'}
  const tracks=[item]
  const save=state.savePlaylist('Saved',tracks)
  tracks.push({library_id:'music',id:'two'})
  item.id='changed';item.title='Changed'
  release();await Promise.all([first,save])
  assert.deepEqual(JSON.parse(writes.find(write=>write.method==='PUT').body),[{library_id:'music',item_id:'one'}])
  assert.deepEqual(state.playlists[0].items,[{library_id:'music',id:'one',title:'Original'}])
})

test('queued playlist append retains the originally selected item',async()=>{
  let release
  const barrier=new Promise(resolve=>{release=resolve})
  const writes=[]
  const state=createMusicState({key:item=>item.id,onChanged(){},requestJSON:async(path,options={})=>{
    writes.push({path,...options})
    if(options.method==='DELETE')await barrier
    if(path.includes('view=summary'))return{data:[{name:'Old',items:[]}]}
    return{data:[]}
  }})
  await state.load()
  const first=state.deletePlaylist('Old')
  const item={library_id:'music',id:'one'}
  const append=state.addPlaylistItem('Saved',item)
  item.id='changed';item.library_id='other'
  release();await Promise.all([first,append])
  assert.deepEqual(JSON.parse(writes.find(write=>write.method==='POST').body),{library_id:'music',item_id:'one'})
})
