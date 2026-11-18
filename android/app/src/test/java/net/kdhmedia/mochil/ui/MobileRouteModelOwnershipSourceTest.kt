package net.kdhmedia.mochil.ui

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

class MobileRouteModelOwnershipSourceTest {
    private fun rootSource() = sequenceOf(
        File("src/main/java/net/kdhmedia/mochil/ui/MochilApp.kt"),
        File("app/src/main/java/net/kdhmedia/mochil/ui/MochilApp.kt"),
        File("android/app/src/main/java/net/kdhmedia/mochil/ui/MochilApp.kt"),
    ).first(File::isFile).readText()

    @Test fun `route-only models are created inside their destination lambdas`() {
        val source = rootSource()
        val library = source.substringAfter("library = {").substringBefore("collections = {")
        val collections = source.substringAfter("collections = {").substringBefore("sources = {")
        val settings = source.substringAfter("settings = {")

        assertTrue(library.contains("CollectionViewModel = viewModel"))
        assertTrue(library.contains("WikiViewModel = viewModel"))
        assertTrue(collections.contains("MobileCollectionsRoute("))
        assertTrue(settings.contains("DownloadLocationViewModel = viewModel"))
        assertTrue(settings.contains("CrashReportViewModel = viewModel"))
    }

    @Test fun `queue and playlist state stay below their consuming surfaces`() {
        val root = rootSource()
        val player = sequenceOf(
            File("src/main/java/net/kdhmedia/mochil/ui/MobilePlaybackHost.kt"),
            File("app/src/main/java/net/kdhmedia/mochil/ui/MobilePlaybackHost.kt"),
            File("android/app/src/main/java/net/kdhmedia/mochil/ui/MobilePlaybackHost.kt"),
        ).first(File::isFile).readText()
        val library = root.substringAfter("library = {").substringBefore("collections = {")
        val expandedRoute = player.substringAfter("fun MobileExpandedPlayerRoute(").substringBefore("fun MobileExpandedPlayerHost(")

        assertTrue(!root.substringBefore("AppShell(").contains("musicQueueViewModel.state.collectAsStateWithLifecycle"))
        assertTrue(!root.substringBefore("AppShell(").contains("musicPlaylistViewModel.state.collectAsStateWithLifecycle"))
        assertTrue(library.contains("musicPlaylistViewModel.state.collectAsStateWithLifecycle"))
        assertTrue(expandedRoute.indexOf("if (!expanded) return") < expandedRoute.indexOf("queueViewModel.state.collectAsStateWithLifecycle"))
        assertTrue(expandedRoute.indexOf("if (!expanded) return") < expandedRoute.indexOf("playlistViewModel.state.collectAsStateWithLifecycle"))
    }
}
