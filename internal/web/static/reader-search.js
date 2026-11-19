// Search inside the open book. A reader engine does the matching, section by
// section; this is the dialog around it: type, press Enter, see every hit with
// its surrounding words grouped by chapter, tap one to go there.
//
// `source.search(query)` is an async iterable of { progress } (0..1), then
// { label, subitems: [{ target, excerpt: { pre, match, post } }] } per chapter with hits.
// `source.clear()` removes highlights. `jumpTo(target)` goes to a hit.
const MAX_RESULTS = 300

export function createReaderSearch({ $, source, jumpTo, document: doc = document }) {
  const dialog = $('#book-search'), form = $('#book-search-form'), input = $('#book-search-input')
  const status = $('#book-search-status'), results = $('#book-search-results')
  let run = 0

  function show(item) {
    const row = doc.createElement('button')
    row.type = 'button'
    row.className = 'search-hit'
    const { pre = '', match = '', post = '' } = item.excerpt || {}
    const mark = doc.createElement('mark')
    mark.textContent = match
    row.append(pre, mark, post)
    row.onclick = async () => {
      row.disabled = true
      try { await jumpTo(item.target); dialog.close() } finally { row.disabled = false }
    }
    return row
  }

  async function search(query) {
    const mine = ++run
    results.replaceChildren()
    status.textContent = 'Searching…'
    let count = 0, capped = false
    try {
      for await (const result of source.search(query)) {
        if (mine !== run) return // a newer search replaced this one
        if (result === 'done') break
        if (result.subitems) {
          const group = doc.createElement('section')
          const heading = doc.createElement('h3')
          heading.textContent = result.label || 'Book'
          group.append(heading)
          for (const item of result.subitems) {
            if (count >= MAX_RESULTS) { capped = true; break }
            group.append(show(item)); count++
          }
          results.append(group)
          if (capped) break
        } else if (Number.isFinite(result.progress)) {
          status.textContent = `Searching… ${Math.round(result.progress * 100)}%`
        }
      }
    } catch (error) {
      if (mine === run) status.textContent = 'Search is unavailable for this book.'
      console.error('Book search failed', error)
      return
    }
    if (mine !== run) return
    status.textContent = !count ? 'No matches.' : capped ? `First ${MAX_RESULTS} matches shown. Try a longer phrase.` : `${count.toLocaleString()} ${count === 1 ? 'match' : 'matches'}`
  }

  function open() {
    if (dialog.open) { input.focus(); input.select(); return }
    dialog.showModal()
    input.focus(); input.select()
  }

  form.addEventListener('submit', event => {
    event.preventDefault()
    const query = input.value.trim()
    if (query.length < 2) { status.textContent = 'Type at least two letters.'; return }
    search(query)
  })
  $('#close-search').onclick = () => dialog.close()
  $('#clear-search').onclick = () => { run++; source.clear(); results.replaceChildren(); status.textContent = ''; input.value = ''; input.focus() }
  $('#search-open').onclick = open

  // Ctrl/Cmd+F finds in the book instead of the browser's page search; call
  // attachDocument for the page and for each book frame so it works from inside the book too.
  const shortcut = event => {
    if ((event.ctrlKey || event.metaKey) && !event.altKey && event.key.toLowerCase() === 'f') { event.preventDefault(); open() }
  }
  const attachDocument = target => target?.addEventListener('keydown', shortcut)
  attachDocument(doc)
  return { open, search, attachDocument }
}

/** The classic reader's engine searches by itself; its hits are CFIs, which goTo understands. */
export function mountClassicSearch({ $, view, jumpTo }) {
  const search = createReaderSearch({
    $, jumpTo,
    source: {
      async * search(query) {
        for await (const result of view.search({ query })) {
          yield result.subitems ? { label: result.label, subitems: result.subitems.map(({ cfi, excerpt }) => ({ target: cfi, excerpt })) } : result
        }
      },
      clear: () => view.clearSearch(),
    },
  })
  view.addEventListener('load', event => search.attachDocument(event.detail?.doc))
  for (const { doc } of view.renderer?.getContents?.() || []) search.attachDocument(doc)
  return search
}
