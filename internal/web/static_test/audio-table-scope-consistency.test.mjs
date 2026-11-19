import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

const staticRoot = new URL('../static/', import.meta.url)

// Bug: on desktop (>=900px), the music Discover "Recently added" rail renders its
// <article class="audio-item"> rows one level deeper than the main Tracks list
// (wrapped in a `.music-discovery-tracks` container inside #audio-items), so the
// `#audio-index[data-kind="music"] #audio-items>.audio-item:has(.audio-col) ...`
// rules that switch the row into a grid of columns and hide the now-redundant
// `<small>` summary never match there (the `>` direct-child combinator fails).
// But the separate, unscoped-by-nesting `#audio-index[data-kind="music"] .audio-col`
// rule that reveals those columns *does* match (it's a plain descendant selector).
// Net effect: the artist/album/time columns render visible *underneath* the
// already-complete `<small>Artist · Album · Time</small>` line, duplicating the
// text (and truncating the artist to "Para…") instead of laying out as a table row.
//
// Fix: every rule that shows/hides the duplicate parts of a music row (the grid
// layout, hiding <small>, and revealing .audio-col) must share the exact same
// selector scope, so they can only ever apply together.

test('music row column visibility is scoped identically to the row grid layout and the hidden duplicate <small>', async () => {
  const css = await readFile(new URL('audio-table.css', staticRoot), 'utf8')

  const gridRule = css.match(/#audio-index\[data-kind="music"\] ([^{]*?\.audio-open)\{display:grid/)
  assert.ok(gridRule, 'the rule that turns a music row into a grid of columns should exist')
  const scope = gridRule[1].replace(/\.audio-open$/, '').trim()
  assert.ok(scope.length > 0, 'the grid-layout rule must be scoped to specific rows, not every .audio-open')

  const hideSmallRule = css.match(/#audio-index\[data-kind="music"\] ([^{]*?small)\{display:none\}/)
  assert.ok(hideSmallRule, 'a rule hiding the duplicate <small> summary should exist')
  assert.ok(hideSmallRule[1].startsWith(scope), 'hiding the duplicate <small> must use the same row scope as the grid layout, or it can go stale independently')

  const showColRule = css.match(/#audio-index\[data-kind="music"\] ([^{]*?\.audio-col)\{display:block/)
  assert.ok(showColRule, 'a rule revealing .audio-col columns should exist')
  assert.ok(showColRule[1].startsWith(scope), 'revealing .audio-col columns must use the same row scope as the grid layout — otherwise a row nested one level deeper (e.g. the Discover "Recently added" rail) can show the columns without the grid layout or the hidden duplicate <small>, duplicating the artist/album/time text')
})
