package net.kdhmedia.mochil.audio

import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.yield
import org.junit.Assert.*
import org.junit.Test

class MusicPlaylistDetailRequestsTest {
    @Test fun `overview does not fetch tracks until selected and leaving discards late detail`() = runBlocking {
        val job = Job()
        val scope = CoroutineScope(coroutineContext + job)
        val detail = CompletableDeferred<MusicPlaylist>()
        var reads = 0
        val requests = MusicPlaylistRequests(scope, { reads++; detail.await() }) {
            listOf(MusicPlaylistOverview("Saved", 200, emptyList()))
        }
        try {
            requests.refresh()
            yield()
            assertEquals(0, reads)
            assertEquals(200, requests.state.value.playlists.single().trackCount)
            requests.select("Saved")
            yield()
            assertTrue(requests.state.value.detailLoading)
            assertNull(requests.state.value.selectedPlaylist)
            requests.select(null)
            detail.complete(MusicPlaylist("Saved", emptyList()))
            yield()
            assertNull(requests.state.value.selectedPlaylist)
            assertNull(requests.state.value.selectedName)
            assertFalse(requests.state.value.detailLoading)
        } finally { job.cancel() }
    }

    @Test fun `failed detail retains selected name and can retry`() = runBlocking {
        val job = Job()
        var fail = true
        val requests = MusicPlaylistRequests(CoroutineScope(coroutineContext + job), { name ->
            if (fail) error("offline")
            MusicPlaylist(name, emptyList())
        }) { listOf(MusicPlaylistOverview("Saved", 0, emptyList())) }
        try {
            requests.select("Saved")
            yield()
            assertEquals("Saved", requests.state.value.selectedName)
            assertNotNull(requests.state.value.error)
            assertNull(requests.state.value.selectedPlaylist)
            fail = false
            requests.select("Saved")
            yield()
            assertEquals("Saved", requests.state.value.selectedPlaylist?.name)
            assertNull(requests.state.value.error)
        } finally { job.cancel() }
    }
}
