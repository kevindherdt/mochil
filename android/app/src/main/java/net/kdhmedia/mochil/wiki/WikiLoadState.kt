package net.kdhmedia.mochil.wiki

/** A network load may finish after a cached page has become an active draft. */
internal fun WikiState.withWikiLoadedState(submitted: WikiState, loaded: WikiState): WikiState {
    if (libraryId != submitted.libraryId || selectedPath != submitted.selectedPath) return this
    val preserve = page != null && (editing || saving || page != submitted.page || draft != submitted.draft)
    if (!preserve) return loaded
    return loaded.copy(
        page = page,
        selectedPath = selectedPath,
        draft = draft,
        draftRecovered = draftRecovered,
        editing = editing,
        saving = saving,
        offlineCopy = offlineCopy,
    )
}
