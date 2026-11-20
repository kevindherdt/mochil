package net.kdhmedia.mochil.ui

import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.ui.platform.LocalContext
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import net.kdhmedia.mochil.auth.LoginViewModel
import net.kdhmedia.mochil.catalogue.LibraryViewModel
import net.kdhmedia.mochil.catalogue.LibrarySummary
import net.kdhmedia.mochil.catalogue.LibraryShellProjection
import net.kdhmedia.mochil.catalogue.shellProjection
import net.kdhmedia.mochil.catalogue.DownloadLocationViewModel
import net.kdhmedia.mochil.catalogue.CollectionViewModel
import net.kdhmedia.mochil.audio.MusicViewModel
import net.kdhmedia.mochil.audio.MusicQueueViewModel
import net.kdhmedia.mochil.audio.MusicPlaylistViewModel
import net.kdhmedia.mochil.playback.PlaybackViewModel
import net.kdhmedia.mochil.update.AppUpdateViewModel
import net.kdhmedia.mochil.dashboard.DashboardViewModel
import net.kdhmedia.mochil.diagnostics.CrashReportViewModel
import net.kdhmedia.mochil.diagnostics.PerformanceDiagnosticsViewModel
import net.kdhmedia.mochil.reader.epub.AnnotationPrivacyViewModel
import net.kdhmedia.mochil.podcast.PodcastViewModel
import net.kdhmedia.mochil.video.VideoViewModel
import net.kdhmedia.mochil.wiki.WikiViewModel
import net.kdhmedia.mochil.ui.theme.AppThemeSelection
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.map

