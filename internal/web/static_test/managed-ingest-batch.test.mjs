import test from 'node:test'
import assert from 'node:assert/strict'
import { managedIngestBatchEntries, managedIngestBatchSummary } from '../static/managed-ingest-ui.js'

test('batch ingest request keeps only bounded paths and omits display metadata', () => {
  const values = Array.from({ length: 105 }, (_, index) => ({ source_path: ` source-${index}.epub `, target_path: ` target-${index}.epub `, title: 'Display only' }))
  const entries = managedIngestBatchEntries(values)
  assert.equal(entries.length, 100)
  assert.deepEqual(entries[0], { source_path: 'source-0.epub', target_path: 'target-0.epub' })
  assert.equal('title' in entries[0], false)
})

test('batch review summary distinguishes clear and conflicted previews', () => {
  assert.equal(managedIngestBatchSummary({ ready: true, entries: [{}, {}], conflicts: 0 }), '2 destinations are clear. No plan or file has been created.')
  assert.equal(managedIngestBatchSummary({ ready: false, entries: [{}], conflicts: 1 }), '1 file needs attention.')
  assert.equal(managedIngestBatchSummary({ ready: false, entries: [{}, {}], conflicts: 2 }), '2 files need attention.')
})
