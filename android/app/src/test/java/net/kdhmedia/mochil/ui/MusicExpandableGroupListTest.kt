package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.audio.AudioGroup
import net.kdhmedia.mochil.audio.MusicGrouping
import net.kdhmedia.mochil.audio.MusicLayout
import net.kdhmedia.mochil.audio.MusicState
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class MusicExpandableGroupListTest {
    @Test fun listAlbumsSeriesAndAuthorsExpandInline() {
        assertTrue(MusicState(grouping = MusicGrouping.Albums, layout = MusicLayout.List).usesInlineTrackGroup())
        assertTrue(MusicState(grouping = MusicGrouping.Series, layout = MusicLayout.List).usesInlineTrackGroup())
        assertTrue(MusicState(grouping = MusicGrouping.Authors, layout = MusicLayout.List).usesInlineTrackGroup())
    }

    @Test fun coverGroupsAndArtistAlbumHierarchyKeepDedicatedDetails() {
        assertFalse(MusicState(grouping = MusicGrouping.Albums, layout = MusicLayout.Covers).usesInlineTrackGroup())
        assertFalse(MusicState(grouping = MusicGrouping.Artists, layout = MusicLayout.List).usesInlineTrackGroup())
        assertFalse(MusicState(grouping = MusicGrouping.Albums, layout = MusicLayout.List, selectedArtist = AudioGroup("Artist", "", 2, "", "music")).usesInlineTrackGroup())
    }
}
