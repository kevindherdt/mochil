package net.kdhmedia.mochil.wiki

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.launch
import kotlinx.coroutines.cancel
import kotlinx.coroutines.CancellationException
import net.kdhmedia.mochil.auth.DeviceCredential
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Assert.assertSame
import org.junit.Test

class WikiRepositoryTest {
    private val credential = DeviceCredential("https://books.example", "device", "token")

    @Test fun `open fetches only the selected page after the tree`() = runBlocking {
        val remote = FakeWikiRemote()
        val repository = WikiRepository(remote, Dispatchers.Unconfined)

        val result = repository.open(credential, "vault", "Notes/Two.md")

        assertEquals("two", result.selected?.id)
        assertEquals("two", result.page?.id)
        assertEquals(listOf("tree", "page:two"), remote.calls)
    }

    @Test fun `open does not fetch an absent remembered page`() = runBlocking {
        val remote = FakeWikiRemote()
        val result = WikiRepository(remote, Dispatchers.Unconfined).open(credential, "vault", "Missing.md")

        assertNull(result.selected)
        assertNull(result.page)
        assertEquals(listOf("tree"), remote.calls)
    }

    @Test fun `save reloads the authoritative revision after writing`() = runBlocking {
        val remote = FakeWikiRemote()
        val original = WikiPage("one", "One", "old", "r1", true)

        val result = WikiRepository(remote, Dispatchers.Unconfined).save(credential, "vault", original, "new")

        assertEquals("r2", result.revision)
        assertEquals(listOf("save:one:new", "page:one"), remote.calls)
    }

    @Test fun `refresh reports when the selected page disappeared without loading another page`() = runBlocking {
        val remote = FakeWikiRemote()
        val result = WikiRepository(remote, Dispatchers.Unconfined).refresh(credential, "vault", "missing", reloadPage = true)

        assertEquals(false, result.selectedStillExists)
        assertNull(result.page)
        assertEquals(listOf("tree"), remote.calls)
    }

    @Test fun `save distinguishes a completed write from a failed revision reload`() = runBlocking {
        val remote = FakeWikiRemote(failPageId = "one")
        val original = WikiPage("one", "One", "old", "r1", true)

        val failure = runCatching {
            WikiRepository(remote, Dispatchers.Unconfined).save(credential, "vault", original, "new")
        }.exceptionOrNull()

        assertTrue(failure is WikiSavedButReloadFailedException)
        assertEquals(listOf("save:one:new", "page:one"), remote.calls)
    }

    @Test fun `create page refreshes the tree and opens the indexed page`() = runBlocking {
        val remote = FakeWikiRemote()
        val result = WikiRepository(remote, Dispatchers.Unconfined).create(credential, "vault", "Notes/Two.md", false)

        assertEquals("two", result.selected?.id)
        assertEquals("two", result.page?.id)
        assertEquals(listOf("create-page:Notes/Two.md", "tree", "page:two"), remote.calls)
    }

    @Test fun `cancelled write does not start a follow-up reload`() = runBlocking {
        val remote = FakeWikiRemote()
        val job = launch {
            remote.afterSave = { cancel() }
            WikiRepository(remote, Dispatchers.Unconfined).save(credential, "vault", WikiPage("one", "One", "old", "r1", true), "new")
        }
        job.join()
        assertTrue(job.isCancelled)
        assertEquals(listOf("save:one:new"), remote.calls)
    }

    @Test fun `reload cancellation is propagated without claiming a network failure`() = runBlocking {
        val cancellation = CancellationException("Closing")
        val remote = FakeWikiRemote(pageFailure = cancellation)
        val failure = runCatching {
            WikiRepository(remote, Dispatchers.Unconfined).save(credential, "vault", WikiPage("one", "One", "old", "r1", true), "new")
        }.exceptionOrNull()
        assertSame(cancellation, failure)
    }

    @Test fun `cancellation immediately after successful save retains retry barrier`() = runBlocking {
        val remote = FakeWikiRemote()
        val repository = WikiRepository(remote, Dispatchers.Unconfined)
        val page = WikiPage("one", "One", "old", "r1", true)
        val job = launch {
            remote.afterSave = { cancel() }
            repository.save(credential, "vault", page, "new")
        }
        job.join()
        assertTrue(job.isCancelled)
        remote.afterSave = {}
        assertTrue(runCatching { repository.save(credential, "vault", page, "new") }.exceptionOrNull() is WikiSavedButReloadFailedException)
        assertEquals(listOf("save:one:new"), remote.calls)
        val current = repository.page(credential, "vault", page.id)
        repository.save(credential, "vault", current, "next")
        assertEquals(listOf("save:one:new", "page:one", "save:one:next", "page:one"), remote.calls)
    }

