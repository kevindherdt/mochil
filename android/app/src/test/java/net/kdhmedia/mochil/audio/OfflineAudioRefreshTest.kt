package net.kdhmedia.mochil.audio

import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.NonCancellable
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.withContext
import org.junit.Assert.assertEquals
import org.junit.Test

class OfflineAudioRefreshTest {
    @Test fun `storage failure preserves previous snapshot and permits retry`() = runBlocking {
        val failures = mutableListOf<String?>()
        val published = mutableListOf<AudioDownloadSnapshot>()
        val snapshot = AudioDownloadSnapshot(setOf("saved"), emptyList())
        var attempts = 0
        val refresh = OfflineAudioRefresh(
            CoroutineScope(coroutineContext + Dispatchers.Unconfined),
            load = { if (attempts++ == 0) throw SecurityException("Storage access revoked"); snapshot },
            publish = { _, value -> published += value },
            onFailure = { library, _ -> failures += library },
        )
        refresh.refresh("music")
        assertEquals(listOf("music"), failures)
        assertEquals(emptyList<AudioDownloadSnapshot>(), published)
        refresh.refresh("music")
        assertEquals(listOf(snapshot), published)
    }

    @Test fun `cancelled refresh is not reported as a storage failure`() = runBlocking {
        val failures = mutableListOf<Exception>()
        val refresh = OfflineAudioRefresh(
            CoroutineScope(coroutineContext + Dispatchers.Unconfined),
            load = { throw kotlinx.coroutines.CancellationException("Leaving screen") },
            publish = { _, _ -> throw AssertionError("Cancelled refresh published") },
            onFailure = { _, failure -> failures += failure },
        )
        refresh.refresh("music")
        assertEquals(emptyList<Exception>(), failures)
    }

    @Test fun `startup result cannot replace the subsequently opened library`() = runBlocking {
        verifyLatest(null, "music")
    }

    @Test fun `older refresh cannot overwrite newer downloads in the same library`() = runBlocking {
        verifyLatest("music", "music")
    }

    private suspend fun verifyLatest(firstLibrary: String?, secondLibrary: String?) {
        val first = CompletableDeferred<AudioDownloadSnapshot>()
        val second = CompletableDeferred<AudioDownloadSnapshot>()
        val published = mutableListOf<Pair<String?, AudioDownloadSnapshot>>()
        var calls = 0
        val refresh = OfflineAudioRefresh(
            CoroutineScope(kotlin.coroutines.coroutineContext + Dispatchers.Unconfined),
            // Model a storage operation that finishes despite cancellation.
            load = { val pending = if (calls++ == 0) first else second; withContext(NonCancellable) { pending.await() } },
            publish = { library, snapshot -> published += library to snapshot },
            onFailure = { _, failure -> throw AssertionError("Unexpected storage failure", failure) },
        )
        refresh.refresh(firstLibrary)
        refresh.refresh(secondLibrary)
        val newest = AudioDownloadSnapshot(setOf("new"), emptyList())
        second.complete(newest)
        first.complete(AudioDownloadSnapshot(setOf("old"), emptyList()))
        assertEquals(listOf(secondLibrary to newest), published)
    }
}
