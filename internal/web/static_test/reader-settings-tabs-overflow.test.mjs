import assert from 'node:assert/strict'
import test from 'node:test'
import { readFile } from 'node:fs/promises'
import { railOverflow } from '../static/destination-rail.js'

// At 390px wide, the Settings tablist (#reader-settings .rs-tabs) overflows
// (measured scrollWidth 452px vs clientWidth 352px) and the "Read aloud" tab
// sits off screen with no visual cue it is reachable by scrolling. The fix
// mirrors the mobile A-Z jump bar and destination rail: toggle
// has-tabs-before/has-tabs-after classes on scroll/resize, and fade the
// tablist edges with those classes in CSS.

test('the settings tablist CSS fades its edges when tabs overflow', async () => {
  const css = await readFile(new URL('../static/reader-readium.css', import.meta.url), 'utf8')
  assert.match(css, /\.rs-tabs\.has-tabs-after\{[^}]*mask-image:[^}]*\}/, 'no edge-fade rule for tabs hidden after the viewport')
  assert.match(css, /\.rs-tabs\.has-tabs-before(:not\(\.has-tabs-after\))?\{[^}]*mask-image:[^}]*\}/, 'no edge-fade rule for tabs hidden before the viewport')
})

test('the settings tablist tracks scroll position to toggle the overflow affordance classes', async () => {
  const source = await readFile(new URL('../static/reader-readium.js', import.meta.url), 'utf8')
  assert.match(source, /has-tabs-before/, 'reader-readium.js never toggles has-tabs-before')
  assert.match(source, /has-tabs-after/, 'reader-readium.js never toggles has-tabs-after')
  assert.match(source, /\.rs-tabs/, 'reader-readium.js never selects the .rs-tabs element to observe it')
})

test('railOverflow correctly flags the measured 390px overflow (352px viewport vs 452px content)', () => {
  // This is the exact overflow reported at 390px: the "Read aloud" tab sits
  // entirely past the visible edge with no scroll affordance shown.
  const overflow = railOverflow(0, 352, 452)
  assert.equal(overflow.before, false)
  assert.equal(overflow.after, true)
})
