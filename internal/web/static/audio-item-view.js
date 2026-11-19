import { serverSaveHTML } from './podcast-server-save.js?v=1'
import { onDevice, podcastBadgesHTML } from './podcast-device-saved.js?v=1'
import { mediaClock } from './media-clock.js?v=1'
import { musicArtworkMonogram } from './music-artwork-fallback.js?v=1'

const icons = {
  more: '<circle cx="12" cy="5" r="1.4"/><circle cx="12" cy="12" r="1.4"/><circle cx="12" cy="19" r="1.4"/>',
  queue: '<path d="M4 6h10M4 11h10M4 16h6M18 14v7m-3-3.5h6"/>',
  next: '<path d="m7 6 8 6-8 6ZM17 6v12"/>',
  heart: '<path d="M12 20.5 4.5 13A5 5 0 0 1 12 6.5 5 5 0 0 1 19.5 13Z"/>',
  download: '<path d="M12 3v12m-5-5 5 5 5-5M5 21h14"/>',
  check: '<path d="m5 12 4 4L19 6"/>',
}

export const audioClock = mediaClock

const fileSize = bytes => {
  const value = Number(bytes)
  if (!Number.isFinite(value) || value <= 0) return ''
  const units = ['B', 'KB', 'MB', 'GB']
  const unit = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1)
  return `${(value / 1024 ** unit).toFixed(unit > 1 ? 1 : 0)} ${units[unit]}`
}