@Composable
fun MochilApp(
    loginViewModel: LoginViewModel = viewModel(),
    television: Boolean = false,
    themeSelection: AppThemeSelection = AppThemeSelection(),
    onThemeSelection: (AppThemeSelection) -> Unit = {},
) {
    val state by loginViewModel.state.collectAsStateWithLifecycle()
    if (state.connected) {
		val sessionKey = state.sessionVersion.toString()
        val libraryViewModel: LibraryViewModel = viewModel(key = "library:$sessionKey")
        val musicViewModel: MusicViewModel = viewModel(key = "music:$sessionKey")
        val playbackViewModel: PlaybackViewModel = viewModel(key = "playback:$sessionKey")
        val podcastViewModel: PodcastViewModel = viewModel(key = "podcasts:$sessionKey")
        val videoViewModel: VideoViewModel = viewModel(key = "video:$sessionKey")
        val dashboardViewModel: DashboardViewModel = viewModel(key = "dashboard:$sessionKey")
        val musicQueueViewModel: MusicQueueViewModel = viewModel(key = "queue:$sessionKey")
        if (television) {
            val appUpdateViewModel: AppUpdateViewModel = viewModel(key = "updates:$sessionKey")
            TvConnectedApp(
                loginViewModel = loginViewModel,
                libraryViewModel = libraryViewModel,
                musicViewModel = musicViewModel,
                playbackViewModel = playbackViewModel,
                appUpdateViewModel = appUpdateViewModel,
                podcastViewModel = podcastViewModel,
                videoViewModel = videoViewModel,
                dashboardViewModel = dashboardViewModel,
                musicQueueViewModel = musicQueueViewModel,
                themeSelection = themeSelection,
                onThemeSelection = onThemeSelection,
            )
            return
        }
        val performanceDiagnosticsViewModel: PerformanceDiagnosticsViewModel = viewModel(key = "performance:$sessionKey")
        val musicPlaylistViewModel: MusicPlaylistViewModel = viewModel(key = "playlists:$sessionKey")
        val context = LocalContext.current
        val destinationPreferences = remember(context) { DestinationVisibilityPreferences(context) }
        var showCollections by rememberSaveable { mutableStateOf(destinationPreferences.isVisible(OptionalDestination.Collections)) }
        var showSources by rememberSaveable { mutableStateOf(destinationPreferences.isVisible(OptionalDestination.Sources)) }
        val setDestinationVisible: (OptionalDestination, Boolean) -> Unit = { destination, visible ->
            destinationPreferences.setVisible(destination, visible)
            when (destination) {
                OptionalDestination.Collections -> showCollections = visible
                OptionalDestination.Sources -> showSources = visible
            }
        }
        val forceLibraryFlow = remember(libraryViewModel) {
            libraryViewModel.state.map { it.offlineOnly }.distinctUntilChanged()
        }
        val forceLibrary by forceLibraryFlow.collectAsStateWithLifecycle(
            initialValue = false,
        )
        val shellLibrariesFlow = remember(libraryViewModel) {
            libraryViewModel.state.map { it.shellProjection() }.distinctUntilChanged()
        }
        val shellLibraries by shellLibrariesFlow.collectAsStateWithLifecycle(
            initialValue = LibraryShellProjection(),
        )
        val lifecycleOwner = LocalLifecycleOwner.current
        DisposableEffect(lifecycleOwner, libraryViewModel, videoViewModel, dashboardViewModel) {
            val observer = LifecycleEventObserver { _, event ->
                if (event == Lifecycle.Event.ON_RESUME) {
					libraryViewModel.refreshLibrarySummaries()
                    // Quiet: the pull-to-refresh indicator belongs to a pull, not to every return to Home.
                    dashboardViewModel.refreshQuietly()
					videoViewModel.refreshAfterPlayback()
				}
            }
            lifecycleOwner.lifecycle.addObserver(observer)
            onDispose { lifecycleOwner.lifecycle.removeObserver(observer) }
        }
        val playVideo: (net.kdhmedia.mochil.video.VideoItem) -> Unit = { item ->
            playbackViewModel.pause()
            videoViewModel.play(item)
        }
        AppShell(
            forceLibrary = forceLibrary,
            playbackViewModel = playbackViewModel,
            expandedPlayer = { expanded, panel, onPanel, onDismiss ->
                MobileExpandedPlayerRoute(
                    expanded = expanded,
                    panel = panel,
                    playbackViewModel = playbackViewModel,
                    queueViewModel = musicQueueViewModel,
                    playlistViewModel = musicPlaylistViewModel,
                    playbackUrl = musicViewModel::playbackUrl,
                    artworkUrl = { if (it.libraryKind == "podcasts") podcastViewModel.artworkUrl(it) else musicViewModel.artworkUrl(it) },
                    authorizationHeader = libraryViewModel.authorizationHeader(),
                    onPanel = onPanel,
                    onDismiss = onDismiss,
                )
            },
            showCollections = showCollections,
            showSources = showSources,
            libraries = shellLibraries.libraries,
            selectedLibrary = shellLibraries.selected,
            onSelectLibrary = libraryViewModel::select,
            dashboard = { onOpenSettings, onOpenLibrary, onOpenCollections, onOpenSources ->
                MobileDashboardRoute(
                    dashboardViewModel = dashboardViewModel,
                    libraryViewModel = libraryViewModel,
                    musicViewModel = musicViewModel,
                    podcastViewModel = podcastViewModel,
                    playbackViewModel = playbackViewModel,
                    videoViewModel = videoViewModel,
                    showCollections = showCollections,
                    showSources = showSources,
                    onOpenSettings = onOpenSettings,
                    onOpenLibrary = onOpenLibrary,
                    onOpenCollections = onOpenCollections,
                    onOpenSources = onOpenSources,
                    onWatch = playVideo,
                )
            },
            library = { onOpenSettings, onOpenHome, onOpenCollections, onOpenSources ->
                val collectionViewModel: CollectionViewModel = viewModel(key = "collections:$sessionKey")
                val wikiViewModel: WikiViewModel = viewModel(key = "wiki:$sessionKey")
                val playlistState by musicPlaylistViewModel.state.collectAsStateWithLifecycle()
                MobileLibraryRoute(
                    libraryViewModel = libraryViewModel,
                    musicViewModel = musicViewModel,
                    podcastViewModel = podcastViewModel,
                    videoViewModel = videoViewModel,
                    wikiViewModel = wikiViewModel,
                    collectionViewModel = collectionViewModel,
                    playlistViewModel = musicPlaylistViewModel,
                    queueViewModel = musicQueueViewModel,
                    playbackViewModel = playbackViewModel,
                    playlistState = playlistState,
                    showCollections = showCollections,
                    showSources = showSources,
                    onOpenSettings = onOpenSettings,
                    onOpenHome = onOpenHome,
                    onOpenCollections = onOpenCollections,
                    onOpenSources = onOpenSources,
                    onWatch = playVideo,
                )
            },
            collections = { _, onShowLibrary -> MobileCollectionsRoute(libraryViewModel, onShowLibrary) },
            sources = { onBack -> SourceHubScreen(onBack) },
            settings = { onBack ->
                val appUpdateViewModel: AppUpdateViewModel = viewModel(key = "updates:$sessionKey")
                val downloadLocationViewModel: DownloadLocationViewModel = viewModel(key = "downloads:$sessionKey")
                val crashReportViewModel: CrashReportViewModel = viewModel(key = "crashes:$sessionKey")
                val annotationPrivacyViewModel: AnnotationPrivacyViewModel = viewModel(key = "annotationPrivacy:$sessionKey")
                MobileDeviceSettingsRoute(
                    loginState = state,
                    loginViewModel = loginViewModel,
                    libraryViewModel = libraryViewModel,
                    musicViewModel = musicViewModel,
                    podcastViewModel = podcastViewModel,
                    playbackViewModel = playbackViewModel,
                    updates = appUpdateViewModel,
                    downloads = downloadLocationViewModel,
                    crashReports = crashReportViewModel,
                    diagnostics = performanceDiagnosticsViewModel,
                    annotationPrivacy = annotationPrivacyViewModel,
                    themeSelection = themeSelection,
                    onThemeSelection = onThemeSelection,
                    showCollections = showCollections,
                    showSources = showSources,
                    onDestinationVisibility = setDestinationVisible,
                    onBack = onBack,
                )
            },
        )
    } else if (television) {
        TvPairingScreen(state, loginViewModel::updateServer, loginViewModel::startPairing, loginViewModel::cancelPairing)
    } else {
        LocalFirstApp(
            state = state,
            onServerChange = loginViewModel::updateServer,
            onUsernameChange = loginViewModel::updateUsername,
            onPasswordChange = loginViewModel::updatePassword,
            onLogin = loginViewModel::login,
        )
    }
}
