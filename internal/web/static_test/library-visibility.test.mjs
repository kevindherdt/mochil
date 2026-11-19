import assert from 'node:assert/strict'
import test from 'node:test'
import { createLibraryVisibility, parseHiddenLibraries } from '../static/library-visibility.js'

test('hidden library settings tolerate damaged browser data', () => {
  assert.deepEqual(parseHiddenLibraries('{broken'), [])
  assert.deepEqual(parseHiddenLibraries(JSON.stringify({ library: 'books' })), [])
})

test('hidden library settings retain only bounded unique identifiers', () => {
  const oversized = 'x'.repeat(257)
  assert.deepEqual(parseHiddenLibraries(JSON.stringify(['books', 'books', '', 4, oversized, 'music'])), ['books', 'music'])
  assert.equal(parseHiddenLibraries(JSON.stringify(Array.from({ length: 600 }, (_, index) => `shelf-${index}`))).length, 512)
})

test('activating a newly saved library cannot leave it hidden', () => {
  const values = new Map([['mochil-hidden-libraries-v1', JSON.stringify(['music'])]])
  const storage = {
    getItem: key => values.get(key) ?? null,
    setItem: (key, value) => values.set(key, value),
  }
  const changes = []
  const visibility = createLibraryVisibility({ storage, onChange: libraries => changes.push(libraries.map(item => item.id)) })
  visibility.setLibraries([{ id: 'books' }, { id: 'music' }])

  visibility.show(['music'])

  assert.equal(visibility.isVisible('music'), true)
  assert.deepEqual(changes, [['books', 'music']])
  assert.equal(values.get('mochil-hidden-libraries-v1'), '[]')
})

test('activating unknown or already visible libraries is a no-op', () => {
  let changes = 0
  const storage = { getItem: () => null, setItem: () => {} }
  const visibility = createLibraryVisibility({ storage, onChange: () => { changes += 1 } })
  visibility.setLibraries([{ id: 'books' }])

  visibility.show(['books', 'missing'])

  assert.equal(changes, 0)
})
