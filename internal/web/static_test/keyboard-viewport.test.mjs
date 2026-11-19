import test from 'node:test'
import assert from 'node:assert/strict'
import { followKeyboard, keyboardHeight } from '../static/keyboard-viewport.js'

test('the keyboard is what covers the layout viewport, ignoring small browser-bar changes', () => {
  assert.equal(keyboardHeight({ height: 500, offsetTop: 0 }, 900), 400)
  assert.equal(keyboardHeight({ height: 860, offsetTop: 0 }, 900), 0, 'a collapsing address bar is not a keyboard')
  assert.equal(keyboardHeight(undefined, 900), 0)
})

test('the visible height is published while the keyboard is up and removed when it closes', () => {
  const listeners = {}
  const viewport = { height: 520, offsetTop: 0, addEventListener: (name, fn) => { listeners[name] = fn }, removeEventListener() {} }
  const props = new Map()
  const win = { innerHeight: 900, visualViewport: viewport, scrolled: 0, scrollTo() { this.scrolled++ }, document: { documentElement: { style: { setProperty: (k, v) => props.set(k, v), removeProperty: k => props.delete(k) } } } }
  followKeyboard(win)
  assert.equal(props.get('--visible-height'), '520px')
  assert.equal(win.scrolled > 0, true, 'the page is put back at the top so the top bar is not hidden')
  viewport.height = 900
  listeners.resize()
  assert.equal(props.has('--visible-height'), false)
})
