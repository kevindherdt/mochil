package net.kdhmedia.mochil.catalogue

import android.os.Looper
import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import java.util.concurrent.CopyOnWriteArrayList
import kotlinx.coroutines.flow.MutableStateFlow
import net.kdhmedia.mochil.auth.DeviceCredential
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.Shadows.shadowOf
import org.robolectric.annotation.Config

/** What the status chips do to the shelf: one request per choice, kept while paging, dropped when the shelf changes meaning. */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], shadows = [ShadowDeviceCredentialStore::class, ShadowLibraryVisibilityStore::class])
class LibraryStatusFilterTest {
    private val queries = CopyOnWriteArrayList<String>()
    private lateinit var server: HttpServer
    private lateinit var vm: LibraryViewModel
    private val books = LibrarySummary("books", "Books", "books", 5)
    private val novels = LibrarySummary("novels", "Light Novels", "books", 3)

    @Before fun start() {
        server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/") { exchange ->
            if (exchange.requestURI.path.endsWith("/books")) queries.add(exchange.requestURI.rawQuery.orEmpty())
            val body = if (exchange.requestURI.path.endsWith("/books"))
                """[{"id":"b1","library_id":"books","title":"One","authors":["A"]}]""" else "[]"
            val bytes = body.toByteArray()
            exchange.responseHeaders.add("X-Next-Cursor", "")
            exchange.sendResponseHeaders(200, bytes.size.toLong())
            exchange.responseBody.use { it.write(bytes) }
        }
        server.start()
        vm = LibraryViewModel(RuntimeEnvironment.getApplication())
        LibraryViewModel::class.java.getDeclaredField("credential").apply { isAccessible = true }
            .set(vm, DeviceCredential("http://127.0.0.1:${server.address.port}", "test", "test"))
        seed(LibraryState(allLibraries = listOf(books, novels), libraries = listOf(books, novels), selected = books, initialLoading = false, grouping = PublicationGrouping.Books))
    }

    @After fun stop() = server.stop(0)

    @Suppress("UNCHECKED_CAST")
    private fun seed(state: LibraryState) {
        (LibraryViewModel::class.java.getDeclaredField("mutableState").apply { isAccessible = true }.get(vm) as MutableStateFlow<LibraryState>).value = state
    }

    private fun waitFor(what: String, condition: () -> Boolean) {
        val deadline = System.nanoTime() + 5_000_000_000L
        while (System.nanoTime() < deadline && !condition()) { shadowOf(Looper.getMainLooper()).idle(); Thread.sleep(20) }
        assertTrue("$what; requests were $queries", condition())
    }

    @Test fun `choosing In Progress asks the server for in-progress books and says so in the state`() {
        vm.setStatus(BookStatusFilter.InProgress)
        waitFor("a request with status=in_progress") { queries.any { "status=in_progress" in it } }
        assertEquals(BookStatusFilter.InProgress, vm.state.value.status)
    }

    @Test fun `choosing All again drops the parameter`() {
        vm.setStatus(BookStatusFilter.Unread)
        waitFor("status=unread") { queries.any { "status=unread" in it } }
        queries.clear()
        vm.setStatus(BookStatusFilter.All)
        waitFor("a books request") { queries.isNotEmpty() }
        assertFalse(queries.any { "status" in it })
        assertEquals(BookStatusFilter.All, vm.state.value.status)
    }

    @Test fun `paging keeps the filter`() {
        seed(vm.state.value.copy(status = BookStatusFilter.InProgress, nextCursor = "c1"))
        vm.loadMore()
        waitFor("the next page request") { queries.any { "cursor=c1" in it } }
        assertTrue(queries.single { "cursor=c1" in it }.contains("status=in_progress"))
    }

    @Test fun `search composes with the filter`() {
        seed(vm.state.value.copy(status = BookStatusFilter.Unread))
        vm.search("dune")
        waitFor("a searched request") { queries.any { "q=dune" in it } }
        assertTrue(queries.single { "q=dune" in it }.contains("status=unread"))
    }

    @Test fun `a shelf change during paging clears the paging flag, so the new list can still load more`() {
        // The page in flight is dropped by the generation check; before the fix nothing cleared loadingMore and the list stuck at 60.
        seed(vm.state.value.copy(nextCursor = "c1", loadingMore = true))
        vm.setStatus(BookStatusFilter.InProgress)
        waitFor("a request with status=in_progress") { queries.any { "status=in_progress" in it } }
        assertFalse("paging flag left set by the dropped page", vm.state.value.loadingMore)
    }

    @Test fun `another library starts at All`() {
        seed(vm.state.value.copy(status = BookStatusFilter.InProgress))
        vm.select(novels)
        assertEquals(BookStatusFilter.All, vm.state.value.status)
    }

    @Test fun `a grouped view has no status to filter by, so it starts at All`() {
        seed(vm.state.value.copy(status = BookStatusFilter.Unread))
        vm.setGrouping(PublicationGrouping.Series)
        assertEquals(BookStatusFilter.All, vm.state.value.status)
    }

    @Test fun `the filter is ignored where it cannot apply`() {
        seed(vm.state.value.copy(grouping = PublicationGrouping.Series))
        queries.clear()
        vm.setStatus(BookStatusFilter.InProgress)
        shadowOf(Looper.getMainLooper()).idle(); Thread.sleep(150); shadowOf(Looper.getMainLooper()).idle()
        assertEquals(BookStatusFilter.All, vm.state.value.status)
        assertTrue(queries.isEmpty())
    }

    @Test fun `the chips belong to the plain list, online, with nothing open`() {
        val base = LibraryState(selected = books, grouping = PublicationGrouping.Books)
        fun shows(state: LibraryState) = state.showsStatusChips()
        val open = BookSummary("b1", "books", "One", emptyList(), "", null, "", "text/plain", "", "", 0)
        assertTrue(shows(base))
        assertTrue(shows(base.copy(grouping = PublicationGrouping.Recent)))
        assertFalse(shows(base.copy(grouping = PublicationGrouping.Series)))
        assertFalse(shows(base.copy(grouping = PublicationGrouping.Authors)))
        assertFalse(shows(base.copy(downloadedOnly = true)))
        assertFalse(shows(base.copy(offlineOnly = true)))
        assertFalse(shows(base.copy(selectedBook = open)))
        // A wide screen keeps the list beside the open book, so its chips stay.
        assertTrue(base.copy(selectedBook = open).showsStatusChips(keepDetailFilters = true))
    }

    @Test fun `downloads only has no status to filter by, so it starts at All`() {
        seed(vm.state.value.copy(status = BookStatusFilter.InProgress))
        vm.setDownloadedOnly(true)
        assertEquals(BookStatusFilter.All, vm.state.value.status)
    }
}
