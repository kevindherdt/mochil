// Which episodes this browser has saved to the device. A browser download cannot be observed afterwards, so the list
// remembers the ones started here: it is a reminder for this browser, not a scan of the device.
const KEY = 'mochil-device-saved'
const read = () => { try { return new Set(JSON.parse(localStorage.getItem(KEY) || '[]')) } catch { return new Set() } }

export const onDevice = id => read().has(id)

export function podcastBadgesHTML(item, escapeHTML) {
  const badges = [item.server_downloaded ? ['server', 'On server'] : null, onDevice(item.id) ? ['device', 'On this device'] : null].filter(Boolean)
  return badges.length ? `<span class="episode-badges">${badges.map(([kind, label]) => `<em class="episode-badge" data-kind="${kind}">${escapeHTML(label)}</em>`).join('')}</span>` : ''
}

export function createDeviceSaved({ container }) {
  container.addEventListener('click', event => {
    const link = event.target.closest?.('a.audio-quick-device')
    const row = link?.closest('[data-item]')
    if (!row) return
    const id = row.dataset.item
    try { const ids = read(); ids.add(id); localStorage.setItem(KEY, JSON.stringify([...ids].slice(-2000))) } catch { /* private mode */ }
    link.dataset.state = 'ready'
    if (!row.querySelector('.episode-badge[data-kind=device]')) {
      const badges = row.querySelector('.episode-badges') || row.querySelector('.audio-open > span:last-of-type')?.appendChild(Object.assign(document.createElement('span'), { className: 'episode-badges' }))
      badges?.insertAdjacentHTML('beforeend', '<em class="episode-badge" data-kind="device">On this device</em>')
    }
  })
}
