export function createAudioPlaybackOrder({ key, getQueue, getActive, random = Math.random }) {
  let shuffled = false
  let order = []

  function rebuild() {
    const queue = getQueue()
    const active = getActive()
    const activeKey = active ? key(active) : ''
    const keepsActive = activeKey && queue.some(item => key(item) === activeKey)
    const rest = queue.filter(item => !keepsActive || key(item) !== activeKey)
    for (let index = rest.length - 1; index > 0; index--) {
      const swap = Math.floor(random() * (index + 1))
      ;[rest[index], rest[swap]] = [rest[swap], rest[index]]
    }
    order = keepsActive ? [active, ...rest] : rest
    return order
  }

  function current() {
    const queue = getQueue()
    const active = getActive()
    const valid = order.length === queue.length &&
      (!active || order.some(item => key(item) === key(active))) &&
      order.every(item => queue.some(candidate => key(candidate) === key(item)))
    return valid ? order : rebuild()
  }

  return {
    get shuffled() { return shuffled },
    setShuffled(value) {
      shuffled = Boolean(value)
      order = shuffled ? rebuild() : []
      return shuffled
    },
    ordered() { return shuffled ? current() : getQueue() },
    adjacent(delta) {
      const active = getActive()
      const list = shuffled ? current() : getQueue()
      const index = active ? list.findIndex(item => key(item) === key(active)) : -1
      return index < 0 ? undefined : list[index + delta]
    },
  }
}
