package net.kdhmedia.mochil.ui

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class TvNowPlayingActionTest {
    @Test fun `back closes queue before leaving and never invokes both actions`() {
        val events = mutableListOf<String>()
        navigateTvNowPlayingBack(true, { events += "queue" }, { events += "browse" })
        assertEquals(listOf("queue"), events)
        navigateTvNowPlayingBack(false, { events += "queue" }, { events += "browse" })
        assertEquals(listOf("queue", "browse"), events)
        val screen = source("TvNowPlayingScreen.kt")
        assertTrue(screen.contains("BackHandler(onBack = navigateBack)"))
        assertTrue(screen.contains("\"Back\", navigateBack"))
    }

    @Test fun `playback time supports long audiobooks and invalid negative time`() {
        assertEquals("0:00", formatNowPlayingTime(-1))
        assertEquals("1:12", formatNowPlayingTime(72_000))
        assertEquals("25:01:02", formatNowPlayingTime(90_062_000))
    }

    private fun source(name: String) = sequenceOf(
        File("src/main/java/net/kdhmedia/mochil/ui/$name"),
        File("app/src/main/java/net/kdhmedia/mochil/ui/$name"),
        File("android/app/src/main/java/net/kdhmedia/mochil/ui/$name"),
    ).first(File::isFile).readText()

    private fun playbackSource() = sequenceOf(
        File("src/main/java/net/kdhmedia/mochil/playback/PlaybackViewModel.kt"),
        File("app/src/main/java/net/kdhmedia/mochil/playback/PlaybackViewModel.kt"),
        File("android/app/src/main/java/net/kdhmedia/mochil/playback/PlaybackViewModel.kt"),
    ).first(File::isFile).readText()

    @Test fun `close stops playback before returning to browse`() {
        val events = mutableListOf<String>()

        closeTvNowPlaying(
            closePlayback = { events += "close" },
            onBack = { events += "back" },
        )

        assertEquals(listOf("close", "back"), events)
    }

    @Test fun `TV queue is progressively disclosed and virtualized`() {
        val player = source("TvNowPlayingScreen.kt")
        val queue = source("TvPlaybackQueue.kt")
        assertTrue(player.contains("if (showingQueue)"))
        assertTrue(player.contains("TvPlaybackQueue("))
        assertTrue(player.contains("focusSession.shouldFocusTransport(showingQueue)"))
        assertTrue(queue.contains("LazyColumn"))
        assertTrue(queue.contains("Move earlier"))
        assertTrue(queue.contains("Move later"))
        assertTrue(queue.contains("Remove from queue"))
        assertTrue(queue.contains("rememberLazyListState(initialFirstVisibleItemIndex = initialIndex)"))
    }

    @Test fun `compact TV dock prioritizes media identity and primary actions`() {
        val dock = source("TvPlaybackDock.kt")

        assertTrue(dock.contains("AuthenticatedArtwork("))
        assertTrue(dock.contains("Open Now Playing"))
        assertTrue(dock.contains("Close player"))
        assertTrue(!dock.contains("SkipPrevious"))
        assertTrue(!dock.contains("SkipNext"))
    }

    @Test fun `TV queue opens at a bounded current item`() {
        assertEquals(0, tvQueueInitialIndex(-1, 12))
        assertEquals(7, tvQueueInitialIndex(7, 12))
        assertEquals(11, tvQueueInitialIndex(20, 12))
        assertEquals(0, tvQueueInitialIndex(4, 0))
    }

    @Test fun `library route owns grid memory above the replaceable player surface`() {
        val app = source("TvConnectedApp.kt")
        val route = source("TvLibraryRoute.kt")
        val screen = source("TvLibraryScreen.kt")

        assertTrue(app.contains("val libraryGridPositions = remember { TvGridPositionStore() }"))
        assertTrue(app.contains("gridPositions = libraryGridPositions"))
        assertTrue(route.contains("gridPositions: TvGridPositionStore"))
        assertTrue(screen.contains("gridPositions: TvGridPositionStore"))
        assertTrue(!screen.contains("val gridPositions = remember { TvGridPositionStore() }"))
    }

    @Test fun `restored Media3 sessions publish their queue immediately`() {
        val connection = playbackSource().substringAfter("future.addListener({").substringBefore("}, ContextCompat")
        assertTrue(connection.contains("publishQueue(it)"))
        assertTrue(connection.indexOf("publishQueue(it)") < connection.indexOf("publish(it)"))
    }
}
