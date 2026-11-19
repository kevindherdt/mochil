package net.kdhmedia.mochil.wiki

/** Rebind a moved page without losing edits made during the remote request. */
internal fun WikiState.withMovedWikiPage(submitted: WikiState, result: WikiMutationResult): WikiState {
    if (libraryId != submitted.libraryId || page?.id != submitted.page?.id || selectedPath != submitted.selectedPath) return this
    val moved = requireNotNull(result.selected)
    val refreshed = requireNotNull(result.page)
    val newerEdits = draft != submitted.draft
    return copy(
        entries = result.entries,
        page = refreshed,
        selectedPath = moved.path,
        draft = if (newerEdits) draft else refreshed.content,
        backlinks = emptyList(),
        saving = false,
        editing = newerEdits,
        draftRecovered = false,
        expanded = expanded + moved.path.substringBeforeLast('/', ""),
    )
}
