package net.kdhmedia.mochil.catalogue

import android.os.Looper
import java.io.File
import kotlinx.coroutines.CompletableDeferred
import net.kdhmedia.mochil.auth.DeviceCredential
import org.junit.Assert.assertEquals
import org.junit.Assert.assertSame
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.Shadows.shadowOf
import org.robolectric.annotation.Config

/**
 * The queue's progress must travel through its own flow, never through [LibraryState].
 *
 * A tick arrives several times a second; while it was copied into the state, every tick handed the
 * shelf a new [LibraryState], so the screen recomposed its controls, chips, banner and — with the
 * Downloaded filter on — its filtered book list, for a download the reader is not looking at.
 *
 * What must still write state is a *finished* download: the downloaded set is a real change.
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], shadows = [ShadowDeviceCredentialStore::class, ShadowLibraryVisibilityStore::class])
class LibraryDownloadQueueStateTest {
    private fun book(id: String, title: String = "Shelf book") = BookSummary(
        id = id, libraryId = "books", title = title, authors = listOf("An Author"), series = "",
        seriesIndex = null, format = "epub", mediaType = "application/epub+zip", language = "en",
        description = "", size = 10,
    )

    private val library = LibrarySummary("books", "Books", "books", 1)

    /** A view model whose queue fetches from [fetch]; no credential store and no server involved. */
    private fun viewModel(
        fetch: suspend (BookSummary, (Long, Long) -> Unit) -> Result<Unit>,
    ): LibraryViewModel {
        val vm = LibraryViewModel(RuntimeEnvironment.getApplication(), fetch)
        LibraryViewModel::class.java.getDeclaredField("credential").apply { isAccessible = true }
            .set(vm, DeviceCredential("http://127.0.0.1:1", "device", "token"))
        // The constructor's own downloaded-books refresh runs on Dispatchers.IO; let it land before
        // the state below is installed, or it would replace the instance this test compares against.
        awaitQuiet(vm)
        return vm
    }

    private fun awaitQuiet(vm: LibraryViewModel) {
        var last = vm.state.value
        var stable = 0
        val deadline = System.nanoTime() + 5_000_000_000L
        while (System.nanoTime() < deadline && stable < 5) {
            shadowOf(Looper.getMainLooper()).idle()
            val now = vm.state.value
            stable = if (now === last) stable + 1 else 0
            last = now
            Thread.sleep(20)
        }
    }

    private fun awaitQueue(vm: LibraryViewModel, description: String, until: (List<DownloadQueueItem>) -> Boolean) {
        val deadline = System.nanoTime() + 5_000_000_000L
        while (System.nanoTime() < deadline && !until(vm.downloadQueueItems.value)) {
            shadowOf(Looper.getMainLooper()).idle()
            Thread.sleep(10)
        }
        assertTrue("$description; queue was ${vm.downloadQueueItems.value}", until(vm.downloadQueueItems.value))
    }

    @Test
    fun `the shelf state carries no download queue`() {
        assertTrue(
            "a queue field in LibraryState makes every progress tick rewrite the whole shelf",
            LibraryState::class.java.methods.none { it.name == "getDownloadQueue" },
        )
    }

    @Test
    fun `a progress tick leaves the library state alone and the banner reads the flow`() {
        val release = CompletableDeferred<Unit>()
        val vm = viewModel { _, onProgress ->
            // The first report is never throttled, so this tick lands before the queue is released.
            onProgress(1_048_576, 3_145_728)
            release.await()
            Result.success(Unit)
        }
        val shelf = book("a")
        vm.update {
            copy(allLibraries = listOf(library), libraries = listOf(library), selected = library, books = listOf(shelf), initialLoading = false)
        }
        val before = vm.state.value

        vm.queueBook(shelf)
        awaitQueue(vm, "the download should have started") { queue ->
            queue.singleOrNull()?.let { it.state == DownloadQueueState.Downloading && it.done > 0 } == true
        }

        assertSame(
            "a queue tick must not hand the shelf a new LibraryState",
            before,
            vm.state.value,
        )
        assertEquals(
            "the banner's line comes from the queue's own flow",
            "Downloading Shelf book (1 of 1): 1.0 MB of 3.0 MB",
            downloadQueueBannerText(vm.downloadQueueItems.value),
        )
        assertEquals("nothing recorded a download, so the shelf keeps its list", listOf(shelf), vm.state.value.books)
        release.complete(Unit)
        awaitQueue(vm, "the released download should finish") { queue -> queue.single().state == DownloadQueueState.Done }
    }

    @Test
    fun `a finished download still updates the downloaded set in the state`() {
        val app = RuntimeEnvironment.getApplication()
        val shelf = book("a")
        lateinit var vm: LibraryViewModel
        vm = viewModel { item, onProgress ->
            onProgress(1_048_576, 3_145_728)
            // What the opener does on success: the file lands and the download index records it.
            val file = File(app.cacheDir, "${item.id}.epub").apply { writeText("epub") }
            assertTrue("the download index must accept the finished book", vm.downloads.record(item, file))
            Result.success(Unit)
        }
        val before = vm.state.value

        vm.queueBook(shelf)
        val deadline = System.nanoTime() + 5_000_000_000L
        while (System.nanoTime() < deadline && vm.state.value.downloadedBooks.isEmpty()) {
            shadowOf(Looper.getMainLooper()).idle()
            Thread.sleep(20)
        }

        assertEquals(
            "the queue should report the book as fetched; queue was ${vm.downloadQueueItems.value}",
            DownloadQueueState.Done,
            vm.downloadQueueItems.value.singleOrNull()?.state,
        )
        assertTrue("a finished download must still reach the state", vm.state.value !== before)
        assertEquals(
            "the completed book belongs in the downloaded set",
            listOf("a"),
            vm.state.value.downloadedBooks.map { it.id },
        )
        assertEquals(DownloadQueueState.Done, vm.downloadQueueItems.value.single().state)
    }
}
