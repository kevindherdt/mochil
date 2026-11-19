export function audioGroupLabel(item, kind, musicBrowse = '') {
  if (kind === 'podcasts') return item.album || item.artist || 'Podcast episodes'
  if (kind === 'audiobooks') return item.series || item.album || 'Standalone audiobooks'
  if (kind === 'music' && musicBrowse === 'albums') return item.album || 'Unknown album'
  if (kind === 'music' && musicBrowse === 'artists') return item.album_artist || item.artist || 'Unknown artist'
  return ''
}

export function groupedAudioHTML(items, { label, row, escapeHTML, music = false, total = items.length }) {
  const groups = new Map()
  for (const item of items) {
    const name = label(item)
    if (!groups.has(name)) groups.set(name, [])
    groups.get(name).push(item)
  }
  return [...groups].map(([name, group]) => `<section class="audio-group"><header><h2>${escapeHTML(name)}</h2><span>${group.length}${items.length < total ? ' loaded' : ''}</span>${music ? `<button data-play-group="${escapeHTML(name)}">Play</button>` : ''}</header>${group.map(item => row(item)).join('')}</section>`).join('') || '<p class="empty-state">Nothing found here yet.</p>'
}
