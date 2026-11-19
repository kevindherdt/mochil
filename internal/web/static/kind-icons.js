// One line icon per kind of library (24px grid); its colour comes from kind-colors.css. A book is the fallback.
// Home, Settings and the editor are menu destinations rather than libraries, and have icons here too.
const ICONS = {
  books: '<path d="M12 6.5C10.3 5.2 8 4.8 4 5v13c4-.2 6.3.2 8 1.5 1.7-1.3 4-1.7 8-1.5V5c-4-.2-6.3.2-8 1.5ZM12 6.5v13"/>',
  lightnovels: '<path d="M7 4h10v16l-5-3-5 3V4Z"/>',
  manga: '<path d="M4 4h7v7H4zM13 4h7v4h-7zM13 10h7v10h-7zM4 13h7v7H4z"/>',
  comics: '<path d="M4 5h16v11H9l-5 4V5Z"/>',
  movies: '<path d="M4 5h16v14H4zM8 5v14M16 5v14M4 9h4M4 15h4M16 9h4M16 15h4"/>',
  tv: '<path d="M4 7h16v11H4zM9 3l3 4 3-4"/>',
  anime: '<path d="m12 3 2.4 6.6L21 12l-6.6 2.4L12 21l-2.4-6.6L3 12l6.6-2.4z"/>',
  music: '<path d="M9 18V6l10-2v12M9 18a3 3 0 1 1-3-3M19 16a3 3 0 1 1-3-3"/>',
  audiobooks: '<path d="M4 14v-2a8 8 0 0 1 16 0v2M4 14h4v6H4zM16 14h4v6h-4z"/>',
  home: '<path d="M4 11 12 4l8 7M6 10v10h12V10"/>',
  settings: '<path d="M4 7h7m4 0h5M4 17h5m4 0h7"/><circle cx="13" cy="7" r="2"/><circle cx="11" cy="17" r="2"/>',
  hermes: '<path d="M5 5h14a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2h-8l-5 4v-4H5a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2Z"/>',
  editor: '<path d="M4 20h4L19 9l-4-4L4 16zM13 7l4 4"/>',
  podcasts: '<path d="M12 3a3 3 0 0 1 3 3v5a3 3 0 0 1-6 0V6a3 3 0 0 1 3-3ZM6 11a6 6 0 0 0 12 0M12 17v4"/>',
}
const ALIASES = { 'light-novels': 'lightnovels', novels: 'lightnovels', video: 'anime', mixed: 'books', calibre: 'books' }
export const kindIconSVG = kind => `<svg class="kind-icon" viewBox="0 0 24 24" aria-hidden="true">${ICONS[ALIASES[kind] || kind] || ICONS.books}</svg>`
