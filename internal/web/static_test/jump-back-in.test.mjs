import assert from 'node:assert/strict'
import test from 'node:test'
import { readFile } from 'node:fs/promises'
import { dismissRequest, kindFamily, kindIcon, jumpCardHTML, jumpDetail, jumpItemHTML, kindLabel, mergeJumpItems } from '../static/jump-back-in.js'

const escapeHTML = value => String(value).replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('"', '&quot;')
const book = (id, at, extra = {}) => ({ id, library_id: 'b', library_kind: 'books', title: id, authors: ['A'], cover_url: '/c', read_url: '/r/' + id, fraction: 0.5, updated_at: at, ...extra })
const video = (id, at, extra = {}) => ({ id, library_id: 'v', library_kind: 'video', title: id, stream_url: '/s', progress: { position_seconds: 600, duration_seconds: 1800, updated_at: at, ...extra } })
const audio = (id, at, kind = 'audiobooks') => ({ id, library_id: 'a', library_kind: kind, title: id, progress: { position_seconds: 10, duration_seconds: 4000, updated_at: at } })

test('everything in progress is merged, newest activity first, without finished, dismissed or music', () => {
  const merged = mergeJumpItems({
    books: [book('old', 1), book('done', 9, { finished: true })],
    videos: [video('show', 5), video('hidden', 8, { dismissed: true }), { ...video('nostream', 7), stream_url: '' }],
    audios: [audio('story', 3), audio('song', 6, 'music')],
  })
  assert.deepEqual(merged.map(entry => entry.item.id), ['show', 'story', 'old'])
  assert.deepEqual(merged.map(entry => entry.type), ['video', 'audio', 'book'])
})

test('libraries that are hidden stay out, and the row is capped', () => {
  const many = Array.from({ length: 9 }, (_, i) => book('b' + i, i))
  assert.equal(mergeJumpItems({ books: many }, () => true, 6, 9).length, 6)
  assert.equal(mergeJumpItems({ books: many }).length, 4)
  assert.equal(mergeJumpItems({ books: many }, id => id !== 'b').length, 0)
})

test('details speak the language of each kind', () => {
  assert.equal(jumpDetail({ type: 'book', item: book('x', 1) }), '50% read')
  assert.equal(jumpDetail({ type: 'video', item: video('x', 1) }), '20 min left')
  assert.equal(jumpDetail({ type: 'audio', item: audio('x', 1) }), '1 h 6 min left')
  assert.equal(kindLabel({ library_kind: 'light-novels' }), 'Light novel')
  assert.equal(kindLabel({ library_kind: 'tv' }), 'TV show')
  assert.equal(kindLabel({ library_kind: 'mystery' }), 'Media')
})