    @Test fun `structural writes report acknowledged success when refresh fails`() = runBlocking {
        val page = WikiPage("one", "One", "old", "r1", true)
        for (operation in listOf("create", "folder", "move", "trash")) {
            val remote = FakeWikiRemote(treeFailure = IllegalStateException("offline"))
            val repository = WikiRepository(remote, Dispatchers.Unconfined)
            val failure = runCatching {
                when (operation) {
                    "create" -> repository.create(credential, "vault", "Notes/Two.md", false)
                    "folder" -> repository.create(credential, "vault", "Folder", true)
                    "move" -> repository.move(credential, "vault", page, "Notes/Two.md")
                    else -> repository.trash(credential, "vault", page)
                }
            }.exceptionOrNull()
            assertTrue(operation, failure is WikiMutationRefreshException)
            assertEquals(2, remote.calls.size)
            assertEquals("tree", remote.calls.last())
        }
    }

    @Test fun `folder creation needs no selected page and refresh cancellation stays cancellation`() = runBlocking {
        val indexed = FakeWikiRemote().apply { entries = entries + WikiEntry("folder", "folder", "Folder", "Folder", "") }
        val result = WikiRepository(indexed, Dispatchers.Unconfined).create(credential, "vault", "Folder", true)
        assertNull(result.page)
        val cancellation = CancellationException("Closing")
        val remote = FakeWikiRemote(treeFailure = cancellation)
        val failure = runCatching {
            WikiRepository(remote, Dispatchers.Unconfined).create(credential, "vault", "Folder", true)
        }.exceptionOrNull()
        assertSame(cancellation, failure)
    }

    @Test fun `created folder missing from refreshed tree blocks duplicate creation`() = runBlocking {
        val remote = FakeWikiRemote()
        val repository = WikiRepository(remote, Dispatchers.Unconfined)
        val failure = runCatching { repository.create(credential, "vault", "Folder", true) }.exceptionOrNull()
        assertTrue(failure is WikiMutationRefreshException)
        repository.tree(credential, "vault")
        assertSame(failure, runCatching { repository.create(credential, "vault", "Folder", true) }.exceptionOrNull())
        assertEquals(listOf("create-folder:Folder", "tree", "tree"), remote.calls)
        // A page with the same path is not evidence that the folder was indexed.
        remote.entries = remote.entries + WikiEntry("page", "wrong", "Folder", "Folder", "md")
        repository.tree(credential, "vault")
        assertSame(failure, runCatching { repository.create(credential, "vault", "Folder", true) }.exceptionOrNull())
        assertEquals(1, remote.calls.count { it.startsWith("create-folder:") })
    }

    @Test fun `committed server save failure preserves acknowledgement without another request`() = runBlocking {
        val remote = FakeWikiRemote()
        val error = WikiException(500, "Saved but not indexed", mutationCommitted = true)
        remote.afterSave = { throw error }
        val failure = runCatching {
            WikiRepository(remote, Dispatchers.Unconfined).save(credential, "vault", WikiPage("one", "One", "old", "r1", true), "new")
        }.exceptionOrNull()
        assertTrue(failure is WikiSavedButReloadFailedException)
        assertSame(error, failure?.cause)
        assertEquals(listOf("save:one:new"), remote.calls)
    }

    @Test fun `unacknowledged server write failure is not converted into a saved outcome`() = runBlocking {
        for (status in listOf(409, 500)) {
            val remote = FakeWikiRemote()
            val error = WikiException(status, "Write failed")
            remote.afterSave = { throw error }
            val failure = runCatching {
                WikiRepository(remote, Dispatchers.Unconfined).save(credential, "vault", WikiPage("one", "One", "old", "r1", true), "new")
            }.exceptionOrNull()
            assertSame(error, failure)
            assertEquals(listOf("save:one:new"), remote.calls)
        }
    }

    @Test fun `acknowledged save blocks writes until page read not merely tree refresh`() = runBlocking {
        val remote = FakeWikiRemote()
        val repository = WikiRepository(remote, Dispatchers.Unconfined)
        val page = WikiPage("one", "One", "old", "r1", true)
        remote.afterSave = { throw WikiException(500, "Saved", mutationCommitted = true) }
        val failure = runCatching { repository.save(credential, "vault", page, "new") }.exceptionOrNull()
        repository.refresh(credential, "vault", page.id, reloadPage = false)
        remote.afterSave = {}
        assertSame(failure, runCatching { repository.save(credential, "vault", page, "new") }.exceptionOrNull())
        assertSame(failure, runCatching { repository.move(credential, "vault", page, "Other.md") }.exceptionOrNull())
        assertSame(failure, runCatching { repository.trash(credential, "vault", page) }.exceptionOrNull())
        assertEquals(listOf("save:one:new", "tree"), remote.calls)
        val refreshed = repository.page(credential, "vault", page.id)
        repository.save(credential, "vault", refreshed, "newer")
        assertEquals(listOf("save:one:new", "tree", "page:one", "save:one:newer", "page:one"), remote.calls)
    }

    @Test fun `move requires source removal as well as destination presence`() = runBlocking {
        val remote = FakeWikiRemote()
        val repository = WikiRepository(remote, Dispatchers.Unconfined)
        val page = WikiPage("one", "One", "old", "r1", true)
        val failure = runCatching { repository.move(credential, "vault", page, "Notes/Two.md") }.exceptionOrNull()
        assertTrue(failure is WikiMutationRefreshException)
        assertSame(failure, runCatching { repository.move(credential, "vault", page, "Notes/Two.md") }.exceptionOrNull())
        assertEquals(listOf("move:one:Notes/Two.md", "tree"), remote.calls)
    }

