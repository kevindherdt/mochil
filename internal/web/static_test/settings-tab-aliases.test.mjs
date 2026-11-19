import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { settingsTabFor } from '../static/settings-tab-aliases.js'

const index = await readFile(new URL('../static/index.html', import.meta.url), 'utf8')

test('retired tab names resolve to the tab that now holds their content', () => {
  assert.equal(settingsTabFor('metadata'), 'libraries')
  assert.equal(settingsTabFor('editor'), 'modules')
  assert.equal(settingsTabFor('reading'), 'libraries')
})

test('current tab names, unknown values and empties pass through untouched', () => {
  for (const name of ['libraries', 'modules', 'playback', 'sync', 'narration', 'appearance', 'accounts', 'diagnostics', 'hermes', 'avatar']) assert.equal(settingsTabFor(name), name)
  assert.equal(settingsTabFor(''), '')
  assert.equal(settingsTabFor(null), null)
  assert.equal(settingsTabFor('constructor'), 'constructor')
})

test('Settings no longer has Metadata, Editor or Reading tabs, and the duplicated reading defaults are gone', () => {
  for (const name of ['metadata', 'editor', 'reading']) assert.doesNotMatch(index, new RegExp(`data-settings-tab="${name}"`))
  assert.doesNotMatch(index, /id="reading-default-/, 'the reader panel writes the same stored defaults')
  assert.doesNotMatch(index, /data-settings-panel="reading"/)
  assert.match(index, /data-settings-tab="playback"/, 'Playback keeps its own tab')
})

test('the editor section is static markup that moves into Modules, and the metadata review mounts into Libraries', async () => {
  assert.match(index, /data-settings-panel="editor"/)
  const settings = await readFile(new URL('../static/settings.js', import.meta.url), 'utf8')
  assert.match(settings, /settingsTabFor\(remembered\)/)
  assert.doesNotMatch(settings, /createReadingDefaults|readingDefaults/)
  assert.match(settings, /#settings-view > \[data-settings-panel\]/, 'tab switching only toggles top-level panels')
})
