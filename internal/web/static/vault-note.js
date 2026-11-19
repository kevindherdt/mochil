// The page chrome of a wiki note: where it lives, what you can do with it, and
// an outline of its headings. Presentation only; the note's text is rendered elsewhere.
const download = '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 4v11m-5-5 5 5 5-5M5 20h14"/></svg>'
const edit = '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m4 20 1-4L16 5l3 3L8 19zM14 7l3 3"/></svg>'

export function vaultNoteHTML({ item, libraryName, content, canEdit, escapeHTML }) {
  const folders = String(item.path || '').split('/').filter(Boolean).slice(0, -1)
  const trail = [libraryName || 'Vault', ...folders]
  const crumbs = trail.map(part => `<span>${escapeHTML(part)}</span>`).join('<i aria-hidden="true">/</i>')
  const actions = `${canEdit ? `<button type="button" data-vault-edit>${edit}<span>Edit</span></button>` : ''}<a href="${escapeHTML(item.download_url)}" download aria-label="Download ${escapeHTML(item.title)}" title="Download">${download}</a>`
  return `<header class="vault-notebar"><div class="vault-notebar-column"><nav class="vault-crumbs" aria-label="Location">${crumbs}</nav><div class="vault-page-actions">${actions}</div></div></header><h1 class="vault-title">${escapeHTML(item.title)}</h1><article>${content}</article>`
}

/** Adds an "On this page" list when the note has enough structure to be worth navigating. */
export function mountVaultOutline(pane, { document: doc = pane.ownerDocument } = {}) {
  pane.querySelector('.vault-outline')?.remove()
  const headings = [...pane.querySelectorAll('article h2, article h3, article h4')].filter(heading => heading.textContent.trim())
  if (headings.length < 3) return null
  const outline = doc.createElement('nav')
  outline.className = 'vault-outline'
  outline.setAttribute('aria-label', 'On this page')
  const title = doc.createElement('strong')
  title.textContent = 'On this page'
  outline.append(title)
  const top = Math.min(...headings.map(heading => Number(heading.tagName[1])))
  for (const heading of headings) {
    const link = doc.createElement('button')
    link.type = 'button'
    link.textContent = heading.textContent.trim()
    link.style.setProperty('--depth', String(Number(heading.tagName[1]) - top))
    link.addEventListener('click', () => {
      heading.setAttribute('tabindex', '-1')
      heading.scrollIntoView({ block: 'start', behavior: 'smooth' })
      heading.focus({ preventScroll: true })
    })
    outline.append(link)
  }
  pane.append(outline)
  return outline
}

/** A note that opens with its own title heading would show the title twice. */
export function dropRepeatedTitle(pane, title) {
  // Properties sit above the title in the file, so skip past them.
  const article = pane.querySelector('article')
  let first = article?.firstElementChild
  if (first?.classList?.contains('vault-properties')) first = first.nextElementSibling
  const same = value => String(value).normalize('NFC').trim().replace(/\s+/g, ' ').toLowerCase()
  if (first?.tagName === 'H1' && same(first.textContent) === same(title)) first.remove()
}
