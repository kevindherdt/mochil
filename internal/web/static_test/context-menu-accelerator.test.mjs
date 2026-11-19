import test from 'node:test'
import assert from 'node:assert/strict'
import { installContextMenuAccelerator } from '../static/context-menu-accelerator.js'

function fixture(fine = true) {
  const summary = { focused: false, focus(options) { this.focused = options } }
  const menu = { open: false, querySelector: () => summary }
  const other = { open: true }
  const item = { querySelector: () => menu }
  const listeners = new Map()
  const container = {
    addEventListener: (type, listener) => listeners.set(type, listener),
    removeEventListener: (type) => listeners.delete(type),
    contains: value => value === item,
    querySelectorAll: () => [other],
  }
  const target = { closest: selector => selector === '.audio-item' ? item : null }
  installContextMenuAccelerator({ container, matchFinePointer: () => fine })
  return { listeners, menu, other, summary, target }
}

test('right click opens the existing accessible item menu on fine pointers', () => {
  const state = fixture()
  let prevented = false
  state.listeners.get('contextmenu')({ target: state.target, preventDefault: () => { prevented = true } })
  assert.equal(prevented, true)
  assert.equal(state.menu.open, true)
  assert.equal(state.other.open, false)
  assert.deepEqual(state.summary.focused, { preventScroll: true })
})

test('touch-style pointers retain their ordinary long-press behavior', () => {
  const state = fixture(false)
  let prevented = false
  state.listeners.get('contextmenu')({ target: state.target, preventDefault: () => { prevented = true } })
  assert.equal(prevented, false)
  assert.equal(state.menu.open, false)
})
