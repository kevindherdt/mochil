import test from 'node:test'
import assert from 'node:assert/strict'
import { captureCataloguePlace, restoreCataloguePlace } from '../static/catalogue-place.js'

function bookCard(id, top, bottom) {
  return {
    dataset: { book: id, library: 'lib' },
    getBoundingClientRect: () => ({ top, bottom }),
  }
}

function seriesCard(index, top, bottom) {
  return {
    dataset: { index: String(index) },
    getBoundingClientRect: () => ({ top, bottom }),
  }
}

function fakeDocument(lists) {
  return {
    querySelector(selector) {
      const id = selector.replace(/^#/, '')
      const cards = lists[id]
      return cards ? { id, hidden: false, querySelectorAll: () => cards } : null
    },
  }
}

function fakeWindow(scrollY = 0) {
  const scrolls = []
  return {
    scrollY,
    scrolls,
    scrollTo(x, y) {
      scrolls.push([x, y])
      this.scrollY = y
    },
  }
}

test('capture forgets the cards hidden under the sticky bar and names the first visible one', () => {
  const document = fakeDocument({ books: [bookCard('a', 20, 80), bookCard('b', 120, 560)] })
  assert.deepEqual(captureCataloguePlace(document, 106), { container: 'books', key: 'book:lib/b', top: 120 })
  assert.equal(captureCataloguePlace(fakeDocument({ books: [] }), 106), null)
  assert.equal(captureCataloguePlace(fakeDocument({}), 106), null, 'no visible catalogue, nothing to carry')
})

test('capture names series and author rows by their index so both layouts match', () => {
  const document = fakeDocument({ 'series-index': [seriesCard(7, 200, 700)] })
  assert.deepEqual(captureCataloguePlace(document, 106), { container: 'series-index', key: 'index:7', top: 200 })
})

test('restore puts the same card back at the same height in the new layout', () => {
  const place = { container: 'books', key: 'book:lib/b', top: 120 }
  const document = fakeDocument({ books: [bookCard('a', 300, 360), bookCard('b', 900, 1340)] })
  const window = fakeWindow(2600)
  restoreCataloguePlace(place, { document, window, requestFrame: callback => callback() })
  assert.deepEqual(window.scrolls, [[0, 3380]])
})

test('restore waits for the reloaded shelf instead of scrolling to a guess', () => {
  const place = { container: 'books', key: 'book:lib/b', top: 120 }
  const window = fakeWindow(0)
  const cards = []
  const container = { id: 'books', hidden: false, querySelectorAll: () => cards }
  let frames = 0
  restoreCataloguePlace(place, {
    document: {
      querySelector() {
        frames += 1
        if (frames === 3) cards.push(bookCard('b', 300, 700))
        return container
      },
    },
    window,
    requestFrame: callback => callback(),
    frames: 30,
  })
  assert.equal(frames, 3, 'the empty shelf was waited out, not measured')
  assert.deepEqual(window.scrolls, [[0, 180]])

  const quiet = fakeWindow(0)
  let tries = 0
  restoreCataloguePlace(place, {
    document: { querySelector: () => (tries += 1, { id: 'books', hidden: false, querySelectorAll: () => [] }) },
    window: quiet,
    requestFrame: callback => callback(),
    frames: 5,
  })
  assert.equal(tries, 5, 'the wait is bounded')
  assert.deepEqual(quiet.scrolls, [], 'a card that never returns scrolls nothing')
})
