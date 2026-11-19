package net.kdhmedia.mochil.playback

import java.io.IOException
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.cancel
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.launch
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class AudioProgressReplayTest {
    private val entries = List(3) { PendingAudioProgress("podcasts", "$it", 30.0, 600.0) }

    @Test fun `only successful sends acknowledge the exact captured entry`() = runBlocking {
        val sent = mutableListOf<PendingAudioProgress>()
        val acknowledged = mutableListOf<PendingAudioProgress>()
        replayPendingAudioProgress(entries, { entry ->
            sent += entry
            when (entry.itemId) {
                "0" -> throw IOException("Offline")
                "1" -> false
                else -> true
            }
        }, acknowledged::add)
        assertEquals(entries, sent)
        assertEquals(listOf(entries.last()), acknowledged)
    }

    @Test fun `cancellation during blocking send does not acknowledge or start later sends`() = runBlocking {
        val sent = mutableListOf<PendingAudioProgress>()
        val acknowledged = mutableListOf<PendingAudioProgress>()
        val job = launch {
            val context = currentCoroutineContext()
            replayPendingAudioProgress(entries, { entry ->
                sent += entry
                context.cancel()
                true
            }, acknowledged::add)
        }
        job.join()
        assertTrue(job.isCancelled)
        assertEquals(listOf(entries.first()), sent)
        assertTrue(acknowledged.isEmpty())
    }

    @Test fun `explicit cancellation exception stops replay instead of behaving like network failure`() = runBlocking {
        var sent = 0
        val job = launch {
            replayPendingAudioProgress(entries, { sent++; throw CancellationException("Stopped") }, { error("Must not acknowledge") })
        }
        job.join()
        assertTrue(job.isCancelled)
        assertEquals(1, sent)
    }

    @Test fun `already cancelled replay does not send anything`() = runBlocking {
        val job = launch {
            currentCoroutineContext().cancel()
            replayPendingAudioProgress(entries, { error("Must not send") }, { error("Must not acknowledge") })
        }
        job.join()
        assertTrue(job.isCancelled)
    }
}
