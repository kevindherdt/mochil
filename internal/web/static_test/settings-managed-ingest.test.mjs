import test from 'node:test'
import assert from 'node:assert/strict'
import { managedIngestConfig } from '../static/settings-managed-ingest.js'

test('managed ingest stays disabled while both roots are blank', () => {
  assert.equal(managedIngestConfig(' ', '', '8'), null)
})

test('managed ingest trims roots and stores an exact byte limit', () => {
  assert.deepEqual(managedIngestConfig(' /incoming ', ' /managed ', '8'), {
    incoming_directory: '/incoming',
    managed_directory: '/managed',
    maximum_file_bytes: 8589934592,
  })
})
