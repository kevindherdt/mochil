package net.kdhmedia.mochil.ui

import androidx.activity.compose.LocalActivity
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.platform.LocalContext
import net.kdhmedia.mochil.ui.theme.MochilSpace
import net.kdhmedia.mochil.catalogue.LibraryShellProjection
import net.kdhmedia.mochil.catalogue.LibraryKind
import net.kdhmedia.mochil.catalogue.LibraryCategory
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.LifecycleEventEffect
import androidx.lifecycle.Lifecycle
import net.kdhmedia.mochil.audio.MusicViewModel
import net.kdhmedia.mochil.audio.MusicQueueViewModel
import net.kdhmedia.mochil.audio.canPersistMusicPlaybackQueue
import net.kdhmedia.mochil.auth.LoginViewModel
import net.kdhmedia.mochil.catalogue.LibrarySummary
import net.kdhmedia.mochil.catalogue.LibraryViewModel
import net.kdhmedia.mochil.dashboard.DashboardViewModel
import net.kdhmedia.mochil.playback.PlaybackViewModel
import net.kdhmedia.mochil.podcast.PodcastViewModel
import net.kdhmedia.mochil.update.AppUpdateViewModel
import net.kdhmedia.mochil.video.VideoItem
import net.kdhmedia.mochil.video.VideoViewModel
import net.kdhmedia.mochil.ui.theme.AppThemeSelection

/** The smallest margin kept clear of the panel edge, so the rail and dock survive a TV that overscans. */
internal val TvScreenInset = 24.dp

/**
 * How much a panel may hide around its edges, in dp. Televisions overscan by a few percent of the picture and
 * the design specification puts it at 5% of the short side -- on a 1080p panel that is 54dp, more than twice
 * the old fixed 24dp, which is why content sat close enough to the edge to lose its first letters on a real
 * set. The fixed inset stays as a floor for small windows.
 */
internal fun tvOverscan(width: Float, height: Float): Dp =
    maxOf(TvScreenInset, (minOf(width, height) * 0.05f).dp)

/** Shared frame for every TV route, including the live playback dock's reduced viewport. */
@Composable
internal fun TvContentFrame(showRail: Boolean, playbackDockVisible: Boolean, content: @Composable () -> Unit) {
    BoxWithConstraints(Modifier.fillMaxSize()) {
        val inset = tvOverscan(maxWidth.value, maxHeight.value)
        Box(
            // Start and bottom only. A review argued for top and end too, on the grounds that a set cuts the outer band off;
            // the detail tests disagree — a 24dp top inset clips a show title out of its own viewport — and every screen
            // already carries its own top breathing room. Left as measured rather than as argued.
            Modifier.fillMaxSize().padding(
                start = if (showRail) TvNavRailWidth + inset else inset,
                bottom = if (showRail && playbackDockVisible) inset + 64.dp else inset,
            ),
        ) { content() }
    }
}

private enum class TvRoute { Home, Library, NowPlaying, Settings }

internal enum class TvResumeRefresh { Libraries, Dashboard, VideoProgress }

/** Delegates to [LibraryKind] so "which kinds get a resume refresh" can never drift from the rail's or
 * MediaKindStyle's grouping of the same kinds. */
internal fun tvResumeRefreshes(route: String, libraryKind: String): Set<TvResumeRefresh> = when (route) {
    "Home" -> setOf(TvResumeRefresh.Libraries, TvResumeRefresh.Dashboard)
    "Library" -> buildSet {
        add(TvResumeRefresh.Libraries)
        if (LibraryKind.fromWire(libraryKind).category == LibraryCategory.Video) add(TvResumeRefresh.VideoProgress)
    }
    "Settings" -> setOf(TvResumeRefresh.Libraries)
    else -> emptySet()
}

