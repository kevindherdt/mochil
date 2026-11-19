import test from 'node:test'
import assert from 'node:assert/strict'
import { seriesActionLabel, seriesContinueTarget, seriesStats } from '../static/series-hub.js'

const book = (n, fraction, finished = false, updated = '') => ({ id: `b${n}`, series_index: n, read_url: `/read/${n}`, reading_progress: { fraction, finished, updated_at: updated } })

test('the series button continues what is in progress, else starts the first unfinished book', () => {
  const books = [book(1, 1, true), book(2, 0.4, false, '2026-01-02'), book(3, 0.1, false, '2026-01-05'), book(4, 0)]
  const target = seriesContinueTarget(books)
  assert.equal(target.book.id, 'b3')
  assert.equal(seriesActionLabel(target), 'Continue Vol. 3')
  const fresh = seriesContinueTarget([book(1, 1, true), book(2, 0)])
  assert.deepEqual([fresh.book.id, seriesActionLabel(fresh)], ['b2', 'Start Vol. 2'])
  assert.equal(seriesActionLabel(seriesContinueTarget([book(1, 1, true)])), 'Read again')
  assert.equal(seriesContinueTarget([]), null)
})

test('series stats count finished books against the whole series', () => {
  assert.deepEqual(seriesStats([book(1, 1, true), book(2, 0)], 4), { read: 1, total: 4, percent: 25 })
})
