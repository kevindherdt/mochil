package net.kdhmedia.mochil.audio

import org.junit.Assert.assertEquals
import org.junit.Test

class AudiobookInitialLoadTest {
    @Test fun retriesOneEmptyInitialSeriesResponseOnly() {
        val retries = mutableSetOf<String>()
        val state = MusicState(libraryId = "audio", libraryKind = "audiobooks", grouping = MusicGrouping.Series)
        assertEquals(EmptyAudiobookPageAction.Retry, emptyAudiobookPageAction(state, 0, emptyList(), retries))
        assertEquals(EmptyAudiobookPageAction.ShowAll, emptyAudiobookPageAction(state, 0, emptyList(), retries))
        assertEquals(EmptyAudiobookPageAction.None, emptyAudiobookPageAction(state.copy(query = "book"), 0, emptyList(), mutableSetOf()))
        assertEquals(EmptyAudiobookPageAction.None, emptyAudiobookPageAction(state.copy(libraryKind = "music"), 0, emptyList(), mutableSetOf()))
    }
}
