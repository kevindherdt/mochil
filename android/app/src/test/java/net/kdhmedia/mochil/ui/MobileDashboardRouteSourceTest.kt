package net.kdhmedia.mochil.ui

import java.io.File
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class MobileDashboardRouteSourceTest {
    private fun source(name: String) = sequenceOf(
        File("src/main/java/net/kdhmedia/mochil/ui/$name"),
        File("app/src/main/java/net/kdhmedia/mochil/ui/$name"),
        File("android/app/src/main/java/net/kdhmedia/mochil/ui/$name"),
    ).first(File::isFile).readText()

    @Test fun `dashboard state is collected only inside the Home destination`() {
        val root = source("MochilApp.kt")
        val home = source("MobileDashboardRoute.kt")

        assertFalse(
            "app root still collects dashboard state",
            root.contains("dashboardViewModel.state.collectAsStateWithLifecycle"),
        )
        assertTrue(
            "Home route does not own dashboard state",
            home.contains("dashboardViewModel.state.collectAsStateWithLifecycle"),
        )
        assertTrue(root.contains("MobileDashboardRoute("))
    }
}
