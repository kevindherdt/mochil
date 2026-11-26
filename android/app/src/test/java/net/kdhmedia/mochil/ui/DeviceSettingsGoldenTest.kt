package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.ui.Modifier
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.onRoot
import androidx.compose.ui.test.performClick
import net.kdhmedia.mochil.BuildConfig
import net.kdhmedia.mochil.catalogue.ClientDeviceClass
import net.kdhmedia.mochil.catalogue.DownloadLocation
import net.kdhmedia.mochil.diagnostics.CrashReportState
import net.kdhmedia.mochil.diagnostics.PerformanceDiagnosticsState
import net.kdhmedia.mochil.playback.ReplayGainMode
import net.kdhmedia.mochil.reader.epub.AnnotationPrivacyState
import net.kdhmedia.mochil.testing.Goldens
import net.kdhmedia.mochil.ui.theme.AppPalette
import net.kdhmedia.mochil.ui.theme.AppThemeMode
import net.kdhmedia.mochil.ui.theme.AppThemeSelection
import net.kdhmedia.mochil.ui.theme.MochilTheme
import net.kdhmedia.mochil.update.AppUpdateState
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode

/** What Settings looks like on a 390 x 844 phone, in the screen's own theme, as pictures a person can look at. */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], qualifiers = "w390dp-h844dp-xxhdpi")
@GraphicsMode(GraphicsMode.Mode.NATIVE)
class DeviceSettingsGoldenTest {
    @get:Rule val compose = createComposeRule()

    private fun settings(palette: AppPalette, mode: AppThemeMode) = compose.setContent {
        MochilTheme(AppThemeSelection(palette, mode), glow = false) {
            Surface(Modifier.fillMaxSize(), color = MaterialTheme.colorScheme.background) {
                DeviceSettingsScreen(
                    serverUrl = "https://media.example.com", serverAddressChanging = false, serverAddressMessage = null,
                    onChangeServerAddress = {}, themeSelection = AppThemeSelection(palette, mode), onThemeSelection = {},
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
    }

    @Test fun `golden hub Tide dark`() {
        settings(AppPalette.Tide, AppThemeMode.Dark)
        Goldens.assertMatches(compose.onRoot(), "settings-hub-tide-dark")
    }

    @Test fun `golden hub Ember light`() {
        settings(AppPalette.Ember, AppThemeMode.Light)
        Goldens.assertMatches(compose.onRoot(), "settings-hub-ember-light")
    }

    @Test fun `golden Appearance page Tide dark`() {
        settings(AppPalette.Tide, AppThemeMode.Dark)
        compose.onNodeWithText("Appearance").performClick()
        Goldens.assertMatches(compose.onRoot(), "settings-appearance-tide-dark")
    }

    @Test fun `golden Server page Tide dark`() {
        settings(AppPalette.Tide, AppThemeMode.Dark)
        compose.onNodeWithText("Server").performClick()
        // The App updates card exists only where the app can update itself (direct), so each flavour has its own picture.
        Goldens.assertMatches(compose.onRoot(), "settings-server-${if (BuildConfig.SELF_UPDATE_ENABLED) "direct" else "store"}-tide-dark")
    }

    @Test fun `golden Downloads page Tide dark`() {
        settings(AppPalette.Tide, AppThemeMode.Dark)
        compose.onNodeWithText("Downloads").performClick()
        Goldens.assertMatches(compose.onRoot(), "settings-downloads-tide-dark")
    }
}
