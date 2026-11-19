package net.kdhmedia.mochil.wiki

/** Structural changes must not discard an active edit or its recovery copy. */
internal fun WikiState.pageMutationWarning(): String? =
    if (editing && draft != page?.content.orEmpty()) {
        "Save or cancel your current edits before creating, moving, or removing a page."
    } else null

/** Ignore queued editor events once a non-editing mutation has taken ownership. */
internal fun WikiState.acceptsDraftChange(): Boolean = editing || !saving

internal fun WikiState.withTrashStarted(): WikiState = copy(saving = true, editing = false, error = null)
