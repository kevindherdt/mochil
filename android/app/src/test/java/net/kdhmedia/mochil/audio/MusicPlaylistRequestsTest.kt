package net.kdhmedia.mochil.audio

import java.io.IOException
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.cancel
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.yield
import org.junit.Assert.*
import org.junit.Test

class MusicPlaylistRequestsTest {
    @Test fun `stale replacement reloads selected details without replaying mutation`() = runBlocking {
        val job = Job()
        val scope = CoroutineScope(coroutineContext + job)
        var reads = 0
        var writes = 0
        val requests = MusicPlaylistRequests(scope, { name ->
            MusicPlaylist(name, emptyList(), "revision-${++reads}")
        }) { emptyList() }
        try {
            requests.select("Saved")
            yield()
            assertEquals("revision-1", requests.state.value.selectedPlaylist?.revision)
            requests.mutate("Saved") { writes++; throw MusicPlaylistConflict() }
            yield()
            yield()
            assertEquals(1, writes)
            assertEquals("revision-2", requests.state.value.selectedPlaylist?.revision)
            assertTrue(requests.state.value.error.orEmpty().contains("another device"))
            assertFalse(requests.state.value.saving)
        } finally { job.cancel() }
    }

    @Test fun `failed rename retains selection and later rename cannot undo navigation`() = runBlocking {
        val job = Job()
        val scope = CoroutineScope(coroutineContext + job)
        val requests = MusicPlaylistRequests(scope, { name -> MusicPlaylist(name, emptyList()) }) { listOf(MusicPlaylistOverview("New", 0, emptyList()), MusicPlaylistOverview("Other", 0, emptyList())) }
        try {
            requests.select("Old")
            requests.mutate("Renamed", "Old" to "New") { throw IOException("offline") }
            yield()
            assertEquals("Old", requests.state.value.selectedName)
            assertEquals("Could not update playlists.", requests.state.value.error)
            val release = CompletableDeferred<Unit>()
            requests.mutate("Renamed", "Old" to "New") { release.await() }
            yield()
            requests.select("Other")
            release.complete(Unit)
            yield()
            assertEquals("Other", requests.state.value.selectedName)
            assertFalse(requests.state.value.saving)
        } finally { job.cancel() }
    }

    @Test fun `mutations reconcile in order and stay saving until the last finishes`() = runBlocking {
        val job = Job()
        val scope = CoroutineScope(coroutineContext + job)
        var server = "Initial"
        val requests = MusicPlaylistRequests(scope, { name -> MusicPlaylist(name, emptyList()) }) { listOf(MusicPlaylistOverview(server, 0, emptyList())) }
        try {
            val first = CompletableDeferred<Unit>()
            val second = CompletableDeferred<Unit>()
            val secondStarted = CompletableDeferred<Unit>()
            requests.mutate("First") { first.await(); server = "First" }
            requests.mutate("Second") {
                assertEquals("First", requests.state.value.playlists.single().name)
                secondStarted.complete(Unit)
                second.await()
                server = "Second"
            }
            first.complete(Unit)
            secondStarted.await()
            assertTrue(requests.state.value.saving)
            second.complete(Unit)
            yield()
            assertEquals("Second", requests.state.value.playlists.single().name)
            assertFalse(requests.state.value.saving)
        } finally { job.cancel() }
    }

    @Test fun `old refresh is discarded when a mutation is queued`() = runBlocking {
        val job = Job()
        val scope = CoroutineScope(coroutineContext + job)
        val old = CompletableDeferred<List<MusicPlaylistOverview>>()
        var calls = 0
        val requests = MusicPlaylistRequests(scope, { name -> MusicPlaylist(name, emptyList()) }) {
            if (++calls == 1) old.await() else listOf(MusicPlaylistOverview("New", 0, emptyList()))
        }
        try {
            requests.refresh()
            yield()
            requests.mutate("Saved") {
                assertTrue(requests.state.value.playlists.isEmpty())
            }
            old.complete(listOf(MusicPlaylistOverview("Old", 0, emptyList())))
            yield()
            yield()
            assertEquals("New", requests.state.value.playlists.single().name)
        } finally { job.cancel() }
    }
}
