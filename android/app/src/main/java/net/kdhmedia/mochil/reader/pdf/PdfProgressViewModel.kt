package net.kdhmedia.mochil.reader.pdf

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.auth.DeviceCredentialStore

data class PdfProgressPrompt(val localPage: Int, val remotePage: Int, val remoteRevision: Long)
data class PdfProgressState(val requestedPage: Int? = null, val prompt: PdfProgressPrompt? = null, val message: String? = null)

class PdfProgressViewModel(application: Application) : AndroidViewModel(application) {
    private val credential = DeviceCredentialStore(application).load()
    private val client = PdfProgressClient()
    private val store = PdfProgressStore(application)
    private val writes = Mutex()
    private val mutableState = MutableStateFlow(PdfProgressState())
    val state: StateFlow<PdfProgressState> = mutableState.asStateFlow()
    private var openedKey = ""
    private var publicationKey = ""
    private var libraryId = ""
    private var bookId = ""
    private var pageCount = 0
    private var locatorKind = FixedLayoutLocatorKind.Pdf
    private var localPage = 0
    private var baseRevision = 0L
    private var ready = false
    private var desiredPage: Int? = null

    fun open(libraryId: String, bookId: String, pageCount: Int, localPage: Int, locatorKind: FixedLayoutLocatorKind = FixedLayoutLocatorKind.Pdf) {
        val saved = credential ?: return
        val legacyKey = "$libraryId:$bookId"
        val key = fixedLayoutSyncKey(libraryId, bookId)
        if (openedKey == key) return
        store.migrateLegacyKey(legacyKey, key)
        openedKey = key; publicationKey = key; this.libraryId = libraryId; this.bookId = bookId; this.pageCount = pageCount.coerceAtLeast(1); this.localPage = localPage.coerceIn(0, this.pageCount - 1); this.locatorKind = locatorKind
        val synchronized = store.load(key)
        baseRevision = synchronized.revision
        viewModelScope.launch {
            try {
                val remote = client.get(saved, libraryId, bookId, locatorKind)
                baseRevision = remote?.revision ?: 0
                val remotePage = remote?.pageFor(this@PdfProgressViewModel.pageCount)
                if (remotePage != null) {
                    when (decidePdfOpenPosition(this@PdfProgressViewModel.localPage, remotePage, baseRevision, synchronized)) {
                        PdfOpenDecision.KEEP_LOCAL -> {
                            ready = true
                            if (remotePage == this@PdfProgressViewModel.localPage) {
                                store.markSynchronized(key, baseRevision, remotePage)
                            } else {
                                desiredPage = this@PdfProgressViewModel.localPage
                                drain()
                            }
                        }
                        PdfOpenDecision.USE_REMOTE -> {
                            this@PdfProgressViewModel.localPage = remotePage
                            ready = true
                            store.markSynchronized(key, baseRevision, remotePage)
                            mutableState.value = PdfProgressState(requestedPage = remotePage)
                        }
                        PdfOpenDecision.ASK -> mutableState.value = PdfProgressState(prompt = PdfProgressPrompt(this@PdfProgressViewModel.localPage, remotePage, baseRevision))
                    }
                } else {
                    ready = true
                    desiredPage = this@PdfProgressViewModel.localPage
                    drain()
                }
            } catch (cancelled: CancellationException) {
                throw cancelled
            } catch (_: Throwable) {
                ready = true
                desiredPage = this@PdfProgressViewModel.localPage
                store.markPending(key, this@PdfProgressViewModel.localPage)
                mutableState.value = PdfProgressState(message = "PDF progress will sync when the server is available.")
            }
        }
    }

    fun pageChanged(page: Int) {
        val observed = page.coerceIn(0, (pageCount - 1).coerceAtLeast(0))
        if (!hasPdfPageMovement(localPage, observed)) return
        localPage = observed
        if (!ready || mutableState.value.prompt != null) return
        desiredPage = localPage
        store.markPending(publicationKey, localPage)
        drain()
    }

