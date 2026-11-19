import {
  createDestinationRail,
  destinationAfterSwipe as destinationAfterSwipeIn,
  railOverflow,
  railScrollTarget,
} from './destination-rail.js?v=4'

const destinations = [
  ['discover', 'Discover'],
  ['albums', 'Albums'],
  ['artists', 'Artists'],
  ['playlists', 'Playlists'],
  ['tracks', 'Tracks'],
  ['queue', 'Queue'],
]

export { railOverflow, railScrollTarget }

export const destinationAfterSwipe = (current, deltaX) =>
  destinationAfterSwipeIn(destinations, current, deltaX)

export function createMusicDestinations({ element, select, onChange, gestureSurface }) {
  return createDestinationRail({
    element,
    destinations,
    attribute: 'data-music-destination',
    getValue: () => select.value,
    setValue: value => { select.value = value },
    onChange,
    gestureSurface,
    swipeExclusion: 'button, a, input, select, textarea, summary, [role=slider], .music-discovery-groups',
  })
}
