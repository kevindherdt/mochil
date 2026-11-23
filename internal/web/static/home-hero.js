// Home leads with what you are in the middle of, not with a list of rails: the design's Figure 7 puts one large
// card at the top ("Continue watching / Silo / Season 1 · Episode 4 · 28m left / Resume") and shelves after it.
import { cleanVideoTitle } from './video-title.js?v=1'

/** "24m left" out of a position and a duration, or nothing when the length is unknown. */
export function heroRemaining(progress) {
  const position = Number(progress?.position_seconds) || 0
  const duration = Number(progress?.duration_seconds) || 0
  if (duration <= 0 || position >= duration) return ''
  const left = Math.max(1, Math.round((duration - position) / 60))
  const total = Math.round(duration / 60)
  const spoken = (minutes) => (minutes >= 60 ? `${Math.floor(minutes / 60)}h ${minutes % 60}m`.replace(' 0m', '') : `${minutes}m`)
  // "108m left of 108m" is a riddle: when barely anything has played, just say the length.
  if (left >= total) return spoken(total)
  return `${spoken(left)} left of ${spoken(total)}`
}

/** The hero only exists when there is something to resume, and only ever from the newest item. */
export function heroHTML(item, { escapeHTML, watchedLabel }) {
  if (!item) return ''
  const title = cleanVideoTitle(item.title || '')
  if (!title) return ''
  const meta = [watchedLabel?.(item.progress), heroRemaining(item.progress)].filter(Boolean).join(' · ')
  const art = item.artwork_url
    ? `<img class="hero-art" src="${escapeHTML(item.artwork_url)}" alt="" decoding="async">`
    : ''
  return `<div class="hero-inner">${art}<div class="hero-copy"><p class="hero-kicker">Continue watching</p><h1 class="hero-title">${escapeHTML(title)}</h1>${meta ? `<p class="hero-meta">${escapeHTML(meta)}</p>` : ''}<button type="button" class="hero-resume" data-library="${escapeHTML(item.library_id)}" data-item="${escapeHTML(item.id)}"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m9 7 8 5-8 5V7Z"/></svg>Resume</button></div></div>`
}

/** Mounts the hero on Home and wires its one button to the same player the rail opens. */
export function mountHero({ document, escaped, watchedLabel, onOpen }) {
  function render(items) {
    // Looked up per render, not at mount: this controller is built at boot, before Home's markup exists.
    const section = document?.getElementById?.('home-hero')
    if (!section) return
    const item = Array.isArray(items) ? items[0] : items
    section.innerHTML = heroHTML(item, { escapeHTML: escaped, watchedLabel })
    section.hidden = !section.innerHTML
    const button = section.querySelector('.hero-resume')
    if (button) button.onclick = () => onOpen?.(item, item?.title)
  }
  return { render }
}
