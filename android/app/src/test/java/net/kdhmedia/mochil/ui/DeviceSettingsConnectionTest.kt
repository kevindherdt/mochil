package net.kdhmedia.mochil.ui

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
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
class DeviceSettingsConnectionTest {
    @get:Rule val compose = createComposeRule()

    @Test fun savedServerIsPairedNotClaimedConnected() {
        compose.setContent {
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
                onDestinationVisibility = { _, _ -> },
            )
        }
        compose.onNodeWithText("Paired").assertIsDisplayed()
        compose.onNodeWithText("Connected").assertDoesNotExist()
        compose.onNodeWithText("Connection not checked").assertDoesNotExist()
    }
}
