import test from 'node:test'
import assert from 'node:assert/strict'
import { createSleepTimer } from '../static/narration-sleep.js'

// A manual clock: time only moves when the test says so.
function clock() {
  let time = 0, next = 1
  const pending = new Map()
  return {
    now: () => time,
    setTimer: (callback, ms) => { const id = next++; pending.set(id, { callback, at: time + ms }); return id },
    clearTimer: id => pending.delete(id),
    advance(ms) {
      const end = time + ms
      for (;;) {
        const due = [...pending].filter(([, t]) => t.at <= end).sort((a, b) => a[1].at - b[1].at)[0]
        if (!due) break
        pending.delete(due[0]); time = due[1].at; due[1].callback()
      }
      time = end
    },
  }
}

function setup() {
  const c = clock()
  const audio = { volume: 1 }
  const events = []
  const timer = createSleepTimer({ audio, onExpire: () => events.push(`pause at volume ${audio.volume.toFixed(2)}`), ...c })
  return { c, audio, events, timer }
}

test('nothing happens before the time is up, then narration pauses', () => {
  const { c, events, timer, audio } = setup()
  timer.set(15)
  c.advance(14 * 60000)
  assert.deepEqual(events, [])
  assert.equal(audio.volume, 1, 'no fade yet')
  c.advance(60000 + 1000)
  assert.equal(events.length, 1)
  assert.equal(audio.volume, 1, 'volume is back to normal for the next time it plays')
  assert.equal(timer.remainingMs(), 0)
})

test('the voice fades over the last seconds', () => {
  const { c, audio, events, timer } = setup()
  timer.set(1)
  c.advance(60000 - 5000)
  assert.ok(audio.volume > 0.4 && audio.volume < 0.6, `halfway through the fade: ${audio.volume}`)
  c.advance(4000)
  assert.ok(audio.volume < 0.2, `nearly silent: ${audio.volume}`)
  c.advance(2000)
  assert.equal(events.length, 1)
})

test('setting a new time or turning it off cancels the old one and restores the volume', () => {
  const { c, audio, events, timer } = setup()
  timer.set(1)
  c.advance(55000)
  assert.ok(audio.volume < 1)
  timer.set(0)
  assert.equal(audio.volume, 1)
  c.advance(10 * 60000)
  assert.deepEqual(events, [], 'a cancelled timer must not fire')
  timer.set(1); timer.set(30)
  c.advance(2 * 60000)
  assert.deepEqual(events, [], 'a restarted timer uses the new time')
})

test('the time left is reported', () => {
  const { c, timer } = setup()
  assert.equal(timer.remainingMs(), 0)
  timer.set(30)
  c.advance(10 * 60000)
  assert.equal(timer.remainingMs(), 20 * 60000)
})
