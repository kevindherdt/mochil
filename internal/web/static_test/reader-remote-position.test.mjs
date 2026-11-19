import assert from 'node:assert/strict'
import test from 'node:test'
import {
  fixedLayoutPage,
  positionFromReaderLocation,
  readingFractionsEquivalent,
  selectInitialReadingPosition,
} from '../static/reader-remote-position.js'

test('newer clean server progress wins even without a web CFI', () => {
  const local = { fraction: 0.2, cfi: 'epubcfi(old)', pending: false, updatedAt: 10 }
  const remote = { fraction: 0.6, cfi: '', revision: 3, updated_at: 20_000 }
  assert.deepEqual(selectInitialReadingPosition(local, remote), { position: remote, source: 'remote' })
})

test('newer unsent local progress remains authoritative while offline', () => {
  const local = { fraction: 0.7, pending: true, updatedAt: 30 }
  const remote = { fraction: 0.6, revision: 3, updated_at: 20_000 }
  assert.deepEqual(selectInitialReadingPosition(local, remote), { position: local, source: 'local' })
})

test('a copy kept after the server refused it does not beat the newer server position', () => {
  const local = { fraction: 0.7, pending: true, conflicted: true, updatedAt: 30 }
  const remote = { fraction: 0.26, revision: 9, updated_at: 20_000 }
  assert.deepEqual(selectInitialReadingPosition(local, remote), { position: remote, source: 'remote' })
})

test('equivalent server progress acknowledges without replacing the exact local page', () => {
  const local = { fraction: 0.5, cfi: 'epubcfi(exact-local-page)', pending: true, updatedAt: 30 }
  const remote = { fraction: 0.5015, cfi: 'epubcfi(adjacent-remote-page)', revision: 4, updated_at: 20_000 }
  assert.deepEqual(selectInitialReadingPosition(local, remote), { position: local, source: 'equivalent' })
})

test('comic pages are exact and retain the legacy PDF fallback', () => {
  const comic = { locators: { comic: JSON.stringify({ page: 8, page_count: 20 }) } }
  const legacy = { locators: { pdf: JSON.stringify({ page: 7, page_count: 20 }) } }
  assert.equal(fixedLayoutPage(comic, 'comic', 20), 8)
  assert.equal(fixedLayoutPage(legacy, 'comic', 20), 7)
  assert.equal(fixedLayoutPage(comic, 'comic', 5), null)
})

test('web comic locations publish a format-specific exact page locator', () => {
  const view = { resolveCFI: () => ({ index: 4 }) }
  assert.deepEqual(positionFromReaderLocation(view, { fraction: 0.4, cfi: 'epubcfi(fake)' }, 'comic', 10), {
    fraction: 0.4,
    cfi: 'epubcfi(fake)',
    locators: { comic: JSON.stringify({ version: 1, page: 4, page_count: 10 }) },
  })
  assert.equal(readingFractionsEquivalent(0.5, 0.501), true)
  assert.equal(readingFractionsEquivalent(0.5, 0.51), false)
})
