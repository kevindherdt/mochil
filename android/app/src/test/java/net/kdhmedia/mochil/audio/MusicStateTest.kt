package net.kdhmedia.mochil.audio

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class MusicStateTest {
    @Test fun `audio identities cannot alias colon-containing server ids`() {
        assertFalse(audioTrackIdentityKey("a:b", "c") == audioTrackIdentityKey("a", "b:c"))
    }
    @Test fun `audiobook flat destinations use track pagination`() {
        assertTrue(MusicGrouping.Continue.isPagedTrackView())
        assertTrue(MusicGrouping.Recent.isPagedTrackView())
        assertTrue(MusicGrouping.Tracks.isPagedTrackView())
        assertFalse(MusicGrouping.Series.isPagedTrackView())
        assertFalse(MusicGrouping.Authors.isPagedTrackView())
    }
    private val first = AudioTrack("1", "music", "First", "Artist", "Album", 60.0)
    private val second = AudioTrack("2", "music", "Second", "Other", "Elsewhere", 70.0)

    @Test fun downloadedViewUsesOfflineMetadataWithoutCatalogueResults() {
        val state = MusicState(libraryId = "music", downloadedOnly = true, offlineTracks = listOf(second, first))
        assertEquals(listOf(second, first), state.visibleTracks())
    }

    @Test fun downloadedAlbumViewStaysScopedToTheOpenAlbum() {
        val album = AudioGroup("Album", "Artist", 1, "1", "music")
        val state = MusicState(
            libraryId = "music", grouping = MusicGrouping.Albums, selectedGroup = album,
            downloadedOnly = true, offlineTracks = listOf(first, second),
        )
        assertEquals(listOf(first), state.visibleTracks())
    }

    @Test fun downloadedAudiobookViewStaysScopedToSeries() {
        val saga = AudioGroup("Saga", "Author", 1, "book", "audiobooks")
        val inSeries = AudioTrack("book", "audiobooks", "Volume 1", "Author", "", 600.0, series = "Saga")
        val other = AudioTrack("other", "audiobooks", "Other", "Author", "", 500.0, series = "Elsewhere")
        val state = MusicState(
            libraryId = "audiobooks", libraryKind = "audiobooks", grouping = MusicGrouping.Series,
            selectedGroup = saga, downloadedOnly = true, offlineTracks = listOf(other, inSeries),
        )
        assertEquals(listOf(inSeries), state.visibleTracks())
    }

    @Test fun offlineFallbackUsesDownloadsFromTheSelectedLibrary() {
        val otherLibrary = second.copy(libraryId = "other")
        val fallback = MusicState(libraryId = "music", offlineTracks = listOf(otherLibrary, first)).offlineFallback()

        assertTrue(requireNotNull(fallback).offlineOnly)
        assertTrue(fallback.downloadedOnly)
        assertEquals(MusicGrouping.Tracks, fallback.grouping)
        assertEquals(listOf(first), fallback.visibleTracks())
    }

    @Test fun offlineFallbackIsUnavailableWithoutLocalTracks() {
        assertNull(MusicState(libraryId = "music").offlineFallback())
    }

    @Test fun favoritesViewOnlyShowsFavoritesFromTheSelectedLibrary() {
        val state = MusicState(
            libraryId = "music", grouping = MusicGrouping.Favorites,
            favoriteTracks = listOf(second.copy(libraryId = "other"), first),
        )

        assertEquals(listOf(first), state.visibleTracks())
    }

    @Test fun recentlyPlayedViewOnlyShowsTheSelectedLibraryInServerOrder() {
        val state = MusicState(
            libraryId = "music", grouping = MusicGrouping.RecentlyPlayed,
            discoveryPlayed = listOf(second, first, first.copy(libraryId = "other")),
        )
        assertEquals(listOf(second, first), state.visibleTracks())
    }

    @Test fun favoriteUpdatesAreImmediateAndDeduplicated() {
        val added = MusicState().withFavorite(first, true).withFavorite(first, true)
        assertEquals(setOf(first.identityKey), added.favoriteIds)
        assertEquals(listOf(first), added.favoriteTracks)

        val removed = added.withFavorite(first, false)
        assertTrue(removed.favoriteIds.isEmpty())
        assertTrue(removed.favoriteTracks.isEmpty())
    }
}
