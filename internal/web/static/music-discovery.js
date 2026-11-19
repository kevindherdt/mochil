import { musicPlaylistCardsHTML } from './music-playlist-view.js?v=8'

export function musicDiscoveryHTML(data, { row, card, escapeHTML, groupArtworkURL }) {
  const retry = label => `<p class="music-discovery-empty" role="status">${escapeHTML(label)} could not load. <button data-discovery-destination="discover">Retry</button></p>`
  const trackSection = (title, destination, items, empty, failed = false, sort = '', carousel = false) => `<section class="music-discovery-section" data-discovery-section="${destination}"><header><h2>${escapeHTML(title)}</h2><button data-discovery-destination="${destination}"${sort ? ` data-discovery-sort="${sort}"` : ''}>See all</button></header><div class="music-discovery-tracks${carousel ? ' is-carousel' : ''}">${failed ? retry(title) : items.length ? items.map(item => carousel ? card(item) : row(item, false, false)).join('') : `<p>${escapeHTML(empty)}</p>`}</div></section>`
  const groupSection = (title, destination, groups, empty, failed = false) => {
    const cards = groups.map(group => `<button class="music-discovery-group" data-discovery-group="${escapeHTML(group.name)}" data-discovery-type="${destination}"><span data-art="${escapeHTML(groupArtworkURL(group))}"></span><strong>${escapeHTML(group.name)}</strong><small>${escapeHTML(group.subtitle || `${group.count} tracks`)}</small></button>`).join('')
    const body = failed ? retry(title) : cards ? `<div class="music-discovery-groups">${cards}</div>` : `<p class="music-discovery-empty">${escapeHTML(empty)}</p>`
    return `<section class="music-discovery-section" data-discovery-section="${destination}"><header><h2>${escapeHTML(title)}</h2><button data-discovery-destination="${destination}">See all</button></header>${body}</section>`
  }
  const playlistSection = `<section class="music-discovery-section"><header><h2>Playlists</h2><button data-discovery-destination="playlists">See all</button></header>${data.playlists?.length ? musicPlaylistCardsHTML(data.playlists, escapeHTML, { limit: 8, discovery: true }) : '<p class="music-discovery-empty">Create a playlist to collect tracks here.</p>'}</section>`
  const favorites = data.favorites.length ? trackSection('Favorites', 'favorites', data.favorites, '') : ''
  const failures = data.failures || {}
  return `<div class="music-discovery">${trackSection('Recently played', 'recently-played', data.played, 'Play some music and it will appear here.', failures.played, '', Boolean(card))}${trackSection('Recently added', 'tracks', data.recent, 'No recent tracks yet.', failures.recent, 'recent')}${favorites}${groupSection('Albums', 'albums', data.albums, 'No albums found yet.', failures.albums)}${groupSection('Artists', 'artists', data.artists, 'No artists found yet.', failures.artists)}${playlistSection}</div>`
}
