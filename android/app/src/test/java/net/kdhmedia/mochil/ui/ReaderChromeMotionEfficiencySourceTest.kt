package net.kdhmedia.mochil.ui

import java.nio.file.Files
import java.nio.file.Paths
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * The reader toolbars fade in and out on every tap. That is fine on a phone, but on an e-ink
 * device each fade is a wasted full-panel repaint/ghosting hit, and `LocalReduceMotion` exists
 * precisely to turn these off (see Motion.kt / Shimmer.kt). Raw `AnimatedVisibility` calls bypass
 * that switch, so the readers must go through the shared `MotionVisibility` helper instead.
 */
class ReaderChromeMotionEfficiencySourceTest {
    private val readerScreens = listOf(
        "reader/comic/ComicReaderScreen.kt",
        "reader/epub/EpubReaderChrome.kt",
        "reader/pdf/PdfReaderScreen.kt",
    )

    @Test fun readerChromeUsesTheReduceMotionAwareHelperInsteadOfRawAnimatedVisibility() {
        readerScreens.forEach { relativePath ->
            val source = Files.readString(Paths.get("src/main/java/net/kdhmedia/mochil/$relativePath"))
            assertFalse(
                "$relativePath should not call raw AnimatedVisibility(...) for reader chrome; it bypasses LocalReduceMotion on e-ink",
                source.contains("AnimatedVisibility("),
            )
            assertTrue(
                "$relativePath should show/hide its chrome through the reduce-motion-aware MotionVisibility helper",
                source.contains("MotionVisibility("),
            )
        }
    }
}
