package net.kdhmedia.mochil.ui

import java.io.File
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class MobilePlaybackIsolationSourceTest {
    private fun source(name: String) = sequenceOf(
        File("src/main/java/net/kdhmedia/mochil/ui/$name"),
        File("app/src/main/java/net/kdhmedia/mochil/ui/$name"),
        File("android/app/src/main/java/net/kdhmedia/mochil/ui/$name"),
    ).first(File::isFile).readText()

    @Test fun `standalone library does not subscribe its root to playback ticks`() {
        val localApp = source("LocalFirstApp.kt")
        assertTrue(localApp.contains("MobileMiniPlayerHost(playbackViewModel)"))
        assertFalse(localApp.contains("playbackViewModel.state.collectAsStateWithLifecycle"))
    }

    @Test fun `mobile mini player separates stable dock and timeline state`() {
        val host = source("MobilePlaybackHost.kt")
        val miniHost = host.substringBefore("internal fun MobileExpandedPlayerHost")
        assertTrue(miniHost.contains("playbackViewModel.dockState.collectAsStateWithLifecycle"))
        assertTrue(miniHost.contains("playbackViewModel.timelineState"))
        assertFalse(miniHost.contains("playbackViewModel.state.collectAsStateWithLifecycle"))
    }

    @Test fun `queue and playlists do not subscribe to playback timeline ticks`() {
        val expandedHost = source("MobilePlaybackHost.kt").substringAfter("internal fun MobileExpandedPlayerHost")
        assertTrue(expandedHost.contains("playbackViewModel.currentTrack.collectAsStateWithLifecycle"))
        assertTrue(expandedHost.contains("playbackViewModel.playerState.collectAsStateWithLifecycle"))
        assertTrue(expandedHost.contains("playbackViewModel.timelineState"))
        assertFalse(expandedHost.contains("playbackViewModel.state.collectAsStateWithLifecycle"))
    }

    @Test fun `queue recovery reaches request owner rather than lazy ensure loaded`() {
        val host = source("MobilePlaybackHost.kt")
        assertTrue(host.contains("onRetryQueue = queueViewModel::refresh"))
        val panel = source("MusicQueuePanel.kt")
        assertTrue(panel.contains("TextButton(onClick = onRetry, enabled = !state.loading && !state.saving)"))
        assertTrue(panel.contains("state.loaded && !state.loading && state.error == null && state.tracks.isEmpty()"))
        assertTrue(panel.contains("liveRegion = LiveRegionMode.Polite"))
    }
}
