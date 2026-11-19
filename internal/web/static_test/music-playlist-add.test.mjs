import assert from 'node:assert/strict'
import test from 'node:test'
import { createMusicState } from '../static/music-state.js'

test('adding a playlist item preserves tracks added on another device', async () => {
  const one = {library_id:'music', id:'one'}, remote = {library_id:'music', id:'remote'}, added = {library_id:'music', id:'added'}
  let serverItems = [one]
  const writes = []
  const state = createMusicState({key: item => item.id, onChanged(){}, requestJSON: async (path, options = {}) => {
    if (path === '/api/audio-favorites') return {data:[]}
    if (path === '/api/audio-playlists?view=summary' && !options.method) return {data:[{name:'Saved',items:serverItems.slice()}]}
    writes.push({path, ...options})
    assert.equal(options.method, 'POST')
    assert.deepEqual(JSON.parse(options.body), {library_id:'music',item_id:'added'})
    serverItems.push(added)
    return {data:null}
  }})
  await state.load()
  serverItems.push(remote)
  await state.addPlaylistItem('Saved', added)
  assert.equal(writes.length, 1)
  assert.deepEqual(state.playlists[0].items, [one,remote,added])
})

test('failed single-item add preserves the local playlist and propagates the error', async () => {
  const one = {library_id:'music',id:'one'}
  let changed = 0
  const state = createMusicState({key:item=>item.id,onChanged(){changed++},requestJSON:async (path,options={})=>{
    if(options.method) throw new Error('Could not save')
    return {data:path==='/api/audio-playlists?view=summary'?[{name:'Saved',items:[one]}]:[]}
  }})
  await state.load()
  await assert.rejects(state.addPlaylistItem('Saved',{library_id:'music',id:'two'}),/Could not save/)
  assert.deepEqual(state.playlists[0].items,[one])
  assert.equal(changed,0)
})
