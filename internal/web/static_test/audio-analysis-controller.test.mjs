import assert from 'node:assert/strict'
import test from 'node:test'
import { audioAnalysisPresentation, createAudioAnalysisController } from '../static/audio-analysis-controller.js'

const element = () => ({ hidden: true, disabled: false, textContent: '', addEventListener(type, handler) { this[type] = handler } })

test('analysis presentation distinguishes pending, missing, and complete shelves', () => {
  assert.deepEqual(audioAnalysisPresentation({ missing: 12 }), { running: false, disabled: false, action: 'Analyze missing', message: '12 items still need analysis.' })
  assert.deepEqual(audioAnalysisPresentation({ missing: 4, job: { status: 'running', progress_message: 'Reading tags…' } }), { running: true, disabled: true, action: 'Analyzing…', message: 'Reading tags…' })
  assert.deepEqual(audioAnalysisPresentation({ missing: 0 }), { running: false, disabled: true, action: 'Analysis complete', message: 'Loudness analysis is complete.' })
})

test('changing shelves cancels stale analysis polling and ignores its response', async () => {
  const container = element(), button = element(), status = element()
  const requests = []
  let resolveOld
  const old = new Promise(resolve => { resolveOld = resolve })
  const controller = createAudioAnalysisController({
    container, button, status,
    requestJSON: path => { requests.push(path); return requests.length === 1 ? old : Promise.resolve({ data: { missing: 0 } }) },
  })
  controller.setContext('old shelf', 'music')
  controller.setContext('new shelf', 'audiobooks')
  await new Promise(resolve => setImmediate(resolve))
  resolveOld({ data: { missing: 99 } })
  await new Promise(resolve => setImmediate(resolve))
  assert.deepEqual(requests, ['/api/audio-analysis?library=old%20shelf', '/api/audio-analysis?library=new%20shelf'])
  assert.equal(status.textContent, 'Loudness analysis is complete.')
  assert.equal(button.disabled, true)
})

test('leaving audio shelves cancels a scheduled poll', async () => {
  const container = element(), button = element(), status = element()
  const scheduled = [], cancelled = []
  const controller = createAudioAnalysisController({
    container, button, status,
    requestJSON: async () => ({ data: { missing: 8, job: { status: 'queued' } } }),
    schedule: callback => { scheduled.push(callback); return 41 },
    cancel: handle => cancelled.push(handle),
  })
  controller.setContext('music', 'music')
  await new Promise(resolve => setImmediate(resolve))
  assert.equal(container.hidden, false)
  assert.equal(scheduled.length, 1)
  controller.setContext('books', 'books')
  assert.equal(container.hidden, true)
  assert.deepEqual(cancelled, [41])
})

test('analysis starts only after confirmation and refreshes the current shelf', async () => {
  const container = element(), button = element(), status = element()
  const requests = []
  const controller = createAudioAnalysisController({
    container, button, status,
    confirmAction: () => true,
    requestJSON: async (path, options) => {
      requests.push([path, options?.method || 'GET'])
      return { data: { missing: 0 } }
    },
  })
  controller.setContext('my music', 'music')
  await new Promise(resolve => setImmediate(resolve))
  requests.length = 0
  await button.click()
  await new Promise(resolve => setImmediate(resolve))
  assert.deepEqual(requests, [
    ['/api/audio-analysis?library=my%20music', 'POST'],
    ['/api/audio-analysis?library=my%20music', 'GET'],
  ])
})
