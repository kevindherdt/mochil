// Where the dock's Library and Notes buttons go. Both are routing decisions rather than view code, and keeping
// them here is what keeps app.js inside its line budget.

/** A shelf that holds notes rather than publications. They all get their own button, so Library must skip them.
    The KIND decides first: a vault whose id happens to be "obsidian" never matched the old id-only pattern, so the
    Notes button found no shelf, opened "", and the tap did nothing at all. */
const NOTES = /wiki|documents|vault|notes/i

export function isNotesShelf(shelf) {
  const id = typeof shelf === 'string' ? shelf : shelf?.id || ''
  const kind = typeof shelf === 'string' ? '' : shelf?.kind || ''
  return NOTES.test(id) || NOTES.test(kind)
}

/** The note shelves on offer, as [id, name] pairs, so the Notes button can ask instead of guessing. */
export function noteShelves(visible) {
  return (visible || []).filter((library) => isNotesShelf(library)).map((library) => [library.id, library.name])
}

/** The shelf the Notes button opens on its own: the first note shelf, or nothing if there is none. */
export function notesShelf(visible) {
  const shelves = noteShelves(visible)
  return shelves.length > 0 ? shelves[0][0] : ''
}

/**
 * The shelf the Library button opens: the one you were last in, unless that is a notes shelf -- reading notes
 * makes the vault the last shelf, and coming back to Library should not drop you in the notes again.
 */
export function libraryTarget(visible, selected) {
  const shelves = visible || []
  if (selected && !isNotesShelf(selected)) return selected
  const shelf = shelves.find((library) => !isNotesShelf(library.id))
  return shelf ? shelf.id : shelves[0]?.id || 'books'
}

/** Everything the dock needs, in one object: it is built from live getters so a shelf added later still counts. */
export function dockShelves(visible, selected) {
  return {
    getLibrary: () => libraryTarget(visible(), selected()),
    notesLibrary: () => notesShelf(visible()),
    notesShelves: () => noteShelves(visible()),
    // Membership, not a pattern on the id: the dock has to light Notes for the same shelf it will open.
    isNotes: () => noteShelves(visible()).some(([id]) => id === selected()),
  }
}
