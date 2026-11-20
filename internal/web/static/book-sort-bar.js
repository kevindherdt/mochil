const STORAGE_KEY = 'mochil-book-sort'
const OPTIONS = [['', 'Title (A–Z)'], ['author', 'Author'], ['series', 'Series'], ['recent', 'Recently added']]

export function readBookSort(storage = globalThis.localStorage) {
  try { const value = storage.getItem(STORAGE_KEY); return OPTIONS.some(([key]) => key === value) ? value : '' } catch (_) { return '' }
}

/** "Sort by" above the All books list: the orders the server can produce for books. */
export function createBookSortBar({ $, document, reload, storage = globalThis.localStorage }) {
  let bar, sort = readBookSort(storage)
  function element() {
    if (bar) return bar
    bar = document.createElement('div')
    bar.className = 'book-sort'
    bar.hidden = true
    bar.innerHTML = `<label><span>Sort by:</span><select aria-label="Sort books">${OPTIONS.map(([key, label]) => `<option value="${key}">${label}</option>`).join('')}</select></label>`
    const select = bar.querySelector('select')
    select.value = sort
    select.onchange = () => {
      sort = select.value
      try { storage.setItem(STORAGE_KEY, sort) } catch (_) {}
      reload(sort)
    }
    $('#book-catalog').before(bar)
    return bar
  }
  return { get: () => sort, show() { element().hidden = false }, hide() { if (bar) bar.hidden = true } }
}
