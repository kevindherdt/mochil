package net.kdhmedia.mochil.ui

import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File

class TvHomeFocusSafetyTest {
    @Test fun loadingWithoutItemsTakesPrecedenceOverErrorFocus() {
        val source = File("src/main/java/net/kdhmedia/mochil/ui/TvHomeScreen.kt").readText()
        val effect = source.substringAfter("LaunchedEffect(state.error, state.loading, empty)")
            .substringBefore("val firstWatching")
        // The request is wrapped in runCatching: Retry lives in item 0 of a lazy list, so a requester with no attached target
// would throw out of the effect and take the app down instead of showing the button.
        assertTrue("Loading hides Retry, so it must be checked before requesting Retry focus",
            effect.indexOf("state.loading && empty -> Unit") in 0 until effect.indexOf("state.error != null -> runCatching { retryFocus.requestFocus() }"))
    }
}
