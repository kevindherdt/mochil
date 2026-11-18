package net.kdhmedia.mochil.wiki

import kotlinx.coroutines.CoroutineDispatcher
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.auth.DeviceCredential

/**
 * Owns compound remote vault operations. The ViewModel remains responsible for
 * request ownership and presentation state; this repository keeps dependent
 * network steps on one IO boundary and returns complete, immutable results.
 */
internal class WikiRepository(
    private val client: WikiRemote,
    private val io: CoroutineDispatcher = Dispatchers.IO,
) {
    private val saveRecovery = WikiSaveRecovery()
    private val structureRecovery = WikiStructureRecovery()

    private fun structureOwner(credential: DeviceCredential, libraryId: String) =
        listOf(credential.serverUrl, credential.deviceId, libraryId)

    private fun structurePageKey(credential: DeviceCredential, libraryId: String, pageId: String) =
        structureOwner(credential, libraryId) + listOf("page", pageId)

    private fun readTree(credential: DeviceCredential, libraryId: String): List<WikiEntry> {
        val tickets = structureRecovery.tickets(structureOwner(credential, libraryId))
        return client.tree(credential, libraryId).also { structureRecovery.refreshed(tickets, it) }
    }

    private fun recoveryKey(credential: DeviceCredential, libraryId: String, pageId: String) =
        listOf(credential.serverUrl, credential.deviceId, libraryId, pageId)

    private fun readPage(credential: DeviceCredential, libraryId: String, pageId: String): WikiPage {
        val key = recoveryKey(credential, libraryId, pageId)
        val ticket = saveRecovery.ticket(key)
        return client.page(credential, libraryId, pageId).also { saveRecovery.refreshed(key, ticket) }
    }
    suspend fun open(credential: DeviceCredential, libraryId: String, selectedPath: String): WikiOpenResult = withContext(io) {
        val entries = readTree(credential, libraryId)
        currentCoroutineContext().ensureActive()
        val selected = entries.firstOrNull { !it.isFolder && it.path == selectedPath }
        WikiOpenResult(entries, selected, selected?.let { readPage(credential, libraryId, it.id) })
    }

    suspend fun tree(credential: DeviceCredential, libraryId: String): List<WikiEntry> = withContext(io) {
        readTree(credential, libraryId)
    }

    suspend fun refresh(
        credential: DeviceCredential,
        libraryId: String,
        selectedPageId: String?,
        reloadPage: Boolean,
    ): WikiRefreshResult = withContext(io) {
        val entries = readTree(credential, libraryId)
        currentCoroutineContext().ensureActive()
        val selected = selectedPageId?.let { id -> entries.firstOrNull { !it.isFolder && it.id == id } }
        val page = selected?.takeIf { reloadPage }?.let { readPage(credential, libraryId, it.id) }
        WikiRefreshResult(entries, selectedPageId == null || selected != null, page)
    }

    suspend fun page(credential: DeviceCredential, libraryId: String, pageId: String): WikiPage = withContext(io) {
        readPage(credential, libraryId, pageId)
    }

    suspend fun backlinks(credential: DeviceCredential, libraryId: String, pageId: String): List<WikiEntry> = withContext(io) {
        client.backlinks(credential, libraryId, pageId)
    }

    suspend fun search(credential: DeviceCredential, libraryId: String, query: String): List<WikiEntry> = withContext(io) {
        client.search(credential, libraryId, query)
    }

    suspend fun save(credential: DeviceCredential, libraryId: String, page: WikiPage, content: String): WikiPage = withContext(io) {
        val key = recoveryKey(credential, libraryId, page.id)
        saveRecovery.requireFresh(key)
        structureRecovery.requireFresh(structurePageKey(credential, libraryId, page.id))
        try {
            client.save(credential, libraryId, page, content)
        } catch (failure: WikiException) {
            if (!failure.mutationCommitted) throw failure
            // Preserve the same acknowledged-save semantics as a failed follow-up GET.
            // In particular, task toggles must not roll back a committed change.
            throw WikiSavedButReloadFailedException(failure).also { saveRecovery.record(key, it) }
        }
        try {
            currentCoroutineContext().ensureActive()
            readPage(credential, libraryId, page.id)
        } catch (cancelled: CancellationException) {
            saveRecovery.record(key, WikiSavedButReloadFailedException(cancelled))
            throw cancelled
        } catch (failure: Exception) {
            throw WikiSavedButReloadFailedException(failure).also { saveRecovery.record(key, it) }
        }
    }

    suspend fun create(
        credential: DeviceCredential,
        libraryId: String,
        path: String,
        folder: Boolean,
    ): WikiMutationResult = withContext(io) {
        structureRecovery.mutate(
            structureOwner(credential, libraryId) + listOf("create", path),
            WikiStructureChange(targetPath = path, folder = folder),
            if (folder) "The folder was created" else "The page was created",
            write = { if (folder) client.createFolder(credential, libraryId, path) else client.createPage(credential, libraryId, path) },
        ) {
            val entries = readTree(credential, libraryId)
            currentCoroutineContext().ensureActive()
            if (folder && entries.none { it.isFolder && it.path == path }) {
                error("The created folder was not indexed.")
            }
            val selected = if (folder) null else entries.firstOrNull { !it.isFolder && it.path == path }
                ?: error("The created page was not indexed.")
            WikiMutationResult(entries, selected, selected?.let { readPage(credential, libraryId, it.id) })
        }
    }

    suspend fun move(
        credential: DeviceCredential,
        libraryId: String,
        page: WikiPage,
        path: String,
    ): WikiMutationResult = withContext(io) {
        saveRecovery.requireFresh(recoveryKey(credential, libraryId, page.id))
        structureRecovery.mutate(
            structurePageKey(credential, libraryId, page.id),
            WikiStructureChange(targetPath = path, removedId = page.id),
            "The page was moved", write = { client.movePage(credential, libraryId, page, path) },
        ) {
            val entries = readTree(credential, libraryId)
            currentCoroutineContext().ensureActive()
            val selected = entries.firstOrNull { !it.isFolder && it.path == path }
                ?: error("The moved page was not indexed.")
            if (entries.any { it.id == page.id }) error("The original page is still in the library index.")
            WikiMutationResult(entries, selected, readPage(credential, libraryId, selected.id))
        }
    }

    suspend fun trash(credential: DeviceCredential, libraryId: String, page: WikiPage): List<WikiEntry> = withContext(io) {
        saveRecovery.requireFresh(recoveryKey(credential, libraryId, page.id))
        structureRecovery.mutate(
            structurePageKey(credential, libraryId, page.id),
            WikiStructureChange(removedId = page.id),
            "The page was moved to vault trash", write = { client.trashPage(credential, libraryId, page) },
        ) {
            readTree(credential, libraryId).also { entries ->
                if (entries.any { it.id == page.id }) error("The trashed page is still in the library index.")
            }
        }
    }
}

internal data class WikiOpenResult(
    val entries: List<WikiEntry>,
    val selected: WikiEntry?,
    val page: WikiPage?,
)

internal data class WikiMutationResult(
    val entries: List<WikiEntry>,
    val selected: WikiEntry?,
    val page: WikiPage?,
)

internal data class WikiRefreshResult(
    val entries: List<WikiEntry>,
    val selectedStillExists: Boolean,
    val page: WikiPage?,
)

internal class WikiSavedButReloadFailedException(cause: Throwable) : Exception(
    "The page was saved, but its latest revision could not be reloaded. Refresh it before editing again.",
    cause,
)