export function createAudioItemView({ escapeHTML, relativePublishedDate, playlists, isFavorite, browse }) {
  const context = item => item.library_kind === 'audiobooks'
    ? [item.artist, item.narrator && `Narrated by ${item.narrator}`, item.series || item.album].filter(Boolean).join(' · ')
    : item.library_kind === 'podcasts' ? ''
    : [item.artist, item.album || item.series].filter(Boolean).join(' · ')

  const musicActions = item => {
    const playlistOptions = playlists().map(list => `<option value="${escapeHTML(list.name)}">${escapeHTML(list.name)}</option>`).join('')
    const playlistActions = browse() === 'playlists' ? '<button data-action="playlist-up">Move up</button><button data-action="playlist-down">Move down</button><button data-action="playlist-remove">Remove from playlist</button>' : ''
    const downloadAction = item.download_url ? `<a href="${escapeHTML(item.download_url)}" download><svg viewBox="0 0 24 24">${icons.download}</svg>Download</a>` : ''
    return `<button data-action="next"><svg viewBox="0 0 24 24">${icons.next}</svg>Play next</button><button data-action="queue"><svg viewBox="0 0 24 24">${icons.queue}</svg>Add to queue</button><button data-action="favorite"><svg viewBox="0 0 24 24">${icons.heart}</svg>${isFavorite(item) ? 'Remove favorite' : 'Favorite'}</button>${downloadAction}<button data-action="track-info">Track info</button>${playlistActions}${playlistOptions ? `<label class="audio-playlist-add"><span>Playlist</span><select>${playlistOptions}</select><button data-action="playlist">Add</button></label>` : ''}`
  }

  function row(item, dashboard = false, showProgress = item.library_kind !== 'music') {
    const isPodcast = item.library_kind === 'podcasts'
    const fraction = item.progress?.duration_seconds > 0 ? Math.min(100, item.progress.position_seconds / item.progress.duration_seconds * 100) : 0
    const played = !!item.progress?.finished && item.library_kind !== 'music' // a song that ended is not "played": it is just music
    const knownDuration = audioClock(item.duration_seconds)
    const meta = [context(item), knownDuration !== '0:00' ? knownDuration : '', played ? 'Played' : ''].filter(Boolean).join(' · ')
    const queueView = item.library_kind === 'music' && browse() === 'queue'
    const actions = queueView
      ? `<button data-action="queue-up">Move up</button><button data-action="queue-down">Move down</button><button data-action="queue-remove">Remove</button>${item.download_url ? `<a href="${escapeHTML(item.download_url)}" download><svg viewBox="0 0 24 24">${icons.download}</svg>Download</a>` : ''}<button data-action="track-info">Track info</button>`
      : item.library_kind === 'music'
        ? musicActions(item)
        : `${isPodcast ? `<button data-action="played"><svg viewBox="0 0 24 24">${icons.check}</svg>${played ? 'Mark unplayed' : 'Mark played'}</button>` : ''}<a href="${escapeHTML(item.download_url)}" download><svg viewBox="0 0 24 24">${icons.download}</svg>Download</a>`
    const quickPlayed = !dashboard && isPodcast ? `<button class="audio-quick-played" data-action="played" aria-label="${played ? 'Mark as unplayed' : 'Mark as played'}"><svg viewBox="0 0 24 24" aria-hidden="true">${icons.check}</svg></button>` : ''
    const quickDownload = !dashboard && isPodcast ? `${serverSaveHTML(item, escapeHTML)}<a class="audio-quick-download audio-quick-device" data-state="${onDevice(item.id) ? 'ready' : 'idle'}" href="${escapeHTML(item.download_url)}" download aria-label="Save to this device: ${escapeHTML(item.title)}" title="Save to this device"><svg viewBox="0 0 24 24" aria-hidden="true">${icons.download}</svg></a>` : ''
    const menu = dashboard || isPodcast ? '' : `<details class="audio-more"><summary aria-label="More options for ${escapeHTML(item.title)}"><svg viewBox="0 0 24 24" aria-hidden="true">${icons.more}</svg></summary><div>${actions}</div></details>`
    const podcastDetails = isPodcast ? [relativePublishedDate(item.modified), fileSize(item.size)].filter(Boolean).join(' · ') : ''
    const description = isPodcast && item.description ? `<span class="podcast-episode-description">${escapeHTML(item.description)}</span>` : ''
    // Wide screens show music rows as a table: title, artist, album and length in their own columns.
    const tableColumns = item.library_kind === 'music' && !dashboard ? `<span class="audio-col audio-col-artist">${escapeHTML(item.artist || '')}</span><span class="audio-col audio-col-album">${escapeHTML(item.album || '')}</span><span class="audio-col audio-col-time">${knownDuration !== '0:00' ? knownDuration : ''}</span>` : ''
    const glyph = isPodcast && !dashboard ? '' : `<span class="audio-glyph"><b aria-hidden="true">${escapeHTML(musicArtworkMonogram(item.album || item.series || item.title))}</b></span>`
    return `<article class="audio-item${played ? ' is-played' : ''}${isPodcast ? ' podcast-episode' : ''}" data-library="${escapeHTML(item.library_id)}" data-item="${escapeHTML(item.id)}"><button class="audio-open">${glyph}<span><strong>${escapeHTML(item.title)}</strong>${description}<small>${escapeHTML([meta, podcastDetails].filter(Boolean).join(' · '))}</small>${isPodcast && !dashboard ? podcastBadgesHTML(item, escapeHTML) : ''}${showProgress && fraction ? `<i style="--progress:${fraction}%"></i>` : ''}</span>${tableColumns}</button>${quickPlayed}${quickDownload}${menu}${dashboard ? '<button class="audio-dismiss" aria-label="Remove from Continue listening"><svg viewBox="0 0 24 24"><path d="m7 7 10 10M17 7 7 17"/></svg></button>' : ''}</article>`
  }

  function audiobookCard(item) {
    const duration = audioClock(item.duration_seconds)
    const meta = [item.artist, item.narrator && `Narrated by ${item.narrator}`, duration !== '0:00' ? duration : ''].filter(Boolean).join(' · ')
    return `<article class="audio-item audiobook-book-card" data-library="${escapeHTML(item.library_id)}" data-item="${escapeHTML(item.id)}"><button class="audio-open"><span class="audiobook-book-art" data-art="${escapeHTML(item.artwork_url || '')}"></span><strong>${escapeHTML(item.title)}</strong><small>${escapeHTML(meta)}</small></button><details class="audio-more"><summary aria-label="More options for ${escapeHTML(item.title)}"><svg viewBox="0 0 24 24" aria-hidden="true">${icons.more}</svg></summary><div><a href="${escapeHTML(item.download_url)}" download><svg viewBox="0 0 24 24">${icons.download}</svg>Download</a></div></details></article>`
  }

  function musicCard(item) {
    const duration = audioClock(item.duration_seconds)
    const meta = [item.artist, duration !== '0:00' ? duration : ''].filter(Boolean).join(' · ') // the album is in the Albums tab; a card keeps to who and how long
    return `<article class="audio-item music-track-card" data-library="${escapeHTML(item.library_id)}" data-item="${escapeHTML(item.id)}"><button class="audio-open"><span class="music-track-art" data-art="${escapeHTML(item.artwork_url || '')}"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M9 18V5l11-2v13M9 9l11-2M6 21c-2 0-3-1-3-2.5S4 16 6 16s3 1 3 2.5S8 21 6 21Zm11-2c-2 0-3-1-3-2.5s1-2.5 3-2.5 3 1 3 2.5S19 19 17 19Z"/></svg></span><strong>${escapeHTML(item.title)}</strong><small>${escapeHTML(meta)}</small></button><details class="audio-more"><summary aria-label="More options for ${escapeHTML(item.title)}"><svg viewBox="0 0 24 24" aria-hidden="true">${icons.more}</svg></summary><div>${musicActions(item)}</div></details></article>`
  }

  function musicAlbumRow(item, index = 0) {
    const duration = audioClock(item.duration_seconds)
    const disc = Number(item.disc ?? item.disc_number)
    const track = Number(item.track ?? item.track_number)
    const ordinal = Number.isFinite(track) && track > 0
      ? `${Number.isFinite(disc) && disc > 1 ? `${disc}·` : ''}${track}`
      : String(index + 1)
    const meta = [item.artist, duration !== '0:00' ? duration : ''].filter(Boolean).join(' · ')
    return `<article class="audio-item music-album-track" data-library="${escapeHTML(item.library_id)}" data-item="${escapeHTML(item.id)}"><button class="audio-open"><span class="music-track-number" aria-hidden="true">${escapeHTML(ordinal)}</span><span><strong>${escapeHTML(item.title)}</strong><small>${escapeHTML(meta)}</small></span></button><details class="audio-more"><summary aria-label="More options for ${escapeHTML(item.title)}"><svg viewBox="0 0 24 24" aria-hidden="true">${icons.more}</svg></summary><div>${musicActions(item)}</div></details></article>`
  }

  return { audiobookCard, context, musicAlbumRow, musicCard, row }
}
