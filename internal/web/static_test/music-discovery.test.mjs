import assert from 'node:assert/strict'
import test from 'node:test'
import { musicDiscoveryHTML } from '../static/music-discovery.js'

const escapeHTML = value => String(value).replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('"', '&quot;')

test('music discovery renders bounded destinations and escapes group metadata', () => {
  const rowArguments = []
  const html = musicDiscoveryHTML({ played:[{title:'Played'}], recent:[], favorites:[], albums:[{name:'Album <One>',subtitle:'Artist & Co',artwork_item_id:'a'}], artists:[], playlists:[{name:'Quiet <mix>',items:[]}] }, {
    row:(...values)=>{ rowArguments.push(values); return `<article>${values[0].title}</article>` }, escapeHTML, groupArtworkURL:group=>`/art/${group.artwork_item_id}`,
  })
  assert.match(html, /Recently played/)
  assert.match(html, /data-discovery-destination="recently-played"/)
  assert.match(html, /data-discovery-destination="tracks" data-discovery-sort="recent"/)
  assert.match(html, /Album &lt;One>/)
  assert.match(html, /Artist &amp; Co/)
  assert.match(html, /data-art="\/art\/a"/)
  assert.doesNotMatch(html, /data-discovery-section="favorites"/)
  assert.doesNotMatch(html, /Favorite tracks to keep them close/)
  assert.match(html, /<h2>Playlists<\/h2>/)
  assert.match(html, /data-discovery-playlist="Quiet &lt;mix>"/)
  assert.deepEqual(rowArguments[0].slice(1), [false, false], 'recently played previews should omit dashboard actions and per-track progress')
})

test('music discovery reveals the Favorites shelf only when it has content', () => {
  const calls = []
  const html = musicDiscoveryHTML({ played:[], recent:[], favorites:[{title:'Saved'}], albums:[], artists:[], playlists:[] }, {
    row:(...args)=>{ calls.push(args); return `<article>${args[0].title}</article>` }, escapeHTML, groupArtworkURL:()=>'',
  })
  assert.match(html, /data-discovery-section="favorites"/)
  assert.match(html, /<article>Saved<\/article>/)
  assert.deepEqual(calls[0].slice(1), [false, false])
})

test('music discovery distinguishes empty sections from recoverable failures', () => {
  const html = musicDiscoveryHTML({
    played:[], recent:[], favorites:[], albums:[], artists:[], playlists:[],
    failures:{albums:true},
  }, { row:()=>'', escapeHTML, groupArtworkURL:()=>'' })

  assert.match(html, /Albums could not load/)
  assert.match(html, /data-discovery-destination="discover">Retry/)
  assert.match(html, /No artists found yet/)
  assert.doesNotMatch(html, /No albums found yet/)
})

test('Recently played becomes a row of cards when a card renderer is given, and stays a list otherwise', () => {
  const data = { played: [{ id: 'p1' }], recent: [{ id: 'r1' }], favorites: [], albums: [], artists: [], playlists: [] }
  const base = { row: item => `<li>${item.id}</li>`, escapeHTML: String, groupArtworkURL: () => '' }
  const withCards = musicDiscoveryHTML(data, { ...base, card: item => `<article>card ${item.id}</article>` })
  assert.match(withCards, /is-carousel"><article>card p1<\/article>/)
  assert.equal((withCards.match(/is-carousel/g) || []).length, 1)
  assert.match(withCards, /<li>r1<\/li>/)
  assert.doesNotMatch(musicDiscoveryHTML(data, base), /is-carousel/)
})
