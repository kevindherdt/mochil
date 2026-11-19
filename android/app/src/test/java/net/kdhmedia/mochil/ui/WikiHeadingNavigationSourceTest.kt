package net.kdhmedia.mochil.ui

import java.nio.file.Files
import java.nio.file.Paths
import org.junit.Assert.assertTrue
import org.junit.Test

class WikiHeadingNavigationSourceTest {
    @Test
    fun headingNavigationWaitsForTheRequestedPageToFinishLoading() {
        val source = Files.readString(Paths.get("src/main/java/net/kdhmedia/mochil/ui/WikiScreen.kt"))
        val effect = source.substringAfter("LaunchedEffect(page?.id, state.loading, state.navigationHeading, document)")
            .substringBefore("val closePage")
        assertTrue(effect.contains("if (state.loading || page == null || document == null) return@LaunchedEffect"))
        assertTrue(effect.indexOf("state.loading || page == null || document == null") < effect.indexOf("bindings.consumeNavigationHeading()"))
    }
}
