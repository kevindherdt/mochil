import assert from 'node:assert/strict'
import test from 'node:test'
import { createMusicState, playlistItemsAfterAction, playlistItemsAfterDrop, queueWithNext } from '../static/music-state.js'

const tick=()=>new Promise(resolve=>setTimeout(resolve,0))

test('queue editing loads lazily, preserves order, and serializes the newest state',async()=>{const one={library_id:'music',id:'one'},two={library_id:'music',id:'two'},three={library_id:'music',id:'three'},writes=[],reads=[];const requestJSON=async(path,options={})=>{if(path==='/api/audio-queue'&&!options.method){reads.push(path);return{data:[one,two]}};if(path==='/api/audio-favorites')return{data:[]};if(path==='/api/audio-playlists?view=summary')return{data:[]};if(path==='/api/audio-queue'){writes.push(JSON.parse(options.body));return{data:null}};throw new Error(path)};const state=createMusicState({requestJSON,key:item=>`${item.library_id}:${item.id}`,onChanged(){}});await state.load();assert.deepEqual(reads,[]);await state.addToQueue(three);state.moveInQueue(three,-1);state.removeFromQueue(one);await tick();await tick();assert.deepEqual(reads,['/api/audio-queue']);assert.deepEqual(state.queue.map(item=>item.id),['three','two']);assert.deepEqual(writes.at(-1).map(entry=>entry.item_id),['three','two']);state.clearQueue();await tick();assert.deepEqual(state.queue,[])})

test('play next moves a duplicate directly after the current track without restarting it',()=>{const one={library_id:'music',id:'one'},two={library_id:'music',id:'two'},three={library_id:'music',id:'three'},key=item=>`${item.library_id}:${item.id}`;assert.deepEqual(queueWithNext([one,two,three],three,one,key).map(item=>item.id),['one','three','two']);assert.deepEqual(queueWithNext([two,three],one,two,key).map(item=>item.id),['two','one','three']);assert.deepEqual(queueWithNext([two],three,null,key).map(item=>item.id),['three','two'])})

test('playlist edits preserve order and stay bounded at both edges', () => {
  const one={id:'one'},two={id:'two'},three={id:'three'},key=item=>item.id
  assert.deepEqual(playlistItemsAfterAction([one,two,three],two,'playlist-up',key).map(item=>item.id),['two','one','three'])
  assert.deepEqual(playlistItemsAfterAction([one,two,three],two,'playlist-down',key).map(item=>item.id),['one','three','two'])
  assert.deepEqual(playlistItemsAfterAction([one,two,three],two,'playlist-remove',key).map(item=>item.id),['one','three'])
  assert.deepEqual(playlistItemsAfterAction([one,two,three],one,'playlist-up',key).map(item=>item.id),['one','two','three'])
})

test('playlist drops preserve insertion intent in both directions', () => {
  const items=['one','two','three','four']
  assert.deepEqual(playlistItemsAfterDrop(items,0,2,true),['two','three','one','four'])
  assert.deepEqual(playlistItemsAfterDrop(items,3,1,false),['one','four','two','three'])
  assert.deepEqual(playlistItemsAfterDrop(items,1,1,true),items)
  assert.deepEqual(playlistItemsAfterDrop(items,-1,2,false),items)
})

test('playlist rename uses the atomic server rename contract without reloading the collection', async () => {
  const requests=[]
  let name='Old name'
  const requestJSON=async(path,options={})=>{
    requests.push({path,method:options.method,body:options.body})
    if(path==='/api/audio-favorites')return{data:[]}
    if(path==='/api/audio-playlists?view=summary'&&!options.method)return{data:[{name,items:[]}]}
    if(options.method==='PATCH'){name=JSON.parse(options.body).name;return{data:null}}
    throw new Error(path)
  }
  const state=createMusicState({requestJSON,key:item=>item.id,onChanged(){}})
  await state.load()
  await state.renamePlaylist('Old name','New name')
  assert.equal(requests.find(request=>request.method==='PATCH').path,'/api/audio-playlists/Old%20name')
  assert.equal(state.playlists[0].name,'New name')
  assert.equal(requests.filter(request=>request.path==='/api/audio-playlists?view=summary'&&!request.method).length,1)
})

test('confirmed playlist mutations update local state without redundant collection reads', async () => {
  const one={library_id:'music',id:'one'},two={library_id:'music',id:'two'}
  const requests=[]
  const requestJSON=async(path,options={})=>{
    requests.push({path,method:options.method})
    if(path==='/api/audio-favorites'&&!options.method)return{data:[]}
    if(path==='/api/audio-playlists?view=summary'&&!options.method)return{data:[{name:'Existing',items:[one]}]}
    if(!options.method&&path.startsWith('/api/audio-playlists/'))return{data:{name:decodeURIComponent(path.split('/').at(-1)),items:[]}}
    return{data:null}
  }
  let changes=0
  const state=createMusicState({requestJSON,key:item=>`${item.library_id}:${item.id}`,onChanged(){changes++}})
  await state.load()
  await state.savePlaylist('Existing',[two,one])
  await state.createPlaylist('New')
  await state.renamePlaylist('New','Renamed')
  await state.deletePlaylist('Renamed')
  assert.deepEqual(state.playlists,[{name:'Existing',items:[two,one]}])
  assert.equal(requests.filter(request=>request.path==='/api/audio-playlists?view=summary'&&!request.method).length,1)
  assert.equal(changes,4)
})

