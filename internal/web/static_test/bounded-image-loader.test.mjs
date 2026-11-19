import test from 'node:test'
import assert from 'node:assert/strict'
import { createBoundedImageLoader } from '../static/bounded-image-loader.js'

function image(source) {
  const listeners = {}
  const item = {
    isConnected: true,
    complete: false,
    dataset: { src: source },
    classes: new Set(),
    classList: { add(name) { this.owner.classes.add(name) }, remove(name) { this.owner.classes.delete(name) } },
    addEventListener(type, listener) { listeners[type] = listener },
    set src(value) { this.currentSrc = value },
    finish(type = 'load') { listeners[type]?.({ type }) }
  }
  item.classList.owner = item
  return item
}

test('bounds concurrent image requests and continues as they settle', () => {
  const loader = createBoundedImageLoader({ concurrency: 2 })
  const images = [image('one'), image('two'), image('three')]
  images.forEach(loader.enqueue)

  assert.deepEqual(images.map(item => item.currentSrc), ['one', 'two', undefined])
  images[0].finish()
  assert.equal(images[2].currentSrc, 'three')
})

test('released queued images are not requested', () => {
  const loader = createBoundedImageLoader({ concurrency: 1 })
  const first = image('one')
  const second = image('two')
  loader.enqueue(first)
  loader.enqueue(second)
  loader.release({ querySelectorAll: () => [second] })

  first.finish()
  assert.equal(second.currentSrc, undefined)
})

test('covers are marked pending while loading and failed on error', () => {
  const loader = createBoundedImageLoader({ concurrency: 2 })
  const [good, bad] = [image('good'), image('bad')]
  loader.enqueue(good)
  loader.enqueue(bad)
  assert.ok(good.classes.has('cover-pending'))
  good.finish()
  bad.finish('error')
  assert.equal(good.classes.has('cover-pending'), false)
  assert.equal(good.classes.has('cover-failed'), false)
  assert.ok(bad.classes.has('cover-failed'))
})

import { sizedCoverSource } from "../static/bounded-image-loader.js";

test("cover requests ask for a size that fits the card", () => {
  globalThis.devicePixelRatio = 2;
  const card = (clientWidth) => ({ clientWidth, parentElement: null });
  assert.equal(sizedCoverSource(card(70), "/covers/books/a?v=1"), "/covers/books/a?v=1&w=160");
  assert.equal(sizedCoverSource(card(110), "/covers/books/a?v=1"), "/covers/books/a?v=1&w=240");
  assert.equal(sizedCoverSource(card(170), "/covers/books/a?v=1"), "/covers/books/a?v=1&w=360");
  assert.equal(sizedCoverSource(card(300), "/covers/books/a?v=1"), "/covers/books/a?v=1", "large slots use the full cover");
  assert.equal(sizedCoverSource(card(0), "/covers/books/a"), "/covers/books/a", "unlaid-out images use the full cover");
  assert.equal(sizedCoverSource(card(50), "/api/other.png"), "/api/other.png");
  assert.equal(sizedCoverSource(card(50), "/covers/x?w=240"), "/covers/x?w=240");
});
