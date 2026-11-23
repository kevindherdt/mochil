import assert from 'node:assert/strict'
import test from 'node:test'
import { dockShelves, isNotesShelf, libraryTarget, noteShelves, notesShelf } from '../static/shelf-targets.js'

const shelves = [
  { id: 'books', name: 'Books' },
  { id: 'wiki', name: 'wiki' },
  { id: 'movies', name: 'Movies' },
  { id: 'documents', name: 'documents' },
]

test('note shelves are recognised by name and by kind', () => {
  assert.equal(isNotesShelf('wiki'), true)
  assert.equal(isNotesShelf('documents'), true)
  assert.equal(isNotesShelf('books'), false)
  assert.equal(isNotesShelf(''), false)
})

test('Library never opens a notes shelf, even when notes were the last place you were', () => {
  assert.equal(libraryTarget(shelves, 'wiki'), 'books')
  assert.equal(libraryTarget(shelves, 'documents'), 'books')
})

test('Library keeps the shelf you were actually in', () => {
  assert.equal(libraryTarget(shelves, 'movies'), 'movies')
})

test('with nothing selected or nothing visible it still returns a shelf', () => {
  assert.equal(libraryTarget(shelves, ''), 'books')
  assert.equal(libraryTarget([], ''), 'books')
  assert.equal(libraryTarget([{ id: 'wiki', name: 'wiki' }], 'wiki'), 'wiki')
})

test('the Notes button knows which vaults exist', () => {
  assert.deepEqual(noteShelves(shelves), [['wiki', 'wiki'], ['documents', 'documents']])
  assert.equal(notesShelf(shelves), 'wiki')
  assert.equal(notesShelf([{ id: 'books', name: 'Books' }]), '')
})

test('the dock gets live getters, so a shelf added later is seen', () => {
  let visible = shelves
  let selected = 'wiki'
  const places = dockShelves(() => visible, () => selected)
  assert.equal(places.getLibrary(), 'books')
  selected = 'movies'
  assert.equal(places.getLibrary(), 'movies')
  visible = [{ id: 'books', name: 'Books' }]
  assert.deepEqual(places.notesShelves(), [])
})

// The Notes button used to look at the library's id only, so a vault whose id was "obsidian" (or "journal") was not
// a note shelf at all: the tap opened "", nothing happened, and the dock never lit up. The kind decides.
test('a vault is a note shelf whatever its id is called', () => {
  assert.equal(isNotesShelf({ id: 'obsidian', kind: 'vault' }), true)
  assert.equal(isNotesShelf({ id: 'journal', kind: 'documents' }), true)
  assert.equal(isNotesShelf({ id: 'books', kind: 'books' }), false)
  assert.deepEqual(noteShelves([{ id: 'obsidian', kind: 'vault', name: 'Obsidian' }, { id: 'books', kind: 'books' }]), [['obsidian', 'Obsidian']])
})
