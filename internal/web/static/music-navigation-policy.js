import { catalogTitleBucket } from './catalog-title-bucket.js'

/** The alphabet is meaningful only where the current music surface is ordered by name. */
export function musicAlphabetEnabled({ browse, sort = 'title', selectedGroup = null, selectedArtist = null }) {
  if (selectedGroup || selectedArtist) return false
  if (browse === 'albums' || browse === 'artists') return true
  return browse === 'tracks' && (sort === 'title' || sort === 'artist')
}

export function musicNavigationLabel({ browse, sort = 'title', title = '', context = '' }) {
  if (browse === 'tracks' && sort === 'artist') return String(context).split(' · ')[0].trim() || title
  return title
}

export function musicNavigationEntries(sections, { browse, sort = 'title' }) {
  return sections.map(element => ({
    element,
    label: musicNavigationLabel({
      browse,
      sort,
      title: element.querySelector('h2,strong')?.textContent || '',
      context: element.querySelector('.audio-open small')?.textContent || '',
    }),
  }))
}

export function musicEffectiveSort(kind, browse, sort) {
  if (kind !== 'music') return sort
  if (browse === 'albums') return 'album'
  if (browse === 'artists') return 'artist'
  return sort
}

export function musicTitleBucket(value) {
  return catalogTitleBucket(value)
}
