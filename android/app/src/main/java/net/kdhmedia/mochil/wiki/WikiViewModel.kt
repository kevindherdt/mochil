package net.kdhmedia.mochil.wiki

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.auth.DeviceCredentialStore
import net.kdhmedia.mochil.auth.DeviceCredential

class WikiViewModel(application: Application, private val savedState: SavedStateHandle) : AndroidViewModel(application) {
    private val credential = DeviceCredentialStore(application).load()
    private val client = WikiClient()
    private val repository = WikiRepository(client)
    private val draftStore = WikiDraftStore(application)
    private val offlineStore = WikiOfflineStore(application)
    private val offlineAssets = WikiOfflineAssetStore(application)
    private val offlineMirror = WikiOfflineMirror(client, offlineStore, offlineAssets)
    private val mutableState = MutableStateFlow(WikiState())
    val state = mutableState.asStateFlow()
    private var treeLoadJob: Job? = null
    private var pageLoadJob: Job? = null
    private var backlinkLoadJob: Job? = null
    private var searchJob: Job? = null
    private var draftSaveJob: Job? = null
    private var offlineMirrorJob: Job? = null
    private val pageSession = WikiPageSession()
    private val offlineSession = WikiPageSession()
    private val pdfOpening = createWikiPdfOpening(application, viewModelScope,
        status = { message -> mutate { copy(attachmentStatus = message) } },
        failure = { message -> mutate { copy(error = message) } })
    fun cancelAttachment() = pdfOpening.cancel()

    fun open(libraryId: String) {
        cancelAttachment()
        if (libraryId == mutableState.value.libraryId && mutableState.value.entries.isNotEmpty() && !mutableState.value.offlineCopy) return
        pageSession.reset()
        offlineSession.reset()
        val session = pageSession.capture()
        treeLoadJob?.cancel()
        pageLoadJob?.cancel()
        backlinkLoadJob?.cancel()
        searchJob?.cancel()
        offlineMirrorJob?.cancel()
        val rememberedPath = savedState.get<String>(LIBRARY_ID).takeIf { it == libraryId }?.let { savedState.get<String>(SELECTED_PATH) }.orEmpty()
        savedState[LIBRARY_ID] = libraryId
        mutate { WikiState(libraryId = libraryId, selectedPath = rememberedPath, loading = true) }
        val saved = credential ?: return mutate { copy(loading = false, error = "This device is no longer connected.") }
        treeLoadJob = viewModelScope.launch {
            val cached = withContext(Dispatchers.IO) { offlineStore.load(saved.deviceId, libraryId) }
            val cachedAssetUrls = cached?.let { withContext(Dispatchers.IO) { offlineAssets.cachedUrls(saved.deviceId, libraryId, it.entries) } }.orEmpty()
            if (!pageSession.isCurrent(session)) return@launch
            if (cached != null && mutableState.value.libraryId == libraryId) mutate {
                val selectedPage = cached.selectedPage()?.page
                copy(
                    entries = cached.entries,
                    page = selectedPage,
                    selectedPath = cached.selectedPath,
                    draft = selectedPage?.content.orEmpty(),
                    expanded = cached.entries.filter(WikiEntry::isFolder).filter { it.depth == 0 }.map(WikiEntry::path).toSet(),
                    offlineCopy = true,
                    offlineAssetUrls = cachedAssetUrls,
                )
            }
            val targetPath = rememberedPath.ifBlank { cached?.selectedPath.orEmpty() }
            val submitted = mutableState.value
            runCatching { repository.open(saved, libraryId, targetPath) }
            .onSuccess { (entries, target, page) ->
                if (!pageSession.isCurrent(session)) return@onSuccess
                val recovery = page?.let { withContext(Dispatchers.IO) { draftStore.load(saved.deviceId).retainForInitialSelection(null, libraryId, it.id)?.recoverFor(libraryId, it) } }
                val assetUrls = withContext(Dispatchers.IO) { offlineAssets.cachedUrls(saved.deviceId, libraryId, entries) }
                if (!pageSession.isCurrent(session)) return@onSuccess
                mutate {
                    if (this.libraryId != libraryId) return@mutate this
                    withWikiLoadedState(submitted, copy(
                        entries = entries,
                        page = recovery?.let { page.copy(revision = it.revision) } ?: page,
                        selectedPath = target?.path.orEmpty(),
                        draft = recovery?.content ?: page?.content.orEmpty(),
                        draftRecovered = recovery != null,
                        editing = recovery != null,
                        expanded = entries.filter(WikiEntry::isFolder).filter { it.depth == 0 }.map(WikiEntry::path).toSet(),
                        loading = false,
                        offlineCopy = false,
                        offlineAssetUrls = assetUrls,
                        error = null,
                    ))
                }
                val savedAt = System.currentTimeMillis()
                withContext(Dispatchers.IO) {
                    offlineStore.saveBestEffort(saved.deviceId, WikiOfflineSnapshot(
                        libraryId, entries, target?.path.orEmpty(),
                        page?.let { listOf(WikiOfflinePage(target?.path.orEmpty(), it, savedAt)) }.orEmpty(), savedAt,
                    ))
                }
                if (pageSession.isCurrent(session)) page?.let { loadBacklinks(saved, libraryId, it.id) }
            }
            .onFailure { failure ->
                if (pageSession.isCurrent(session) && failure !is CancellationException) mutate {
                    if (libraryId != this.libraryId) this
                    else if (offlineCopy && entries.isNotEmpty()) copy(loading = false, error = null)
                    else copy(loading = false, error = failure.message ?: "Could not load this wiki.")
                }
            }
        }
    }

