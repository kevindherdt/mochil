// Pure family-specific audio view helpers. Keeping these free of DOM and
// network state lets the shared player controller stay focused on playback.
const backIcon = '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m15 6-6 6 6 6"/></svg>'
const podcastIcon = '<svg class="audio-art-placeholder" viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="2"/><path d="M8.5 15.5a5 5 0 0 1 0-7M15.5 8.5a5 5 0 0 1 0 7M5.7 18.3a9 9 0 0 1 0-12.6M18.3 5.7a9 9 0 0 1 0 12.6"/></svg>'
const audiobookIcon = '<svg class="audio-art-placeholder" viewBox="0 0 24 24" aria-hidden="true"><path d="M4 5.5A3.5 3.5 0 0 1 7.5 2H11v17H7.5A3.5 3.5 0 0 0 4 22Zm16 0A3.5 3.5 0 0 0 16.5 2H13v17h3.5A3.5 3.5 0 0 1 20 22Z"/></svg>'
export function podcastFeedView(feeds, query, natural) {
  const needle = query.toLocaleLowerCase()
  return feeds.filter(feed => !needle || [feed.title, feed.description].some(value => value?.toLocaleLowerCase().includes(needle)))
    .slice().sort((a, b) => natural.compare(a.title, b.title))
}

export function podcastOverviewHTML(feeds, layout, escapeHTML) {
  if (!feeds.length) return '<p class="empty-state">No subscribed podcasts yet.</p>'
  const count = feed => feed.episode_count || 0
  if (layout === 'covers') return `<div class="podcast-cover-grid">${feeds.map(feed => `<button class="podcast-card" data-podcast="${escapeHTML(feed.title)}"><span class="podcast-card-art" data-art="${escapeHTML(feed.artwork_url || '')}">${podcastIcon}</span><strong>${escapeHTML(feed.title)}</strong><small>${count(feed)} ${count(feed) === 1 ? 'episode' : 'episodes'}</small></button>`).join('')}</div>`
  return feeds.map(feed => `<details class="podcast-list-group" data-podcast="${escapeHTML(feed.title)}"><summary><span class="podcast-list-art" data-art="${escapeHTML(feed.artwork_url || '')}">${podcastIcon}</span><span><strong>${escapeHTML(feed.title)}</strong><small>${count(feed)} ${count(feed) === 1 ? 'episode' : 'episodes'}</small></span><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m8 10 4 4 4-4"/></svg></summary></details>`).join('')
}

export function podcastFocusHTML(name, episodes, total, row, escapeHTML) {
  const count = episodes.length === total ? episodes.length.toLocaleString() : `${episodes.length.toLocaleString()} of ${total.toLocaleString()}`
  return `<section class="podcast-focus"><button class="podcast-back">${backIcon}<span>All podcasts</span></button><header><span class="podcast-focus-art" data-art="${escapeHTML(episodes[0]?.artwork_url || '')}">${podcastIcon}</span><div><h2>${escapeHTML(name)}</h2><p>${count} episodes</p></div></header>${episodes.map(item => row(item)).join('')}</section>`
}

export function audiobookOverviewHTML(groups, layout, escapeHTML, groupType = 'series') {
  if (!groups.length) return '<p class="empty-state">No audiobooks found here yet.</p>'
  const metadata = group => `${group.count} ${group.count === 1 ? 'audiobook' : 'audiobooks'}${groupType === 'series' && group.artist ? ` · ${escapeHTML(group.artist)}` : ''}`
  if (layout === 'covers') return `<div class="audiobook-series-grid">${groups.map(group => `<button class="audiobook-series-card" data-audiobook-series="${escapeHTML(group.name)}"><span class="audiobook-series-art" data-art="${escapeHTML(group.artwork_url || '')}">${audiobookIcon}</span><strong>${escapeHTML(group.name)}</strong><small>${metadata(group)}</small></button>`).join('')}</div>`
  return groups.map(group => `<button class="audiobook-series-row" data-audiobook-series="${escapeHTML(group.name)}"><span class="audiobook-series-row-art" data-art="${escapeHTML(group.artwork_url || '')}">${audiobookIcon}</span><span><strong>${escapeHTML(group.name)}</strong><small>${metadata(group)}</small></span><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m9 6 6 6-6 6"/></svg></button>`).join('')
}

export function audiobookFocusHTML(name, books, layout, row, card, escapeHTML, groupType = 'series') {
  const representative = books[0]
  return `<section class="audiobook-focus"><button class="audiobook-back">${backIcon}<span>All audiobook ${groupType === 'authors' ? 'authors' : 'series'}</span></button><header><span class="audiobook-focus-art" data-art="${escapeHTML(representative?.artwork_url || '')}">${audiobookIcon}</span><div><h2>${escapeHTML(name)}</h2><p>${books.length} ${books.length === 1 ? 'audiobook' : 'audiobooks'}${groupType === 'series' && representative?.artist ? ` · ${escapeHTML(representative.artist)}` : ''}</p></div></header><div class="${layout === 'covers' ? 'audiobook-book-grid' : 'audiobook-book-list'}">${books.map(item => layout === 'covers' ? card(item) : row(item)).join('')}</div></section>`
}

/**
 * Owns podcast and audiobook surface selection while the shared audio
 * coordinator retains loading, paging, playback, and DOM lifecycle state.
 */
export function spokenMediaSurface({
  kind,
  browse,
  items,
  podcast,
  audiobook,
  layouts,
  search,
  natural,
  row,
  audiobookCard,
  escapeHTML,
}) {
  if (kind === 'podcasts') {
    const feeds = podcastFeedView(podcast.feeds, search, natural)
    return podcast.show
      ? { html: podcastFocusHTML(podcast.show, podcast.episodes, podcast.total, row, escapeHTML), status: '' }
      : { html: podcastOverviewHTML(feeds, layouts.podcasts, escapeHTML), status: '' }
  }
  if (kind !== 'audiobooks') return null

  if (['series', 'authors'].includes(browse)) {
    return audiobook.group
      ? { html: audiobookFocusHTML(audiobook.group, audiobook.books, layouts.audiobookItems, row, audiobookCard, escapeHTML, audiobook.groupType), status: '' }
      : { html: audiobookOverviewHTML(audiobook.groups, layouts.audiobookSeries, escapeHTML, audiobook.groupType), status: '' }
  }

  const covers = layouts.audiobookItems === 'covers'
  const markup = items.map(item => covers ? audiobookCard(item) : row(item)).join('')
  return {
    html: markup
      ? `<div class="${covers ? 'audiobook-book-grid' : 'audiobook-book-list'}">${markup}</div>`
      : '<p class="empty-state">No audiobooks found here yet.</p>',
    status: '',
  }
}
