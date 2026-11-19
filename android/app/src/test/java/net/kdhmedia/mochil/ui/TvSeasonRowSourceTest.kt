package net.kdhmedia.mochil.ui

import java.nio.file.Files
import java.nio.file.Paths
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * A TV show with more seasons than fit on the screen. The pill row used to be a plain `Row` of pills inside a
 * vertically scrolling column: the later pills were measured past the right edge, so the remote could focus a season
 * the viewer could not see and nothing on screen scrolled. A lazy row keeps the row scrollable and follows the focus.
 * This is a Compose layout property, so it is pinned in the source rather than through an emulator.
 */
class TvSeasonRowSourceTest {
    private val source = Files.readString(Paths.get("src/main/java/net/kdhmedia/mochil/ui/TvShowDetail.kt"))

    @Test fun `season pills are a scrollable lazy row`() {
        assertTrue("the seasons must be laid out lazily so the row can scroll", source.contains("items(show.seasons,"))
        assertFalse("a plain forEach lays every pill out past the screen edge", source.contains("show.seasons.forEach"))
    }

    @Test fun `both screens name a season through the shared label`() {
        assertTrue("the TV show page must share the phone's season label", source.contains("tvSeasonLabel(season)"))
        val phone = Files.readString(Paths.get("src/main/java/net/kdhmedia/mochil/ui/VideoScreen.kt"))
        assertTrue(phone.contains("tvSeasonLabel(season)"))
    }
}
