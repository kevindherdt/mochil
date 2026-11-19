package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.audio.MusicGrouping
import net.kdhmedia.mochil.catalogue.PublicationGrouping
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class BrowseModeMenuTest {
    @Test fun `track mode is contextual for music and audiobooks`() {
        assertEquals("Tracks", musicGroupingLabel(MusicGrouping.Tracks, "music"))
        assertEquals("All audiobooks", musicGroupingLabel(MusicGrouping.Tracks, "audiobooks"))
        assertEquals("Favorites", musicGroupingLabel(MusicGrouping.Favorites, "music"))
        assertEquals("Recently played", musicGroupingLabel(MusicGrouping.RecentlyPlayed, "music"))
    }

    @Test fun `collections render their own root and detail empty states`() {
        assertFalse(publicationGroupingUsesBookEmptyState(PublicationGrouping.Collections))
        assertTrue(publicationGroupingUsesBookEmptyState(PublicationGrouping.Books))
        assertTrue(publicationGroupingUsesBookEmptyState(PublicationGrouping.Continue))
    }
}
