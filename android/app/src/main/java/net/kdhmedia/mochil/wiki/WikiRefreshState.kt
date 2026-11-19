package net.kdhmedia.mochil.wiki

internal fun WikiState.withWikiRefresh(submitted: WikiState, result: WikiRefreshResult): WikiState {
    if (libraryId != submitted.libraryId || selectedPath != submitted.selectedPath || page?.id != submitted.page?.id) return this
    // Recheck at completion: editing or saving may have begun after refresh started.
    val preserve = editing || saving || page != submitted.page || draft != submitted.draft
    val keepSelection = preserve || result.selectedStillExists
    return copy(
        entries = result.entries,
        page = if (preserve) page else if (result.selectedStillExists) result.page ?: page else null,
        selectedPath = if (keepSelection) selectedPath else "",
        draft = if (preserve) draft else result.page?.content ?: if (result.selectedStillExists) draft else "",
        backlinks = if (preserve || (result.page == null && result.selectedStillExists)) backlinks else emptyList(),
        editing = editing && keepSelection,
        expanded = expanded.intersect(result.entries.filter(WikiEntry::isFolder).mapTo(mutableSetOf(), WikiEntry::path)),
        loading = false,
        offlineCopy = if (preserve) offlineCopy else false,
        error = if (preserve && !result.selectedStillExists) "This page is no longer in the vault. Your draft has been kept." else error,
    )
}
