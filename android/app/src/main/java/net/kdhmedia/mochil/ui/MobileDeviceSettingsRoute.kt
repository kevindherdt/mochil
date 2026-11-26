package net.kdhmedia.mochil.ui

import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import net.kdhmedia.mochil.audio.MusicViewModel
import net.kdhmedia.mochil.auth.LoginState
import net.kdhmedia.mochil.auth.LoginViewModel
import net.kdhmedia.mochil.catalogue.DownloadLocationViewModel
import net.kdhmedia.mochil.catalogue.LibraryViewModel
import net.kdhmedia.mochil.catalogue.LibraryVisibilityProjection
import net.kdhmedia.mochil.catalogue.visibilityProjection
import net.kdhmedia.mochil.diagnostics.CrashReportViewModel
import net.kdhmedia.mochil.diagnostics.PerformanceDiagnosticsViewModel
import net.kdhmedia.mochil.podcast.PodcastViewModel
import net.kdhmedia.mochil.playback.PlaybackViewModel
import net.kdhmedia.mochil.reader.epub.AnnotationPrivacyViewModel
import net.kdhmedia.mochil.ui.theme.AppThemeSelection
import net.kdhmedia.mochil.update.AppUpdateViewModel
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.map

/** Owns Settings-only subscriptions so media routes do not observe administrative state. */
@Composable
internal fun MobileDeviceSettingsRoute(
    loginState: LoginState,
    loginViewModel: LoginViewModel,
    libraryViewModel: LibraryViewModel,
    musicViewModel: MusicViewModel,
    podcastViewModel: PodcastViewModel,
    playbackViewModel: PlaybackViewModel,
    updates: AppUpdateViewModel,
    downloads: DownloadLocationViewModel,
    crashReports: CrashReportViewModel,
    diagnostics: PerformanceDiagnosticsViewModel,
    annotationPrivacy: AnnotationPrivacyViewModel,
    themeSelection: AppThemeSelection,
    onThemeSelection: (AppThemeSelection) -> Unit,
    showCollections: Boolean,
    showSources: Boolean,
    onDestinationVisibility: (OptionalDestination, Boolean) -> Unit,
) {
    val libraryVisibilityFlow = remember(libraryViewModel) {
        libraryViewModel.state.map { it.visibilityProjection() }.distinctUntilChanged()
    }
    val libraryVisibility by libraryVisibilityFlow.collectAsStateWithLifecycle(LibraryVisibilityProjection())
    val updateState by updates.state.collectAsStateWithLifecycle()
    val downloadLocation by downloads.state.collectAsStateWithLifecycle()
    val crashState by crashReports.state.collectAsStateWithLifecycle()
    val diagnosticsState by diagnostics.state.collectAsStateWithLifecycle()
    val annotationPrivacyState by annotationPrivacy.state.collectAsStateWithLifecycle()
    val replayGainMode by playbackViewModel.replayGainMode.collectAsStateWithLifecycle()
    LaunchedEffect(downloadLocation.treeUri, downloadLocation.message) {
        musicViewModel.refreshOfflineDownloads()
        podcastViewModel.refreshOfflineDownloads()
    }
    val downloadFolderPicker = rememberLauncherForActivityResult(ActivityResultContracts.OpenDocumentTree()) { downloads.select(it) }
    val annotationExportPicker = rememberLauncherForActivityResult(ActivityResultContracts.CreateDocument("application/json")) { destination ->
        destination?.let(annotationPrivacy::exportTo)
    }

    DeviceSettingsScreen(
        serverUrl = loginViewModel.connectedServerUrl(),
        serverAddressChanging = loginState.loading,
        serverAddressMessage = loginState.notice ?: loginState.error,
        onChangeServerAddress = loginViewModel::changeServerAddress,
        themeSelection = themeSelection,
        onThemeSelection = onThemeSelection,
        updateState = updateState,
        onCheckForUpdates = updates::check,
        onInstallUpdate = updates::downloadAndInstall,
        onDisconnectServer = loginViewModel::disconnect,
        downloadLocation = downloadLocation,
        onChooseDownloadFolder = { downloadFolderPicker.launch(null) },
        onUseAppStorage = downloads::useAppStorage,
        replayGainMode = replayGainMode,
        onReplayGainMode = playbackViewModel::setReplayGainMode,
        crashState = crashState,
        onSendCrashReport = crashReports::send,
        onDiscardCrashReport = crashReports::discard,
        diagnosticsState = diagnosticsState,
        onDiagnosticsEnabled = diagnostics::setEnabled,
        onSendDiagnostics = diagnostics::sendNow,
        onClearDiagnostics = diagnostics::clear,
        annotationPrivacyState = annotationPrivacyState,
        onExportAnnotations = { annotationExportPicker.launch("mochil-annotations.json") },
        onDeleteServerAnnotations = annotationPrivacy::deleteServerCopy,
        libraries = libraryVisibility.allLibraries,
        visibleLibraryIds = libraryVisibility.visibleLibraryIds,
        deviceClass = libraryViewModel.deviceClass(),
        onLibraryVisibility = libraryViewModel::setLibraryVisible,
        showCollections = showCollections,
        showSources = showSources,
        onDestinationVisibility = onDestinationVisibility,
    )
}
