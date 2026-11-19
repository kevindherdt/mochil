package net.kdhmedia.mochil.ui

import java.io.File
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class TvRouteOwnershipSourceTest {
    private fun source(name: String) = sequenceOf(
        File("src/main/java/net/kdhmedia/mochil/ui/$name"),
        File("app/src/main/java/net/kdhmedia/mochil/ui/$name"),
        File("android/app/src/main/java/net/kdhmedia/mochil/ui/$name"),
    ).first(File::isFile).readText()

    @Test fun `TV coordinator delegates destination state ownership`() {
        val root = source("TvConnectedApp.kt")
        assertFalse(root.contains("collectAsStateWithLifecycle"))
        assertFalse(root.contains("fun TvHomeRoute("))
        assertFalse(root.contains("fun TvLibraryRoute("))
        assertFalse(root.contains("fun TvSettingsRoute("))
        assertTrue(root.contains("TvHomeRoute("))
        assertTrue(root.contains("TvLibraryRoute("))
        assertTrue(root.contains("TvSettingsRoute("))
    }

    @Test fun `TV destinations collect only their relevant state`() {
        val home = source("TvHomeRoute.kt")
        val library = source("TvLibraryRoute.kt")
        val settings = source("TvSettingsRoute.kt")
        assertTrue(home.contains("dashboardViewModel.state.collectAsStateWithLifecycle"))
        assertTrue(library.contains("libraryViewModel.state.collectAsStateWithLifecycle"))
        assertTrue(library.contains("musicViewModel.state.collectAsStateWithLifecycle"))
        assertTrue(library.contains("podcastViewModel.state.collectAsStateWithLifecycle"))
        assertTrue(library.contains("videoViewModel.state.collectAsStateWithLifecycle"))
        assertTrue(settings.contains("updates.state.collectAsStateWithLifecycle"))
        assertFalse(settings.contains("musicViewModel"))
        assertFalse(settings.contains("videoViewModel"))
    }

    @Test fun `TV publication routing includes collection hierarchy ownership`() {
        val screen = source("TvLibraryScreen.kt")
        val content = source("TvLibraryContent.kt")
        assertTrue(screen.contains("library.selectedCollection?.id"))
        assertTrue(screen.contains("library.selectedCollection?.name"))
        assertTrue(content.contains("state.selectedCollection != null"))
        assertTrue(content.contains("tvPublicationBooks(library)"))
    }

    @Test fun `TV music playback updates persistent and live queues together`() {
        val library = source("TvLibraryRoute.kt")
        val queue = library.substringAfter("val queue = tvPlaybackTracks(music)").substringBefore("},\n                playNext")
        assertTrue(queue.contains("musicQueueViewModel.replace(queue)"))
        assertTrue(queue.contains("playbackViewModel.play(queue, track"))
    }
}
