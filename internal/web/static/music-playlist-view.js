const playlistItems = playlist => Array.isArray(playlist?.items) ? playlist.items : [];

const itemArtwork = item => String(item?.artwork_url || item?.player_artwork_url || "").trim();

const icons = {
  back: '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m15 18-6-6 6-6"/></svg>',
  more: '<svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="5" r="1.4"/><circle cx="12" cy="12" r="1.4"/><circle cx="12" cy="19" r="1.4"/></svg>',
};

export function playlistArtworkURLs(playlist, limit = 4) {
  if (playlist?.detailLoaded === false) return (playlist.artwork_urls || []).slice(0, limit);
  const seen = new Set();
  const artwork = [];
  for (const item of playlistItems(playlist)) {
    const url = itemArtwork(item);
    if (!url || seen.has(url)) continue;
    seen.add(url);
    artwork.push(url);
    if (artwork.length === limit) break;
  }
  return artwork;
}

const playlistArtworkHTML = (playlist, escapeHTML) => {
  const artwork = playlistArtworkURLs(playlist);
  if (!artwork.length)
    return '<span class="music-playlist-art"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M5 6h10M5 10h10M5 14h7M18 5v10.5a2.5 2.5 0 1 1-2-2.2V5Z" /></svg></span>';
  if (artwork.length === 1)
    return `<span class="music-playlist-art" data-art="${escapeHTML(artwork[0])}"></span>`;
  return `<span class="music-playlist-art music-playlist-mosaic" data-art-count="${artwork.length}">${artwork.map(url => `<span data-art="${escapeHTML(url)}"></span>`).join("")}</span>`;
};

export function musicPlaylistCardsHTML(playlists, escapeHTML, { limit = Infinity, discovery = false, layout = "covers" } = {}) {
  playlists = Array.isArray(playlists) ? playlists : [];
  const list = !discovery && layout === "list";
  return `<div class="music-playlist-grid${discovery ? " music-discovery-playlists" : ""}${list ? " list" : ""}">${playlists.slice(0, limit)
    .map(
      playlist => {
        const items = playlistItems(playlist);
        const count = playlist.detailLoaded === false ? playlist.item_count : items.length;
        const attribute = discovery ? "data-discovery-playlist" : "data-open-playlist";
        return `<button class="music-playlist-card" ${attribute}="${escapeHTML(playlist.name)}">${playlistArtworkHTML(playlist, escapeHTML)}<strong>${escapeHTML(playlist.name)}</strong><small>${count.toLocaleString()} ${count === 1 ? "track" : "tracks"}</small></button>`;
      },
    )
    .join("")}</div>`;
}

// Favorites is the one built-in playlist: it sits first in Playlists and opens the favorites list.
const favoritesCardHTML = (count, layout) => `<div class="music-playlist-grid${layout === "list" ? " list" : ""} music-favorites-entry"><button class="music-playlist-card" data-open-favorites><span class="music-playlist-art music-favorites-art" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="M12 20.5s-7.5-4.6-7.5-10.2A4.3 4.3 0 0 1 12 7.6a4.3 4.3 0 0 1 7.5 2.7c0 5.6-7.5 10.2-7.5 10.2Z"/></svg></span><strong>Favorites</strong><small>${count.toLocaleString()} ${count === 1 ? "track" : "tracks"}</small></button></div>`;

// A heading with a back arrow for lists that have no tab of their own (Favorites, Recently played).
export function browseHeadingHTML(title, count, backTo, backLabel) {
  return `<header class="music-playlist-heading music-favorites-heading"><button class="music-playlist-back" data-browse-back="${backTo}" aria-label="${backLabel}">${icons.back}</button><div><h2>${title}</h2><p>${count.toLocaleString()} ${count === 1 ? "track" : "tracks"}</p></div></header>`;
}

export function musicPlaylistOverviewHTML(playlists, escapeHTML, layout = "covers", favoriteCount = 0) {
  playlists = Array.isArray(playlists) ? playlists : [];
  const create = '<button class="music-playlist-new" data-new-playlist>New playlist</button>';
  const favorites = favoritesCardHTML(favoriteCount, layout);
  if (!playlists.length)
    return `${create}${favorites}<p class="empty-state">Create a playlist to collect more tracks here.</p>`;
  return create + favorites + musicPlaylistCardsHTML(playlists, escapeHTML, { layout });
}

export function musicPlaylistDetailHTML(playlist, row, escapeHTML) {
  if (playlist.detailLoaded === false) return `<section class="music-playlist-detail"><header class="music-playlist-heading"><button class="music-playlist-back" data-playlist-back aria-label="Back to playlists">${icons.back}</button>${playlistArtworkHTML(playlist, escapeHTML)}<h2>${escapeHTML(playlist.name)}</h2></header><p role="status">${playlist.detailError ? `${escapeHTML(playlist.detailError)} <button data-playlist-retry>Retry</button>` : 'Loading playlist…'}</p></section>`;
  const items = playlistItems(playlist);
  return `<section class="music-playlist-detail" data-playlist="${escapeHTML(playlist.name)}"><header class="music-playlist-heading"><button class="music-playlist-back" data-playlist-back aria-label="Back to playlists">${icons.back}</button>${playlistArtworkHTML(playlist, escapeHTML)}<div><h2>${escapeHTML(playlist.name)}</h2><p>${items.length.toLocaleString()} ${items.length === 1 ? "track" : "tracks"}</p></div><div class="music-playlist-actions"><button data-play-playlist${items.length ? "" : " disabled"}>Play</button><details><summary aria-label="Playlist options">${icons.more}</summary><div><a href="/api/audio-playlists/${encodeURIComponent(playlist.name)}?download=1" download>Download playlist</a><button data-rename-playlist>Rename</button><button class="music-playlist-delete" data-delete-playlist>Delete playlist</button></div></details></div></header><form class="music-playlist-rename" data-playlist-rename-form hidden><label><span>Playlist name</span><input name="name" maxlength="80" required value="${escapeHTML(playlist.name)}"></label><button type="submit">Save</button><button type="button" data-cancel-playlist-rename>Cancel</button><p role="status"></p></form><div class="music-playlist-tracks">${items.length ? items.map(item => row(item)).join("") : '<p class="empty-state">Add tracks from their options menu.</p>'}</div></section>`;
}
