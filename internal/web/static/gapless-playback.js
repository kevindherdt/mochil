const defaultMaximumBytes = 128 * 1024 * 1024

export function createGaplessPlayback({ primary, secondary, key, maximumBytes = defaultMaximumBytes }) {
  let active = primary
  let standby = secondary
  let preparedKey = ''

  secondary.preload = 'auto'

  function reset(item) {
    stop(primary)
    stop(secondary)
	active = primary
	standby = secondary
    preparedKey = ''
    active.src = item.stream_url
    active.preload = 'metadata'
    return active
  }

  function prepare(item) {
    if (!item || !item.stream_url || Number(item.size || 0) > maximumBytes) {
	  if (preparedKey) clearStandby()
      return false
    }
    const nextKey = key(item)
    if (preparedKey === nextKey && standby.src) return true
    stop(standby)
    standby.preload = 'auto'
    standby.src = item.stream_url
    standby.load()
    preparedKey = nextKey
    return true
  }

  function take(item) {
    if (!item || preparedKey !== key(item) || standby.readyState < 2) return null
    const previous = active
    active = standby
    standby = previous
    preparedKey = ''
    standby.pause()
    return active
  }

  function clearStandby() {
    stop(standby)
    preparedKey = ''
  }

  function close() {
    stop(primary)
    stop(secondary)
    preparedKey = ''
	active = primary
	standby = secondary
  }

  return { reset, prepare, take, clearStandby, close, active: () => active }
}

function stop(audio) {
  audio.pause()
  audio.onloadedmetadata = null
  audio.onerror = null
  audio.onwaiting = null
  audio.onplaying = null
  audio.oncanplay = null
  audio.removeAttribute('src')
  audio.load()
}
