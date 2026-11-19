export function createSerializedWriter(send, onError = () => {}) {
  let pending, running = false
  async function drain() {
    if (running) return
    running = true
    while (pending !== undefined) {
      const value = pending
      pending = undefined
      try { await send(value) } catch (error) { onError(error, value) }
    }
    running = false
  }
  return value => { pending = value; void drain() }
}
