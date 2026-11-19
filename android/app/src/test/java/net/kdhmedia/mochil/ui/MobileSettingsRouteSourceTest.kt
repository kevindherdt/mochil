package net.kdhmedia.mochil.ui

import java.io.File
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class MobileSettingsRouteSourceTest {
    private fun source(name: String) = sequenceOf(
        File("src/main/java/net/kdhmedia/mochil/ui/$name"),
        File("app/src/main/java/net/kdhmedia/mochil/ui/$name"),
        File("android/app/src/main/java/net/kdhmedia/mochil/ui/$name"),
    ).first(File::isFile).readText()

    @Test fun `administrative state is collected only inside the settings destination`() {
        val root = source("MochilApp.kt")
        val settings = source("MobileDeviceSettingsRoute.kt")
        listOf("updates.state", "downloads.state", "crashReports.state", "diagnostics.state", "annotationPrivacy.state", "playbackViewModel.replayGainMode").forEach { subscription ->
            assertFalse("app root still collects $subscription", root.contains(subscription))
            assertTrue("settings route does not own $subscription", settings.contains(subscription))
        }
        assertTrue(root.contains("MobileDeviceSettingsRoute("))
    }
}
