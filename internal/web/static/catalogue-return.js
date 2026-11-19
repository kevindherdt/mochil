// Going back from an opened series or author should land where the person was, not at the top of a freshly loaded first page.
// remember() is taken before the list is replaced; restore() reloads the list, fetches pages until as many are shown as before, then scrolls.
export function createListReturn({ count, load, loadMore, scrollTo = (y) => window.scrollTo(0, y), scrollY = () => window.scrollY }) {
  let saved = null
  return {
    remember() { saved = { shown: count(), y: scrollY() } },
    async restore() {
      const back = saved
      saved = null
      await load()
      if (!back || !back.shown) return
      while (count() < back.shown && await loadMore()) { /* keep paging */ }
      scrollTo(back.y)
    },
  }
}
