import assert from 'node:assert/strict'
import test from 'node:test'
import { primaryNavigationVisible } from '../../../scripts/browser-layout-contract.mjs'

const rect = (left, width = 44, painted = true) => ({ left, right: left + width, width, painted })

test('compact primary navigation keeps Home left and Settings reachable', () => {
  assert.equal(primaryNavigationVisible({
    viewportWidth: 320,
    switcher: rect(8, 112),
    search: rect(180),
    activity: rect(224),
    settings: rect(268),
  }), true)
})

test('compact primary navigation rejects displaced or missing controls', () => {
  const base = {
    viewportWidth: 320,
    switcher: rect(8, 112),
    search: rect(180),
    activity: rect(224),
    settings: rect(268),
  }
  assert.equal(primaryNavigationVisible({ ...base, switcher: rect(190, 112) }), false)
  assert.equal(primaryNavigationVisible({ ...base, settings: rect(300) }), false)
  assert.equal(primaryNavigationVisible({ ...base, activity: null }), true, 'the activity button only exists while the server is busy')
  assert.equal(primaryNavigationVisible({ ...base, activity: rect(100) }), false, 'a shown activity button must sit after search')
  assert.equal(primaryNavigationVisible({ ...base, settings: null }), false)
  assert.equal(primaryNavigationVisible({ ...base, search: rect(180, 44, false) }), false)
})

test('desktop navigation is not constrained by the compact ordering contract', () => {
  assert.equal(primaryNavigationVisible({ viewportWidth: 1440 }), true)
})
