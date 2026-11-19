import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

const css = await readFile(new URL('../static/sticky-shell.css', import.meta.url), 'utf8')

test('podcast actions remain a right-aligned in-flow toolbar without overriding audiobook destinations', () => {
  assert.match(css, /audio-browse-head\[data-kind="podcasts"\] \.audio-toolbar\{[^}]*width:max-content[^}]*flex:0 0 auto/)
  assert.match(css, /audio-view-menu,[^{]*#podcast-manage-toggle\{position:relative;inset:auto\}/)
  assert.doesNotMatch(css, /audio-browse-head:not\(\[data-kind="music"\]\)/)
})

test('mobile Home and library selector stays left while shell actions remain reachable', () => {
  assert.match(css, /@media\(max-width:760px\)\{\.masthead\{[^}]*grid-template-areas:"destination actions"/)
  assert.match(css, /\.masthead-destination\{grid-area:destination;[^}]*justify-self:start/)
  assert.match(css, /\.masthead-actions\{grid-area:actions;justify-self:end/)
})
