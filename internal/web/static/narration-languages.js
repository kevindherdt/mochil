// Which languages the narration voices speak, shared by Settings and the reader's Read aloud menu.
const LANGUAGE = { english: 'English', dutch: 'Dutch' }
const escapeHTML = value => String(value).replace(/[&<>"']/g, ch => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[ch])

export const languageName = language => LANGUAGE[language] || (language ? language[0].toUpperCase() + language.slice(1) : 'English')

/** The language buttons: "All", then one per language the voices really have, each with its count. */
export function languageChoices(voices) {
  const counts = new Map()
  for (const voice of voices) counts.set(voice.language || 'english', (counts.get(voice.language || 'english') || 0) + 1)
  const choices = [{ id: 'all', label: 'All', count: voices.length }]
  for (const [id, count] of [...counts].sort((a, b) => languageName(a[0]).localeCompare(languageName(b[0])))) choices.push({ id, label: languageName(id), count })
  const mine = voices.filter(voice => voice.custom).length
  if (mine && mine < voices.length) choices.push({ id: 'mine', label: 'My voices', count: mine })
  return choices
}

/** The voices to show for a language button (or "mine") and the words typed in a search box. */
export function filterVoices(voices, { language = 'all', query = '' } = {}) {
  const wanted = query.trim().toLowerCase()
  return voices.filter(voice =>
    (language === 'all' || (language === 'mine' ? voice.custom : (voice.language || 'english') === language)) &&
    (!wanted || `${voice.name} ${languageName(voice.language || 'english')}`.toLowerCase().includes(wanted)))
}

/** Only languages are offered in the reader: with one language there is nothing to choose. */
export function readerLanguageChoices(voices) {
  const languages = languageChoices(voices).filter(choice => choice.id !== 'mine')
  return languages.length > 2 ? languages : []
}

/**
 * The voice dropdown for a language. Voices you made come first. When the voice being used is not in the chosen
 * language, a "Choose a voice" line is shown selected instead of silently switching voices.
 */
export function readerVoiceOptionsHTML(voices, language, currentId) {
  const shown = filterVoices(voices, { language })
  const option = voice => `<option value="${escapeHTML(voice.id)}">${escapeHTML(voice.name)}${language === 'all' && voice.language === 'dutch' ? ' (Dutch)' : ''}</option>`
  const mine = shown.filter(voice => voice.custom), builtIn = shown.filter(voice => !voice.custom)
  const groups = (mine.length ? `<optgroup label="Your voices">${mine.map(option).join('')}</optgroup>` : '') +
    (builtIn.length ? `<optgroup label="${mine.length ? 'Built in' : 'Voices'}">${builtIn.map(option).join('')}</optgroup>` : '')
  const placeholder = shown.some(voice => voice.id === currentId) ? '' : '<option value="" selected disabled>Choose a voice</option>'
  return placeholder + groups
}
