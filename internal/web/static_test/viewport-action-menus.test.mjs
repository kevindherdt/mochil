import test from 'node:test'
import assert from 'node:assert/strict'
import { viewportMenuPosition } from '../static/viewport-action-menus.js'

const rect = (left, top, width, height) => ({ left, top, right:left+width, bottom:top+height, width, height })

test('item menu stays beside a low viewport trigger by opening upward', () => {
  const value = viewportMenuPosition({ getBoundingClientRect:()=>rect(900, 690, 44, 44) }, { getBoundingClientRect:()=>rect(0, 0, 230, 280) }, { width:1024, height:768 })
  assert.equal(value.top, 404)
  assert.equal(value.left, 714)
})

test('item menu clamps to the viewport at narrow desktop widths', () => {
  const value = viewportMenuPosition({ getBoundingClientRect:()=>rect(8, 100, 44, 44) }, { getBoundingClientRect:()=>rect(0, 0, 230, 180) }, { width:700, height:500 })
  assert.equal(value.left, 10)
  assert.equal(value.top, 150)
})
