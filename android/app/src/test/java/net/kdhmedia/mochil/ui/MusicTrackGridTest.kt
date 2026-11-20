package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.audio.MusicGrouping
import net.kdhmedia.mochil.audio.MusicLayout
import net.kdhmedia.mochil.audio.MusicState
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class MusicTrackGridTest {
    @Test fun favoritesAndHistoryHonorTheCoverLayout() {
        assertTrue(MusicState(grouping = MusicGrouping.Favorites, layout = MusicLayout.Covers).usesTrackCoverGrid())
        assertTrue(MusicState(grouping = MusicGrouping.RecentlyPlayed, layout = MusicLayout.Covers).usesTrackCoverGrid())
        assertFalse(MusicState(grouping = MusicGrouping.Favorites, layout = MusicLayout.List).usesTrackCoverGrid())
    }

    @Test fun groupedAlbumCoversRemainOwnedByTheGroupGrid() {
        assertFalse(MusicState(grouping = MusicGrouping.Albums, layout = MusicLayout.Covers).usesTrackCoverGrid())
    }
}
