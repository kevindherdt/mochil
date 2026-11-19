// Add ?debug=1 to a Readium reader address to see where time goes on this
// device: how long each page turn takes to land, which book files were slow to
// arrive, and which stretches froze the page. Tap the panel to copy it.
export function formatTimingLine(label, ms, note = '') {
  return `${String(Math.round(ms)).padStart(6)} ms  ${label}${note ? '  ' + note : ''}`
}

export function createReaderTiming({ document, now = () => performance.now(), observe = true, limit = 14 }) {
  const lines = []
  const panel = document.createElement('pre')
  panel.id = 'timing-panel'
  panel.setAttribute('aria-live', 'off')
  document.body.append(panel)

  const render = () => { panel.textContent = lines.slice(-limit).join('\n') }
  const add = line => { lines.push(line); render() }

  let started = null
  const api = {
    // The reader calls this when a turn, tap or jump begins and again when the page lands.
    begin(label) { started = { label, at: now() } },
    land(note = '') {
      if (!started) return
      add(formatTimingLine(started.label, now() - started.at, note))
      started = null
    },
    add,
    text: () => lines.join('\n'),
  }
  panel.addEventListener('click', () => navigator.clipboard?.writeText(api.text()).then(() => add('copied')).catch(() => {}))

  if (observe && typeof PerformanceObserver === 'function') {
    try {
      new PerformanceObserver(list => {
        for (const entry of list.getEntries()) {
          if (!entry.name.includes('/api/readium/')) continue
          const cached = entry.transferSize === 0 && entry.decodedBodySize > 0
          if (entry.duration >= 300 || (!cached && entry.responseStart - entry.requestStart >= 300))
            add(formatTimingLine(entry.name.replace(/^.*\/api\/readium\/[^/]+\/[^/]+\//, ''), entry.duration, `server ${Math.round(entry.responseStart - entry.requestStart)} ms`))
        }
      }).observe({ type: 'resource', buffered: true })
    } catch {}
    try {
      new PerformanceObserver(list => {
        for (const entry of list.getEntries()) if (entry.duration >= 120) add(formatTimingLine('page froze', entry.duration))
      }).observe({ type: 'longtask', buffered: true })
    } catch {}
  }
  return api
}
