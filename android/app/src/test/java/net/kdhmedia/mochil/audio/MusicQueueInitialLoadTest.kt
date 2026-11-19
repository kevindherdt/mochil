package net.kdhmedia.mochil.audio

import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.yield
import org.junit.Assert.*
import org.junit.Test

class MusicQueueInitialLoadTest {
    private fun track(id: String) = AudioTrack(id, "music", id, "Artist", "Album", 180.0)

    @Test fun `rapid unloaded appends share a read and preserve saved queue order`() = runBlocking {
        val read = CompletableDeferred<List<AudioTrack>>()
        val saved = CompletableDeferred<List<AudioTrack>>()
        var reads = 0
        val requests = MusicQueueRequests(this, true, { reads++; read.await() }, { saved.complete(it) })
        requests.edit { queueWithAppended(it, track("two")) }
        requests.edit { queueWithAppended(it, track("three")) }
        yield()
        assertEquals(1, reads)
        assertFalse(saved.isCompleted)
        read.complete(listOf(track("one")))
        assertEquals(listOf("one", "two", "three"), saved.await().map { it.id })
    }

    @Test fun `failed read never writes and does not silently replay failed edits later`() = runBlocking {
        var fail = true
        val writes = mutableListOf<List<AudioTrack>>()
        val requests = MusicQueueRequests(this, true, { if (fail) error("offline") else listOf(track("one")) }, { writes += it })
        requests.edit { queueWithAppended(it, track("two")) }
        yield()
        assertTrue(writes.isEmpty())
        assertFalse(requests.state.value.loaded)
        assertNotNull(requests.state.value.error)
        fail = false
        requests.refresh()
        yield()
        assertEquals(listOf("one"), requests.state.value.tracks.map { it.id })
        assertTrue(writes.isEmpty())
    }

    @Test fun `explicit replacement supersedes pending append while initial read is delayed`() = runBlocking {
        val read = CompletableDeferred<List<AudioTrack>>()
        val writes = mutableListOf<List<AudioTrack>>()
        val requests = MusicQueueRequests(this, true, { read.await() }, { writes += it })
        requests.edit { queueWithAppended(it, track("append")) }
        yield()
        requests.persist(listOf(track("replacement")))
        read.complete(listOf(track("old")))
        yield()
        yield()
        assertEquals(listOf("replacement"), requests.state.value.tracks.map { it.id })
        assertEquals(listOf(listOf(track("replacement"))), writes)
    }
}
