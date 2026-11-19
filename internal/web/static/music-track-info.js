import { mediaClock } from './media-clock.js?v=1'
export { musicSourceFacts } from './music-source-facts.js?v=1'

const durationLabel = seconds => mediaClock(Math.round(Number(seconds)), '')

const sizeLabel = bytes => {
  const value = Number(bytes)
  if (!Number.isFinite(value) || value <= 0) return ''
  if (value >= 1073741824) return `${(value / 1073741824).toFixed(1)} GB`
  if (value >= 1048576) return `${(value / 1048576).toFixed(1)} MB`
  if (value >= 1024) return `${(value / 1024).toFixed(1)} KB`
  return `${value} B`
}

export function musicTrackDetails(item) {
  const details = []
  const add = (label, value) => { if (String(value || '').trim()) details.push([label, String(value).trim()]) }
  const artist = String(item.artist || '').trim()
  if (artist && artist.toLowerCase() !== 'unknown artist') add('Artist', artist)
  const albumArtist = String(item.album_artist || '').trim()
  if (albumArtist && albumArtist.toLowerCase() !== artist.toLowerCase()) add('Album artist', albumArtist)
  add('Album', item.album)
  add('Genre', item.genre)
  const year = Number(item.year)
  if (year >= 1000 && year <= 2100) add('Year', year)
  const disc = Number(item.disc ?? item.disc_number)
  const track = Number(item.track ?? item.track_number)
  if (disc > 0) add('Disc', disc)
  if (track > 0) add('Track', track)
  add('Format', String(item.container || '').toUpperCase())
	const sampleRate = Number(item.sample_rate)
	if (sampleRate >= 1000) add('Sample rate', `${sampleRate % 1000 ? (sampleRate / 1000).toFixed(1) : sampleRate / 1000} kHz`)
	const bitDepth = Number(item.bit_depth)
	if (bitDepth > 0) add('Bit depth', `${bitDepth}-bit`)
  const bitrate = Number(item.bitrate)
  if (bitrate >= 1000 && bitrate <= 1_000_000_000) add('Bitrate', `${Math.round(bitrate / 1000)} kbps`)
  add('Duration', durationLabel(item.duration_seconds))
  add('File size', sizeLabel(item.size))
  return details
}

export function musicTrackInfoHTML(item, escapeHTML) {
  const details = musicTrackDetails(item)
  return `<div class="music-track-info-card"><header><span class="music-track-info-art" data-art="${escapeHTML(item.artwork_url || '')}"></span><div><small>Track information</small><h2 id="music-track-info-title">${escapeHTML(item.title || 'Untitled track')}</h2></div></header><dl>${details.map(([label, value]) => `<div><dt>${escapeHTML(label)}</dt><dd>${escapeHTML(value)}</dd></div>`).join('')}</dl><form method="dialog"><button>Done</button></form></div>`
}

export function createMusicTrackInfo({ document, escapeHTML, loadArtwork = () => {} }) {
  const dialog = document.createElement('dialog')
  dialog.className = 'music-track-info'
  dialog.setAttribute('aria-labelledby', 'music-track-info-title')
  document.body.append(dialog)
  dialog.addEventListener('click', event => {
    if (event.target === dialog) dialog.close()
  })
  return {
    open(item) {
      dialog.innerHTML = musicTrackInfoHTML(item, escapeHTML)
      const artwork = dialog.querySelector('[data-art]')
      if (item.artwork_url) loadArtwork(artwork, item.artwork_url, true)
      dialog.showModal()
    },
  }
}
