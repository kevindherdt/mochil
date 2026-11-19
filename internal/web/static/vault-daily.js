// Daily notes and templates: where today's note lives and how {{placeholders}} fill in.
import { isVaultPage } from './vault-pages.js?v=1'

export const DAILY_FOLDER = 'Daily'
const pad = value => String(value).padStart(2, '0')
const DAYS = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']
const MONTHS = ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December']

/** Fills YYYY, MM, MMMM, DD, dddd, HH and mm; anything else in the pattern stays as written. */
export function formatDate(date, pattern = 'YYYY-MM-DD') {
  const parts = {
    YYYY: String(date.getFullYear()), MMMM: MONTHS[date.getMonth()], MM: pad(date.getMonth() + 1),
    DD: pad(date.getDate()), dddd: DAYS[date.getDay()], HH: pad(date.getHours()), mm: pad(date.getMinutes()),
  }
  return pattern.replace(/YYYY|MMMM|MM|DD|dddd|HH|mm/g, token => parts[token])
}

/** {{date}}, {{date:dddd, MMMM DD}}, {{time}}, {{time:HH:mm}} and {{title}}. */
export function expandTemplate(text, { date = new Date(), title = '' } = {}) {
  return String(text).replace(/\{\{\s*(date|time|title)\s*(?::([^}]*))?\}\}/g, (_all, name, format) => {
    if (name === 'title') return title
    return formatDate(date, (format || (name === 'date' ? 'YYYY-MM-DD' : 'HH:mm')).trim())
  })
}

export const dailyPath = (date = new Date()) => `${DAILY_FOLDER}/${formatDate(date)}.md`
export const defaultDaily = date => `# ${formatDate(date, 'dddd, MMMM DD, YYYY')}\n\n`

/** Pages in a "Templates" folder at the top of the vault. */
export const templatePages = entries => entries.filter(entry => isVaultPage(entry) && /^templates\//i.test(entry.path || ''))

/** The page used for new daily notes: Templates/Daily.md (or "Daily note.md"), if there is one. */
export const dailyTemplate = entries => templatePages(entries).find(entry => /^daily( note)?$/i.test(entry.title))
