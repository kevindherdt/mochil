package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.audio.AudioGroup
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.audio.MusicGrouping
import net.kdhmedia.mochil.audio.MusicState
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class TvAudioNavigationTest {
    @Test fun `artist drill down replaces artists with that artists albums`() {
        val artist = AudioGroup("Paramore", "", 3, "", "music")
        val albums = listOf(AudioGroup("Riot!", "Paramore", 11, "", "music"))
        val state = MusicState(
            grouping = MusicGrouping.Artists,
            groups = listOf(artist),
            selectedArtist = artist,
            artistAlbums = albums,
        )

        assertEquals(albums, tvAudioGroups(state))
    }

    @Test fun `discover prefers played music and removes duplicates`() {
        val played = track("played")
        val recent = track("recent")
        val state = MusicState(
            grouping = MusicGrouping.Discover,
            discoveryPlayed = listOf(played),
            discoveryRecent = listOf(played, recent),
            favoriteTracks = listOf(recent),
        )

        assertEquals(listOf(played, recent), tvDiscoveryTracks(state))
    }

    @Test fun `discover stops after the bounded TV viewport feed`() {
        val played = (0 until 100).map { track("played-$it") }
        val state = MusicState(grouping = MusicGrouping.Discover, discoveryPlayed = played, discoveryRecent = listOf(track("never-reached")))

        assertEquals(80, tvDiscoveryTracks(state).size)
        assertEquals("played-79", tvDiscoveryTracks(state).last().id)
    }

    @Test fun `favorites playback uses the visible favorites queue`() {
        val favorite = track("favorite")
        val state = MusicState(grouping = MusicGrouping.Favorites, favoriteTracks = listOf(favorite), favoritesOnly = true)

        assertEquals(listOf(favorite), tvPlaybackTracks(state))
    }

    @Test fun `audiobook cards expose progress without adding per-track music progress`() {
        val audiobook = AudioTrack("book", "audio", "Book", "Author", "", 1_000.0, resumePositionSeconds = 250.0)
        assertEquals(.25f, audiobookProgress(audiobook, "audiobooks").fraction)
        assertFalse(audiobookProgress(audiobook, "audiobooks").completed)
        assertEquals(0f, audiobookProgress(audiobook, "music").fraction)
        assertTrue(audiobookProgress(audiobook.copy(finished = true), "audiobooks").completed)
    }

    private fun track(id: String) = AudioTrack(id, "music", id, "Artist", "Album", 180.0)
}
