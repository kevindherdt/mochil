import assert from 'node:assert/strict'
import test from 'node:test'
import { createMusicGroupCatalogue } from '../static/music-group-catalogue.js'

const response = (data, headers = {}) => ({ data, response: { headers: new Headers(headers) } })

test('music group catalogue owns album paging and absent next offsets', async () => {
  const paths = []
  const catalogue = createMusicGroupCatalogue({ requestJSON: async path => {
    paths.push(path)
    return response([{ name:'One' }], { 'X-Total-Count':'1' })
  } })
  assert.equal(await catalogue.loadGroups({ library:'music', browse:'albums' }), true)
  assert.deepEqual(catalogue.groups, [{ name:'One' }])
  assert.equal(catalogue.total, 1)
  assert.equal(catalogue.nextOffset, null)
  assert.match(paths[0], /group_type=albums/)
})

test('artist drill-down retains hierarchy and album tracks', async () => {
  const artist = { name:'Artist' }
  const album = { id:'stable-album', name:'Album', count:1 }
  const track = { id:'track' }
  const paths=[]
  const catalogue = createMusicGroupCatalogue({ requestJSON: async path => {
    paths.push(path)
    return path.startsWith('/api/audio-artist-albums') ? response([album]) : response([track])
  } })
  assert.deepEqual(await catalogue.open(artist, { library:'music', browse:'artists' }), { level:'artist', tracks:[] })
  assert.equal(catalogue.selectedArtist, artist)
  assert.deepEqual(catalogue.groups, [album])
  assert.deepEqual(await catalogue.open(album, { library:'music', browse:'artists' }), { level:'album', tracks:[track] })
  assert.equal(catalogue.selectedGroup, album)
  assert.match(paths.at(-1), /group_id=stable-album/)
  assert.equal(await catalogue.back({ library:'music', search:'' }), 'artist-or-albums')
  assert.equal(catalogue.selectedGroup, null)
})

test('superseded group responses cannot replace newer state', async () => {
  let release
  const first = new Promise(resolve => { release = resolve })
  let calls = 0
  const catalogue = createMusicGroupCatalogue({ requestJSON: async () => {
    calls += 1
    return calls === 1 ? first : response([{ name:'New' }])
  } })
  const stale = catalogue.loadGroups({ library:'old', browse:'albums' })
  const current = catalogue.loadGroups({ library:'new', browse:'albums', reset:true })
  await current
  release(response([{ name:'Old' }]))
  assert.equal(await stale, false)
  assert.deepEqual(catalogue.groups, [{ name:'New' }])
})

test('navigation is bounded to the latest request', async () => {
  const catalogue = createMusicGroupCatalogue({ requestJSON: async () => response([{ bucket:'A', offset:2 }]) })
  assert.equal(await catalogue.loadNavigation({ library:'music', sort:'title', mode:'tracks' }), true)
  assert.deepEqual(catalogue.navigation, [{ bucket:'A', offset:2 }])
  assert.equal(await catalogue.loadNavigation({ library:'music', sort:'title', mode:'queue' }), false)
  assert.deepEqual(catalogue.navigation, [])
})

test('navigation and group requests can settle together without cancelling each other', async () => {
  let releaseNavigation
  let releaseGroups
  const navigation = new Promise(resolve => { releaseNavigation = resolve })
  const groups = new Promise(resolve => { releaseGroups = resolve })
  const catalogue = createMusicGroupCatalogue({ requestJSON: async path => (
    path.startsWith('/api/audio-navigation') ? navigation : groups
  ) })

  const navigationLoad = catalogue.loadNavigation({ library:'music', sort:'artist', mode:'artists' })
  const groupLoad = catalogue.loadGroups({ library:'music', browse:'artists' })
  releaseGroups(response([{ name:'Paramore' }], { 'X-Total-Count':'1' }))
  releaseNavigation(response([{ bucket:'P', offset:0 }]))

  assert.deepEqual(await Promise.all([navigationLoad, groupLoad]), [true, true])
  assert.deepEqual(catalogue.navigation, [{ bucket:'P', offset:0 }])
  assert.deepEqual(catalogue.groups, [{ name:'Paramore' }])
})

test('going back from an artist shows as many artists as were loaded when it was opened', async () => {
  const page = start => Array.from({ length: 72 }, (_, i) => ({ name: `Artist ${start + i}` }))
  const catalogue = createMusicGroupCatalogue({ requestJSON: async path => {
    if (path.startsWith('/api/audio-artist-albums')) return response([{ name: 'Album' }])
    const offset = Number(new URLSearchParams(path.split('?')[1]).get('offset'))
    return response(page(offset), { 'X-Total-Count': '400', 'X-Next-Offset': String(offset + 72) })
  } })
  const context = { library: 'music', search: '' }
  await catalogue.loadGroups({ ...context, browse: 'artists' })
  await catalogue.loadGroups({ ...context, browse: 'artists', reset: false })
  await catalogue.loadGroups({ ...context, browse: 'artists', reset: false })
  assert.equal(catalogue.groups.length, 216)
  await catalogue.open({ name: 'Artist 200' }, { library: 'music', browse: 'artists' })
  assert.equal(await catalogue.back(context), 'artists')
  assert.equal(catalogue.groups.length, 216, 'the list is as long as before, so the old scroll position exists again')
  assert.equal(catalogue.groups[200].name, 'Artist 200')
})
