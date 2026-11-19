package net.kdhmedia.mochil.ui

import android.view.KeyEvent
import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Test
import net.kdhmedia.mochil.catalogue.LibrarySummary
import net.kdhmedia.mochil.catalogue.LibraryState
import net.kdhmedia.mochil.catalogue.PublicationCollection
import net.kdhmedia.mochil.catalogue.PublicationGrouping
import net.kdhmedia.mochil.catalogue.BookSummary
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.audio.MusicState
import net.kdhmedia.mochil.podcast.PodcastState
import net.kdhmedia.mochil.video.VideoState

class TvInputNavigationTest {
    private fun source(name: String) = sequenceOf(
        File("src/main/java/net/kdhmedia/mochil/ui/$name"),
        File("app/src/main/java/net/kdhmedia/mochil/ui/$name"),
        File("android/app/src/main/java/net/kdhmedia/mochil/ui/$name"),
    ).first(File::isFile).readText()

    @Test fun shoulderButtonsCycleLibraries() {
        assertEquals(TvLibraryDirection.Previous, tvLibraryDirection(KeyEvent.KEYCODE_BUTTON_L1))
        assertEquals(TvLibraryDirection.Previous, tvLibraryDirection(KeyEvent.KEYCODE_BUTTON_L2))
        assertEquals(TvLibraryDirection.Next, tvLibraryDirection(KeyEvent.KEYCODE_BUTTON_R1))
        assertEquals(TvLibraryDirection.Next, tvLibraryDirection(KeyEvent.KEYCODE_BUTTON_R2))
        assertNull(tvLibraryDirection(KeyEvent.KEYCODE_MEDIA_NEXT))
    }

    @Test fun cyclingWrapsAndHandlesEmptyLibraries() {
        assertEquals(2, adjacentLibraryIndex(0, 3, TvLibraryDirection.Previous))
        assertEquals(0, adjacentLibraryIndex(2, 3, TvLibraryDirection.Next))
        assertEquals(-1, adjacentLibraryIndex(0, 0, TvLibraryDirection.Next))
        assertEquals(-1, adjacentLibraryIndex(0, 1, TvLibraryDirection.Next))
        assertEquals(-1, adjacentLibraryIndex(0, 1, TvLibraryDirection.Previous))
    }

    @Test fun cyclingStartsAtTheNearestEdgeWhenSelectionIsNoLongerVisible() {
        assertEquals(0, adjacentLibraryIndex(-1, 3, TvLibraryDirection.Next))
        assertEquals(2, adjacentLibraryIndex(-1, 3, TvLibraryDirection.Previous))
        assertEquals(0, adjacentLibraryIndex(7, 3, TvLibraryDirection.Next))
        assertEquals(2, adjacentLibraryIndex(7, 3, TvLibraryDirection.Previous))
    }

    @Test fun `library passes shoulder input through when no adjacent shelf exists`() {
        assertTrue(source("TvLibraryScreen.kt").contains("if (next < 0) return@onPreviewKeyEvent false"))
    }

    @Test fun railHighlightsWhereYouAre() {
        assertEquals(TV_RAIL_HOME, tvRailSelectedKey("Home", "music"))
        assertEquals(TV_RAIL_SETTINGS, tvRailSelectedKey("Settings", "music"))
        assertEquals(tvRailLibraryKey("music"), tvRailSelectedKey("Library", "music"))
        assertNull(tvRailSelectedKey("Library", null))
        assertNull(tvRailSelectedKey("NowPlaying", "music"))
    }

    @Test fun everyLibraryKindHasARailIcon() {
        val kinds = listOf("movies", "tv", "anime", "music", "audiobooks", "podcasts", "books", "manga", "comics", "calibre", "mixed", "lightnovels", "unknown")
        kinds.forEach { assertNotNull(tvRailIcon(it)) }
        assertNotEquals(tvRailIcon("movies"), tvRailIcon("music"))
    }

    @Test fun hierarchyFocusIdentityIsUnambiguousAndIgnoresNoOtherRouteState() {
        assertNotEquals(tvHierarchyKey("a:b", "c"), tvHierarchyKey("a", "b:c"))
        assertEquals(tvHierarchyKey("Artist", "Album"), tvHierarchyKey("Artist", "Album"))
    }

