package net.kdhmedia.mochil.ui

import androidx.activity.ComponentActivity
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createAndroidComposeRule
import androidx.compose.ui.test.getUnclippedBoundsInRoot
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.unit.dp
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performScrollTo
import net.kdhmedia.mochil.catalogue.ClientDeviceClass
import net.kdhmedia.mochil.catalogue.DownloadLocation
import net.kdhmedia.mochil.diagnostics.CrashReportState
import net.kdhmedia.mochil.diagnostics.PerformanceDiagnosticsState
import net.kdhmedia.mochil.playback.ReplayGainMode
import net.kdhmedia.mochil.reader.epub.AnnotationPrivacyState
import net.kdhmedia.mochil.ui.theme.AppPalette
import net.kdhmedia.mochil.ui.theme.AppThemeMode
import net.kdhmedia.mochil.ui.theme.AppThemeSelection
import net.kdhmedia.mochil.ui.theme.MochilTheme
import net.kdhmedia.mochil.update.AppUpdateState
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

/** Settings is a hub of cards that open pages, as on the web; Back from a page returns to the hub. */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], qualifiers = "w390dp-h844dp-xxhdpi")
class DeviceSettingsHubTest {
    @get:Rule val compose = createAndroidComposeRule<ComponentActivity>()

    private var selection = AppThemeSelection()

    @androidx.compose.runtime.Composable
    private fun SettingsForTest() = DeviceSettingsScreen(
        serverUrl = "https://example.invalid", serverAddressChanging = false, serverAddressMessage = null,
        onChangeServerAddress = {}, themeSelection = selection, onThemeSelection = {},
        updateState = AppUpdateState(), onCheckForUpdates = {}, onInstallUpdate = {}, onDisconnectServer = {},
        downloadLocation = DownloadLocation(), onChooseDownloadFolder = {}, onUseAppStorage = {},
        replayGainMode = ReplayGainMode.Off, onReplayGainMode = {}, crashState = CrashReportState(),
        onSendCrashReport = {}, onDiscardCrashReport = {}, diagnosticsState = PerformanceDiagnosticsState(),
        onDiagnosticsEnabled = {}, onSendDiagnostics = {}, onClearDiagnostics = {},
        annotationPrivacyState = AnnotationPrivacyState(), onExportAnnotations = {}, onDeleteServerAnnotations = {},
        libraries = emptyList(), visibleLibraryIds = emptySet(), deviceClass = ClientDeviceClass.Handheld,
        onLibraryVisibility = { _, _ -> }, showCollections = true, showSources = true,
        onDestinationVisibility = { _, _ -> },
    )

    private fun settings(onSelection: (AppThemeSelection) -> Unit = { selection = it }) = compose.setContent {
        MochilTheme(AppThemeSelection(AppPalette.Tide, AppThemeMode.Dark), glow = false) {
            DeviceSettingsScreen(
                serverUrl = "https://example.invalid", serverAddressChanging = false, serverAddressMessage = null,
                onChangeServerAddress = {}, themeSelection = selection, onThemeSelection = onSelection,
                updateState = AppUpdateState(), onCheckForUpdates = {}, onInstallUpdate = {}, onDisconnectServer = {},
                downloadLocation = DownloadLocation(), onChooseDownloadFolder = {}, onUseAppStorage = {},
                replayGainMode = ReplayGainMode.Off, onReplayGainMode = {}, crashState = CrashReportState(),
                onSendCrashReport = {}, onDiscardCrashReport = {}, diagnosticsState = PerformanceDiagnosticsState(),
                onDiagnosticsEnabled = {}, onSendDiagnostics = {}, onClearDiagnostics = {},
                annotationPrivacyState = AnnotationPrivacyState(), onExportAnnotations = {}, onDeleteServerAnnotations = {},
                libraries = emptyList(), visibleLibraryIds = emptySet(), deviceClass = ClientDeviceClass.Handheld,
                onLibraryVisibility = { _, _ -> }, showCollections = true, showSources = true,
                onDestinationVisibility = { _, _ -> },
            )
        }
    }

    private fun pressBack() = compose.activityRule.scenario.onActivity { it.onBackPressedDispatcher.onBackPressed() }

    @Test fun `the hub shows the server status and every category, and none of their contents`() {
        settings()
        compose.onNodeWithText("Paired").assertIsDisplayed()
        compose.onNodeWithText("https://example.invalid").assertIsDisplayed()
        SettingsCategory.entries.filterNot { it == SettingsCategory.Server }.forEach {
            compose.onNodeWithText(it.label).performScrollTo().assertIsDisplayed()
        }
        compose.onNodeWithText("Brightness").assertDoesNotExist()
        compose.onNodeWithText("Change address").assertDoesNotExist()
    }

