package net.kdhmedia.mochil.podcast

import android.os.Looper
import java.io.File
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.auth.DeviceCredentialStore
import net.kdhmedia.mochil.audio.identityKey
import org.junit.Assert.*
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.Shadows.shadowOf
import org.robolectric.annotation.Config
import org.robolectric.annotation.Implementation
import org.robolectric.annotation.Implements
import android.content.Context

@Implements(DeviceCredentialStore::class)
class PodcastViewModelCredentialStoreShadow {
    @Implementation fun __constructor__(context: Context) {}
    @Implementation fun load(): DeviceCredential? = null
}

/**
 * downloads.ids() ultimately reads a real OfflineAudioStore entry, including a disk/Binder round
 * trip for externally-stored episodes -- it must never run synchronously on the thread that
 * constructs or re-opens the ViewModel (that thread is the UI main thread in production).
 */

/**
 * Runs work only when the test pumps it, so "the lookup did not happen inline" is observed rather
 * than sampled. Real IO dispatchers complete whenever they feel like it, which is how the old
 * assertion passed locally and failed on a loaded CI runner.
 */
private class PumpDispatcher : kotlinx.coroutines.CoroutineDispatcher() {
    private val queued = ArrayDeque<Runnable>()
    override fun dispatch(context: kotlin.coroutines.CoroutineContext, block: Runnable) {
        synchronized(queued) { queued.addLast(block) }
    }
    fun queuedCount(): Int = synchronized(queued) { queued.size }
    fun pump() {
        while (true) {
            val next = synchronized(queued) { queued.removeFirstOrNull() } ?: return
            next.run()
        }
    }
}

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], shadows = [PodcastViewModelCredentialStoreShadow::class])
class PodcastViewModelDownloadedIdsDispatchTest {
    private val io = PumpDispatcher()

    private fun seedOneOfflineEpisode() {
        val app = RuntimeEnvironment.getApplication()
        val dir = File(app.filesDir, "music").apply { mkdirs() }
        val seeded = File(dir, "seeded.mp3").apply { writeText("fake audio bytes") }
        app.getSharedPreferences("music_downloads", Context.MODE_PRIVATE).edit()
            .putString("items", """[{"key":"seeded-episode","path":"${seeded.absolutePath}","external_uri":null}]""")
            .commit()
    }

    @Test fun constructingTheViewModelDoesNotBlockOnDownloadedIdsBeforeTheMainLooperIdles() {
        seedOneOfflineEpisode()
        val vm = PodcastViewModel(RuntimeEnvironment.getApplication(), io)
        // The lookup must be *dispatched*, not run on the calling (main) thread. Let the main looper
        // deliver the launch, then look at what it queued instead of sampling state.value, which only
        // says who won a race.
        shadowOf(Looper.getMainLooper()).idle()
        assertTrue(
            "construction must queue the download lookup off the calling thread, not compute it inline",
            io.queuedCount() > 0,
        )
        assertTrue("the lookup is queued, so nothing can be known yet", vm.state.value.downloadedIds.isEmpty())
        waitForIds(vm)
        assertEquals(setOf("seeded-episode"), vm.state.value.downloadedIds)
    }

    @Test fun openingALibraryDoesNotBlockOnDownloadedIdsBeforeTheMainLooperIdles() {
        seedOneOfflineEpisode()
        val vm = PodcastViewModel(RuntimeEnvironment.getApplication(), io)
        waitForIds(vm)
        assertEquals(setOf("seeded-episode"), vm.state.value.downloadedIds)

        val queuedBefore = io.queuedCount()
        vm.open("library-2")
        shadowOf(Looper.getMainLooper()).idle()
        assertTrue(
            "open() must queue the download lookup off the calling thread instead of computing it inline",
            io.queuedCount() > queuedBefore,
        )
        waitForIds(vm)
        assertEquals(setOf("seeded-episode"), vm.state.value.downloadedIds)
    }

    private fun waitForIds(vm: PodcastViewModel) {
        val deadline = System.nanoTime() + java.util.concurrent.TimeUnit.SECONDS.toNanos(5)
        while (vm.state.value.downloadedIds.isEmpty() && System.nanoTime() < deadline) {
            io.pump()
            shadowOf(Looper.getMainLooper()).idle()
            Thread.sleep(10)
        }
    }

    // playbackUrl() is now a suspend fun so it can withContext(Dispatchers.IO) the same disk/Binder
    // lookup that ids() does; this proves the IO-wrap didn't break the actual resolved value.
    @Test fun playbackUrlResolvesASeededOfflineEpisode() = kotlinx.coroutines.runBlocking {
        val app = RuntimeEnvironment.getApplication()
        val track = net.kdhmedia.mochil.audio.AudioTrack(
            id = "ep1", libraryId = "podcasts-lib", title = "Episode 1", artist = "", album = "",
            durationSeconds = 120.0, libraryKind = "podcasts", feedId = "feed1",
        )
        val dir = File(app.filesDir, "music").apply { mkdirs() }
        val file = File(dir, "ep1.mp3").apply { writeText("fake audio bytes") }
        val entry = org.json.JSONObject()
            .put("key", track.identityKey)
            .put("path", file.absolutePath)
            .put("track", net.kdhmedia.mochil.audio.AudioDownloadMetadata.encode(track))
        app.getSharedPreferences("music_downloads", Context.MODE_PRIVATE).edit()
            .putString("items", org.json.JSONArray().put(entry).toString())
            .commit()

        val vm = PodcastViewModel(app)
        val url = vm.playbackUrl(track)
        assertTrue("expected a file:// URL for the seeded local download, got: $url", url.startsWith("file:"))
        assertTrue(url.endsWith("ep1.mp3"))
    }
}
