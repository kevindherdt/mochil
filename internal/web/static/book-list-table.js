const HEAD = '<div class="book-list-head" aria-hidden="true"><span>#</span><span></span><span>Title</span><span>Creator</span><span>Series</span><span>Progress</span></div>'

/** Gives every wide book list a header row, and numbers the rows of lists that are not windowed away as you scroll. */
export function watchBookLists(root, { document, window }) {
  let queued = false
  function sync() {
    queued = false
    const lists = new Set(root.querySelectorAll('.book-list'))
    for (const head of root.querySelectorAll('.book-list-head')) if (!lists.has(head.nextElementSibling)) head.remove()
    for (const list of lists) {
      // The All list drops rows as you scroll, so its rows could be numbered wrongly; every other list is complete.
      list.classList.toggle('is-numbered', list.id !== 'books')
      if (!list.previousElementSibling?.classList.contains('book-list-head')) list.insertAdjacentHTML('beforebegin', HEAD)
      list.previousElementSibling.classList.toggle('is-numbered', list.id !== 'books')
    }
  }
  const schedule = () => { if (!queued) { queued = true; window.requestAnimationFrame(sync) } }
  new MutationObserver(schedule).observe(root, { childList: true, subtree: true, attributes: true, attributeFilter: ['class'] })
  schedule()
}
