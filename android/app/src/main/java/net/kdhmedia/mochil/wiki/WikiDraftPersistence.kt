package net.kdhmedia.mochil.wiki

import kotlinx.coroutines.CancellationException

internal suspend fun persistWikiDraft(write: suspend () -> Unit): Boolean = try {
    write()
    true
} catch (cancelled: CancellationException) {
    throw cancelled
} catch (_: Exception) {
    false
}

internal fun WikiState.withDraftPersistenceFailure(submitted: WikiState): WikiState =
    if (libraryId == submitted.libraryId && selectedPath == submitted.selectedPath &&
        page?.id == submitted.page?.id && editing && draft == submitted.draft) {
        copy(error = "Could not keep a recovery copy on this device. Your text is still here; save the page before closing it.")
    } else this
