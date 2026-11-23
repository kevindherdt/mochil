import assert from 'node:assert/strict'
import test from 'node:test'
import { readFile } from 'node:fs/promises'

const source = await readFile(new URL('../static/book-cards.js', import.meta.url), 'utf8')

// The card shows a reading state two ways, and only two: the strip on the cover while you are part-way in, and a
// small green tick once the book is finished. Nothing is printed for "nobody has opened this" -- that is the
// default state of a shelf, and a pill on every unread card is noise (his ruling, 21 Nov). The rule is read from
// the source because importing the module needs the browser factory it is wrapped in.
test('a card words the states the cover cannot show, and stays quiet otherwise', async () => {
  assert.match(source, /if \(book\?\.read_along\) return 'Narrated'/)
  assert.match(source, /if \(status === 'on_hold'\) return 'On hold'/)
  assert.match(source, /if \(status === 'dropped'\) return 'Dropped'/)
  assert.doesNotMatch(source, /return 'To read'/)
  assert.doesNotMatch(source, /return 'In progress'/)
  assert.doesNotMatch(source, /return 'Read'/)
  assert.doesNotMatch(source, /return 'Unread'/)
})

test('a finished book carries one small tick instead of a full-width strip', async () => {
  assert.match(source, /function finishedMark\(book\)/) // eslint-disable-line no-useless-escape
  assert.match(source, /if \(state\.finished\) return ''/)
  assert.match(source, /aria-label="Finished"/)
})

// A book opened once sits at a fraction of a percent; a bar at that width is a sliver that reads as a bug.
test('a bar is only drawn when there is a percentage to draw', async () => {
  assert.match(source, /state\.percentage >= 1/)
})

test('the foot of the cover carries the progress words and the bar', async () => {
  const css = await readFile(new URL('../static/card-shell.css', import.meta.url), 'utf8')
  assert.match(css, /\.book \.cover-foot\s*\{/)
  assert.match(css, /\.book \.cover-foot-bar::after/)
  assert.match(css, /width: var\(--progress, 0%\)/)
})

test('series cards use the same card language as single books', async () => {
  const css = await readFile(new URL('../static/card-shell.css', import.meta.url), 'utf8')
  assert.match(css, /\.series-card \.cover-frame\s*\{[^}]*border-radius/)
  assert.match(css, /\.series-card p\s*\{[^}]*border-radius: var\(--r-pill/)
})
test('the finished tick is small, round and out of the progress strip', async () => {
  const css = await readFile(new URL('../static/card-surface.css', import.meta.url), 'utf8')
  assert.match(css, /\.cover-done\s*\{/)
  assert.match(css, /width: 18px/)
  assert.match(css, /border-radius: 50%/)
})
