// Pick several books and act on them together. One way in on every device:
// the card menu's Select. Shortcuts: Ctrl/Shift-click, or press and hold on touch.
import { createBookStatus } from './book-status.js?v=2'
const HOLD_MS = 480

// 'Select all' used to mean 'everything that happens to be on screen'. On a
// shelf of 5,430 books with 120 loaded that quietly left 5,310 books out of a
// batch action, so the control now names what it takes and how many that is.
export function selectionLabel(loaded) {
  const count = Number.isFinite(Number(loaded)) ? Math.max(0, Math.trunc(Number(loaded))) : 0
  return {
    text: `Select loaded (${count})`,
    title: `Selects the ${count} ${count === 1 ? 'book' : 'books'} loaded on screen; books that are not loaded yet are not selected`,
  }
}

export function createBookSelection({ document, collections, reload }) {
  createBookStatus({ document, reload })
  const chosen = new Map()
  let bar
  let holdTimer
  let swallowClick = false
  let frame

  const keyOf = article => `${article.dataset.library}\u0000${article.dataset.book}`
  const memberOf = article => ({ type: 'book', library_id: article.dataset.library, book_id: article.dataset.book })
  const articleFrom = target => target.closest?.('.book[data-book]')
  const selecting = () => document.body.hasAttribute('data-selecting')

  function ensureBar() {
    if (bar) return bar
    bar = document.createElement('div')
    bar.className = 'selection-bar'
    bar.hidden = true
    bar.setAttribute('role', 'region')
    bar.setAttribute('aria-label', 'Selected books')
    bar.innerHTML = '<output aria-live="polite"></output><button type="button" data-act="all">Select loaded (0)</button><button type="button" class="primary" data-act="collect">Add to collection</button><button type="button" data-act="done">Done</button>'
    bar.addEventListener('click', event => {
      const action = event.target.closest('button')?.dataset.act
      if (action === 'done') clear()
      else if (action === 'all') selectAll()
      else if (action === 'collect' && chosen.size) collections.openFor([...chosen.values()])
    })
    document.body.append(bar)
    return bar
  }

  function paint() {
    frame = 0
    const articles = document.querySelectorAll('.book[data-book]')
    articles.forEach(article => {
      if (chosen.has(keyOf(article))) article.setAttribute('data-selected', '')
      else article.removeAttribute('data-selected')
    })
    const toolbar = ensureBar()
    toolbar.hidden = !selecting()
    toolbar.querySelector('output').textContent = chosen.size === 1 ? '1 selected' : `${chosen.size} selected`
    toolbar.querySelector('[data-act="collect"]').disabled = chosen.size === 0
    const all = toolbar.querySelector('[data-act="all"]')
    const label = selectionLabel(articles.length)
    all.textContent = label.text
    all.title = label.title
    all.disabled = articles.length === 0
  }
  // Pages of books are added and dropped constantly; do nothing unless someone is selecting.
  const schedulePaint = () => { if (selecting() || chosen.size) frame ||= requestAnimationFrame(paint) }

  function toggle(article) {
    const key = keyOf(article)
    if (chosen.has(key)) chosen.delete(key)
    else chosen.set(key, memberOf(article))
    paint()
  }
  function begin(article) {
    document.body.setAttribute('data-selecting', '')
    if (article) toggle(article)
    else paint()
  }
  function clear() {
    chosen.clear()
    document.body.removeAttribute('data-selecting')
    paint()
  }
  function selectAll() {
    document.querySelectorAll('.book[data-book]').forEach(article => chosen.set(keyOf(article), memberOf(article)))
    paint()
  }

  document.addEventListener('click', event => {
    if (swallowClick) { swallowClick = false; event.preventDefault(); event.stopPropagation(); return }
    const start = event.target.closest?.('.book-select')
    if (start) {
      event.preventDefault(); event.stopPropagation()
      start.closest('details')?.removeAttribute('open')
      begin(articleFrom(start))
      return
    }
    const article = articleFrom(event.target)
    if (!article || event.target.closest('.book-more, .collection-save')) return
    if (selecting() || event.ctrlKey || event.metaKey || event.shiftKey) {
      event.preventDefault(); event.stopPropagation()
      if (!selecting()) document.body.setAttribute('data-selecting', '')
      toggle(article)
    }
  }, true)

  // Press and hold on touch screens.
  document.addEventListener('pointerdown', event => {
    if (event.pointerType === 'mouse' || event.target.closest('.book-more')) return
    const article = articleFrom(event.target)
    if (!article) return
    clearTimeout(holdTimer)
    holdTimer = setTimeout(() => {
      swallowClick = true
      if (!selecting()) document.body.setAttribute('data-selecting', '')
      if (!chosen.has(keyOf(article))) toggle(article)
      navigator.vibrate?.(12)
    }, HOLD_MS)
  })
  for (const type of ['pointerup', 'pointercancel', 'pointermove', 'scroll']) document.addEventListener(type, () => clearTimeout(holdTimer), { passive: true, capture: true })
  document.addEventListener('contextmenu', event => { if (articleFrom(event.target) && swallowClick) event.preventDefault() })
  document.addEventListener('keydown', event => { if (event.key === 'Escape' && selecting() && !document.querySelector('dialog[open]')) clear() })

  // Pages of books are added and dropped as people scroll; keep the marks right.
  new MutationObserver(schedulePaint).observe(document.querySelector('main') || document.body, { childList: true, subtree: true })
  return { clear, count: () => chosen.size }
}
