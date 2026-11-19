import { createDestinationRail } from './destination-rail.js?v=4'

const destinations = [
  ['discover', 'Discover', '<path d="m12 3 2.4 6.6L21 12l-6.6 2.4L12 21l-2.4-6.6L3 12l6.6-2.4z"/>'],
  ['series', 'Series', '<path d="M6 4h12v4H6zM4 10h16v4H4zM6 16h12v4H6z"/>'],
  ['authors', 'Authors', '<path d="M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8ZM4 20c0-4 3.5-6 8-6s8 2 8 6"/>'],
  ['books', 'All', '<path d="M4 4h7v7H4zM13 4h7v7h-7zM4 13h7v7H4zM13 13h7v7h-7z"/>'],
  ['collections', 'Collections', '<path d="M3 6h7l2 2h9v11H3z"/>'],
]

export function createPublicationDestinations({ element, gestureSurface, getBrowseMode, getCollectionMode, onSelect }) {
  let selected = getCollectionMode() ? 'collections' : getBrowseMode()
  const rail = createDestinationRail({
    element,
    destinations,
    attribute: 'data-publication-destination',
    getValue: () => selected,
    setValue: value => { selected = value },
    gestureSurface,
    onChange: async () => onSelect(selected),
  })
  return {
    sync() {
      selected = getCollectionMode() ? 'collections' : getBrowseMode()
      rail.sync()
    },
    setVisible: rail.setVisible,
  }
}