@Composable
internal fun TvConnectedApp(
    loginViewModel: LoginViewModel,
    libraryViewModel: LibraryViewModel,
    musicViewModel: MusicViewModel,
    playbackViewModel: PlaybackViewModel,
    appUpdateViewModel: AppUpdateViewModel,
    podcastViewModel: PodcastViewModel,
    videoViewModel: VideoViewModel,
    dashboardViewModel: DashboardViewModel,
    musicQueueViewModel: MusicQueueViewModel,
    themeSelection: AppThemeSelection,
    onThemeSelection: (AppThemeSelection) -> Unit,
) {
    val activity = LocalActivity.current
    var route by rememberSaveable { mutableStateOf(TvRoute.Home) }
    var nowPlayingReturnRoute by rememberSaveable { mutableStateOf(TvRoute.Home) }
    var playbackDockVisible by remember { mutableStateOf(false) }
    LifecycleEventEffect(Lifecycle.Event.ON_RESUME) {
        tvResumeRefreshes(route.name, libraryViewModel.state.value.selected?.kind.orEmpty()).forEach { refresh ->
            when (refresh) {
                TvResumeRefresh.Libraries -> libraryViewModel.refreshLibrarySummaries()
                TvResumeRefresh.Dashboard -> dashboardViewModel.refreshQuietly()
                TvResumeRefresh.VideoProgress -> videoViewModel.refreshAfterPlayback()
            }
        }
    }
    val libraryGridPositions = remember { TvGridPositionStore() }
    val railFocus = remember { FocusRequester() }
    val selectLibrary: (LibrarySummary) -> Unit = { selected ->
        libraryViewModel.select(selected)
        when (LibraryKind.fromWire(selected.kind)) {
            LibraryKind.Movies, LibraryKind.Tv, LibraryKind.Anime -> videoViewModel.open(selected.id, selected.kind)
            LibraryKind.Music, LibraryKind.Audiobooks -> {
                if (selected.kind == "music") musicQueueViewModel.ensureLoaded()
                musicViewModel.open(selected.id, selected.kind)
            }
            LibraryKind.Podcasts -> podcastViewModel.open(selected.id)
            else -> {}
        }
    }
    val playVideo: (VideoItem) -> Unit = { item ->
        playbackViewModel.pause()
        videoViewModel.play(item)
    }

    Surface(Modifier.fillMaxSize(), color = MaterialTheme.colorScheme.background) {
    val backdrop = remember { androidx.compose.runtime.mutableStateOf("") }
    val appContext = androidx.compose.ui.platform.LocalContext.current
    val libraryOrder = remember { TvLibraryOrder(appContext) }
    androidx.compose.runtime.CompositionLocalProvider(LocalTvBackdrop provides backdrop, LocalTvLibraryOrder provides libraryOrder) {
    Box(Modifier.fillMaxSize()) {
    TvBackdropLayer(backdrop, libraryViewModel.authorizationHeader())
    RecordTvFirstFrame(route.name.lowercase())
    val showRail = route != TvRoute.NowPlaying
    // Back from the top of any screen lands on the navigation rail; Back from the rail goes Home.
    val leaveToRail = { railFocus.requestFocus(); Unit }
    TvContentFrame(showRail, playbackDockVisible) { when (route) {
        TvRoute.Settings -> TvSettingsRoute(loginViewModel, libraryViewModel, appUpdateViewModel, themeSelection, onThemeSelection, leaveToRail)
        TvRoute.NowPlaying -> TvNowPlayingScreen(
            playbackViewModel = playbackViewModel,
            artworkUrl = { track -> if (track.libraryKind == "podcasts") podcastViewModel.artworkUrl(track) else musicViewModel.artworkUrl(track) },
            authorizationHeader = libraryViewModel.authorizationHeader(),
            onPersistQueue = { tracks ->
                if (canPersistMusicPlaybackQueue(tracks)) musicQueueViewModel.replace(tracks)
            },
            onBack = { route = nowPlayingReturnRoute },
        )
        TvRoute.Library -> TvLibraryRoute(
            libraryViewModel = libraryViewModel,
            musicViewModel = musicViewModel,
            musicQueueViewModel = musicQueueViewModel,
            playbackViewModel = playbackViewModel,
            podcastViewModel = podcastViewModel,
            videoViewModel = videoViewModel,
            gridPositions = libraryGridPositions,
            onSelectLibrary = selectLibrary,
            onPlayVideo = playVideo,
            onHome = leaveToRail,
        )
        TvRoute.Home -> TvHomeRoute(
            libraryViewModel = libraryViewModel,
            musicViewModel = musicViewModel,
            playbackViewModel = playbackViewModel,
            podcastViewModel = podcastViewModel,
            videoViewModel = videoViewModel,
            dashboardViewModel = dashboardViewModel,
            focusRail = leaveToRail,
            onPlayVideo = playVideo,
        )
    } }
    // Above the content, below the rail: Box children paint in declaration order, so this has to come after the frame —
    // composed before it, every route's content drew over the clock and it was only visible where the corner happened to be
    // empty (Home).
    if (showRail) BoxWithConstraints(Modifier.fillMaxSize()) {
        val corner = tvOverscan(maxWidth.value, maxHeight.value)
        TvStatusCluster(Modifier.align(Alignment.TopEnd).padding(end = corner + MochilSpace.s24, top = corner + MochilSpace.s12))
    }
    if (showRail) TvNavRailHost(
        libraryViewModel = libraryViewModel,
        route = route.name,
        onHome = { route = TvRoute.Home },
        onLibrary = { selected ->
            selectLibrary(selected)
            route = TvRoute.Library
        },
        onSettings = { route = TvRoute.Settings },
        onBack = { if (route == TvRoute.Home) activity?.finish() else route = TvRoute.Home },
        entry = railFocus,
        modifier = Modifier.align(Alignment.CenterStart).padding(start = TvScreenInset),
    )
    if (route != TvRoute.NowPlaying) TvPlaybackDock(
        playbackViewModel = playbackViewModel,
        artworkUrl = { track -> if (track.libraryKind == "podcasts") podcastViewModel.artworkUrl(track) else musicViewModel.artworkUrl(track) },
        authorizationHeader = libraryViewModel.authorizationHeader(),
        onVisibilityChanged = { playbackDockVisible = it },
    ) {
        nowPlayingReturnRoute = route
        route = TvRoute.NowPlaying
    }
    }
    }
    }
}