    fun refresh() {
        val snapshot = mutableState.value
        val session = pageSession.capture()
        val saved = credential ?: return
        if (snapshot.libraryId.isBlank()) return
        val dirty = snapshot.editing && snapshot.draft != snapshot.page?.content.orEmpty()
        treeLoadJob?.cancel()
        if (!dirty) pageLoadJob?.cancel()
        backlinkLoadJob?.cancel()
        mutate { copy(loading = true, error = null) }
        treeLoadJob = viewModelScope.launch {
            runCatching {
                repository.refresh(saved, snapshot.libraryId, snapshot.page?.id, reloadPage = !dirty)
            }.onSuccess { result ->
                if (!pageSession.isCurrent(session)) return@onSuccess
                mutate { withWikiRefresh(snapshot, result) }
                cacheCurrentPage()
                mutableState.value.page?.let { loadBacklinks(saved, snapshot.libraryId, it.id) }
            }.onFailure { failure ->
                if (pageSession.isCurrent(session) && failure !is CancellationException) mutate { copy(loading = false, error = failure.message ?: "Could not refresh this vault.") }
            }
        }
    }

    fun toggle(path: String) = mutate { copy(expanded = if (path in expanded) expanded - path else expanded + path) }
    fun search(query: String) {
        val normalized = query.trim().take(200)
        searchJob?.cancel()
        if (normalized.isBlank()) return mutate { copy(searchQuery = "", searchResults = emptyList(), searching = false) }
        val saved = credential ?: return
        val libraryId = mutableState.value.libraryId
        mutate { copy(searchQuery = normalized, searchResults = emptyList(), searching = true, error = null) }
        searchJob = viewModelScope.launch {
            delay(250)
            runCatching { repository.search(saved, libraryId, normalized) }
                .onSuccess { results -> mutate {
                    if (this.libraryId == libraryId && searchQuery == normalized) copy(searchResults = results, searching = false) else this
                } }
                .onFailure { failure -> if (failure !is CancellationException) mutate {
                    withWikiSearchFailure(libraryId, normalized, failure.message ?: "Could not search this vault.")
                } }
        }
    }
    fun assetUrl(entry: WikiEntry): String = mutableState.value.offlineAssetUrls[entry.id]
        ?: credential?.let { client.assetUrl(it, mutableState.value.libraryId, entry) }.orEmpty()
    fun openLink(target: String) {
        if (pdfOpening.openLink(target, mutableState.value, credential)) return
        cancelAttachment()
        val destination = parseWikiLinkDestination(target)
        if (destination.page.isBlank()) {
            if (mutableState.value.page != null && destination.heading.isNotBlank()) {
                mutate { copy(navigationHeading = destination.heading) }
            }
            return
        }
        val entry = findWikiLinkTarget(mutableState.value.entries, destination, mutableState.value.selectedPath)
            ?: return missingWikiLinkTarget(mutableState.value.selectedPath, target)?.let { missing ->
                if (mutableState.value.page?.writable == true && !mutableState.value.offlineCopy) {
                    mutate { copy(missingLink = missing, error = null) }
                } else {
                    mutate { copy(error = "No page named “${target.take(80)}” was found in this vault.") }
                }
            } ?: mutate { copy(error = "This Wiki link does not contain a safe page path.") }
        if (entry.id == mutableState.value.page?.id) {
            mutate { copy(navigationHeading = destination.heading) }
        } else select(entry, destination.heading)
    }
    fun select(entry: WikiEntry) = select(entry, "")

