package net.kdhmedia.mochil.audio

import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.yield
import org.junit.Assert.*
import org.junit.Test

class MusicFavoriteRequestsTest {
    private val track = AudioTrack("one", "music", "One", "Artist", "Album", 60.0)

    @Test fun `confirmed result survives replacement of visible library state and failed refresh`() = runBlocking {
        var state = MusicState(libraryId = "music")
        val writeGate = CompletableDeferred<Unit>()
        val finalRead = CompletableDeferred<Unit>()
        val requests = MusicFavoriteRequests(this,
            read = { finalRead.complete(Unit); error("offline") },
            write = { _, _ -> writeGate.await() },
            current = { state }, publish = { state = it }, notify = {})
        requests.toggle(track)
        yield()
        state = MusicState(libraryId = "other", loading = true)
        requests.refresh() // Deferred until the pending edit finishes.
        writeGate.complete(Unit)
        finalRead.await()
        assertEquals("other", state.libraryId)
        assertTrue(state.loading)
        assertTrue(track.identityKey in state.favoriteIds)
        assertEquals(listOf(track), state.favoriteTracks)
    }

    @Test fun `old snapshot cannot erase an optimistic toggle`() = runBlocking {
        var state = MusicState()
        val oldRead = CompletableDeferred<Unit>()
        val writeGate = CompletableDeferred<Unit>()
        var reads = 0
        val requests = MusicFavoriteRequests(this,
            read = { if (++reads == 1) { oldRead.await(); emptyList() } else listOf(track) },
            write = { _, _ -> writeGate.await() }, current = { state }, publish = { state = it }, notify = {})
        requests.refresh()
        yield()
        requests.toggle(track)
        oldRead.complete(Unit)
        yield()
        assertTrue(track.identityKey in state.favoriteIds)
        writeGate.complete(Unit)
        Unit
    }

    @Test fun `rapid toggles write in order without overlapping requests`() = runBlocking {
        var state = MusicState()
        val firstWrite = CompletableDeferred<Unit>()
        val finalRead = CompletableDeferred<Unit>()
        val writes = mutableListOf<Boolean>()
        var active = 0
        var peak = 0
        val requests = MusicFavoriteRequests(this, read = { finalRead.complete(Unit); emptyList() },
            write = { _, favorite ->
                active++; peak = maxOf(peak, active)
                writes += favorite
                if (writes.size == 1) firstWrite.await()
                active--
            }, current = { state }, publish = { state = it }, notify = {})
        requests.toggle(track)
        yield()
        requests.toggle(track)
        yield()
        assertEquals(listOf(true), writes)
        firstWrite.complete(Unit)
        finalRead.await()
        assertEquals(listOf(true, false), writes)
        assertEquals(1, peak)
        assertFalse(track.identityKey in state.favoriteIds)
    }

    @Test fun `two failed toggles restore confirmed state even if refresh also fails`() = runBlocking {
        var state = MusicState()
        val firstWrite = CompletableDeferred<Unit>()
        val finalRead = CompletableDeferred<Unit>()
        var writes = 0
        val messages = mutableListOf<String>()
        val requests = MusicFavoriteRequests(this,
            read = { finalRead.complete(Unit); error("offline") },
            write = { _, _ -> if (++writes == 1) firstWrite.await(); error("offline") },
            current = { state }, publish = { state = it }, notify = messages::add)
        requests.toggle(track)
        yield()
        requests.toggle(track)
        firstWrite.complete(Unit)
        finalRead.await()
        assertEquals(2, writes)
        assertFalse(track.identityKey in state.favoriteIds)
        assertEquals(listOf("Could not update this favorite."), messages)
    }

    @Test fun `failed undo retains the successfully confirmed favorite`() = runBlocking {
        var state = MusicState()
        val firstWrite = CompletableDeferred<Unit>()
        val finalRead = CompletableDeferred<Unit>()
        var writes = 0
        val requests = MusicFavoriteRequests(this,
            read = { finalRead.complete(Unit); error("offline") },
            write = { _, _ -> if (++writes == 1) firstWrite.await() else error("offline") },
            current = { state }, publish = { state = it }, notify = {})
        requests.toggle(track)
        yield()
        requests.toggle(track)
        firstWrite.complete(Unit)
        finalRead.await()
        assertTrue(track.identityKey in state.favoriteIds)
    }
}
