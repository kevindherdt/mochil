package net.kdhmedia.mochil.ui

import androidx.activity.compose.BackHandler
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import net.kdhmedia.mochil.auth.LoginViewModel
import net.kdhmedia.mochil.catalogue.LibraryViewModel
import net.kdhmedia.mochil.catalogue.LibraryVisibilityProjection
import net.kdhmedia.mochil.catalogue.visibilityProjection
import net.kdhmedia.mochil.diagnostics.CrashReportViewModel
import net.kdhmedia.mochil.diagnostics.PerformanceDiagnosticsViewModel
import net.kdhmedia.mochil.ui.theme.AppThemeSelection
import net.kdhmedia.mochil.update.AppUpdateViewModel
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.map

/** Creates and observes administrative state only while TV Settings is visible. */
@Composable
internal fun TvSettingsRoute(
    loginViewModel: LoginViewModel,
    libraryViewModel: LibraryViewModel,
    updates: AppUpdateViewModel,
    themeSelection: AppThemeSelection,
    onThemeSelection: (AppThemeSelection) -> Unit,
    onBack: () -> Unit,
) {
    val state by updates.state.collectAsStateWithLifecycle()
    val libraryVisibilityFlow = remember(libraryViewModel) {
        libraryViewModel.state.map { it.visibilityProjection() }.distinctUntilChanged()
    }
    val libraries by libraryVisibilityFlow.collectAsStateWithLifecycle(LibraryVisibilityProjection())
    val crashReports: CrashReportViewModel = viewModel()
    val performance: PerformanceDiagnosticsViewModel = viewModel()
    val crashState by crashReports.state.collectAsStateWithLifecycle()
    val performanceState by performance.state.collectAsStateWithLifecycle()
    BackHandler(onBack = onBack)
    TvSettingsScreen(
        serverUrl = loginViewModel.connectedServerUrl(),
        update = state,
        themeSelection = themeSelection,
        onThemeSelection = onThemeSelection,
        crashState = crashState,
        diagnosticsState = performanceState,
        onDiagnosticsEnabled = performance::setEnabled,
        onSendDiagnostics = performance::sendNow,
        onClearDiagnostics = performance::clear,
        onSendCrashReport = crashReports::send,
        onDiscardCrashReport = crashReports::discard,
        onBack = onBack,
        onCheckUpdate = updates::check,
        onInstallUpdate = updates::downloadAndInstall,
        onDisconnect = loginViewModel::disconnect,
        libraries = libraries.allLibraries,
        visibleLibraryIds = libraries.visibleLibraryIds,
        deviceClass = libraryViewModel.deviceClass(),
        onLibraryVisibility = libraryViewModel::setLibraryVisible,
    )
}
