/** The title Home shows for a book entry: a chapter of a series reads as "Series — chapter", not as a bare "Chapter 1". */
export function dashboardReadingTitle(item) {
  const title = String(item?.title || '').trim() || 'Untitled'
  const series = String(item?.series || '').trim()
  return series && series.toLowerCase() !== title.toLowerCase() && /^chapter\s+/i.test(title) ? `${series} — ${title}` : title
}
