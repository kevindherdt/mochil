package net.kdhmedia.mochil.ui

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performScrollTo
import androidx.compose.ui.test.performClick
import net.kdhmedia.mochil.catalogue.ClientDeviceClass
import net.kdhmedia.mochil.catalogue.DownloadLocation
import net.kdhmedia.mochil.diagnostics.CrashReportState
import net.kdhmedia.mochil.diagnostics.PerformanceDiagnosticsState
import net.kdhmedia.mochil.playback.ReplayGainMode
import net.kdhmedia.mochil.reader.epub.AnnotationPrivacyState
import net.kdhmedia.mochil.ui.theme.AppThemeSelection
import net.kdhmedia.mochil.update.AppUpdateState
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class DeviceSettingsAddSourceTest {
    @get:Rule val compose = createComposeRule()

    private fun settings() = compose.setContent {
        DeviceSettingsScreen(
            serverUrl = "https://example.invalid", serverAddressChanging = false, serverAddressMessage = null,
            onChangeServerAddress = {}, themeSelection = AppThemeSelection(), onThemeSelection = {},
            updateState = AppUpdateState(), onCheckForUpdates = {}, onInstallUpdate = {}, onDisconnectServer = {},
            downloadLocation = DownloadLocation(), onChooseDownloadFolder = {}, onUseAppStorage = {},
            replayGainMode = ReplayGainMode.Off, onReplayGainMode = {}, crashState = CrashReportState(),
            onSendCrashReport = {}, onDiscardCrashReport = {}, diagnosticsState = PerformanceDiagnosticsState(),
            onDiagnosticsEnabled = {}, onSendDiagnostics = {}, onClearDiagnostics = {},
            annotationPrivacyState = AnnotationPrivacyState(), onExportAnnotations = {}, onDeleteServerAnnotations = {},
            libraries = emptyList(), visibleLibraryIds = emptySet(), deviceClass = ClientDeviceClass.Handheld,
            onLibraryVisibility = { _, _ -> }, showCollections = true, showSources = true,
            onDestinationVisibility = { _, _ -> }, onBack = {},
        )
    }

    @Test fun librariesSettingsOffersOneAddASourceEntryThatOpensTheHubWithBothFlows() {
        settings()
        compose.onNodeWithText("Libraries").performClick()
        compose.onNodeWithText("OPDS catalogues").assertDoesNotExist()
        compose.onNodeWithText("WebDAV libraries").assertDoesNotExist()
        compose.onNodeWithText("Add a source").performScrollTo().performClick()
        compose.onNodeWithText("Sources").assertIsDisplayed()
        compose.onNodeWithText("OPDS catalogues", useUnmergedTree = true).assertExists()
        compose.onNodeWithText("WebDAV libraries", useUnmergedTree = true).assertExists()
    }
}
