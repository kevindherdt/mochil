import assert from 'node:assert/strict'
import test from 'node:test'
import { readFile } from 'node:fs/promises'

const source = await readFile(new URL('../static/book-cards.js', import.meta.url), 'utf8')

// The card follows the mock-up: the foot of the cover says how far in you are, and one pill under the author
// says what the card is. Importing the module needs the browser factory it is wrapped in, so the rule is read
// from the source -- a shape check, and it is what stops the pill silently disappearing again.
test('every card carries one pill, and a recording outranks the reading state', async () => {
  assert.match(source, /if \(book\?\.read_along\) return 'Narrated'/)
  assert.match(source, /if \(status === 'on_hold'\) return 'On hold'/)
  assert.match(source, /if \(status === 'dropped'\) return 'Dropped'/)
  assert.match(source, /return 'To read'/)
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
