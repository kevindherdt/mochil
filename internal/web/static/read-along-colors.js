// Highlight colours for the sentence being read aloud. Chosen per browser in Settings.
export const highlightKey = 'bobobooks-read-along-color'
export const highlightColors = {
  amber: ['Amber', '214,110,70'],
  yellow: ['Yellow', '245,205,40'],
  green: ['Green', '70,180,110'],
  blue: ['Blue', '70,140,230'],
  pink: ['Pink', '235,100,160'],
  violet: ['Violet', '150,110,230'],
}

export function highlightColor(storage) {
  try {
    const saved = storage.getItem(highlightKey)
    return Object.hasOwn(highlightColors, saved) ? saved : 'amber'
  } catch { return 'amber' }
}

/** Strength is higher than it looks: a tint must stay readable on both light and dark pages. */
export function highlightCSS(className, name) {
  const rgb = highlightColors[name]?.[1] ?? highlightColors.amber[1]
  return `.${className}{background-image:linear-gradient(rgba(${rgb},.38),rgba(${rgb},.38))!important;border-radius:.2em;box-decoration-break:clone;-webkit-box-decoration-break:clone;transition:background .15s}`
}
