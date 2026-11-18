// Browser-local defaults for the Readium EPUB reader; book positions are separate.
const key = 'bobobooks-readium-settings'
const defaults = { fontSize: 1, flow: 'paginated', columnCount: 0, fontFamily: '', lineHeight: 1.6, pageGutter: 40, theme: 'app' }
const choices = {
  flow: ['paginated', 'scrolled'], fontFamily: ['', 'serif', 'sans-serif', 'monospace'],
  columnCount: [0, 1, 2], lineHeight: [1.4, 1.6, 1.8, 2], pageGutter: [16, 24, 40, 64], theme: ['app', 'light', 'sepia', 'dark'],
}
export function loadReaderSettings(storage) {
  let saved = {}
  try { saved = JSON.parse(storage.getItem(key) || '{}') || {} } catch {}
  const result = { ...defaults }
  for (const [field, values] of Object.entries(choices)) if (values.includes(saved[field])) result[field] = saved[field]
  if (typeof saved.fontFamily === 'string' && /^BoboFont[0-9a-f]{16}$/.test(saved.fontFamily)) result.fontFamily = saved.fontFamily
  try {
    const size = Number(storage.getItem('bobobooks-readium-font-size'))
    if (Number.isFinite(size) && size >= 0.7 && size <= 2.4) result.fontSize = size
  } catch {}
  return result
}
export function saveReaderSettings(storage, settings) {
  const { fontSize, ...rest } = settings
  try { storage.setItem(key, JSON.stringify(rest)) } catch {}
  try { storage.setItem('bobobooks-readium-font-size', String(fontSize)) } catch {}
}
export function readiumPreferences(settings, appTheme) {
  const theme = settings.theme === 'app' ? appTheme : settings.theme
  const palette = theme === 'light'
    ? { backgroundColor: '#f5efe6', textColor: '#241f1a', linkColor: '#c85d3a', selectionBackgroundColor: '#c85d3a', selectionTextColor: '#fff8f2' }
    : theme === 'sepia'
      ? { backgroundColor: '#f4ecd8', textColor: '#433624', linkColor: '#8b4529', selectionBackgroundColor: '#b87350', selectionTextColor: '#fff8f2' }
      : { backgroundColor: '#171513', textColor: '#f4eee6', linkColor: '#f09a72', selectionBackgroundColor: '#f09a72', selectionTextColor: '#2a1510' }
  // Readium includes gutters in its maximum page width; compensate so a
  // larger gutter also narrows the centered text block on wide screens.
  const maximalLineLength = 80 + (40 - settings.pageGutter) / 4
  return { ...palette, fontSize: settings.fontSize, fontFamily: settings.fontFamily || null, lineHeight: settings.lineHeight, pageGutter: settings.pageGutter, scrollPaddingLeft: settings.pageGutter, scrollPaddingRight: settings.pageGutter, columnCount: settings.columnCount || null, scroll: settings.flow === 'scrolled', hyphens: true, optimalLineLength: 70, maximalLineLength }
}