    @Test fun `move opens destination when source is absent from refreshed tree`() = runBlocking {
        val remote = FakeWikiRemote().apply { entries = entries.filterNot { it.id == "one" } }
        val result = WikiRepository(remote, Dispatchers.Unconfined).move(
            credential, "vault", WikiPage("one", "One", "old", "r1", true), "Notes/Two.md",
        )
        assertEquals("two", result.page?.id)
        assertEquals(listOf("move:one:Notes/Two.md", "tree", "page:two"), remote.calls)
    }

    @Test fun `trash with stale tree blocks duplicate writes until removal is indexed`() = runBlocking {
        val remote = FakeWikiRemote()
        val repository = WikiRepository(remote, Dispatchers.Unconfined)
        val page = WikiPage("one", "One", "old", "r1", true)
        val failure = runCatching { repository.trash(credential, "vault", page) }.exceptionOrNull()
        assertTrue(failure is WikiMutationRefreshException)
        repository.tree(credential, "vault")
        assertSame(failure, runCatching { repository.trash(credential, "vault", page) }.exceptionOrNull())
        assertSame(failure, runCatching { repository.save(credential, "vault", page, "new") }.exceptionOrNull())
        assertEquals(listOf("trash:one", "tree", "tree"), remote.calls)
    }

    @Test fun `trash succeeds when refreshed tree no longer contains the page`() = runBlocking {
        val remote = FakeWikiRemote().apply { entries = entries.filterNot { it.id == "one" } }
        val repository = WikiRepository(remote, Dispatchers.Unconfined)
        val result = repository.trash(credential, "vault", WikiPage("one", "One", "old", "r1", true))
        assertEquals(listOf("two"), result.map { it.id })
        assertEquals(listOf("trash:one", "tree"), remote.calls)
    }

    private class FakeWikiRemote(private val failPageId: String = "", private val pageFailure: Exception? = null, private val treeFailure: Exception? = null) : WikiRemote {
        var afterSave: () -> Unit = {}
        var mutationFailure: Exception? = null
        val calls = mutableListOf<String>()
        var entries = listOf(
            WikiEntry("page", "one", "Notes/One.md", "One", "md"),
            WikiEntry("page", "two", "Notes/Two.md", "Two", "md"),
        )

        override fun tree(credential: DeviceCredential, libraryId: String) = entries.also { calls += "tree"; treeFailure?.let { throw it } }
        override fun search(credential: DeviceCredential, libraryId: String, query: String) = emptyList<WikiEntry>()
        override fun backlinks(credential: DeviceCredential, libraryId: String, id: String) = emptyList<WikiEntry>()
        override fun page(credential: DeviceCredential, libraryId: String, id: String): WikiPage {
            calls += "page:$id"
            pageFailure?.let { throw it }
            if (id == failPageId) error("offline")
            return WikiPage(id, id.replaceFirstChar(Char::uppercase), "content", "r2", true)
        }
        override fun save(credential: DeviceCredential, libraryId: String, page: WikiPage, content: String) { calls += "save:${page.id}:$content"; afterSave() }
        override fun createFolder(credential: DeviceCredential, libraryId: String, path: String) { calls += "create-folder:$path"; mutationFailure?.let { throw it } }
        override fun createPage(credential: DeviceCredential, libraryId: String, path: String) { calls += "create-page:$path"; mutationFailure?.let { throw it } }
        override fun movePage(credential: DeviceCredential, libraryId: String, page: WikiPage, path: String) { calls += "move:${page.id}:$path"; mutationFailure?.let { throw it } }
        override fun trashPage(credential: DeviceCredential, libraryId: String, page: WikiPage) { calls += "trash:${page.id}"; mutationFailure?.let { throw it } }
    }

    @Test fun `structural server acknowledgements block repeat requests even after stale tree reads`() = runBlocking {
        for (operation in listOf("create", "folder", "move", "trash")) {
            val remote = FakeWikiRemote().apply { mutationFailure = WikiException(500, "Saved", mutationCommitted = true) }
            val repository = WikiRepository(remote, Dispatchers.Unconfined)
            val page = WikiPage("one", "One", "old", "r1", true)
            suspend fun perform() {
                when (operation) {
                    "create" -> repository.create(credential, "vault", "New.md", false)
                    "folder" -> repository.create(credential, "vault", "New", true)
                    "move" -> repository.move(credential, "vault", page, "New.md")
                    else -> repository.trash(credential, "vault", page)
                }
            }
            val failure = runCatching { perform() }.exceptionOrNull()
            assertTrue(failure is WikiMutationRefreshException)
            remote.mutationFailure = null
            repository.tree(credential, "vault")
            assertSame(failure, runCatching { perform() }.exceptionOrNull())
            assertEquals(2, remote.calls.size)
            assertEquals("tree", remote.calls.last())
        }
    }
}
