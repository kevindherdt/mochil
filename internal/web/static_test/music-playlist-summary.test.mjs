import test from 'node:test'
import assert from 'node:assert/strict'
import {createMusicState} from '../static/music-state.js'
import {decodePlaylistOverview, decodePlaylistDetail} from '../static/music-playlist-data.js'
import {musicPlaylistOverviewHTML, musicPlaylistDetailHTML} from '../static/music-playlist-view.js'

const summary = {version:1,playlists:[{name:'Saved',item_count:200,artwork_urls:['/cover/one']}]}
const track = {library_id:'music',id:'one'}
const escape = value => String(value).replaceAll('<','&lt;').replaceAll('>','&gt;')

test('summary overview preserves artwork and count without exposing playable preview tracks',()=>{
  const lists = decodePlaylistOverview(summary)
  assert.deepEqual(lists[0].items,[])
  const html = musicPlaylistOverviewHTML(lists,escape)
  assert.match(html,/200 tracks/)
  assert.match(html,/\/cover\/one/)
  const detail = musicPlaylistDetailHTML(lists[0],()=>{throw new Error('must not render tracks')},escape)
  assert.match(detail,/Loading playlist/)
  assert.doesNotMatch(detail,/data-play-playlist/)
})

test('overview makes no detail requests until one playlist is opened',async()=>{
  const paths=[]
  const state=createMusicState({key:item=>item.id,onChanged(){},requestJSON:async path=>{
    paths.push(path)
    if(path==='/api/audio-favorites')return{data:[]}
    if(path.includes('?view=summary'))return{data:summary}
    return{data:{name:'Saved',items:[track]}}
  }})
  await state.load()
  assert.equal(paths.length,2)
  await state.ensurePlaylist('Saved')
  assert.deepEqual(state.playlists[0].items,[track])
  await state.ensurePlaylist('Saved')
  assert.equal(paths.length,3)
})

test('late detail response cannot resurrect a deleted playlist',async()=>{
  let resolve
  let reads=0
  const delayed=new Promise(done=>{resolve=done})
  const state=createMusicState({key:item=>item.id,onChanged(){},requestJSON:async(path,options={})=>{
    if(options.method)return{data:null}
    if(path==='/api/audio-favorites')return{data:[]}
    if(path.includes('?view=summary'))return{data:summary}
    return ++reads===1?delayed:{data:{name:'Saved',items:[track]}}
  }})
  await state.load()
  const loading=state.ensurePlaylist('Saved')
  await state.deletePlaylist('Saved')
  resolve({data:{name:'Saved',items:[track]}})
  await loading
  assert.deepEqual(state.playlists,[])
})

test('failed detail preserves summary and offers a retry without pretending it is empty',async()=>{
  const state=createMusicState({key:item=>item.id,onChanged(){},requestJSON:async path=>{
    if(path==='/api/audio-favorites')return{data:[]}
    if(path.includes('?view=summary'))return{data:summary}
    throw new Error('Network unavailable')
  }})
  await state.load()
  await state.ensurePlaylist('Saved')
  assert.equal(state.playlists[0].item_count,200)
  assert.match(musicPlaylistDetailHTML(state.playlists[0],()=>'',escape),/data-playlist-retry/)
})

test('legacy full arrays remain usable but invalid summary and detail are rejected',()=>{
  assert.deepEqual(decodePlaylistOverview([{name:'Old',items:null}]),[{name:'Old',items:[]}])
  assert.throws(()=>decodePlaylistOverview({version:2,playlists:[]}))
  assert.throws(()=>decodePlaylistDetail({name:'Saved'},'Saved'))
  assert.throws(()=>decodePlaylistDetail({name:'Other',items:[]},'Saved'))
})
