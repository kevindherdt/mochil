package net.kdhmedia.mochil.audio

import net.kdhmedia.mochil.auth.DeviceCredential
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.launch
import kotlinx.coroutines.cancel

class MusicDownloadCoordinatorTest {
    private val credential = DeviceCredential("https://example.test", "phone", "token")
    private val first = AudioTrack("1", "music", "First", "Artist", "Album", 60.0)
    private val second = AudioTrack("2", "music", "Second", "Artist", "Album", 70.0)

    @Test fun plansDeduplicateTracksAndRespectExistingDownloads() = runBlocking {
        val store = FakeAudioDownloadStore(mutableSetOf(first.downloadKey))
        val coordinator = MusicDownloadCoordinator(store) { _, track -> "https://example.test/${track.id}" }

        assertEquals(listOf(second), coordinator.pending(listOf(first, second, second)))
        assertEquals(listOf(first), coordinator.downloaded(listOf(first, first, second)))
        assertEquals(2, store.snapshotReads)
        assertEquals(0, store.containsReads)
    }

    @Test fun batchSelectionReadsOneSnapshotOffCallerThread() = runBlocking {
        val caller = Thread.currentThread()
        val store = FakeAudioDownloadStore()
        val coordinator = MusicDownloadCoordinator(store) { _, _ -> error("selection must not start transfers") }
        val tracks = (1..1000).map { first.copy(id = it.toString()) }
        assertEquals(tracks, coordinator.pending(tracks))
        assertEquals(1, store.snapshotReads)
        assertEquals(0, store.containsReads)
        assertTrue(store.snapshotThread !== caller)
        assertEquals(emptyList<AudioTrack>(), coordinator.pending(emptyList()))
        assertEquals(1, store.snapshotReads)
    }

    @Test fun batchStopsAtFirstFailureAndReportsCompletedWork() = runBlocking {
        val store = FakeAudioDownloadStore(failDownload = second.downloadKey)
        val coordinator = MusicDownloadCoordinator(store) { _, track -> "https://example.test/${track.id}" }
        val progress = mutableListOf<String>()

        val result = coordinator.downloadBatch(credential, listOf(first, second)) { _, track -> progress += track.id }

        assertEquals(1, result.completed)
        assertEquals(listOf("1"), progress)
        assertTrue(result.failure != null)
        assertTrue(first.downloadKey in store.downloaded)
        assertFalse(second.downloadKey in store.downloaded)
    }

    @Test fun toggleRemovesExistingTrackWithoutAConnectedServer() {
        val store = FakeAudioDownloadStore(mutableSetOf(first.downloadKey))
        val coordinator = MusicDownloadCoordinator(store) { _, _ -> error("stream URL must not be requested") }

        assertFalse(coordinator.toggle(null, first))
        assertTrue(store.downloaded.isEmpty())
    }

    @Test fun cancelledBatchDoesNotStartAnotherDownload() = runBlocking {
        val store = FakeAudioDownloadStore()
        val coordinator = MusicDownloadCoordinator(store) { _, track -> "https://example.test/${track.id}" }
        val job = launch {
            coordinator.downloadBatch(credential, listOf(first, second)) { _, _ -> cancel() }
        }
        job.join()
        assertTrue(job.isCancelled)
        assertEquals(setOf(first.downloadKey), store.downloaded)
    }

    @Test fun cancelledRemovalPreservesRemainingFiles() = runBlocking {
        val store = FakeAudioDownloadStore(mutableSetOf(first.downloadKey, second.downloadKey))
        val coordinator = MusicDownloadCoordinator(store) { _, _ -> error("no download expected") }
        val job = launch {
            coordinator.removeBatch(listOf(first, second)) { _, _ -> cancel() }
        }
        job.join()
        assertTrue(job.isCancelled)
        assertEquals(setOf(second.downloadKey), store.downloaded)
    }

    @Test fun recoveredPodcastUsesStableFeedIdentityAcrossSharedDownloadActions() = runBlocking {
        val stored = AudioTrack(
            id = "episode-7", libraryId = "podcasts-old", title = "Episode", artist = "Show",
            album = "Show", durationSeconds = 60.0, libraryKind = "podcasts", feedId = "feed-stable",
        )
        val restoredLibraryEpisode = stored.copy(libraryId = "podcasts-new")
        val store = FakeAudioDownloadStore(
            downloaded = mutableSetOf(stored.downloadKey),
            offlineTracks = mutableListOf(stored),
        )
        val coordinator = MusicDownloadCoordinator(store) { _, _ -> error("recovered media must not download again") }

        assertEquals(emptyList<AudioTrack>(), coordinator.pending(listOf(restoredLibraryEpisode)))
        assertEquals(listOf(restoredLibraryEpisode), coordinator.downloaded(listOf(restoredLibraryEpisode)))
        assertFalse(coordinator.toggle(null, restoredLibraryEpisode))
        assertTrue(store.downloaded.isEmpty())
    }
}

class OfflineAudioIdentityTest {
    private val oldPodcast = AudioTrack(
        id = "episode-7",
        libraryId = "podcasts-old",
        title = "Episode",
        artist = "Show",
        album = "Show",
        durationSeconds = 60.0,
        libraryKind = "podcasts",
        feedId = "feed-stable",
    )

    @Test fun `podcast downloads survive a recreated library identity`() {
        assertTrue(sameOfflineAudioIdentity(oldPodcast, oldPodcast.copy(libraryId = "podcasts-new")))
    }

    @Test fun `podcast aliasing never crosses feeds episodes or media kinds`() {
        assertFalse(sameOfflineAudioIdentity(oldPodcast, oldPodcast.copy(feedId = "different-feed", libraryId = "podcasts-new")))
        assertFalse(sameOfflineAudioIdentity(oldPodcast, oldPodcast.copy(id = "different-episode", libraryId = "podcasts-new")))
        assertFalse(sameOfflineAudioIdentity(oldPodcast, oldPodcast.copy(libraryKind = "music", libraryId = "music-new")))
    }
}

private class FakeAudioDownloadStore(
    val downloaded: MutableSet<String> = mutableSetOf(),
    private val failDownload: String? = null,
    private val offlineTracks: MutableList<AudioTrack> = mutableListOf(),
) : AudioDownloadStore {
    var snapshotReads = 0
    var containsReads = 0
    var snapshotThread: Thread? = null
    override fun snapshot(libraryId: String?): AudioDownloadSnapshot {
        snapshotReads++
        snapshotThread = Thread.currentThread()
        return AudioDownloadSnapshot(ids(), tracks(libraryId))
    }
    override fun ids(): Set<String> = downloaded.toSet()
    override fun tracks(libraryId: String?): List<AudioTrack> = offlineTracks.filter { libraryId == null || it.libraryId == libraryId }
    override fun contains(track: AudioTrack): Boolean {
        containsReads++
        return offlineTracks.any { sameOfflineAudioIdentity(it, track) } || track.downloadKey in downloaded
    }
    override fun playbackUrl(track: AudioTrack): String? = null
    override fun download(credential: DeviceCredential, track: AudioTrack, url: String) {
        if (track.downloadKey == failDownload) error("download failed")
        downloaded += track.downloadKey
    }
    override fun remove(track: AudioTrack): Boolean {
        val matching = offlineTracks.filter { sameOfflineAudioIdentity(it, track) }
        offlineTracks.removeAll(matching)
        matching.forEach { downloaded.remove(it.downloadKey) }
        return downloaded.remove(track.downloadKey) || matching.isNotEmpty()
    }
}
