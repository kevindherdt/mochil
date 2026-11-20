package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.audio.AudioTrack
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Test

class MusicTrackInfoTest {
    @Test fun detailsContainOnlyUsefulKnownMetadata() {
        val track = AudioTrack(
            id = "one", libraryId = "music", title = "Song", artist = "Artist", album = "Album", durationSeconds = 185.0,
            sizeBytes = 25_165_824, albumArtist = "Various Artists", genre = "Rock", trackNumber = 3, discNumber = 2, container = "flac", year = 2022, bitrate = 1_411_200,
        )
        assertEquals(
            listOf("Artist", "Album artist", "Album", "Genre", "Year", "Disc", "Track", "Format", "Bitrate", "Duration", "File size"),
            trackMetadataDetails(track).map { it.first },
        )
        assertEquals("FLAC", trackMetadataDetails(track).first { it.first == "Format" }.second)
        assertEquals("1411 kbps", trackMetadataDetails(track).first { it.first == "Bitrate" }.second)
        assertEquals("24.0 MB", trackMetadataDetails(track).first { it.first == "File size" }.second)
    }

    @Test fun unknownAndDuplicateArtistValuesStayOutOfTheDialog() {
        val track = AudioTrack("one", "music", "Song", "Unknown artist", "", 0.0, albumArtist = "Unknown artist")
        val labels = trackMetadataDetails(track).map { it.first }
        assertFalse("Artist" in labels)
        assertFalse("Album artist" in labels)
        assertEquals(emptyList<Pair<String, String>>(), trackMetadataDetails(track))
    }
}
