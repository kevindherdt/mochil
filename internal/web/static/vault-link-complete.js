// [[ suggestions for the note editor. Plain functions: the editor feeds them to CodeMirror.
import { rankPages, pageName } from './vault-find.js?v=1'

/** Suggestions for what has been typed after "[[". Names that are not unique use their folder path. */
export function wikiLinkMatches(pages, query, limit = 8) {
  const counts = new Map()
  for (const entry of pages) counts.set(String(entry.title).toLowerCase(), (counts.get(String(entry.title).toLowerCase()) || 0) + 1)
  return rankPages(pages, query, limit).map(entry => {
    const unique = counts.get(String(entry.title).toLowerCase()) === 1
    const name = unique ? entry.title : pageName(entry)
    return { label: entry.title, detail: pageName(entry).split('/').slice(0, -1).join('/'), name }
  })
}

export function wikiLinkSource(getPages) {
  return context => {
    const before = context.matchBefore(/\[\[[^\][|#\n]*/)
    if (!before) return null
    const options = wikiLinkMatches(getPages(), before.text.slice(2)).map(match => ({
      label: match.label,
      detail: match.detail,
      apply: (view, _completion, from, to) => {
        const closed = view.state.sliceDoc(to, to + 2) === ']]'
        view.dispatch({
          changes: { from, to, insert: closed ? match.name : `${match.name}]]` },
          selection: { anchor: from + match.name.length + 2 },
          userEvent: 'input.complete',
        })
      },
    }))
    return options.length ? { from: before.from + 2, options, filter: false } : null
  }
}
