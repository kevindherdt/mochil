import assert from 'node:assert/strict'
import test from 'node:test'
import { createListReturn } from '../static/catalogue-return.js'

test('going back reloads pages until the same number are shown, then scrolls to where the person was', async () => {
  let shown = 0, y = 2400, scrolled = null
  const back = createListReturn({
    count: () => shown, scrollY: () => y, scrollTo: (to) => { scrolled = to },
    load: async () => { shown = 80 }, loadMore: async () => { shown += 80; return true },
  })
  shown = 240
  back.remember()
  await back.restore()
  assert.equal(shown, 240)
  assert.equal(scrolled, 2400)
})

test('a list that cannot grow further stops paging and nothing is remembered twice', async () => {
  let shown = 100, calls = 0, scrolled = null
  const back = createListReturn({ count: () => shown, scrollY: () => 500, scrollTo: (to) => { scrolled = to }, load: async () => { shown = 80 }, loadMore: async () => { calls++; return false } })
  back.remember()
  await back.restore()
  assert.equal(calls, 1)
  await back.restore()
  assert.equal(scrolled, 500)
})
