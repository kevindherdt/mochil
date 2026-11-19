import assert from 'node:assert/strict'
import test from 'node:test'
import { placeOf, readPlace, rememberedIndex, savePlace, sentenceKey, startIndex } from '../static/narration-resume.js'

const chapter = ['It was dark.', 'A door opened.', 'He stepped in.', 'A door opened.'].map(text => ({ text }))

test('a place is the chapter and the opening words of the sentence, without the anchor', () => {
  assert.deepEqual(placeOf('ch2.xhtml#p5', { text: '  A   door opened. ' }, 1), { href: 'ch2.xhtml', key: 'A door opened.', index: 1 })
  assert.equal(sentenceKey('x'.repeat(200)).length, 60)
})

test('the saved sentence is found again in its chapter, the nearest one when the words repeat', () => {
  const place = placeOf('ch2.xhtml', chapter[3], 3)
  assert.equal(rememberedIndex(chapter, place, 'ch2.xhtml'), 3)
  assert.equal(rememberedIndex(chapter, { ...place, index: 0 }, 'ch2.xhtml#x'), 1)
})

test('a place in another chapter, or whose words are gone, is not found', () => {
  const place = placeOf('ch2.xhtml', chapter[2], 2)
  assert.equal(rememberedIndex(chapter, place, 'ch3.xhtml'), -1)
  assert.equal(rememberedIndex(chapter.slice(0, 2), place, 'ch2.xhtml'), -1)
  assert.equal(rememberedIndex(chapter, null, 'ch2.xhtml'), -1)
})

test('what the person selected wins, then where narration stopped if it is on the page, then the first on the page', () => {
  const visibleAt = i => i < 3
  assert.equal(startIndex({ selected: 2, remembered: 1, visibleAt, first: 0 }), 2)
  assert.equal(startIndex({ selected: -1, remembered: 1, visibleAt, first: 0 }), 1)
  assert.equal(startIndex({ selected: -1, remembered: 3, visibleAt, first: 0 }), 0, 'stopped somewhere they are no longer looking')
  assert.equal(startIndex({ selected: -1, remembered: -1, visibleAt, first: 2 }), 2)
  assert.equal(startIndex({ selected: 0, remembered: -1, visibleAt, first: 2 }), 0, 'the first sentence can be chosen too')
})

test('a place is kept in storage and a damaged one is ignored', () => {
  const data = new Map()
  const storage = { getItem: k => data.get(k) ?? null, setItem: (k, v) => data.set(k, v) }
  savePlace(storage, 'k', { href: 'a.xhtml', key: 'Hi there.', index: 4 })
  assert.deepEqual(readPlace(storage, 'k'), { href: 'a.xhtml', key: 'Hi there.', index: 4 })
  data.set('k', '{oops')
  assert.equal(readPlace(storage, 'k'), null)
  data.set('k', '{"href":3}')
  assert.equal(readPlace(storage, 'k'), null)
  assert.equal(readPlace({ getItem() { throw new Error('blocked') } }, 'k'), null)
})