    private fun select(entry: WikiEntry, navigationHeading: String) {
        cancelAttachment()
        if (entry.isFolder) return
        val saved = credential ?: return
        pageSession.reset()
        val session = pageSession.capture()
        val libraryId = mutableState.value.libraryId
        // Only automatic startup selection may claim a process-death draft.
        val mayRecoverDraft = mutableState.value.page == null
        draftSaveJob?.cancel()
        if (!mayRecoverDraft) draftStore.clear()
        pageLoadJob?.cancel()
        backlinkLoadJob?.cancel()
        savedState[SELECTED_PATH] = entry.path
        mutate { copy(selectedPath = entry.path, page = null, draft = "", draftRecovered = false, offlineCopy = false,
            loading = true, saving = false, error = null, editing = false, backlinks = emptyList(), navigationHeading = navigationHeading) }
        pageLoadJob = viewModelScope.launch {
            val cached = withContext(Dispatchers.IO) { offlineStore.load(saved.deviceId, libraryId)?.page(entry.id) }
            if (!pageSession.isCurrent(session)) return@launch
            mutate { withOfflinePageIfCurrent(libraryId, entry.path, cached) }
            val recoveryCandidate = if (mayRecoverDraft) withContext(Dispatchers.IO) {
                draftStore.load(saved.deviceId).retainForInitialSelection(null, libraryId, entry.id)
            } else null
            if (!pageSession.isCurrent(session)) return@launch
            if (mayRecoverDraft && recoveryCandidate == null) withContext(Dispatchers.IO) { draftStore.clear() }
            val submitted = mutableState.value
            runCatching { repository.page(saved, libraryId, entry.id) }
            .onSuccess { page ->
                if (!pageSession.isCurrent(session)) return@onSuccess
                val recovered = recoveryCandidate.recoverFor(libraryId, page)
                mutate {
                    if (this.libraryId != libraryId || selectedPath != entry.path) return@mutate this
                    withWikiLoadedState(submitted, copy(
                    page = recovered?.let { page.copy(revision = it.revision) } ?: page,
                    selectedPath = entry.path,
                    draft = recovered?.content ?: page.content,
                    loading = false,
                    editing = recovered != null,
                    draftRecovered = recovered != null,
                    offlineCopy = false,
                )) }
                cacheCurrentPage()
                loadBacklinks(saved, libraryId, page.id)
            }
            .onFailure { failure ->
                if (failure !is CancellationException) mutate {
                    if (!pageSession.isCurrent(session) || this.libraryId != libraryId || selectedPath != entry.path) this
                    else if (cached != null) copy(loading = false, error = null, offlineCopy = true)
                    else copy(loading = false, error = failure.message ?: "Could not open this page.")
                }
            }
        }
    }
    fun consumeNavigationHeading() = mutate { copy(navigationHeading = "") }
    fun dismissMissingLink() = mutate { copy(missingLink = null) }
    fun createMissingLink() {
        val path = mutableState.value.missingLink?.path ?: return
        mutate { copy(missingLink = null) }
        create(path, folder = false)
    }
    fun saveOffline() {
        val saved = credential ?: return
        val snapshot = mutableState.value
        if (snapshot.libraryId.isBlank() || snapshot.entries.isEmpty() || snapshot.offlineSaving) return
        offlineMirrorJob?.cancel()
        mutate { copy(offlineSaving = true, offlineSaved = 0, offlineTotal = 0, offlineStatus = null, error = null) }
        val session = offlineSession.capture()
        offlineMirrorJob = viewModelScope.launch {
            runCatching {
                offlineMirror.save(saved, snapshot.libraryId, snapshot.entries, snapshot.selectedPath) { completed, total ->
                    mutate {
                        if (offlineSession.isCurrent(session) && libraryId == snapshot.libraryId) copy(offlineSaved = completed, offlineTotal = total) else this
                    }
                }
            }.onSuccess { result -> mutate {
                if (!offlineSession.isCurrent(session) || libraryId != snapshot.libraryId) this else copy(
                    offlineSaving = false,
                    offlineSaved = result.saved,
                    offlineTotal = minOf(result.available, MAX_OFFLINE_PAGES),
                    offlineAssetUrls = result.assetUrls,
                    offlineStatus = buildOfflineWikiStatus(result),
                )
            } }.onFailure { failure ->
                if (failure !is CancellationException) mutate {
                    if (offlineSession.isCurrent(session) && libraryId == snapshot.libraryId) copy(offlineSaving = false, error = failure.message ?: "Could not save this vault offline.") else this
                }
            }
        }
    }
    fun edit(enabled: Boolean) {
        if (!enabled) { draftSaveJob?.cancel(); draftStore.clear() }
        mutate { copy(editing = enabled, draft = page?.content.orEmpty(), draftRecovered = false) }
    }
    fun closePage() {
        cancelAttachment()
        pageSession.reset()
        treeLoadJob?.cancel()
        pageLoadJob?.cancel()
        backlinkLoadJob?.cancel()
        draftSaveJob?.cancel()
        draftStore.clear()
        savedState[SELECTED_PATH] = ""
        mutate { withClosedWikiPage() }
    }
    fun draft(value: String) {
        if (!mutableState.value.acceptsDraftChange()) return
        if (!acceptsWikiDraft(value)) {
            mutate { copy(error = WIKI_DRAFT_LIMIT_ERROR) }
            return
        }
        val content = value
        mutate { copy(draft = content, draftRecovered = false, error = error.takeUnless { it == WIKI_DRAFT_LIMIT_ERROR }) }
        val snapshot = mutableState.value
        val page = snapshot.page ?: return
        draftSaveJob?.cancel()
        if (!shouldPersistWikiDraft(content, page.content)) {
            draftStore.clear()
            return
        }
        val writeTicket = draftStore.beginWrite()
        draftSaveJob = viewModelScope.launch {
            delay(500)
            val persisted = persistWikiDraft {
                withContext(Dispatchers.IO) {
                    credential?.let { draftStore.save(it.deviceId, WikiDraft(snapshot.libraryId, page.id, page.revision, content, System.currentTimeMillis()), writeTicket) }
                }
            }
            if (!persisted) mutate { withDraftPersistenceFailure(snapshot) }
        }
    }
    fun save() {
        val saved = credential ?: return
        val snapshot = mutableState.value
        if (snapshot.saving) return
        val session = pageSession.capture()
        val page = snapshot.page ?: return
        val content = snapshot.draft
        mutate { copy(saving = true, error = null) }
        viewModelScope.launch { runCatching { repository.save(saved, snapshot.libraryId, page, content) }
            .onSuccess { refreshed ->
                if (!pageSession.isCurrent(session)) return@onSuccess
                mutate {
                withSavedWikiDraft(snapshot, refreshed)
            }.also {
                val current = mutableState.value
                if (current.libraryId == snapshot.libraryId && current.page?.id == page.id && current.selectedPath == snapshot.selectedPath) {
                    // Persist any newer edit against the newly acknowledged revision.
                    draft(current.draft)
                    cacheCurrentPage()
                }
            } }
            .onFailure { failure ->
                if (!pageSession.isCurrent(session) || failure is CancellationException) return@onFailure
                mutate {
                if (libraryId == snapshot.libraryId && this.page?.id == page.id && selectedPath == snapshot.selectedPath) copy(
                    saving = false,
                    error = when (failure) {
                        is WikiSavedButReloadFailedException -> failure.message
                        is WikiException -> if (failure.status == 409) "This page changed elsewhere. Reload it before saving." else failure.message
                        else -> failure.message ?: "Could not save this page."
                    },
                )
                else this
            } } }
    }
    fun toggleTask(taskOrdinal: Int, checked: Boolean) {
        val saved = credential ?: return
        val snapshot = mutableState.value
        val page = snapshot.page ?: return
        if (!page.writable || snapshot.saving || snapshot.editing) return
        val session = pageSession.capture()
        val content = toggleWikiTask(page.content, taskOrdinal, checked)
        if (content == page.content) return
        mutate { copy(page = page.copy(content = content), draft = content, saving = true, error = null) }
        viewModelScope.launch {
            runCatching { repository.save(saved, snapshot.libraryId, page, content) }.onSuccess { refreshed ->
                if (!pageSession.isCurrent(session)) return@onSuccess
                mutate {
                    if (libraryId == snapshot.libraryId && this.page?.id == page.id) {
                        copy(page = refreshed, draft = refreshed.content, saving = false)
                    } else copy(saving = false)
                }
                cacheCurrentPage()
            }.onFailure { failure ->
                if (!pageSession.isCurrent(session) || failure is CancellationException) return@onFailure
                mutate {
                    if (libraryId == snapshot.libraryId && this.page?.id == page.id) copy(
                        page = if (failure is WikiSavedButReloadFailedException) this.page else page,
                        draft = if (failure is WikiSavedButReloadFailedException) draft else page.content,
                        saving = false,
                        error = when {
                            failure is WikiSavedButReloadFailedException -> failure.message
                            (failure as? WikiException)?.status == 409 -> "This task changed elsewhere. Reload the page before updating it."
                            else -> failure.message ?: "Could not update this task."
                        },
                    ) else copy(saving = false)
                }
            }
        }
    }
    fun create(path: String, folder: Boolean) {
        val saved = credential ?: return
        if (mutableState.value.saving) return
        if (!folder) mutableState.value.pageMutationWarning()?.let { warning ->
            mutate { copy(error = warning) }
            return
        }
        val session = pageSession.capture()
        val snapshot = mutableState.value
        val libraryId = snapshot.libraryId
        val normalized = path.trim().replace('\\', '/').trim('/').let { if (folder || it.endsWith(".md", true)) it else "$it.md" }
        if (normalized.isBlank()) return mutate { copy(error = "Enter a page or folder name.") }
        mutate { copy(saving = true, error = null) }
        viewModelScope.launch { runCatching { repository.create(saved, libraryId, normalized, folder) }.onSuccess { result ->
            if (!pageSession.isCurrent(session)) return@onSuccess
            mutate {
                withCreatedWikiEntry(snapshot, result, normalized)
            }
            savedState[SELECTED_PATH] = mutableState.value.selectedPath
            cacheCurrentPage()
        }
            .onFailure { failure ->
                if (!pageSession.isCurrent(session) || failure is CancellationException) return@onFailure
                mutate { if (this.libraryId == libraryId) copy(saving = false, error = failure.message ?: "Could not create this ${if (folder) "folder" else "page"}.") else this }
            } }
    }
    fun move(path: String) {
        val saved = credential ?: return
        val snapshot = mutableState.value
        if (snapshot.saving) return
        snapshot.pageMutationWarning()?.let { warning ->
            mutate { copy(error = warning) }
            return
        }
        val session = pageSession.capture()
        val page = snapshot.page ?: return
        val libraryId = snapshot.libraryId
        val normalized = normalizeWikiPagePath(path)
        if (normalized.isBlank() || normalized == mutableState.value.selectedPath) return
        backlinkLoadJob?.cancel()
        draftSaveJob?.cancel()
        draftStore.clear()
        mutate { copy(saving = true, error = null) }
        viewModelScope.launch { runCatching { repository.move(saved, libraryId, page, normalized) }.onSuccess { result ->
            if (!pageSession.isCurrent(session)) return@onSuccess
            mutate { withMovedWikiPage(snapshot, result) }
            val current = mutableState.value
            savedState[SELECTED_PATH] = current.selectedPath
            // Replace any in-flight old-path recovery write with the moved identity.
            draft(current.draft)
            current.page?.let { loadBacklinks(saved, libraryId, it.id) }
            cacheCurrentPage()
        }.onFailure { failure ->
            if (!pageSession.isCurrent(session) || failure is CancellationException) return@onFailure
            mutate {
            if (this.libraryId == libraryId && this.page?.id == page.id) copy(saving = false, error = if ((failure as? WikiException)?.status == 409) "This page changed elsewhere or that name already exists." else failure.message ?: "Could not move this page.")
            else copy(saving = false)
        } } }
    }
    fun trash() {
        val saved = credential ?: return
        val snapshot = mutableState.value
        if (snapshot.saving) return
        snapshot.pageMutationWarning()?.let { warning ->
            mutate { copy(error = warning) }
            return
        }
        val session = pageSession.capture()
        val page = snapshot.page ?: return
        val libraryId = snapshot.libraryId
        backlinkLoadJob?.cancel()
        draftSaveJob?.cancel()
        draftStore.clear()
        mutate { withTrashStarted() }
        viewModelScope.launch { runCatching { repository.trash(saved, libraryId, page) }.onSuccess { entries ->
            if (!pageSession.isCurrent(session)) return@onSuccess
            if (mutableState.value.libraryId != libraryId || mutableState.value.page?.id != page.id) return@onSuccess mutate { copy(saving = false) }
            savedState[SELECTED_PATH] = ""
            mutate { copy(entries = entries, page = null, selectedPath = "", draft = "", backlinks = emptyList(), saving = false, editing = false) }
            cacheCurrentPage()
        }
            .onFailure { failure ->
                if (!pageSession.isCurrent(session) || failure is CancellationException) return@onFailure
                mutate {
                if (this.libraryId == libraryId && this.page?.id == page.id) copy(saving = false, error = if ((failure as? WikiException)?.status == 409) "This page changed elsewhere. Reload it before removing it." else failure.message ?: "Could not move this page to vault trash.")
                else copy(saving = false)
            } } }
    }

