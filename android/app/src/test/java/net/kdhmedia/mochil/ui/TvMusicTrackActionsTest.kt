package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.audio.AudioTrack
import org.junit.Assert.assertEquals
import org.junit.Test

class TvMusicTrackActionsTest {
    private val music = AudioTrack("track", "music", "Track", "Artist", "Album", 180.0, libraryKind = "music")
    private val audiobook = music.copy(libraryKind = "audiobooks")

    @Test fun musicTrackActionsKeepQueueAndFavoriteChoicesBehindHold() {
        assertEquals(listOf("Play next", "Add to queue", "Add to favorites"), tvMusicTrackActions(music, false).map { it.label })
        assertEquals("Remove from favorites", tvMusicTrackActions(music, true).last().label)
    }

    @Test fun audiobookTrackActionsDoNotOfferTheMusicOnlyPersistentQueue() {
        assertEquals(emptyList<TvMusicTrackAction>(), tvMusicTrackActions(audiobook, false))
    }
}
