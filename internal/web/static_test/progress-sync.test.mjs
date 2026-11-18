import assert from 'node:assert/strict'
import test from 'node:test'
import { createProgressSync } from '../static/progress-sync.js'

test('serializes writes and advances the expected revision', async () => {
  const seen = []
  const sync = createProgressSync({ save: async value => { seen.push(value.expected_revision); return { revision: value.expected_revision + 1 } } })
  sync.reset(4)
  await Promise.all([sync.submit({ position: 10 }), sync.submit({ position: 20 })])
  assert.deepEqual(seen, [4, 5])
  assert.equal(sync.revision, 6)
})

test('stops writes after a cross-device conflict', async () => {
  let calls = 0, conflicts = 0
  const sync = createProgressSync({
    save: async () => { calls++; const error = new Error('changed'); error.status = 409; throw error },
    onConflict: () => { conflicts++ },
  })
  await sync.submit({ position: 10 })
  await sync.submit({ position: 20 })
  assert.equal(calls, 1)
  assert.equal(conflicts, 1)
  assert.equal(sync.conflicted, true)
})

test('refreshes a stale revision and retries one active write', async () => {
  const revisions = []
  const sync = createProgressSync({
    save: async value => {
      revisions.push(value.expected_revision)
      if (revisions.length === 1) { const error = new Error('changed'); error.status = 409; throw error }
      return { revision: value.expected_revision + 1 }
    },
    resolveConflict: async () => ({ revision: 8, retry: true }),
  })
  sync.reset(3)
  assert.equal(await sync.submit({ position: 30 }), true)
  assert.deepEqual(revisions, [3, 8])
  assert.equal(sync.revision, 9)
  assert.equal(sync.conflicted, false)
})

test('quietly accepts the remote revision for an inactive write', async () => {
  let conflicts = 0
  const sync = createProgressSync({
    save: async () => { const error = new Error('changed'); error.status = 409; throw error },
    resolveConflict: async () => ({ revision: 11, retry: false }),
    onConflict: () => { conflicts++ },
  })
  sync.reset(4)
  assert.equal(await sync.submit({ position: 10 }), false)
  assert.equal(sync.revision, 11)
  assert.equal(sync.conflicted, false)
  assert.equal(conflicts, 0)
})

test('keeps a recoverable media conflict quiet when revision refresh is temporarily unavailable', async () => {
  let conflicts = 0
  const sync = createProgressSync({
    save: async () => { const error = new Error('changed'); error.status = 409; throw error },
    resolveConflict: async () => ({ handled: true }),
    onConflict: () => { conflicts++ },
  })
  sync.reset(4)
  assert.equal(await sync.submit({ position: 10 }), false)
  assert.equal(sync.conflicted, false)
  assert.equal(conflicts, 0)
})

test('ignores a late response after switching items', async () => {
  let release
  const waiting = new Promise(resolve => { release = resolve })
  const sync = createProgressSync({ save: async () => { await waiting; return { revision: 99 } } })
  sync.reset(2)
  const old = sync.submit({ position: 10 })
  sync.reset(7)
  release()
  await old
  assert.equal(sync.revision, 7)
})

test('a failed network save does not poison subsequent serialized writes', async () => {
  const seen = []
  const sync = createProgressSync({save: async value => {
    seen.push([value.position, value.expected_revision])
    if (seen.length === 1) throw new Error('offline')
    return {revision:value.expected_revision + 1}
  }})
  sync.reset(4)
  const first = sync.submit({position:10})
  const next = sync.submit({position:20})
  await assert.rejects(first, /offline/)
  assert.equal(await next, true)
  assert.deepEqual(seen, [[10,4],[20,4]])
  assert.equal(sync.revision, 5)
})

test('failed conflict retry leaves the reconciled revision available to a later save', async () => {
  const revisions = []
  const sync = createProgressSync({
    save: async value => {
      revisions.push(value.expected_revision)
      if (revisions.length === 1) throw Object.assign(new Error('changed'), {status:409})
      if (revisions.length === 2) throw new Error('offline during retry')
      return {revision:9}
    },
    resolveConflict: async () => ({revision:8,retry:true}),
  })
  sync.reset(3)
  await assert.rejects(sync.submit({position:10}), /offline during retry/)
  assert.equal(await sync.submit({position:20}), true)
  assert.deepEqual(revisions, [3,8,8])
  assert.equal(sync.revision,9)
})

test('reset after a rejected save allows a new item to save with its own revision', async () => {
  let fail = true
  const sync = createProgressSync({save:async value => {
    if (fail) throw new Error('offline')
    return {revision:value.expected_revision + 1}
  }})
  await assert.rejects(sync.submit({position:10}), /offline/)
  fail = false
  sync.reset(12)
  assert.equal(await sync.submit({position:0}),true)
  assert.equal(sync.revision,13)
})

test('an in-flight successful response is not reported as success for a newer generation', async () => {
  let started, finish
  const began = new Promise(resolve => { started = resolve })
  const reply = new Promise(resolve => { finish = resolve })
  const sync = createProgressSync({save:async () => { started(); return reply }})
  sync.reset(2)
  const old = sync.submit({position:10})
  await began
  sync.reset(7)
  finish({revision:99})
  assert.equal(await old,false)
  assert.equal(sync.revision,7)
})

test('conflict resolvers can check ownership before mutating caller state after awaiting', async () => {
  let started, finish, changes = 0
  const began = new Promise(resolve => { started = resolve })
  const reply = new Promise(resolve => { finish = resolve })
  const sync = createProgressSync({
    save:async () => { throw Object.assign(new Error('conflict'),{status:409}) },
    resolveConflict:async (_payload,_error,isCurrent) => {
      started()
      await reply
      if (isCurrent()) changes++
      return {revision:99,handled:true}
    },
  })
  const old = sync.submit({position:10})
  await began
  sync.reset(7)
  finish()
  assert.equal(await old,false)
  assert.equal(changes,0)
  assert.equal(sync.revision,7)
})
