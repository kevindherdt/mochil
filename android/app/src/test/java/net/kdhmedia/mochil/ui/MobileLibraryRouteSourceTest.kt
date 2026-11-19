package net.kdhmedia.mochil.ui

import java.io.File
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class MobileLibraryRouteSourceTest {
    private fun source(name: String) = sequenceOf(
        File("src/main/java/net/kdhmedia/mochil/ui/$name"),
        File("app/src/main/java/net/kdhmedia/mochil/ui/$name"),
        File("android/app/src/main/java/net/kdhmedia/mochil/ui/$name"),
    ).first(File::isFile).readText()

    @Test fun `media state is collected only inside the Library destination`() {
        val root = source("MochilApp.kt")
        val library = source("MobileLibraryRoute.kt")
        listOf(
            "libraryViewModel.state.collectAsStateWithLifecycle",
            "musicViewModel.state.collectAsStateWithLifecycle",
            "podcastViewModel.state.collectAsStateWithLifecycle",
            "videoViewModel.state.collectAsStateWithLifecycle",
            "collectionViewModel.state.collectAsStateWithLifecycle",
        ).forEach { subscription ->
            assertFalse("app root still collects $subscription", root.contains(subscription))
            assertTrue("Library route does not own $subscription", library.contains(subscription))
        }
        assertTrue(root.contains("MobileLibraryRoute("))
        assertTrue(root.contains("map { it.offlineOnly }.distinctUntilChanged()"))
    }

    @Test fun `Home and Settings own their library directory subscriptions`() {
        val home = source("MobileDashboardRoute.kt")
        val settings = source("MobileDeviceSettingsRoute.kt")
        assertTrue(home.contains("libraryViewModel.state.map { it.shellProjection() }.distinctUntilChanged()"))
        assertTrue(settings.contains("libraryViewModel.state.map { it.visibilityProjection() }.distinctUntilChanged()"))
        assertFalse(home.contains("libraryViewModel.state.collectAsStateWithLifecycle"))
        assertFalse(settings.contains("libraryViewModel.state.collectAsStateWithLifecycle"))
    }
}