test('rapid playlist mutations reach the server and local state in user order', async () => {
  const track={library_id:'music',id:'one'},writes=[]
  let releaseSave
  const saveBlocked=new Promise(resolve=>{releaseSave=resolve})
  const requestJSON=async(path,options={})=>{
    if(path==='/api/audio-favorites'&&!options.method)return{data:[]}
    if(path==='/api/audio-playlists?view=summary'&&!options.method)return{data:[{name:'List',items:[]}]}
    writes.push(options.method)
    if(options.method==='PUT')await saveBlocked
    return{data:null}
  }
  const state=createMusicState({requestJSON,key:item=>`${item.library_id}:${item.id}`,onChanged(){}})
  await state.load()
  const save=state.savePlaylist('List',[track])
  const remove=state.deletePlaylist('List')
  await tick()
  assert.deepEqual(writes,['PUT'])
  releaseSave()
  await Promise.all([save,remove])
  assert.deepEqual(writes,['PUT','DELETE'])
  assert.deepEqual(state.playlists,[])
})

test('null collection payloads normalize to safe empty arrays and only load once', async () => {
  const reads=[]
  const requestJSON=async path=>{reads.push(path);return path==='/api/audio-playlists?view=summary'?{data:[{name:'Empty',items:null}]}:{data:null}}
  const state=createMusicState({requestJSON,key:item=>item.id,onChanged(){}})
  await state.load()
  await state.load()
  assert.deepEqual(state.favoriteItems,[])
  assert.deepEqual(state.playlists,[{name:'Empty',items:[]}])
  assert.equal(reads.filter(path=>path==='/api/audio-playlists?view=summary').length,1)
  assert.deepEqual(state.loadErrors,{favorites:'',playlists:''})
})

test('account collection load failures remain distinguishable from empty collections', async () => {
  const requestJSON=async path=>{throw new Error(path.includes('favorites')?'favorites offline':'playlists offline')}
  const state=createMusicState({requestJSON,key:item=>item.id,onChanged(){}})
  await state.load()
  assert.deepEqual(state.favoriteItems,[])
  assert.deepEqual(state.playlists,[])
  assert.deepEqual(state.loadErrors,{favorites:'favorites offline',playlists:'playlists offline'})
})

test('favorite toggles update local music state without reloading the full collection', async () => {
  const track={library_id:'music',id:'track',title:'Track'}
  const requests=[]
  const requestJSON=async(path,options={})=>{
    requests.push({path,method:options.method})
    if(path==='/api/audio-favorites'&&!options.method)return{data:[]}
    if(path==='/api/audio-playlists?view=summary')return{data:[]}
    if(path==='/api/audio-favorites'&&options.method==='PUT')return{data:null}
    throw new Error(path)
  }
  let changes=0
  const state=createMusicState({requestJSON,key:item=>`${item.library_id}:${item.id}`,onChanged(){changes++}})
  await state.load()
  assert.equal(await state.toggleFavorite(track),true)
  assert.equal(state.isFavorite(track),true)
  assert.deepEqual(state.favoriteItems,[track])
  assert.equal(await state.toggleFavorite(track),false)
  assert.equal(state.isFavorite(track),false)
  assert.deepEqual(state.favoriteItems,[])
  assert.equal(requests.filter(request=>request.path==='/api/audio-favorites'&&!request.method).length,1)
  assert.equal(requests.filter(request=>request.method==='PUT').length,2)
  assert.equal(changes,2)
})

test('rapid favorite toggles serialize per track and preserve the final intent', async () => {
  const track={library_id:'music',id:'track'}
  const writes=[]
  let releaseFirst
  const firstWrite=new Promise(resolve=>{releaseFirst=resolve})
  const requestJSON=async(path,options={})=>{
    if(path==='/api/audio-favorites'&&!options.method)return{data:[]}
    if(path==='/api/audio-playlists?view=summary')return{data:[]}
    if(path==='/api/audio-favorites'&&options.method==='PUT'){
      writes.push(JSON.parse(options.body).favorite)
      if(writes.length===1)await firstWrite
      return{data:null}
    }
    throw new Error(path)
  }
  const state=createMusicState({requestJSON,key:item=>`${item.library_id}:${item.id}`,onChanged(){}})
  await state.load()
  const first=state.toggleFavorite(track)
  const second=state.toggleFavorite(track)
  await tick()
  assert.deepEqual(writes,[true])
  releaseFirst()
  assert.deepEqual(await Promise.all([first,second]),[true,false])
  assert.deepEqual(writes,[true,false])
  assert.equal(state.isFavorite(track),false)
})
