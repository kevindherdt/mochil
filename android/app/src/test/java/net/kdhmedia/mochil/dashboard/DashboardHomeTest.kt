package net.kdhmedia.mochil.dashboard

import net.kdhmedia.mochil.audio.AudioTrack
import org.junit.Assert.assertEquals
import org.junit.Test
import org.json.JSONObject

class DashboardHomeTest {
    @Test fun synchronizedListeningPositionIsCopiedIntoPlayableTrack() {
        val item = continueListening(JSONObject("""{"id":"episode","library_id":"podcasts","title":"Episode","artist":"Show","library_kind":"podcasts","duration_seconds":1200,"progress":{"position_seconds":345,"duration_seconds":1200}}"""))!!
        assertEquals(345.0, item.positionSeconds, 0.0)
        assertEquals(345.0, item.track.resumePositionSeconds, 0.0)
    }

    @Test fun playTimeProgressRefreshUpdatesWrapperAndPlayableTrack() {
        val original = ContinueListening(AudioTrack("episode", "podcasts", "Episode", "Show", "Show", 1200.0, libraryKind = "podcasts", resumePositionSeconds = 20.0), 20.0, 1200.0)
        val refreshed = listeningAt(original, 456.0, 1200.0)
        assertEquals(456.0, refreshed.positionSeconds, 0.0)
        assertEquals(456.0, refreshed.track.resumePositionSeconds, 0.0)
    }

    @Test fun playTimeRefreshRetainsKnownDurationWhenRemoteDurationIsMissing() {
        val original = ContinueListening(AudioTrack("episode", "podcasts", "Episode", "Show", "Show", 1200.0, libraryKind = "podcasts"), 20.0, 1200.0)
        val refreshed = listeningAt(original, 456.0, 0.0)
        assertEquals(1200.0, refreshed.durationSeconds, 0.0)
        assertEquals(456.0, refreshed.track.resumePositionSeconds, 0.0)
    }
    @Test fun eachHomeSectionKeepsOnlyFiveNewestItems() {
        assertEquals(listOf(1, 2, 3, 4, 5), homeItems((1..12).toList()))
    }

    @Test fun shortHomeSectionsRemainUnchanged() {
        assertEquals(listOf("one", "two"), homeItems(listOf("one", "two")))
    }

    @Test fun homeListeningKeepsLongFormMediaAndExcludesIndividualSongs() {
        fun item(kind: String) = ContinueListening(AudioTrack("id", kind, "Title", "Artist", "Album", 1200.0, libraryKind = kind), 20.0, 1200.0)
        assertEquals(false, isLongFormListening(item("music")))
        assertEquals(true, isLongFormListening(item("podcasts")))
        assertEquals(true, isLongFormListening(item("audiobooks")))
    }
}
