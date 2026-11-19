// Goes to a place (a contents entry or a search hit) and keeps it as the saved
// position, so reopening the book comes back to where the reader went.
export function createReaderJump({ view, readingSync, currentReaderPosition, positionStore, saveReadingPosition }) {
  return async target => {
    const relocated = new Promise(resolve => {
      const timeout = setTimeout(() => resolve(null), 1500)
      view.addEventListener('relocate', event => { clearTimeout(timeout); resolve(event) }, { once: true })
    })
    await view.goTo(target)
    await relocated
    readingSync.cancelScheduled()
    if (view.lastLocation?.fraction != null) {
      const position = currentReaderPosition()
      if (!position) return
      positionStore.save(position)
      await saveReadingPosition(position)
    }
  }
}
