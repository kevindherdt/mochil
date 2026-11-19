import test from 'node:test'
import assert from 'node:assert/strict'
import {createMusicState} from '../static/music-state.js'

test('failed playlist refresh retains loaded details and later success clears the error',async()=>{
  let response={data:[{name:'Saved',items:[{library_id:'music',id:'one'}]}]}
  const state=createMusicState({key:item=>item.id,onChanged(){},requestJSON:async path=>{
    if(path.includes('favorites'))return{data:[]}
    if(response instanceof Error)throw response
    return response
  }})
  await state.load()
  const before=state.playlists
  response=new Error('Offline')
  await state.load({force:true})
  assert.equal(state.playlists,before)
  assert.equal(state.loadErrors.playlists,'Offline')
  response={data:{version:99,playlists:[]}}
  await state.load({force:true})
  assert.equal(state.playlists,before)
  assert.equal(state.loadErrors.playlists,'Unsupported playlist overview.')
  response={data:[]}
  await state.load({force:true})
  assert.deepEqual(state.playlists,[])
  assert.equal(state.loadErrors.playlists,'')
})

test('confirmed append followed by failed overview refresh does not discard cached playlists',async()=>{
  let fail=false
  const writes=[]
  const state=createMusicState({key:item=>item.id,onChanged(){},requestJSON:async(path,options={})=>{
    if(options.method){writes.push(options.method);return{data:null}}
    if(path.includes('favorites'))return{data:[]}
    if(fail)throw new Error('Refresh offline')
    return{data:[{name:'Saved',items:[{library_id:'music',id:'one'}]}]}
  }})
  await state.load();fail=true
  await state.addPlaylistItem('Saved',{library_id:'music',id:'two'})
  assert.deepEqual(writes,['POST'])
  assert.equal(state.playlists[0].name,'Saved')
  assert.equal(state.loadErrors.playlists,'Refresh offline')
  // Do not pretend a failed refresh verified the new complete server contents.
  assert.equal(state.playlists[0].items.length,1)
  assert.equal(state.playlists[0].detailLoaded,false)
})
