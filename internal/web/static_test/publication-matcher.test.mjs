import assert from 'node:assert/strict'
import test from 'node:test'
import { publicationSuggestionMarkup } from '../static/publication-matcher.js'

const escapeHTML = value => String(value).replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;').replaceAll('"', '&quot;')
const result = { title: '<Book>', authors: ['Writer'], provider: 'Hardcover', provider_id: '1', source_url: 'https://hardcover.app/books/book' }

test('ambiguous matches remain inspect-only instead of changing a volume field', () => {
  const markup = publicationSuggestionMarkup(result, 0, { code: 'duplicate_series_index' }, escapeHTML)
  assert.match(markup, /&lt;Book&gt;/)
  assert.doesNotMatch(markup, /data-suggestion=/)
  assert.match(markup, />Source</)
})

test('missing title matches retain their explicit field action', () => {
  const markup = publicationSuggestionMarkup(result, 0, { code: 'missing_title' }, escapeHTML)
  assert.match(markup, /data-suggestion="0"/)
  assert.match(markup, />Use title</)
})

test('Open Library covers use the same reviewed same-origin cover flow', () => {
  const markup = publicationSuggestionMarkup({
    title: 'A Book', authors: ['Writer'], provider: 'Open Library', provider_id: 'OL1W',
    source_url: 'https://openlibrary.org/works/OL1W',
    cover_url: 'https://covers.openlibrary.org/b/id/123-M.jpg?default=false',
  }, 0, { code: 'missing_author' }, escapeHTML)
  assert.match(markup, /src="\/api\/metadata\/suggestion-artwork\?url=/)
  assert.match(markup, /data-cover-action="save"/)
  assert.doesNotMatch(markup, /src="https:\/\/covers\.openlibrary\.org/)
})
