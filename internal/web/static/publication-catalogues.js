import { createAuthorCatalogue } from './author-catalogue.js?v=12'
import { createBookCatalogue } from './book-catalogue.js?v=8'
import { createContinueCatalogue } from './continue-catalogue.js?v=1'
import { createBookSortBar } from './book-sort-bar.js?v=4'
import { watchBookLists } from './book-list-table.js?v=3'
import { createSeriesCatalogue } from './series-catalogue.js?v=21'

export function createPublicationCatalogues({
  $, document, escapeHTML, requestJSON, createBoundedWindow, createContinuousPager,
  cards, collections, libraryState, getBrowseMode, getSeriesLayout, getItemLayout,
  getActiveBucket, onNavigationChange, setCurrentBucket, onSortChange = () => {},
}) {
  let books, series, authors, continuing
  const sortBar = createBookSortBar({
    $, document,
    reload: (sort) => { onSortChange(sort); books.load(true) },
  })
  watchBookLists(document.querySelector('#library-view') || document.body, { document, window })
  books = createBookCatalogue({
    $, requestJSON, createBoundedWindow, createContinuousPager, cards, collections, libraryState,
    getItemLayout, getActiveBucket, getSort: () => ['recent', 'discover'].includes(getBrowseMode()) ? 'recent' : sortBar.get(),
    isActive: () => ['books', 'recent', 'discover'].includes(getBrowseMode()),
    abortSeries: () => { series?.abort(); authors?.abort(); continuing?.abort() },
    onNavigationChange, setCurrentBucket,
  })
  series = createSeriesCatalogue({
    $, document, escapeHTML, requestJSON, createBoundedWindow, createContinuousPager, cards, collections, libraryState,
    getBrowseMode, getSeriesLayout, getItemLayout, getActiveBucket,
    abortBooks: () => { books.abort(); authors?.abort(); continuing?.abort() },
  })
  authors = createAuthorCatalogue({
    $, document, escapeHTML, requestJSON, createBoundedWindow, createContinuousPager, cards, collections, libraryState,
    getBrowseMode, getItemLayout, getGroupLayout: getSeriesLayout, getActiveBucket,
    abortOthers: () => { books.abort(); series.abort(); continuing?.abort() },
  })
  continuing = createContinueCatalogue({
    $, requestJSON, cards, collections, libraryState, getBrowseMode, getItemLayout,
    abortOthers: () => { books.abort(); series.abort(); authors.abort() },
  })
  books.bind()
  series.bind()
  authors.bind()
  const abortAll = () => [books, series, authors, continuing].forEach(catalogue => catalogue.abort())
  return { books, series, authors, continuing, sortBar, abortAll }
}
