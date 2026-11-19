const supportedKinds = new Set(['music', 'audiobooks'])

export function audioAnalysisPresentation(data = {}) {
  const job = data.job || {}
  const running = ['queued', 'running'].includes(job.status)
  const missing = Math.max(0, Number(data.missing) || 0)
  return {
    running,
    disabled: running || missing === 0,
    action: running ? 'Analyzing…' : missing ? 'Analyze missing' : 'Analysis complete',
    message: running
      ? (job.progress_message || 'Analysis queued…')
      : missing
        ? `${missing.toLocaleString()} items still need analysis.`
        : 'Loudness analysis is complete.',
  }
}

export function createAudioAnalysisController({
  container,
  button,
  status,
  requestJSON,
  confirmAction = message => globalThis.confirm(message),
  schedule = (callback, delay) => globalThis.setTimeout(callback, delay),
  cancel = handle => globalThis.clearTimeout(handle),
  pollMilliseconds = 2_000,
}) {
  let library = ''
  let kind = ''
  let generation = 0
  let timer

  function stopTimer() {
    if (timer !== undefined) cancel(timer)
    timer = undefined
  }

  function enabled() {
    return Boolean(library) && supportedKinds.has(kind)
  }

  async function refresh(ticket = generation) {
    if (!enabled() || ticket !== generation) return false
    try {
      const { data } = await requestJSON(`/api/audio-analysis?library=${encodeURIComponent(library)}`)
      if (ticket !== generation) return false
      const view = audioAnalysisPresentation(data)
      button.disabled = view.disabled
      button.textContent = view.action
      status.textContent = view.message
      stopTimer()
      if (view.running) timer = schedule(() => { timer = undefined; void refresh(ticket) }, pollMilliseconds)
      return true
    } catch (error) {
      if (ticket !== generation) return false
      button.disabled = false
      status.textContent = error.message
      return false
    }
  }

  async function analyze() {
    if (!enabled() || !confirmAction('Analyze missing loudness for this shelf? This runs in the background and may take some time.')) return false
    button.disabled = true
    button.textContent = 'Analyzing…'
    try {
      await requestJSON(`/api/audio-analysis?library=${encodeURIComponent(library)}`, { method: 'POST' })
      return refresh()
    } catch (error) {
      button.disabled = false
      button.textContent = 'Analyze missing'
      status.textContent = error.message
      return false
    }
  }

  button.addEventListener('click', () => { void analyze() })

  return {
    setContext(nextLibrary, nextKind) {
      const changed = nextLibrary !== library || nextKind !== kind
      library = nextLibrary || ''
      kind = nextKind || ''
      container.hidden = !enabled()
      if (!changed) return
      generation += 1
      stopTimer()
      status.textContent = ''
      if (enabled()) void refresh()
    },
    refresh,
    dispose() {
      generation += 1
      stopTimer()
    },
  }
}
