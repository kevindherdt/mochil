package net.kdhmedia.mochil.audio

import java.io.IOException
import kotlinx.coroutines.Job
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.joinAll
import kotlinx.coroutines.runBlocking
import org.junit.Assert.*
import org.junit.Test

class MusicQueueRetryTest {
    private fun track(id: String) = AudioTrack(id, "music", id, "Artist", "Album", 180.0)
    private suspend fun finishRequests() = currentCoroutineContext()[Job]!!.children.toList().joinAll()

    @Test fun `refresh retries failed save before reading server again`() = runBlocking {
        var reads = 0
        val writes = mutableListOf<List<AudioTrack>>()
        val remote = listOf(track("remote"))
        val local = listOf(track("local"))
        val requests = MusicQueueRequests(this, true, { reads++; remote }, {
            writes += it
            if (writes.size == 1) throw IOException("offline")
        })
        requests.persist(local)
        finishRequests()
        assertNotNull(requests.state.value.error)
        requests.refresh()
        finishRequests()
        assertEquals(listOf(local, local), writes)
        assertEquals(0, reads)
        assertEquals(local, requests.state.value.tracks)
        assertNull(requests.state.value.error)
        assertFalse(requests.state.value.saving)
        requests.refresh()
        finishRequests()
        assertEquals(1, reads)
        assertEquals(remote, requests.state.value.tracks)
    }

    @Test fun `failed clear remains an intentional empty queue on repeated retry`() = runBlocking {
        val writes = mutableListOf<List<AudioTrack>>()
        val requests = MusicQueueRequests(this, true, { error("must not reload") }, {
            writes += it
            throw IOException("offline")
        })
        requests.persist(emptyList())
        finishRequests()
        repeat(2) { requests.refresh(); finishRequests() }
        assertEquals(3, writes.size)
        assertTrue(writes.all { it.isEmpty() })
        assertTrue(requests.state.value.loaded)
        assertTrue(requests.state.value.tracks.isEmpty())
        assertNotNull(requests.state.value.error)
    }
}
