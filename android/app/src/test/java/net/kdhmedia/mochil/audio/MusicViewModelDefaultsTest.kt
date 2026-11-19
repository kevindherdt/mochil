package net.kdhmedia.mochil.audio

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class MusicViewModelDefaultsTest {
    @Test
    fun audiobooksOpenOnPopulatedSeriesShelf() {
        assertEquals(MusicGrouping.Series, initialAudioGrouping("audiobooks", MusicGrouping.Continue))
        assertEquals(MusicGrouping.Series, initialAudioGrouping("audiobooks", MusicGrouping.Discover))
    }

    @Test
    fun musicRetainsSupportedGrouping() {
        assertEquals(MusicGrouping.Artists, initialAudioGrouping("music", MusicGrouping.Artists))
        assertEquals(MusicGrouping.Discover, initialAudioGrouping("music", MusicGrouping.Series))
    }

    @Test
    fun reloadsAnEmptyOrFailedSameLibraryButNotLoadedContent() {
        val empty = MusicState(libraryId = "audio", libraryKind = "audiobooks", loading = false)
        assertTrue(shouldReloadAudioLibrary(empty, "audiobooks"))
        assertTrue(shouldReloadAudioLibrary(empty.copy(error = "failed"), "audiobooks"))
        assertFalse(shouldReloadAudioLibrary(empty.copy(groups = listOf(AudioGroup("Series", "", 1, "", "", "id"))), "audiobooks"))
    }
}
