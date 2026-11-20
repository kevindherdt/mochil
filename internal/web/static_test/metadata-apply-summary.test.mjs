import assert from 'node:assert/strict'
import test from 'node:test'
import { metadataApplyImpact, metadataApplyReceipt } from '../static/metadata-apply-summary.js'

test('comic override receipt states that ComicInfo was not rewritten', () => {
  const receipt = metadataApplyReceipt(
    { apply_mode: 'catalogue-overrides', applied_books: 1, source_books: 0, override_books: 1, comic_info_changed: false },
    { source_type: 'filesystem', changes: [{ format: 'cbz', mode: 'catalogue override' }] },
  )
  assert.match(receipt, /1 catalogue override saved/)
  assert.match(receipt, /ComicInfo\.xml was not rewritten/)
})

test('Calibre receipt distinguishes its database from publication archives', () => {
  const receipt = metadataApplyReceipt(
    { apply_mode: 'calibre-database', applied_books: 2, source_books: 2, override_books: 0, publication_archives_changed: false },
    { source_type: 'calibre', changes: [] },
  )
  assert.match(receipt, /2 Calibre records updated/)
  assert.match(receipt, /Calibre database updated; publication archives unchanged/)
})

test('receipt reports the verified catalogue refresh without implying another write', () => {
  const receipt = metadataApplyReceipt(
    { apply_mode: 'filesystem', applied_books: 1, source_books: 1, override_books: 0, scan_books: 324 },
    { source_type: 'filesystem', changes: [{ format: 'epub', mode: 'source file' }] },
  )
  assert.match(receipt, /324 catalogue items verified after refresh/)
})

test('receipt surfaces a committed apply whose catalogue refresh needs attention', () => {
  const receipt = metadataApplyReceipt(
    { apply_mode: 'filesystem', applied_books: 1, source_books: 1, override_books: 0, scan_warning: 'temporary read failure' },
    { source_type: 'filesystem', changes: [{ format: 'epub', mode: 'source file' }] },
  )
  assert.match(receipt, /Catalogue refresh needs attention: temporary read failure/)
})

test('ordinary overrides explicitly preserve source publications', () => {
  assert.match(metadataApplyReceipt(
    { applied_books: 1, source_books: 0, override_books: 1 },
    { source_type: 'filesystem', changes: [{ format: 'pdf', mode: 'catalogue override' }] },
  ), /Original publication files unchanged/)
})

test('comic preview explains archive impact before confirmation', () => {
  assert.match(metadataApplyImpact({
    source_type: 'filesystem',
    changes: [{ format: 'cbz', mode: 'catalogue override' }],
  }), /ComicInfo\.xml remain unchanged/)
})
