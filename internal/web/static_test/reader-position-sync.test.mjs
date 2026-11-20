import assert from 'node:assert/strict'
import test from 'node:test'
import { createReaderPositionSync, readerPositionMoved } from '../static/reader-position-sync.js'

function response(status, body) {
  return { status, ok: status >= 200 && status < 300, json: async () => body }
}

function fixture(fetchImpl) {
  const timers = []
  const saved = []
  const synced = []
  const statuses = []
  const conflicts = []
  const sync = createReaderPositionSync({
    endpoint: '/api/reading/books/one',
    deviceID: 'web-one',
    deviceName: 'Browser',
    fetchImpl,
    positionStore: {
      save: (value, extra) => saved.push({ ...value, ...extra }),
      markSynced: value => synced.push({ ...value }),
      markConflicted: () => conflicts.push(true),
    },
    renderStatus: value => statuses.push(value),
    setTimer: callback => { timers.push(callback); return timers.length },
    clearTimer() {},
    setRepeating: () => 1,
    clearRepeating() {},
    addOnlineListener() {},
    removeOnlineListener() {},
  })
  return { sync, timers, saved, synced, statuses, conflicts }
}

test('scheduled saves coalesce to the newest exact position and anchor', async () => {
  const requests = []
  const { sync, timers, synced, statuses } = fixture(async (_url, options) => {
    const body = JSON.parse(options.body)
    requests.push(body)
    return response(200, { ...body, revision: 4 })
  })
  sync.revision = 3
  sync.latestAnchor = { exact: 'matching words long enough for an anchor' }
  sync.schedule({ cfi: 'first', fraction: 0.1 })
  const locators = { comic: '{"page":2,"page_count":10}' }
  sync.schedule({ cfi: 'second', fraction: 0.2, locators })
  await timers.at(-1)()

  assert.equal(requests.length, 1)
  assert.equal(requests[0].cfi, 'second')
  assert.equal(requests[0].base_revision, 3)
  assert.equal(requests[0].web_anchor.exact, 'matching words long enough for an anchor')
  assert.deepEqual(requests[0].locators, locators)
  assert.equal(sync.revision, 4)
  assert.deepEqual(synced, [{ cfi: 'second', fraction: 0.2, locators }])
  assert.equal(statuses.at(-1), 'Saved')
})

test('an equivalent authoritative position acknowledges a stale same-page save', async () => {
  let calls = 0
  const { sync, synced, statuses } = fixture(async () => {
    calls++
    return response(200, { cfi: 'newer-device-locator', fraction: 0.5015, revision: 8, device_id: 'tablet' })
  })
  sync.revision = 3

  await sync.save({ cfi: 'local-renderer-locator', fraction: 0.5 })
  await sync.save()

  assert.equal(calls, 1)
  assert.equal(sync.revision, 8)
  assert.deepEqual(synced, [{ cfi: 'local-renderer-locator', fraction: 0.5 }])
  assert.equal(statuses.at(-1), 'Saved')
})

test('a semantic same-location acknowledgement does not retry after reflow changes its fraction', async () => {
  let calls = 0
  const anchor = { spine_index: 3, prefix: 'words before', exact: 'the same sentence', suffix: 'words after' }
  const { sync, synced, statuses } = fixture(async () => {
    calls++
    return response(200, { cfi: 'remote-layout', fraction: 0.75, web_anchor: anchor, revision: 8, device_id: 'tablet' })
  })
  sync.revision = 3
  sync.latestAnchor = anchor

  await sync.save({ cfi: 'local-layout', fraction: 0.2 })
  await sync.save()

  assert.equal(calls, 1)
  assert.deepEqual(synced, [{ cfi: 'local-layout', fraction: 0.2 }])
  assert.equal(statuses.at(-1), 'Saved')
})

test('a cross-device conflict blocks passive repeats but real movement reclaims ownership', async () => {
  const requests = []
  const { sync, timers } = fixture(async (_url, options) => {
    requests.push(JSON.parse(options.body))
    return requests.length === 1
      ? response(409, { revision: 9, device_id: 'other-device' })
      : response(200, { ...requests.at(-1), revision: 10 })
  })
  await sync.save({ cfi: 'one', fraction: 0.4 })
  await sync.save({ cfi: 'one', fraction: 0.4 })

  assert.equal(requests.length, 1)
  assert.equal(sync.blocked, true)
  assert.equal(sync.revision, 9)

  sync.schedule({ cfi: 'two', fraction: 0.5 })
  await timers.at(-1)()

  assert.equal(requests.length, 2)
  assert.equal(requests[1].base_revision, 9)
  assert.equal(sync.blocked, false)
  assert.equal(sync.revision, 10)
})

test('a stale tab sharing the same device id cannot overwrite the active tab', async () => {
  let calls = 0
  const { sync } = fixture(async () => {
    calls++
    return response(409, { revision: 9, device_id: 'web-one', fraction: 0.7 })
  })

  await sync.save({ cfi: 'stale-tab', fraction: 0.4 })
  await sync.save()

  assert.equal(calls, 1)
  assert.equal(sync.revision, 9)
  assert.equal(sync.blocked, true)
})