    private fun loadBacklinks(saved: DeviceCredential, libraryId: String, pageId: String) {
        backlinkLoadJob?.cancel()
        val session = pageSession.capture()
        backlinkLoadJob = viewModelScope.launch {
            loadWikiBacklinks({ repository.backlinks(saved, libraryId, pageId) }) { backlinks ->
                if (pageSession.isCurrent(session)) mutate { withBacklinksIfCurrent(libraryId, pageId, backlinks) }
            }
        }
    }
    private fun cacheCurrentPage() {
        val snapshot = mutableState.value
        if (snapshot.libraryId.isBlank() || snapshot.entries.isEmpty() || snapshot.offlineCopy) return
        val savedAt = System.currentTimeMillis()
        viewModelScope.launch(Dispatchers.IO) {
            credential?.let { offlineStore.saveBestEffort(it.deviceId, WikiOfflineSnapshot(
                snapshot.libraryId, snapshot.entries, snapshot.selectedPath,
                snapshot.page?.let { listOf(WikiOfflinePage(snapshot.selectedPath, it, savedAt)) }.orEmpty(), savedAt,
            )) }
        }
    }
    private fun mutate(block: WikiState.() -> WikiState) { mutableState.value = mutableState.value.block() }

    private companion object {
        const val LIBRARY_ID = "wiki_library_id"
        const val SELECTED_PATH = "wiki_selected_path"
    }
}
