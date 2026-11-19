package net.kdhmedia.mochil.audio

import java.io.IOException
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.cancel
import kotlinx.coroutines.Job
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.joinAll
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.yield
import org.junit.Assert.*
import org.junit.Test

class MusicQueueRequestsTest {
    private fun track(id: String) = AudioTrack(id, "music", id, "Artist", "Album", 180.0)
    private suspend fun finishRequests() = currentCoroutineContext()[Job]!!.children.toList().joinAll()

    @Test fun `old refresh cannot overwrite a local edit`() = runBlocking {
        val read = CompletableDeferred<List<AudioTrack>>()
        val writes = mutableListOf<List<AudioTrack>>()
        val requests = MusicQueueRequests(this, true, { read.await() }, { writes += it })
        requests.refresh()
        yield()
        val latest = listOf(track("new"))
        requests.persist(latest)
        read.complete(listOf(track("old")))
        finishRequests()
        assertEquals(latest, requests.state.value.tracks)
        assertEquals(listOf(latest), writes)
        assertFalse(requests.state.value.saving)
    }

    @Test fun `failed older write cannot reload over a waiting edit or clear saving`() = runBlocking {
        val firstRelease = CompletableDeferred<Unit>()
        val secondRelease = CompletableDeferred<Unit>()
        val secondStarted = CompletableDeferred<Unit>()
        var reads = 0
        val writes = mutableListOf<List<AudioTrack>>()
        val requests = MusicQueueRequests(this, true, { reads++; emptyList() }, {
            writes += it
            if (writes.size == 1) { firstRelease.await(); throw IOException("offline") }
            secondStarted.complete(Unit)
            secondRelease.await()
        })
        requests.persist(listOf(track("first")))
        yield()
        val latest = listOf(track("latest"))
        requests.persist(latest)
        firstRelease.complete(Unit)
        secondStarted.await()
        assertTrue(requests.state.value.saving)
        assertNull(requests.state.value.error)
        assertEquals(latest, requests.state.value.tracks)
        requests.refresh()
        assertEquals(0, reads)
        secondRelease.complete(Unit)
        finishRequests()
        assertEquals(latest, writes.last())
        assertFalse(requests.state.value.saving)
    }

    @Test fun `newest failure retains an immutable edit and error`() = runBlocking {
        val input = mutableListOf(track("wanted"))
        val expected = input.toList()
        val requests = MusicQueueRequests(this, true, { error("must not reload") }, { throw IOException("offline") })
        requests.persist(input)
        input.clear()
        finishRequests()
        assertEquals(expected, requests.state.value.tracks)
        assertEquals("Could not save the queue.", requests.state.value.error)
        assertFalse(requests.state.value.saving)
    }

    @Test fun `queued obsolete snapshots are skipped`() = runBlocking {
        val writes = mutableListOf<List<AudioTrack>>()
        val requests = MusicQueueRequests(this, true, { emptyList() }, { writes += it })
        requests.persist(listOf(track("one")))
        requests.persist(listOf(track("two")))
        val last = listOf(track("three"))
        requests.persist(last)
        finishRequests()
        assertEquals(listOf(last), writes)
    }

    @Test fun `cancelled transport does not publish a save failure`() = runBlocking {
        val requests = MusicQueueRequests(this, true, { emptyList() }, {
            currentCoroutineContext().cancel()
            throw IOException("transport interrupted")
        })
        requests.persist(listOf(track("one")))
        finishRequests()
        assertNull(requests.state.value.error)
    }
}
