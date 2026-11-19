import assert from 'node:assert/strict'
import test from 'node:test'
import {
  audiobookFocusHTML,
  audiobookOverviewHTML,
  podcastFocusHTML,
  podcastOverviewHTML,
  spokenMediaSurface,
} from '../static/audio-media-views.js'
import { favoriteTrackView } from '../static/music-favorite-view.js'

const escapeHTML = value => String(value).replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('"', '&quot;')

test('music favorites retain filtering and artist-aware ordering outside spoken views', () => {
  const natural = new Intl.Collator('en', {numeric:true,sensitivity:'base'})
  const items = [
    {library_id:'music',title:'Second',artist:'Beta',album:'Album'},
    {library_id:'music',title:'First',artist:'Alpha',album:'Album'},
    {library_id:'other',title:'Hidden',artist:'Alpha',album:'Album'},
  ]
  assert.deepEqual(
    favoriteTrackView(items, {libraryID:'music',query:'',sort:'artist',natural}).map(item => item.title),
    ['First', 'Second'],
  )
  assert.deepEqual(favoriteTrackView(items, {libraryID:'music',query:'second',sort:'title',natural}).map(item => item.title), ['Second'])
})

test('audiobook overview renders compact server summaries', () => {
  const html = audiobookOverviewHTML([{name:'Series <Two>',count:3,artist:'Author & Co',artwork_url:'/art?id="one"'}], 'covers', escapeHTML)
  assert.match(html, /3 audiobooks · Author &amp; Co/)
  assert.match(html, /Series &lt;Two>/)
  assert.match(html, /data-art="\/art\?id=&quot;one&quot;"/)
  assert.doesNotMatch(html, /undefined/)
})

test('spoken-media artwork fallbacks use the shared SVG icon system', () => {
  const podcast = podcastOverviewHTML([{title:'Offline show',episode_count:2}], 'covers', escapeHTML)
  const audiobook = audiobookOverviewHTML([{name:'Offline series',count:1}], 'list', escapeHTML)
  assert.match(podcast, /class="audio-art-placeholder"/)
  assert.match(audiobook, /class="audio-art-placeholder"/)
  assert.doesNotMatch(`${podcast}${audiobook}`, /◉/)
})

test('spoken-media back controls use labelled SVG controls instead of text glyphs', () => {
  const podcast = podcastFocusHTML('Show', [], 0, () => '', escapeHTML)
  const audiobook = audiobookFocusHTML('Series', [], 'list', () => '', () => '', escapeHTML)
  assert.match(podcast, /class="podcast-back"><svg[^>]*aria-hidden="true"/)
  assert.match(podcast, /<span>All podcasts<\/span>/)
  assert.match(audiobook, /class="audiobook-back"><svg[^>]*aria-hidden="true"/)
  assert.doesNotMatch(`${podcast}${audiobook}`, /←/)
})

test('spoken media surface keeps podcast search and audiobook layouts family-owned', () => {
  const natural = new Intl.Collator('en', {numeric:true,sensitivity:'base'})
  const podcasts = spokenMediaSurface({
    kind:'podcasts', browse:'', items:[], search:'second', natural, row:() => '', audiobookCard:() => '', escapeHTML,
    podcast:{feeds:[{title:'First'},{title:'Second',episode_count:2}],show:'',episodes:[],total:0},
    audiobook:{}, layouts:{podcasts:'covers'},
  })
  assert.match(podcasts.html, /Second/)
  assert.doesNotMatch(podcasts.html, /First/)

  const audiobooks = spokenMediaSurface({
    kind:'audiobooks', browse:'all', items:[{title:'Book'}], search:'', natural, row:item=>`<article>${item.title}</article>`, audiobookCard:item=>`<button>${item.title}</button>`, escapeHTML,
    podcast:{feeds:[]}, audiobook:{groups:[],books:[]}, layouts:{audiobookItems:'covers'},
  })
  assert.match(audiobooks.html, /class="audiobook-book-grid"/)
  assert.match(audiobooks.html, /<button>Book<\/button>/)
  assert.equal(spokenMediaSurface({...audiobooks, kind:'music'}), null)
})
