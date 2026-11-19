// Browser-local display tint for Readium. Filtering the host also covers chapter
// iframes and images without inserting an overlay that could intercept input.
const key = 'mochil-readium-warm-filter'

export function mountWarmFilter({ document, storage }) {
  if (!storage) try { storage = globalThis.localStorage } catch {}
  const slider = document.querySelector('#reading-warmth')
  const host = document.querySelector('#readium-host')
  const output = document.querySelector('#reading-warmth-value')
  const status = document.querySelector('#reading-warmth-status')
  let saved = 0
  try {
    const raw = storage.getItem(key)
    const value = Number(raw)
    if (raw !== null && Number.isFinite(value) && value >= 0 && value <= 100) saved = Math.round(value)
  } catch {
    status.textContent = 'Browser storage is unavailable; your setting may not be saved.'
  }
  function apply(value) {
    slider.value = String(value)
    output.textContent = value ? `${value}%` : 'Off'
    slider.setAttribute('aria-valuetext', value ? `${value}% warm` : 'Off')
    host.style.filter = value ? `sepia(${value / 100})` : ''
  }
  apply(saved)
  slider.addEventListener('input', () => {
    const value = Math.min(100, Math.max(0, Number(slider.value) || 0))
    apply(value)
    try {
      storage.setItem(key, String(value))
      status.textContent = 'Saved in this browser.'
    } catch {
      status.textContent = 'Setting not saved; browser storage is unavailable. It will last only until this page is closed or reloaded.'
    }
  })
}

if (typeof document !== 'undefined') mountWarmFilter({ document })
