import { createSerializedWriter } from './serialized-writer.js?v=1'

/** Owns queue loading, edits, save ordering, and retry state. */
export function createMusicQueueState({requestJSON, key, onChanged}) {
  let queue = []
  let queueLoaded = false, queueLoading
  let queueVersion = 0
  let queueReplacement = 0
  let queueSaveError = ''
  let queueLoadError = ''
  const asItems = value => Array.isArray(value) ? value : []
  const entries = items => items.map(item => ({library_id:item.library_id, item_id:item.id}))
  const writeQueue = createSerializedWriter(async ({version, value}) => {
    await requestJSON('/api/audio-queue', {method:'PUT', headers:{'Content-Type':'application/json'}, body:JSON.stringify(value)})
    if (version === queueVersion) { queueSaveError = ''; onChanged() }
  }, (error, {version}) => {
    if (version !== queueVersion) return
    queueSaveError = 'Queue changes could not be saved. Your edits are still here.'
    onChanged()
  })

  async function loadQueue() {
    const version = queueVersion
    try {
      const loaded = (await requestJSON('/api/audio-queue')).data
      if (!Array.isArray(loaded)) throw new Error('Invalid queue response')
      if (version === queueVersion) { queue = loaded; queueLoaded = true; queueLoadError = '' }
    } catch {
      if (version === queueVersion) {
        queueLoadError = 'Could not load the saved queue. No changes were sent.'
        onChanged()
      }
    }
  }
  async function ensureQueue() {
    if (queueLoaded) return queue
    if (!queueLoading) queueLoading = loadQueue().finally(() => { queueLoading = null })
    await queueLoading
    return queueLoaded ? queue : null
  }
  function setQueue(items) {
    // Replacing/clearing the queue supersedes earlier actions awaiting its load.
    queueReplacement++
    commitQueue(items)
  }
  function commitQueue(items) {
    queueLoaded = true
    queueLoadError = ''
    queue = asItems(items).map(item => ({...item}))
    queueSaveError = ''
    writeQueue({version:++queueVersion, value:entries(queue)})
  }
  async function addToQueue(item) {
    if (!item) return
    const selected = {...item}
    const replacement = queueReplacement
    if (!await ensureQueue() || replacement !== queueReplacement) return
    if (!queue.some(candidate => key(candidate) === key(selected))) commitQueue([...queue, selected])
  }
  async function playNext(item, current) {
    if (!item) return
    const selected = {...item}, anchor = current ? {...current} : null
    const replacement = queueReplacement
    if (!await ensureQueue() || replacement !== queueReplacement) return
    commitQueue(queueWithNext(queue, selected, anchor, key))
  }
  function removeFromQueue(item) {
    setQueue(queue.filter(candidate => key(candidate) !== key(item)))
  }
  function moveInQueue(item, delta) {
    const index = queue.findIndex(candidate => key(candidate) === key(item))
    const target = index + delta
    if (index < 0 || target < 0 || target >= queue.length) return
    const next = queue.slice()
    ;[next[index], next[target]] = [next[target], next[index]]
    setQueue(next)
  }
  function clearQueue() { setQueue([]) }
  return {
    ensureQueue, setQueue, playNext, addToQueue, removeFromQueue, moveInQueue, clearQueue,
    retryQueueSave: () => queueLoaded ? commitQueue(queue) : ensureQueue().then(onChanged),
    get items() { return queue },
    get error() { return queueSaveError || queueLoadError },
  }
}

export function queueWithNext(queue, item, current, key) {
  const itemKey = key(item)
  const withoutItem = queue.filter(candidate => key(candidate) !== itemKey)
  const currentKey = current ? key(current) : ''
  const currentIndex = withoutItem.findIndex(candidate => key(candidate) === currentKey)
  if (currentIndex >= 0) return [...withoutItem.slice(0, currentIndex + 1), item, ...withoutItem.slice(currentIndex + 1)]
  return current && currentKey !== itemKey ? [current, item, ...withoutItem.filter(candidate => key(candidate) !== currentKey)] : [item, ...withoutItem]
}
