import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { createDestinationRail } from '../static/destination-rail.js'

const cssPath = fileURLToPath(new URL('../static/destination-rail.css', import.meta.url))
const css = readFileSync(cssPath, 'utf8')

// Tiny cascade stub for the one property we care about: does `.has-icon span`
// end up visible at a 390px phone width? Real rule order is base, then
// @media(min-width:N), then @media(max-width:N), applied in file order.
function labelDisplayAt(viewportWidth) {
  let display = 'inline' // UA default for <span>
  const ruleRe = /(@media\s*\(([a-z-]+):(\d+)px\)\s*{\s*)?\.destination-rail button\.has-icon(\.active)?\s*span\s*{[^}]*display:\s*([a-z]+)/g
  let match
  while ((match = ruleRe.exec(css))) {
    const [, mediaPrefix, feature, pxValue, onlyWhenActive, value] = match
    if (onlyWhenActive) continue // irrelevant to the non-active buttons we're checking
    if (mediaPrefix) {
      const px = Number(pxValue)
      const applies = feature === 'min-width' ? viewportWidth >= px : viewportWidth <= px
      if (!applies) continue
    }
    display = value
  }
  return display
}

test('the mobile nav row renders a visible, accessible label on every icon button', () => {
  const classes = new Set()
  const buttons = []
  const element = {
    classList: { add: v => classes.add(v), toggle() {}, remove() {} },
    set innerHTML(html) {
      this._html = html
      buttons.length = 0
      // Pull each button's accessible attributes and its label <span> text out of the markup.
      for (const m of html.matchAll(/<button[^>]*aria-label="([^"]*)"[^>]*>(?:(?!<\/button>).)*?<span>([^<]*)<\/span>/gs)) {
        buttons.push({ ariaLabel: m[1], visibleText: m[2] })
      }
    },
    get innerHTML() { return this._html },
    addEventListener() {},
    querySelectorAll() { return [] },
  }
  createDestinationRail({
    element,
    destinations: [
      ['discover', 'Discover', '<path d="M0 0"/>'],
      ['series', 'Series', '<path d="M0 0"/>'],
      ['authors', 'Authors', '<path d="M0 0"/>'],
      ['books', 'All', '<path d="M0 0"/>'],
      ['collections', 'Collections', '<path d="M0 0"/>'],
    ],
    attribute: 'data-publication-destination',
    getValue: () => 'discover',
    setValue() {},
    async onChange() {},
  })

  assert.equal(buttons.length, 5, 'every destination should render as a button')
  for (const button of buttons) {
    assert.ok(button.ariaLabel, 'every icon button needs an accessible name')
    assert.notEqual(button.visibleText.trim(), '', `"${button.ariaLabel}" has no sighted-user-visible label`)
  }
})

test('icon button labels stay visible (not display:none) at a 390px phone width', () => {
  assert.notEqual(labelDisplayAt(390), 'none', 'labels must not be hidden on narrow phones — they become undiscoverable icon-only buttons')
})