test('layout-only locator changes do not reclaim progress ownership', () => {
  assert.equal(readerPositionMoved({ cfi: 'old', fraction: 0.4 }, { cfi: 'new-layout', fraction: 0.400001 }), false)
  assert.equal(readerPositionMoved({ fraction: 0.4 }, { fraction: 0.41 }), true)
  assert.equal(readerPositionMoved(
    { fraction: 0.4, locators: { comic: '{"page":4}' } },
    { fraction: 0.41, cfi: 'layout-only', locators: { comic: '{"page":4}' } },
  ), false)
  assert.equal(readerPositionMoved(
    { fraction: 0.4, locators: { comic: '{"page":4}' } },
    { fraction: 0.41, locators: { comic: '{"page":5}' } },
  ), true)
})

test('accepting a remote position clears pending work without echoing it back', async () => {
  let calls = 0
  const { sync, synced } = fixture(async () => { calls++; return response(500, null) })
  sync.schedule({ cfi: 'local', fraction: 0.2 })
  sync.acceptRemote({ cfi: 'remote', fraction: 0.7 }, 'Restored')
  await sync.save()

  assert.equal(calls, 0)
  assert.deepEqual(sync.latestPosition, { cfi: 'remote', fraction: 0.7 })
  assert.deepEqual(synced, [{ cfi: 'remote', fraction: 0.7 }])
  assert.equal(sync.blocked, false)
})

test('loading treats an absent server position as an empty shelf state', async () => {
  const { sync } = fixture(async () => response(404, null))
  assert.equal(await sync.load(), null)
})

test('staying on the local page explicitly reclaims write ownership', async () => {
  const requests = []
  const { sync } = fixture(async (_url, options) => {
    const body = JSON.parse(options.body)
    requests.push(body)
    return response(200, { ...body, revision: 4 })
  })
  sync.revision = 3
  sync.acceptRemote({ cfi: 'remote', fraction: 0.8 })
  assert.equal(sync.claimLocal({ cfi: 'local', fraction: 0.25 }), true)
  await sync.save()

  assert.equal(requests.length, 1)
  assert.equal(requests[0].base_revision, 3)
  assert.equal(requests[0].fraction, 0.25)
})

test('what is kept while blocked is marked conflicted, and moving on purpose clears it', async () => {
  const { sync, timers, saved, conflicts } = fixture(async (_url, options) =>
    JSON.parse(options.body).cfi === 'one' ? response(409, { revision: 9 }) : response(200, { ...JSON.parse(options.body), revision: 10 }))
  await sync.save({ cfi: 'one', fraction: 0.4 })
  assert.equal(conflicts.length, 1, 'the refused copy is marked')
  sync.schedule({ cfi: 'one', fraction: 0.400001 })
  assert.equal(saved.at(-1).conflicted, true, 'a passive copy while blocked stays marked')
  sync.schedule({ cfi: 'two', fraction: 0.5 })
  assert.equal(saved.at(-1).conflicted, false, 'a deliberate move is a fresh claim')
  await timers.at(-1)()
})

test('a position deleted elsewhere leaves this tab able to save again', async () => {
  const requests = []
  const { sync, statuses, conflicts } = fixture(async (_url, options) => {
    const body = JSON.parse(options.body)
    requests.push(body)
    // Nothing is stored any more. The server answers every save that names an old
    // revision with 409 and revision 0, which is how it says "there is no position".
    return body.base_revision === 0
      ? response(200, { ...body, revision: 1 })
      : response(409, { library_id: 'books', book_id: 'one', fraction: 0, updated_at: 0, revision: 0 })
  })
  sync.revision = 5

  await sync.save({ cfi: 'here', fraction: 0.4 })

  assert.deepEqual(requests.map(request => request.base_revision), [5, 0], 'the retry starts from nothing instead of the lost revision')
  assert.equal(sync.blocked, false, 'there was nothing to merge with, so the tab is not held back')
  assert.equal(sync.revision, 1)
  assert.equal(statuses.at(-1), 'Saved')
  assert.deepEqual(conflicts, [], 'a deleted position is not a conflict to preserve')
})

test('a position deleted before this tab loads is saved from the ground up', async () => {
  const requests = []
  const { sync } = fixture(async (_url, options) => {
    if (options?.method !== 'PUT') return response(404, null)
    const body = JSON.parse(options.body)
    requests.push(body)
    return response(200, { ...body, revision: 1 })
  })
  sync.revision = 5 // an earlier load in this tab, before the book was marked unread
  assert.equal(await sync.load(), null)
  assert.equal(sync.revision, 0, 'no position means no revision to claim')
  await sync.save({ cfi: 'here', fraction: 0.4 })
  assert.equal(requests[0].base_revision, 0)
})
