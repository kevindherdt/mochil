import assert from 'node:assert/strict'
import test from 'node:test'
import { readFile } from 'node:fs/promises'

const staticRoot = new URL('../static/', import.meta.url)

// Bug: when a search has only 1-2 results, the results reuse .book-grid, which (before
// this fix) is an auto-fill, 1fr-column grid that is always exactly as wide as its
// container. A couple of cards then sit alone in the top-left corner of the page, in
// front of the full-height gradient backdrop, looking broken rather than intentional.
// There is no dedicated "search results" stylesheet: the fix lives in the shared grid
// rule in card-surface.css that search reuses, verified above in wide-grid-layout, plus
// an explicit sparse-results treatment so very short result sets read as deliberate.

test('a short catalog grid narrows towards content width instead of filling the viewport', async () => {
  const css = await readFile(new URL('card-surface.css', staticRoot), 'utf8')
  // auto-fit collapses the empty tracks beyond the last real card (unlike auto-fill, which
  // keeps reserving them), and justify-content:center then centres the remaining, real
  // tracks, so with 1-2 results the row is only as wide as those cards and sits centred,
  // not stretched edge-to-edge with blank space pinned to the right.
  // The sparse-results guard is the :not(:has(> :nth-child(5))) rule of the shared cover grid.
  assert.match(css, /\.book-grid:not\(\.book-list\):not\(:has\(> :nth-child\(5\)\)\)[^{]*\{[^}]*grid-template-columns:\s*repeat\(auto-fit,/)
  assert.match(css, /\.book-grid:not\(\.book-list\):not\(:has\(> :nth-child\(5\)\)\)[^{]*\{[^}]*justify-content:\s*center/)
})
