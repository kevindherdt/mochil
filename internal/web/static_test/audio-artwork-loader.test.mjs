import assert from 'node:assert/strict'
import test from 'node:test'
import { createAudioArtworkLoader } from '../static/audio-artwork-loader.js'

test('audio artwork waits for the viewport and preserves the container fallback', async () => {
  let observer
  class Observer {
    constructor(callback) { this.callback = callback; this.observed = []; this.released = [] ; observer = this }
    observe(element) { this.observed.push(element) }
    unobserve(element) { this.released.push(element) }
  }
  const image = {
    dataset: {},
    isConnected: true,
    complete: true,
    addEventListener() {},
  }
  let child = null
  const element = {
    ownerDocument: { createElement: () => image },
    querySelector: selector => selector === 'img' || selector === 'img[data-src]' ? child : null,
    append: value => { child = value },
  }
  const loader = createAudioArtworkLoader({ IntersectionObserverClass: Observer })
  loader.load(element, '/cover.jpg')
  assert.deepEqual(observer.observed, [element])
  assert.equal(image.src, undefined)
  observer.callback([{ isIntersecting: true, target: element }])
  await Promise.resolve()
  assert.equal(image.src, '/cover.jpg')
  assert.deepEqual(observer.released, [element])
})

test('focused artwork bypasses viewport observation with high fetch priority', async () => {
  let observed = false
  class Observer {
    constructor() {}
    observe() { observed = true }
    unobserve() {}
  }
  const image = { dataset: {}, isConnected: true, complete: true, addEventListener() {} }
  let child = null
  const element = {
    ownerDocument: { createElement: () => image },
    querySelector: selector => selector === 'img' || selector === 'img[data-src]' ? child : null,
    append: value => { child = value },
  }

  createAudioArtworkLoader({ IntersectionObserverClass: Observer }).load(element, '/focused.jpg', true)
  await Promise.resolve()

  assert.equal(observed, false)
  assert.equal(image.fetchPriority, 'high')
  assert.equal(image.src, '/focused.jpg')
})

test('a tile whose artwork fails is marked so it shows a placeholder, unless it has text of its own', () => {
  const build = text => {
    const listeners = {}
    const image = { dataset: {}, classList: { add() {} }, remove() { child = null }, addEventListener: (type, fn) => { listeners[type] = fn } }
    let child = null
    const element = {
      dataset: {}, textContent: text,
      get children() { return child ? [child] : [] },
      ownerDocument: { createElement: () => image },
      querySelector: () => child, append: value => { child = value },
    }
    createAudioArtworkLoader({ IntersectionObserverClass: null }).load(element, `/missing-${text || 'blank'}.jpg`)
    listeners.error()
    return element
  }
  assert.equal('artFailed' in build('').dataset, true)
  assert.equal('artFailed' in build('AF').dataset, false)
})

test('a cover the server said does not exist is not asked for again after a redraw', () => {
  const listeners = {}
  const makeElement = () => {
    const children = []
    const image = { dataset: {}, classList: { add() {} }, addEventListener: (name, fn) => { listeners[name] = fn }, remove() { children.length = 0 } }
    return {
      dataset: {}, children, textContent: '',
      ownerDocument: { createElement: () => image },
      querySelector: () => children[0] || null,
      append: value => { children.push(value) },
    }
  }
  const loader = createAudioArtworkLoader({ IntersectionObserverClass: null })
  const first = makeElement()
  loader.load(first, '/api/audio-artwork/music/x?size=card&v=1')
  listeners.error()
  assert.equal('artFailed' in first.dataset, true)
  const again = makeElement()
  loader.load(again, '/api/audio-artwork/music/x?size=card&v=1')
  assert.equal(again.children.length, 0, 'no second image request')
  assert.equal('artFailed' in again.dataset, true, 'shows the placeholder straight away')
})
