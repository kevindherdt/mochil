import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'
import { isLibraryDestination, libraryHue } from '../static/library-identity.js'

const here = dirname(fileURLToPath(import.meta.url))
const read = (name) => readFile(join(here, '..', 'static', name), 'utf8')

test('the same library always gets the same hue', () => {
  assert.equal(libraryHue('books'), libraryHue('books'))
  assert.equal(libraryHue('manga'), libraryHue('manga'))
})

test('two libraries of the same kind do not look the same', () => {
  // The complaint this exists for: books, Manga, lightnovels, test and notes all rendered in the
  // same amber because colour came from the kind rather than from the library.
  const ids = ['books', 'manga', 'lightnovels', 'test', 'notes', 'audiobooks', 'movies', 'music']
  const hues = ids.map(libraryHue)
  const distinct = new Set(hues).size
  assert.ok(distinct >= 6, `only ${distinct} distinct hues across ${ids.length} libraries: ${hues}`)
})

test('neighbouring ids do not land on neighbouring hues', () => {
  const pairs = ['a', 'b', 'c', 'd', 'e', 'f']
  assert.ok(new Set(pairs.map(libraryHue)).size >= 5)
})

test('every hue is a real number the stylesheet can use', () => {
  for (const id of ['', 'x', 'a-very-long-library-identifier-1234', 'ünïcode']) {
    const hue = libraryHue(id)
    assert.equal(typeof hue, 'number')
    assert.ok(hue >= 0 && hue < 360, `${id} gave ${hue}`)
  }
})

test('destinations that are not libraries stay neutral', () => {
  // The switcher passes these as sentinels, not ids; Home wearing a library hue made it look like a
  // shelf in the list.
  for (const value of ['__home__', '__settings__']) {
    assert.equal(isLibraryDestination(value), false, value)
  }
  for (const value of ['home', 'settings', 'editor', 'hermes', '']) {
    assert.equal(isLibraryDestination(value), false, value)
  }
  for (const value of ['books', 'zines', 'manga']) {
    assert.equal(isLibraryDestination(value), true, value)
  }
})

test('the switcher reads the hue off the library, and the stylesheet uses it', async () => {
  const menu = await read('library-menu.js')
  assert.match(menu, /libraryHue\(option\.value\)/, 'the item hue comes from the library id')
  assert.match(menu, /--kind-h/, 'the item carries the hue for the stylesheet')

  const css = await read('kind-colors.css')
  assert.match(css, /\[data-library\][^{]*\.kind-icon\{stroke:hsl\(var\(--kind-h\)/, 'the icon paints the hue')
  // A 20px stroke needs a deeper tone on a light surface than on a dark one.
  assert.match(css, /\[data-theme="light"\][^{]*\[data-library\][^{]*\.kind-icon\{stroke:hsl\(var\(--kind-h\) 64% 38%\)/)
})
