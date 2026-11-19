import test from 'node:test'
import assert from 'node:assert/strict'
import { createBoundedWindow } from '../static/bounded-window.js'

function spacer() { return { style: { height: '0px' } } }
function container(count = 0) {
  const root = { children: [], scrollHeight: count * 10 }
  for (let index = 0; index < count; index++) root.children.push(node(root, index))
  Object.defineProperty(root, 'firstElementChild', { get: () => root.children[0] })
  return root
}
function node(root, index) {
  return {
    isConnected: true,
    getBoundingClientRect() { return { top: root.children.indexOf(this) * 10 } },
    remove() { const at = root.children.indexOf(this); if (at >= 0) { root.children.splice(at, 1); root.scrollHeight -= 10 } }
  }
}

test('keeps a bounded number of nodes and records removed height', () => {
  const root = container(8), top = spacer(), bottom = spacer()
  const windowed = createBoundedWindow({ container: root, topSpacer: top, bottomSpacer: bottom, maximum: 6, trim: 2 })
  const state = windowed.begin('append')
  root.children.push(node(root, 8), node(root, 9)); root.scrollHeight += 20
  const removed = windowed.finish(state)
  assert.deepEqual(removed, { start: 4, end: 0 })
  assert.equal(root.children.length, 6)
  assert.equal(top.style.height, '40px')
})

test('prepending consumes the top spacer and trims the far end', () => {
  globalThis.window = { scrollBy() { throw new Error('spacer should preserve the anchor') } }
  const root = container(6), top = spacer(), bottom = spacer()
  top.style.height = '20px'
  const windowed = createBoundedWindow({ container: root, topSpacer: top, bottomSpacer: bottom, maximum: 6, trim: 2 })
  const state = windowed.begin('prepend')
  root.children.unshift(node(root, -2), node(root, -1)); root.scrollHeight += 20
  const removed = windowed.finish(state)
  assert.deepEqual(removed, { start: 0, end: 2 })
  assert.equal(root.children.length, 6)
  assert.equal(top.style.height, '0px')
  assert.equal(bottom.style.height, '20px')
})

test('reset clears both virtual spacers', () => {
  const top = spacer(), bottom = spacer()
  top.style.height = '120px'; bottom.style.height = '80px'
  createBoundedWindow({ container: container(), topSpacer: top, bottomSpacer: bottom }).reset()
  assert.equal(top.style.height, '0px')
  assert.equal(bottom.style.height, '0px')
})
