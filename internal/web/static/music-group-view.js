import { musicGroupArtworkURL } from './music-group-artwork.js?v=1'

export { musicGroupArtworkURL }

export const musicGroupMonogram = musicArtworkMonogram

const backIconHTML = '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m15 18-6-6 6-6M9 12h10"/></svg>'

function artworkHTML(group, escapeHTML, className = 'music-group-art') {
  const artwork = musicGroupArtworkURL(group)
  return `<span class="${className}"${artwork ? ` data-art="${escapeHTML(artwork)}"` : ''}><b aria-hidden="true">${escapeHTML(musicGroupMonogram(group.name))}</b></span>`
}

export function musicGroupGridHTML(groups, { escapeHTML, type, title = '', backLabel = '', layout = 'covers' }) {
  const heading = title ? `<header class="music-group-heading">${backLabel ? `<button type="button" data-music-group-back aria-label="${escapeHTML(backLabel)}">${backIconHTML}</button>` : ''}<div><h2>${escapeHTML(title)}</h2><p>${groups.length.toLocaleString()} ${groups.length === 1 ? 'collection' : 'collections'}</p></div></header>` : ''
  const cards = groups.map(group => {
    const subtitle = group.subtitle || `${group.count.toLocaleString()} ${group.count === 1 ? 'track' : 'tracks'}`
    return `<button type="button" class="music-group-card" data-music-group="${escapeHTML(group.name)}" data-music-group-type="${escapeHTML(type)}">${artworkHTML(group, escapeHTML)}<strong>${escapeHTML(group.name)}</strong><small>${escapeHTML(subtitle)}</small></button>`
  }).join('')
  return `${heading}<div class="music-group-grid${layout === 'list' ? ' list' : ''}">${cards || '<p class="empty-state">Nothing found here yet.</p>'}</div>`
}

export function musicTrackGroupHeadingHTML(group, artist, escapeHTML) {
  const context = [artist, `${group.count.toLocaleString()} ${group.count === 1 ? 'track' : 'tracks'}`].filter(Boolean).join(' · ')
  const download = group.id && group.artwork_library_id ? `<a href="/api/audio-group-download?library=${encodeURIComponent(group.artwork_library_id)}&amp;group_id=${encodeURIComponent(group.id)}" download>Download</a>` : ''
  return `<header class="music-group-heading featured"><button type="button" data-music-group-back aria-label="Back to ${escapeHTML(artist || 'albums')}">${backIconHTML}</button>${artworkHTML(group, escapeHTML, 'music-group-heading-art')}<div class="music-group-identity"><h2>${escapeHTML(group.name)}</h2><p>${escapeHTML(context)}</p></div><div class="music-group-actions"><button type="button" data-play-music-group>Play</button><button type="button" data-shuffle-music-group aria-label="Shuffle ${escapeHTML(group.name)}">Shuffle</button>${download}</div></header>`
}
import { musicArtworkMonogram } from './music-artwork-fallback.js?v=1'
