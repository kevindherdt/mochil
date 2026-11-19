package net.kdhmedia.mochil.catalogue

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.auth.DeviceCredentialStore

class CollectionViewModel(application: Application) : AndroidViewModel(application) {
    private val credential = DeviceCredentialStore(application).load()
    private val client = CollectionClient()
    private val mutableState = MutableStateFlow(CollectionPickerState())
    val state = mutableState.asStateFlow()
    private val mutableChanges = MutableSharedFlow<String>(extraBufferCapacity = 1)
    val changes = mutableChanges.asSharedFlow()

    fun open(book: BookSummary) = open(CollectionTarget(book.title, CollectionMember("book", book.libraryId, bookId = book.id)), book)
    fun open(series: SeriesSummary) = open(CollectionTarget(series.name, CollectionMember("series", series.libraryId, series = series.name)))
    fun close() { mutableState.value = CollectionPickerState() }

    fun toggle(collection: PublicationCollection) {
        val saved = credential ?: return fail("This device is no longer connected.")
        val target = mutableState.value.target ?: return
        if (mutableState.value.busyId != null) return
        val removing = collection.contains(target.member)
        mutableState.value = mutableState.value.copy(busyId = collection.id, message = null)
        viewModelScope.launch {
            runCatching { withContext(Dispatchers.IO) {
                client.setMember(saved, collection.id, target.member, !removing)
                client.list(saved, target.member.libraryId)
            } }
                .onSuccess { collections ->
                    val current = mutableState.value
                    if (current.target != target) return@onSuccess
                    mutableState.value = current.copy(
                        busyId = null,
                        collections = collections,
                        message = if (removing) "Removed from ${collection.name}." else "Added to ${collection.name}.",
                    )
                    mutableChanges.tryEmit(target.member.libraryId)
                }
                .onFailure { if (mutableState.value.target == target) fail("Could not update this collection.") }
        }
    }

    /** Sets the reader's own label for the open book: unread, reading, read, on_hold or dropped. */
    internal fun setReadingStatus(status: ReadingStatus) {
        val saved = credential ?: return fail("This device is no longer connected.")
        val book = mutableState.value.book ?: return
        if (mutableState.value.statusBusy) return
        mutableState.value = mutableState.value.copy(statusBusy = true, message = null)
        viewModelScope.launch {
            runCatching { withContext(Dispatchers.IO) { ReadingStatusClient().set(saved, book.libraryId, book.id, status.wire) } }
                .onSuccess {
                    val current = mutableState.value
                    if (current.book?.id != book.id) return@onSuccess
                    mutableState.value = current.copy(statusBusy = false, book = book.withReadingStatus(status), message = readingStatusMessage(status))
                    mutableChanges.tryEmit(book.libraryId)
                }
                .onFailure { if (mutableState.value.book?.id == book.id) fail("Could not change the reading status.") }
        }
    }

    fun create(name: String) {
        val normalized = name.trim().take(80)
        val target = mutableState.value.target ?: return
        val saved = credential ?: return fail("This device is no longer connected.")
        if (normalized.isBlank() || mutableState.value.creating) return
        mutableState.value = mutableState.value.copy(creating = true, message = null)
        viewModelScope.launch {
            runCatching { withContext(Dispatchers.IO) {
                val created = client.create(saved, normalized, target)
                created to client.list(saved, target.member.libraryId)
            } }
                .onSuccess { (created, collections) -> if (mutableState.value.target == target) {
                    mutableState.value = mutableState.value.copy(creating = false, collections = collections, message = "Created ${created.name}.")
                    mutableChanges.tryEmit(target.member.libraryId)
                } }
                .onFailure { if (mutableState.value.target == target) fail("Could not create this collection.") }
        }
    }

    private fun open(target: CollectionTarget, book: BookSummary? = null) {
        val saved = credential
        mutableState.value = CollectionPickerState(
            target = target,
            book = book,
            collectionsEnabled = saved != null,
            loading = saved != null,
        )
        if (saved == null) return
        viewModelScope.launch {
            runCatching { withContext(Dispatchers.IO) { client.list(saved, target.member.libraryId) } }
                .onSuccess { if (mutableState.value.target == target) mutableState.value = mutableState.value.copy(collections = it, loading = false) }
                .onFailure { if (mutableState.value.target == target) fail("Could not load collections.") }
        }
    }

    private fun fail(message: String) { mutableState.value = mutableState.value.copy(loading = false, busyId = null, creating = false, statusBusy = false, message = message) }
}
