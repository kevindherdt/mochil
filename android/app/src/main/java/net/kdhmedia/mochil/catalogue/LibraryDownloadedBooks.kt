package net.kdhmedia.mochil.catalogue

import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch

/**
 * Refreshes the index of books already on this device.
 *
 * Reading that index decodes JSON and stats every file, and saving it commits to disk, so it runs off
 * the main thread: doing it there made a large downloaded library stutter at start-up and after each
 * finished download.
 */
internal fun LibraryViewModel.refreshDownloadedBooks() {
    viewModelScope.launch(Dispatchers.IO) {
        val books = downloads.list().map(DownloadedPublication::asBookSummary)
        update { copy(downloadedBooks = books) }
    }
}
