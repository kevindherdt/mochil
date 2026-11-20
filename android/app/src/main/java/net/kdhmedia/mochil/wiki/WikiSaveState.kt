package net.kdhmedia.mochil.wiki

/** Apply the saved baseline without discarding text typed while the request was in flight. */
internal fun WikiState.withSavedWikiDraft(submitted: WikiState, refreshed: WikiPage): WikiState {
    if (libraryId != submitted.libraryId || page?.id != submitted.page?.id || selectedPath != submitted.selectedPath) return this
    val newerEdits = draft != submitted.draft
    return copy(
        page = refreshed,
        draft = if (newerEdits) draft else refreshed.content,
        saving = false,
        editing = newerEdits,
        draftRecovered = false,
    )
}
