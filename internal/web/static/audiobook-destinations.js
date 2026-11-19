import { createDestinationRail } from './destination-rail.js?v=4'

const destinations = [
  ['continue', 'Continue'],
  ['recent', 'Recent'],
  ['series', 'Series'],
  ['authors', 'Authors'],
  ['all', 'All'],
]

export function createAudiobookDestinations({ element, select, onChange, gestureSurface }) {
  return createDestinationRail({
    element,
    destinations,
    attribute: 'data-audiobook-destination',
    getValue: () => select.value,
    setValue: value => { select.value = value },
    onChange,
    gestureSurface,
    swipeExclusion: 'button, a, input, select, textarea, summary, [role=slider]',
  })
}
