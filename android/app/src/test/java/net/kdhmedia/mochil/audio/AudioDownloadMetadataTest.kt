package net.kdhmedia.mochil.audio

import org.junit.Assert.assertEquals
import org.junit.Test

class AudioDownloadMetadataTest {
    @Test fun `offline audiobook metadata survives a complete round trip`() {
        val track = AudioTrack(
            id = "part-2", libraryId = "audiobooks", title = "Book", artist = "Author", album = "Series",
            durationSeconds = 7_200.0, mediaType = "audio/flac", series = "Saga", seriesIndex = 2.0,
            narrator = "Narrator", libraryKind = "audiobooks", resumePositionSeconds = 912.5,
            description = "Description", chapters = listOf(AudioChapter("Arrival", 900.0, 1_200.0)),
            albumArtist = "Author", genre = "Fantasy", trackNumber = 2, discNumber = 1,
            container = "flac", year = 2024, bitrate = 900_000, codec = "flac", sampleRate = 48_000,
            bitDepth = 24, playbackMode = "direct", playbackQuality = "maximum",
            streamPath = "/stream/part-2", artworkPath = "/art/part-2", serverDownloaded = true,
        )

        assertEquals(track, AudioDownloadMetadata.decode(AudioDownloadMetadata.encode(track)))
    }
}
