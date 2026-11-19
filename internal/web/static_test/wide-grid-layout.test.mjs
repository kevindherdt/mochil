import assert from 'node:assert/strict'
import test from 'node:test'
import { readFile } from 'node:fs/promises'

const staticRoot = new URL('../static/', import.meta.url)

// Bug: on wide desktop screens (1440px+), the Home/Discover card grids (book-grid,
// series-cover-grid and the shared media-grid.css grids) use auto-fill columns sized
// in 1fr, which creates empty trailing column *tracks* that still occupy row width.
// A handful of items then sits pinned to the left edge, leaving a large, intentional-
// looking empty area to the right and (because rows stop early) below. The same grid
// is reused to show search results, so 1-2 results end up as a single small card in
// the top-left corner of an otherwise empty page.
//
// Fix: cap the column's maximum width (instead of letting it stretch via 1fr to fill
// the full row) and center the occupied columns with justify-content, so a short grid
// reads as an intentional, centered block instead of a left-pinned fragment, on any
// screen from 1024 to 1920px, while narrower layouts (<=760px) are untouched.

test('one shared cover grid serves every library kind and a short row stays capped and centred', async () => {
  const css = await readFile(new URL('card-surface.css', staticRoot), 'utf8')
  const shared = css.match(/@media \(min-width: 761px\) \{\s*\.book-grid:not\(\.book-list\), \.series-cover-grid, [^{]*\{([^}]*)\}\s*([^{]*)\{([^}]*)\}/)
  assert.ok(shared, 'the shared grid rule and its few-results rule should live in the 761px breakpoint')
  assert.match(shared[1], /grid-template-columns:\s*repeat\(auto-fill,\s*minmax\(clamp\(\d+px,\s*[\d.]+vw,\s*\d+px\),\s*1fr\)\)/, 'the column minimum scales with the viewport and the columns stretch, so no wide gutters are left')
  assert.match(shared[1], /gap:\s*clamp\(18px,\s*1\.6vw,\s*28px\)/)
  assert.match(shared[2], /:not\(:has\(> :nth-child\(5\)\)\)/, 'a handful of results must not stretch into giant cards')
  assert.match(shared[3], /grid-template-columns:\s*repeat\(auto-fit,\s*minmax\(196px,\s*\d+px\)\)/)
  assert.match(shared[3], /justify-content:\s*center/, 'a short row of cards must be centred, not left-aligned')
})

test('the cover grid has no per-kind or per-library override and the same rule covers series, focus and collections', async () => {
  const css = await readFile(new URL('card-surface.css', staticRoot), 'utf8')
  const rule = css.match(/@media \(min-width: 761px\) \{\s*(\.book-grid:not\(\.book-list\), \.series-cover-grid, [^{]*)\{/)[1]
  for (const sel of ['.series-books .grid', '.series-focus-books .grid', '.collection-index>.grid']) assert.ok(rule.includes(sel), sel + ' must share the rule')
  const grid = css + await readFile(new URL('book-list.css', staticRoot), 'utf8')
  assert.doesNotMatch(grid, /\[data-(kind|library)[^\]]*\][^{]*(book-grid|series-cover-grid)[^{]*\{[^}]*grid-template-columns/)
})

test('the shared media grid (music, podcasts, audiobooks, documents) also caps width and centres', async () => {
  const css = await readFile(new URL('media-grid.css', staticRoot), 'utf8')
  const block = css.match(/@media \(min-width: 761px\) \{[\s\S]*?\.document-grid[\s\S]*?\)\s*\{([^}]*)\}/)
  assert.ok(block, 'the shared grid rule should exist inside the 761px breakpoint')
  const body = block[1]
  assert.match(body, /grid-template-columns:\s*repeat\(auto-fit,\s*minmax\(var\(--grid-min, 196px\),\s*var\(--grid-max,\s*\d+px\)\)\)/, 'columns must have a capped maximum width and collapse empty tracks')
  assert.match(body, /justify-content:\s*center/, 'a short row of cards must be centred, not left-aligned')
})

test('titles keep two lines of room in every cover grid', async () => {
  const css = await readFile(new URL('card-surface.css', staticRoot), 'utf8')
  assert.match(css, /\.series-cover-grid \.series-card h2 \{ min-height: 2\.48em; \}/)
  assert.match(css, /\.series-cover-grid \.series-card h2 \{ font-weight: 650; color: var\(--text\); \}|\.series-cover-grid \.series-card h2 \{ font-weight: 650/)
})
