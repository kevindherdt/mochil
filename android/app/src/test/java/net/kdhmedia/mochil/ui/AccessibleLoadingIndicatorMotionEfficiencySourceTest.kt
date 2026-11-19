package net.kdhmedia.mochil.ui

import java.nio.file.Files
import java.nio.file.Paths
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * `AccessibleLoadingIndicator` is the shared loading spinner used across ~18 phone and TV
 * screens (library/music/video/podcast/wiki loading states). Material3's indeterminate
 * `CircularProgressIndicator()` spins forever via its own internal infinite transition, which
 * is a continuous full-panel repaint on e-ink for however long a screen is loading. Like
 * Shimmer.kt's `.shimmer()`, it must check `LocalReduceMotion` and fall back to a static (non
 * spinning) visual instead of animating unconditionally.
 */
class AccessibleLoadingIndicatorMotionEfficiencySourceTest {
    private val source = Files.readString(
        Paths.get("src/main/java/net/kdhmedia/mochil/ui/AccessibleLoadingIndicator.kt"),
    )

    @Test fun checksReduceMotionBeforeChoosingTheIndicatorVisual() {
        assertTrue(
            "AccessibleLoadingIndicator.kt should read LocalReduceMotion, the same switch Shimmer.kt and Motion.kt use",
            source.contains("LocalReduceMotion.current"),
        )
    }

    @Test fun offersAStaticAlternativeToTheSpinningIndeterminateIndicator() {
        assertTrue(
            "AccessibleLoadingIndicator.kt should render a static (non-animating) ring when reduce motion is on, " +
                "instead of always using the continuously-spinning indeterminate CircularProgressIndicator()",
            source.contains("progress = { 1f }") || source.contains("progress = {1f}"),
        )
    }
}
