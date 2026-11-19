package net.kdhmedia.mochil.ui

import java.nio.file.Files
import java.nio.file.Paths
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Every poster card on the TV grid drew its artwork twice: one full-card copy to blur, one poster for the right-hand
 * slot, each asking Coil for the same file at a different constraint — two decodes per card. The pair now comes from
 * one request, and the live blur is the focused card's alone. Both are decode/draw properties of a Compose tree, so
 * they are pinned here instead of through an emulator.
 */
class TvPosterArtworkSourceTest {
    private val source = Files.readString(Paths.get("src/main/java/net/kdhmedia/mochil/ui/TvMediaGrid.kt"))

    /** The body of the poster artwork widget, so the list row's own artwork call is not counted. */
    private val wideArtwork = source.substringAfter("internal fun TvWideArtwork(")

    @Test fun `the poster card loads one image for both copies`() {
        assertEquals("the poster pair must share one request", 1, wideArtwork.split("authenticatedArtworkRequest(").size - 1)
        assertTrue("the two draws must share the loaded painter", wideArtwork.contains("rememberAsyncImagePainter(request)"))
        assertEquals("the poster's own draw must not start a second load", 1, wideArtwork.split("AuthenticatedArtwork(").size - 1)
    }

    @Test fun `the blurred backdrop belongs to the focused card only`() {
        assertFalse("an unconditional full-card blur is the per-frame work this removed", source.contains("fillMaxSize().blur(24.dp)"))
        assertTrue("the card passes its focus state to the artwork", source.contains("blurredBackdrop = focused"))
    }
}