test('a book card is a link, a video card is a button, and both carry their type tag and escape the title', () => {
  const b = jumpCardHTML({ type: 'book', item: { ...book('x', 1), title: '<b>"x"' } }, escapeHTML)
  assert.match(b, /^<a class="jump-card" href="\/r\/x"/)
  assert.match(b, /jump-tag" role="img" aria-label="Book" title="Book"><svg class="kind-icon"/)
  assert.doesNotMatch(b, /<b>/)
  const v = jumpCardHTML({ type: 'video', item: video('x', 1) }, escapeHTML)
  assert.match(v, /^<button type="button" class="jump-card" data-jump-type="video" data-library="v" data-item="x"/)
  assert.match(v, /jump-tag" role="img" aria-label="Video" title="Video"><svg class="kind-icon"/)
})

test('a manga chapter card names its series in the visible title and remove action', () => {
  const item = book('x', 1, { library_kind: 'manga', title: 'Chapter 0', series: 'Aku no Kyouten' })
  const html = jumpItemHTML({ type: 'book', item }, escapeHTML)
  assert.match(html, /<strong>Aku no Kyouten — Chapter 0<\/strong>/)
  assert.match(html, /aria-label="Remove Aku no Kyouten — Chapter 0 from Jump back in"/)
})

test('book cards preserve normal and already qualified titles and handle absent series', () => {
  for (const [title, series, expected] of [
    ['The Great Hunt', 'The Wheel of Time', 'The Great Hunt'],
    ['Aku no Kyouten — Chapter 0', 'Aku no Kyouten', 'Aku no Kyouten — Chapter 0'],
    ['Chapter 0', '', 'Chapter 0'],
    ['', 'Series', 'Untitled'],
  ]) {
    const html = jumpItemHTML({ type: 'book', item: book('x', 1, { title, series }) }, escapeHTML)
    assert.ok(html.includes(`<strong>${expected}</strong>`), html)
    assert.ok(html.includes(`data-title="${expected}"`), html)
  }
})

test('a series identical to its chapter title is not repeated', () => {
  const html = jumpItemHTML({ type: 'book', item: book('x', 1, { title: 'Chapter 0', series: 'chapter 0' }) }, escapeHTML)
  assert.match(html, /<strong>Chapter 0<\/strong>/)
})

test('chapter series and title are escaped in cards and remove actions', () => {
  const html = jumpItemHTML({ type: 'book', item: book('x', 1, { title: 'Chapter <0>', series: 'A & B' }) }, escapeHTML)
  assert.match(html, /<strong>A &amp; B — Chapter &lt;0><\/strong>/)
  assert.match(html, /aria-label="Remove A &amp; B — Chapter &lt;0> from Jump back in"/)
})

test('no kind crowds out the others: the newest four of each, ordered together', () => {
  const now = Math.floor(Date.UTC(2026, 10, 18, 3, 0, 0) / 1000) * 1000
  const books = Array.from({ length: 6 }, (_, i) => book('b' + i, now - (i + 1) * 1000))
  const merged = mergeJumpItems({ books, videos: [video('show', now / 1000 - 100)], audios: [audio('story', now / 1000 - 50)] })
  assert.deepEqual(merged.map(entry => entry.item.id), ['b0', 'b1', 'b2', 'b3', 'story', 'show'])
})

// Bug: /api/reading stamps updated_at in milliseconds (reading.Position.UpdatedAt is
// time.Now().UnixMilli) while video and audio progress stamp it in seconds
// (video.Progress.UpdatedAt / audio.Progress.UpdatedAt are time.Now().Unix). Comparing
// the raw numbers put every book ahead of an episode watched a minute ago, however old
// the book was, so "newest activity first" was really "books first".
test('a book last read yesterday does not outrank an episode watched a minute ago', () => {
  const now = Date.UTC(2026, 10, 18, 2, 0, 0)
  const merged = mergeJumpItems({
    books: [book('yesterday', now - 24 * 3600 * 1000)],
    videos: [video('just-now', Math.floor(now / 1000))],
    audios: [audio('an-hour-ago', Math.floor(now / 1000) - 3600)],
  })
  assert.deepEqual(merged.map(entry => entry.item.id), ['just-now', 'an-hour-ago', 'yesterday'])
})

test('video and podcast art keeps its blurred backdrop, from the URL its own image already loads', () => {
  const videoHtml = jumpCardHTML({ type: 'video', item: { ...video('x', 1), artwork_url: "/art/it's (1).jpg" } }, escapeHTML)
  assert.match(videoHtml, /class="cover"/, 'the artwork gets the shared error fallback')
  assert.match(videoHtml, /style="--art:url\('\/art\/it%27s%20%281%29\.jpg'\)"/)
  assert.match(jumpCardHTML({ type: 'audio', item: { ...audio('x', 1), artwork_url: '/a.jpg' } }, escapeHTML), /style="--art:url\('\/a\.jpg'\)"/)
})

// The book card's <img> is served a ?w= sized cover by the bounded loader; the CSS backdrop used the
// full-size cover_url, so every card on Home paid for a second, larger fetch just to blur it.
test('a book card no longer pulls its full-size cover only for a backdrop', () => {
  const html = jumpCardHTML({ type: 'book', item: book('x', 1) }, escapeHTML)
  assert.doesNotMatch(html, /--art:/)
  assert.match(html, /<img class="cover" decoding="async" data-src="\/c"/, 'the sized cover still renders')
})

test('each kind is hidden through its own endpoint, and the same call brings it back', () => {
  const item = { library_id: 'l/1', id: 'x y' }
  assert.deepEqual(dismissRequest('book', item, true), { url: '/api/reading/l%2F1/x%20y/dismiss', options: { method: 'POST' } })
  assert.equal(dismissRequest('book', item, false).url, '/api/reading/l%2F1/x%20y/show')
  const video = dismissRequest('video', item, true)
  assert.equal(video.url, '/api/video-progress/l%2F1/x%20y')
  assert.equal(video.options.method, 'PATCH')
  assert.equal(JSON.parse(video.options.body).dismissed, true)
  assert.equal(JSON.parse(dismissRequest('audio', item, false).options.body).dismissed, false)
  assert.match(dismissRequest('audio', item, true).url, /^\/api\/audio-progress\//)
})

test('a card is wrapped with a remove button that names the item and escapes it', () => {
  const html = jumpItemHTML({ type: 'video', item: { ...video('x', 1), title: '<i>"x"' } }, escapeHTML)
  assert.match(html, /^<article class="jump-item"><button type="button" class="jump-card"/)
  assert.match(html, /data-jump-dismiss data-jump-type="video" data-library="v" data-item="x"/)
  assert.doesNotMatch(html, /<i>/)
})

test('a book with its own recording shows headphones in its tag; one without does not', () => {
  assert.match(jumpCardHTML({ type: 'book', item: { ...book('x', 1), read_along: true } }, escapeHTML), /jump-tag-audio/)
  assert.doesNotMatch(jumpCardHTML({ type: 'book', item: book('x', 1) }, escapeHTML), /jump-tag-audio/)
  assert.doesNotMatch(jumpCardHTML({ type: 'video', item: { ...video('x', 1), read_along: true } }, escapeHTML), /jump-tag-audio/)
})

// Bug: Jump back in can show the same book twice - once as plain EPUB reading progress,
// once as a read-along entry - with the only difference being the small headphones icon
// in the corner tag. That is too subtle to notice, so it reads as a duplicate. A short
// text label next to the title makes the two cards visibly different on their own.
test('a read-along card carries a visible "Read-along" text label next to its title, not just an icon', () => {
  const html = jumpCardHTML({ type: 'book', item: { ...book('x', 1), read_along: true } }, escapeHTML)
  assert.match(html, /<\/strong>.*Read-along/s)
  assert.doesNotMatch(jumpCardHTML({ type: 'book', item: book('x', 1) }, escapeHTML), /Read-along/)
  // A read-along video-progress entry has no such concept; it must not spuriously gain the label.
  assert.doesNotMatch(jumpCardHTML({ type: 'video', item: { ...video('x', 1), read_along: true } }, escapeHTML), /Read-along/)
})

test('each kind of card gets its colour family, and every family has a chip colour that reads against its own text colour', async () => {
  assert.equal(kindFamily({ type: 'book', item: book('x', 1) }), 'book')
  assert.equal(kindFamily({ type: 'video', item: video('x', 1) }), 'video')
  assert.equal(kindFamily({ type: 'audio', item: audio('x', 1) }), 'audio')
  assert.equal(kindFamily({ type: 'audio', item: audio('x', 1, 'podcasts') }), 'podcast')
  assert.match(jumpCardHTML({ type: 'video', item: video('x', 1) }, escapeHTML), /data-family="video"/)
  assert.match(jumpCardHTML({ type: 'book', item: book('x', 1) }, escapeHTML), /data-family="book"/)
  const css = await readFile(new URL('../static/card-surface.css', import.meta.url), 'utf8')
  const lum = hex => { const [r, g, b] = [1, 3, 5].map(i => parseInt(hex.slice(i, i + 2), 16) / 255).map(v => (v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4)); return 0.2126 * r + 0.7152 * g + 0.0722 * b }
  for (const family of ['book', 'video', 'audio', 'podcast']) {
    const fill = css.match(new RegExp(`--kind-${family}:\\s*(#[0-9a-f]{6})`, 'i'))?.[1]
    const ink = css.match(new RegExp(`--kind-${family}-ink:\\s*(#[0-9a-f]{6})`, 'i'))?.[1]
    assert.ok(fill && ink, `${family} lacks a chip colour`)
    const [hi, lo] = [lum(fill), lum(ink)].sort((a, b) => b - a)
    assert.ok((hi + 0.05) / (lo + 0.05) >= 4.5, `${family} chip text is hard to read`)
  }
})

test('cards no longer duplicate the kind badge with a second icon on hover', () => {
  const book1 = jumpCardHTML({ type: 'book', item: book('x', 1) }, escapeHTML)
  assert.doesNotMatch(book1, /jump-go/)
  const video1 = jumpCardHTML({ type: 'video', item: video('x', 1) }, escapeHTML)
  assert.doesNotMatch(video1, /jump-go/)
})

test('every kind of library has its own icon and a strong colour whose text reads against it', async () => {
  const tokens = await readFile(new URL('../static/card-surface.css', import.meta.url), 'utf8')
  const token = name => tokens.match(new RegExp(`--${name}:\\s*(#[0-9a-f]{6})`, 'i'))?.[1]
  // A kind may point at a palette token instead of writing a hex; resolve it so the same colour is
  // checked, and so the palette stays the one place the hue is written.
  const css = (await readFile(new URL('../static/kind-colors.css', import.meta.url), 'utf8'))
    .replace(/var\(--kind-([a-z]+)-ink\)/gi, (_, name) => token(`kind-${name}-ink`) || '')
    .replace(/var\(--kind-([a-z]+)\)/gi, (_, name) => token(`kind-${name}`) || '')
  const lum = hex => { const [r, g, b] = [1, 3, 5].map(i => parseInt(hex.slice(i, i + 2), 16) / 255).map(v => (v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4)); return 0.2126 * r + 0.7152 * g + 0.0722 * b }
  const rules = [...css.matchAll(/--kind:(#[0-9a-f]{6});--kind-ink:(#[0-9a-f]{6})/gi)]
  assert.ok(rules.length >= 10)
  for (const [, fill, ink] of rules) {
    const [light, dark] = [lum(fill), lum(ink)].sort((a, b) => b - a)
    assert.ok((light + 0.05) / (dark + 0.05) >= 4.5, `${fill} on ${ink}`)
  }
  assert.equal(new Set(rules.map(rule => rule[1])).size, rules.length)
  const icons = new Set()
  for (const kind of ['books', 'lightnovels', 'manga', 'comics', 'movies', 'tv', 'anime', 'music', 'audiobooks', 'podcasts']) icons.add(kindIcon({ library_kind: kind }))
  assert.equal(icons.size, 10)
})
