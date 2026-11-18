import test from 'node:test'
import assert from 'node:assert/strict'
import {createMusicState} from '../static/music-state.js'

const one={library_id:'music',id:'one'},two={library_id:'music',id:'two'}
function fixture() {
  const reads=[]
  const state=createMusicState({key:item=>item.id,onChanged(){},requestJSON:async(path,options={})=>{
    if(options.method)return{data:null}
    if(path.includes('playlists'))return{data:[]}
    return new Promise((resolve,reject)=>reads.push({resolve:data=>resolve({data}),reject}))
  }})
  return {state,reads}
}

test('late favorites refresh merges newer toggles while retaining other server items',async()=>{
  const {state,reads}=fixture()
  const load=state.load()
  await state.toggleFavorite(one)
  reads[0].resolve([two]);await load
  assert.equal(state.isFavorite(one),true)
  assert.equal(state.isFavorite(two),true)
  const refresh=state.load({force:true})
  await state.toggleFavorite(one)
  reads[1].resolve([one,two]);await refresh
  assert.equal(state.isFavorite(one),false)
  assert.deepEqual(state.favoriteItems,[two])
})

test('failed and malformed refreshes retain the last usable favorites',async()=>{
  const {state,reads}=fixture()
  const load=state.load();reads[0].resolve([one]);await load
  const refresh=state.load({force:true});reads[1].reject(new Error('Offline'));await refresh
  assert.deepEqual(state.favoriteItems,[one]);assert.equal(state.loadErrors.favorites,'Offline')
  const malformed=state.load({force:true});reads[2].resolve({});await malformed
  assert.deepEqual(state.favoriteItems,[one]);assert.equal(state.loadErrors.favorites,'Invalid favorites response')
})

test('older overlapping refresh cannot replace a newer result',async()=>{
  const {state,reads}=fixture()
  const first=state.load(),second=state.load({force:true})
  reads[1].resolve([two]);await second
  reads[0].resolve([one]);await first
  assert.deepEqual(state.favoriteItems,[two])
})

test('old completion cannot release the active refresh or make readers return early',async()=>{
  const {state,reads}=fixture()
  const old=state.load(),latest=state.load({force:true})
  reads[0].resolve([one]);await old
  let finished=false
  const follower=state.load().then(()=>{finished=true})
  await new Promise(resolve=>setImmediate(resolve))
  assert.equal(finished,false)
  assert.equal(reads.length,2)
  reads[1].resolve([two]);await Promise.all([latest,follower])
  assert.equal(finished,true)
  assert.deepEqual(state.favoriteItems,[two])
})

test('already loaded readers wait for an active forced refresh',async()=>{
  const {state,reads}=fixture()
  const first=state.load();reads[0].resolve([one]);await first
  const refresh=state.load({force:true})
  let finished=false
  const follower=state.load().then(()=>{finished=true})
  await new Promise(resolve=>setImmediate(resolve))
  assert.equal(finished,false)
  reads[1].resolve([two]);await Promise.all([refresh,follower])
  assert.equal(reads.length,2)
  assert.deepEqual(state.favoriteItems,[two])
})
