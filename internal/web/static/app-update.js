// Tells an open app that a newer version has been deployed. The shell page names
// its bundle by content hash, so a different name on the server means new code.
const ENTRY = /<script type="module" src="(\/assets\/dist\/[^"?]+)/

export function watchForUpdates(win = window, { intervalMs = 10 * 60 * 1000 } = {}) {
  const doc = win.document
  if (!doc?.querySelectorAll) return () => {}
  const current = [...doc.querySelectorAll('script[src]')].map(script => new URL(script.src, win.location.href).pathname).find(path => path.startsWith('/assets/dist/'))
  if (!current) return () => {} // running from unbundled sources
  let announced = false

  function announce() {
    announced = true
    const bar = doc.createElement('div')
    bar.className = 'app-update'
    bar.setAttribute('role', 'status')
    bar.style.cssText = 'position:fixed;left:50%;bottom:20px;transform:translateX(-50%);z-index:90;display:flex;align-items:center;gap:12px;padding:8px 8px 8px 16px;border:1px solid var(--line);border-radius:12px;background:var(--bg);color:var(--text);box-shadow:0 10px 30px rgb(20 16 12/.25);font:600 14px/1.2 "Source Sans","Segoe UI",sans-serif'
    bar.append('A new version is ready')
    const reload = doc.createElement('button')
    reload.type = 'button'
    reload.textContent = 'Reload'
    reload.style.cssText = 'min-height:36px;padding:0 14px;border:0;border-radius:8px;background:var(--accent);color:#fff;font:inherit;cursor:pointer'
    reload.onclick = () => win.location.reload()
    bar.append(reload)
    doc.body.append(bar)
  }

  async function check() {
    if (announced) return
    try {
      const response = await win.fetch('/', { cache: 'no-store', credentials: 'same-origin' })
      if (!response.ok) return
      const latest = ENTRY.exec(await response.text())?.[1]
      if (latest && latest !== current) announce()
    } catch {
      // Offline or signed out: nothing to report.
    }
  }

  const onVisible = () => { if (doc.visibilityState === 'visible') check() }
  doc.addEventListener('visibilitychange', onVisible)
  const timer = win.setInterval(check, intervalMs)
  return () => { doc.removeEventListener('visibilitychange', onVisible); win.clearInterval(timer) }
}