    fun useRemote() {
        val prompt = mutableState.value.prompt ?: return
        baseRevision = prompt.remoteRevision; localPage = prompt.remotePage; ready = true
        store.markSynchronized(publicationKey, baseRevision, localPage)
        mutableState.value = PdfProgressState(requestedPage = prompt.remotePage)
    }

    fun keepLocal() {
        val prompt = mutableState.value.prompt ?: return
        baseRevision = prompt.remoteRevision; localPage = prompt.localPage; ready = true
        mutableState.value = PdfProgressState()
        desiredPage = localPage
        store.markPending(publicationKey, localPage)
        drain()
    }

    fun consumedRequestedPage() { mutableState.value = mutableState.value.copy(requestedPage = null) }

    /** Retries one durable position without spinning while the device is offline. */
    fun retryPending() {
        if (!ready || publicationKey.isBlank() || mutableState.value.prompt != null) return
        val pending = store.load(publicationKey).pendingPage ?: return
        desiredPage = desiredPage ?: pending.coerceIn(0, (pageCount - 1).coerceAtLeast(0))
        drain()
    }

    private fun drain() {
        val saved = credential ?: return
        if (writes.isLocked) return
        viewModelScope.launch {
            writes.withLock {
                while (true) {
                    val page = desiredPage ?: break
                    desiredPage = null
                    try {
                        when (val result = client.put(saved, libraryId, bookId, page, pageCount, baseRevision, locatorKind)) {
                            is PdfSaveResult.Saved -> {
                                baseRevision = result.position.revision
                                val acknowledgedPage = result.position.pageFor(pageCount)
                                if (fixedLayoutSaveAcknowledgesPage(page, acknowledgedPage)) {
                                    store.markSynchronized(publicationKey, baseRevision, page, pendingPageAfterSuccessfulWrite(desiredPage))
                                    mutableState.value = mutableState.value.copy(message = null)
                                } else {
                                    store.markPending(publicationKey, page)
                                    mutableState.value = PdfProgressState(prompt = PdfProgressPrompt(page, acknowledgedPage, baseRevision))
                                    desiredPage = null
                                }
                            }
                            is PdfSaveResult.Conflict -> {
                                val remotePage = result.position.pageFor(pageCount)
                                baseRevision = result.position.revision
                                if (remotePage == page) {
                                    store.markSynchronized(publicationKey, baseRevision, page, pendingPageAfterSuccessfulWrite(desiredPage))
                                    mutableState.value = mutableState.value.copy(message = null)
                                } else {
                                    store.markPending(publicationKey, page)
                                    mutableState.value = PdfProgressState(prompt = PdfProgressPrompt(page, remotePage, baseRevision))
                                    desiredPage = null
                                }
                            }
                        }
                    } catch (cancelled: CancellationException) {
                        throw cancelled
                    } catch (_: Throwable) {
                        // Preserve the newest queued page if the reader moved while
                        // this request was in flight; otherwise retain this page.
                        desiredPage = pageToRetainAfterFailedWrite(page, desiredPage)
                        store.markPending(publicationKey, desiredPage!!)
                        mutableState.value = mutableState.value.copy(message = "PDF progress could not be synchronized yet.")
                        break
                    }
                    if (mutableState.value.prompt != null) break
                }
            }
        }
    }
}

/** Passive layout/lifecycle callbacks must not claim cross-device write ownership. */
internal fun hasPdfPageMovement(previous: Int, observed: Int): Boolean = previous != observed

internal fun pageToRetainAfterFailedWrite(attemptedPage: Int, newerQueuedPage: Int?): Int =
    newerQueuedPage ?: attemptedPage

internal fun pendingPageAfterSuccessfulWrite(newerQueuedPage: Int?): Int? = newerQueuedPage

/** A successful HTTP response only acknowledges the exact fixed-layout page. */
internal fun fixedLayoutSaveAcknowledgesPage(submittedPage: Int, authoritativePage: Int): Boolean =
    submittedPage == authoritativePage
