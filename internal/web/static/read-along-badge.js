// A book that carries its own recording says so wherever books are shown: on a cover, or as a label.
export const headphones = '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3a8 8 0 0 0-8 8v6a3 3 0 0 0 3 3h1v-7H6v-2a6 6 0 0 1 12 0v2h-2v7h1a3 3 0 0 0 3-3v-6a8 8 0 0 0-8-8z"/></svg>'

export function readAlongMarkup(book, kind) {
  if (!book?.read_along) return ''
  const label = 'Has its own recording'
  if (kind === 'cover') return `<span class="book-read-along" title="${label}" aria-label="${label}">${headphones}</span>`
  return `<small class="book-read-along-label">${headphones}Read-along</small>`
}
