// Where a tap lands: the outer edges turn pages, the middle shows or hides the
// controls. "Left" and "right" follow the reading direction via goLeft/goRight.
export function tapZone(x, width, edge = 0.28) {
  if (!(width > 0)) return 'toggle'
  const position = x / width
  if (position < edge) return 'left'
  if (position > 1 - edge) return 'right'
  return 'toggle'
}

const interactive = 'a,button,input,select,textarea,summary,label,[contenteditable="true"]'

export function bindReaderNavigation({ document, view, previousButton, nextButton, onActivity = () => {}, controlsVisible = () => false, hideControls = () => {}, isScrolled = () => false }) {
  const handlePagingKey = event => {
    if (event.defaultPrevented || event.altKey || event.ctrlKey || event.metaKey) return
    if (event.target?.matches?.('input,select,textarea,button,[contenteditable="true"]')) return
    const actions = {
      PageUp: () => view.prev(), PageDown: () => view.next(),
      ArrowLeft: () => view.goLeft(), h: () => view.goLeft(),
      ArrowRight: () => view.goRight(), l: () => view.goRight(),
    }
    const action = actions[event.key]
    if (!action) return
    event.preventDefault()
    action()
  }
  // The book lives in an iframe, so pointer activity there never reaches the page.
  let visibleAtPointerDown = false
  // Chapters are laid out in one wide, scrolled iframe, so a click's position
  // inside the book is not its position on screen: add the frame's own offset.
  const handleTap = (event, offset, getSelection) => {
    if (event.defaultPrevented || event.detail > 1 || event.target?.closest?.(interactive)) return
    if (getSelection?.()?.toString()) return
    const width = view.clientWidth || view.ownerDocument?.defaultView?.innerWidth || 0
    const zone = tapZone(offset + event.clientX, width)
    if (zone === 'toggle') {
      if (visibleAtPointerDown) hideControls()
    } else if (!isScrolled()) {
      if (zone === 'left') view.goLeft()
      else view.goRight()
    }
  }
  const notePointerDown = () => { visibleAtPointerDown = controlsVisible(); onActivity() }
  const attachActivity = doc => {
    doc.addEventListener('pointermove', onActivity, { passive: true })
    doc.addEventListener('pointerdown', notePointerDown, { passive: true })
    doc.addEventListener('click', event => {
      const frame = doc.defaultView?.frameElement
      handleTap(event, frame ? frame.getBoundingClientRect().left : 0, () => doc.getSelection?.())
    })
  }
  // Side margins on wide screens belong to the reader, not the book.
  view.addEventListener('pointerdown', notePointerDown, { passive: true })
  view.addEventListener('click', event => handleTap(event, 0, null))
  const attachBookKeys = event => {
    const doc = event.detail?.doc
    if (!doc) return
    doc.addEventListener('keydown', handlePagingKey)
    attachActivity(doc)
  }
  document.addEventListener('keydown', handlePagingKey)
  view.addEventListener('load', attachBookKeys)
  for (const { doc } of view.renderer?.getContents?.() || []) {
    doc.addEventListener('keydown', handlePagingKey)
    attachActivity(doc)
  }
  previousButton.onclick = () => view.prev()
  nextButton.onclick = () => view.next()
}