    @Test fun `the hub has no back arrow because Settings is a tab`() {
        settings()
        compose.onNodeWithContentDescription("Back").assertDoesNotExist()
    }

    // Reported from a phone: tapping About in Settings closed the app. The page reads three licence files out of the
    // APK, so this drives the real route rather than calling the content directly.
    @Test fun `About opens its page and reads every notice out of the bundle`() {
        settings()
        compose.onNodeWithText("About").performScrollTo().performClick()
        compose.onNodeWithText("Notices").assertIsDisplayed()
        compose.onNodeWithText("Lucide icons").assertIsDisplayed().performClick()
        compose.onNodeWithText("Permission to use, copy, modify", substring = true).assertIsDisplayed()
    }

    @Test fun `a category opens its page with a title and a back arrow, and Back returns to the hub`() {
        settings()
        compose.onNodeWithText("Appearance").performClick()
        compose.onNodeWithContentDescription("Back").assertIsDisplayed()
        compose.onNodeWithText("Tide").assertIsDisplayed()
        compose.onNodeWithText("Downloads").assertDoesNotExist()
        pressBack()
        compose.waitForIdle()
        compose.onNodeWithText("Downloads").assertIsDisplayed()
        compose.onNodeWithContentDescription("Back").assertDoesNotExist()
    }

    @Test fun `the back arrow on a page returns to the hub too`() {
        settings()
        compose.onNodeWithText("Downloads").performScrollTo().performClick()
        compose.onNodeWithContentDescription("Back").performClick()
        compose.onNodeWithText("Privacy").performScrollTo().assertIsDisplayed()
    }

    // A short screen: on a tall one the hub fits without scrolling and there is no offset to carry into the page.
    @Config(sdk = [35], qualifiers = "w390dp-h480dp-xxhdpi")
    @Test fun `a page opens at its top even when the hub was scrolled down`() {
        compose.setContent {
            androidx.compose.runtime.CompositionLocalProvider(
                androidx.compose.ui.platform.LocalDensity provides androidx.compose.ui.unit.Density(androidx.compose.ui.platform.LocalDensity.current.density, 2f),
            ) { MochilTheme(AppThemeSelection(AppPalette.Tide, AppThemeMode.Dark), glow = false) { SettingsForTest() } }
        }
        // Scroll the hub to its last card, then open the longest page: it must start at its own top, with its header showing.
        compose.onNodeWithText("Privacy").performScrollTo()
        compose.onNodeWithText("Reading").performScrollTo().performClick()
        val top = compose.onNodeWithContentDescription("Back").getUnclippedBoundsInRoot().top
        org.junit.Assert.assertTrue("the page header starts at the top of the screen, not $top", top >= 0.dp)
    }

    @Test fun `the server card opens the server page with its actions`() {
        settings()
        compose.onNodeWithText("Paired").performClick()
        compose.onNodeWithText("Change address").assertIsDisplayed()
        compose.onNodeWithText("Disconnect").assertIsDisplayed()
    }

    @Test fun `Appearance offers one swatch per palette and reports the chosen one`() {
        settings()
        compose.onNodeWithText("Appearance").performClick()
        AppPalette.entries.forEach { compose.onNodeWithText(it.label).assertIsDisplayed() }
        compose.onNodeWithText("Magenta Cream").performClick()
        assertEquals(AppPalette.MagentaCream, selection.palette)
    }

    @Test fun `Appearance offers device, light and dark and reports the chosen mode`() {
        settings()
        compose.onNodeWithText("Appearance").performClick()
        compose.onNodeWithText("Light").performClick()
        assertEquals(AppThemeMode.Light, selection.mode)
    }

    @Test fun `Sources is its own entry and opens the hub with both flows`() {
        settings()
        compose.onNodeWithText("Sources").performScrollTo().performClick()
        compose.onNodeWithText("OPDS catalogues", useUnmergedTree = true).assertExists()
        compose.onNodeWithText("WebDAV libraries", useUnmergedTree = true).assertExists()
    }

    @Test fun `Libraries no longer carries an Add a source button`() {
        settings()
        compose.onNodeWithText("Libraries").performScrollTo().performClick()
        compose.onNodeWithText("Add a source").assertDoesNotExist()
    }
}
