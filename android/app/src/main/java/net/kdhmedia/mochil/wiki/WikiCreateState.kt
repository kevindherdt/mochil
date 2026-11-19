package net.kdhmedia.mochil.wiki

/** Creating a folder or a late page response must not replace an active draft. */
internal fun WikiState.withCreatedWikiEntry(
    submitted: WikiState,
    result: WikiMutationResult,
    requestedPath: String,
): WikiState {
    if (libraryId != submitted.libraryId || page?.id != submitted.page?.id || selectedPath != submitted.selectedPath) return this
    val openCreated = result.page != null && result.selected != null && draft == submitted.draft
    return copy(
        entries = result.entries,
        page = if (openCreated) result.page else page,
        selectedPath = if (openCreated) result.selected.path else selectedPath,
        draft = if (openCreated) result.page.content else draft,
        backlinks = if (openCreated) emptyList() else backlinks,
        saving = false,
        editing = if (openCreated) true else editing,
        expanded = expanded + requestedPath.substringBeforeLast('/', ""),
    )
}