    @Test fun contentRouteIdentityChangesWhenTheLibraryChangesAtTheSameHierarchy() {
        val root = tvHierarchyKey(null, null, null)

        assertNotEquals(tvContentRouteKey("music", root), tvContentRouteKey("podcasts", root))
        assertEquals(tvContentRouteKey("music", root), tvContentRouteKey("music", root))
    }

    @Test fun nowPlayingFocusReturnsToTransportAfterClosingQueueWithoutFollowingTrackChanges() {
        val session = TvNowPlayingFocusSession()
        assertTrue(session.shouldFocusTransport(queueVisible = false))
        assertFalse(session.shouldFocusTransport(queueVisible = false))
        assertFalse(session.shouldFocusTransport(queueVisible = true))
        assertFalse(session.shouldFocusTransport(queueVisible = true))
        assertTrue(session.shouldFocusTransport(queueVisible = false))
        assertFalse(session.shouldFocusTransport(queueVisible = false))
    }

    @Test fun `tv search remains compact and announces an active query`() {
        assertEquals("Search this library", tvSearchButtonLabel(""))
        assertEquals("Search this library", tvSearchButtonLabel("   "))
        assertEquals("Edit search: Paramore", tvSearchButtonLabel("  Paramore  "))
        assertEquals(93, tvSearchButtonLabel("a".repeat(100)).length)
    }

    @Test fun publicationBackClosesExactlyOneHierarchyLevelBeforeHome() {
        assertEquals(TvPublicationBackTarget.Collection, tvPublicationBackTarget(hasCollection = true, hasSeries = true, hasAuthor = true))
        assertEquals(TvPublicationBackTarget.Series, tvPublicationBackTarget(hasCollection = false, hasSeries = true, hasAuthor = true))
        assertEquals(TvPublicationBackTarget.Author, tvPublicationBackTarget(hasCollection = false, hasSeries = false, hasAuthor = true))
        assertEquals(TvPublicationBackTarget.Home, tvPublicationBackTarget(hasCollection = false, hasSeries = false, hasAuthor = false))
    }

    @Test fun `open collections are publication groups with their own focus route and books`() {
        val book = BookSummary(
            id = "book-1",
            libraryId = "books",
            title = "Book",
            authors = emptyList(),
            series = "",
            seriesIndex = null,
            format = "epub",
            mediaType = "application/epub+zip",
            language = "",
            description = "",
            size = 1,
        )
        val collection = PublicationCollection("later", "Read later", "books", 1, emptyList())
        val state = LibraryState(
            grouping = PublicationGrouping.Collections,
            selectedCollection = collection,
            books = emptyList(),
            seriesBooks = listOf(book),
        )

        assertTrue(tvPublicationInsideGroup(state))
        assertTrue(tvPublicationShowsBooks(state))
        assertEquals(listOf(book), tvPublicationBooks(state))
        assertTrue(tvHasVisibleFocusKey("book:book-1", state, MusicState(), VideoState(), PodcastState()))
        assertFalse(tvHasVisibleFocusKey("book:missing", state, MusicState(), VideoState(), PodcastState()))
        assertFalse(tvHasVisibleFocusKey(null, state, MusicState(), VideoState(), PodcastState()))
        assertNotEquals(
            tvHierarchyKey(null, null, null),
            tvHierarchyKey(collection.id, null, null),
        )
    }

    @Test fun podcastCardsExposeBoundedResumeAndPlayedProgress() {
        val listening = AudioTrack(
            libraryId = "podcasts",
            id = "episode-1",
            title = "Episode",
            artist = "Show",
            album = "",
            durationSeconds = 1_000.0,
            mediaType = "audio/mpeg",
            resumePositionSeconds = 275.0,
        )
        assertEquals(.275f, tvPodcastProgress(listening).fraction, .0001f)
        assertFalse(tvPodcastProgress(listening).completed)
        assertEquals(1f, tvPodcastProgress(listening.copy(resumePositionSeconds = 2_000.0)).fraction, .0001f)
        assertTrue(tvPodcastProgress(listening.copy(finished = true)).completed)
    }
}
