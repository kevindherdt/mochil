package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.hermes.ProvideHermesChatEntry
import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.PermanentNavigationDrawer
import androidx.compose.material3.Scaffold
import androidx.compose.runtime.Composable
import androidx.compose.foundation.layout.Row
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import net.kdhmedia.mochil.playback.PlaybackViewModel
import net.kdhmedia.mochil.catalogue.LibrarySummary

internal enum class MobileShellDestination { Home, Library, Collections, Sources, Settings }

internal fun handlesMobileShellBack(destination: MobileShellDestination): Boolean =
    destination != MobileShellDestination.Home

internal fun mobileShellBackTarget(destination: MobileShellDestination, returnDestination: MobileShellDestination): MobileShellDestination =
    when (destination) {
        MobileShellDestination.Home -> MobileShellDestination.Home
        MobileShellDestination.Library -> MobileShellDestination.Home
        MobileShellDestination.Collections, MobileShellDestination.Sources, MobileShellDestination.Settings ->
            returnDestination.takeIf { it == MobileShellDestination.Home || it == MobileShellDestination.Library }
                ?: MobileShellDestination.Home
    }

internal fun mobileShellAuxiliaryReturn(current: MobileShellDestination, previous: MobileShellDestination): MobileShellDestination =
    current.takeIf { it == MobileShellDestination.Home || it == MobileShellDestination.Library }
        ?: previous.takeIf { it == MobileShellDestination.Home || it == MobileShellDestination.Library }
        ?: MobileShellDestination.Home

/** Back from Settings returns to Collections or Sources when that is where Settings was opened from, otherwise as usual. */
internal fun mobileShellSettingsBack(openedFrom: MobileShellDestination, returnDestination: MobileShellDestination): MobileShellDestination =
    when (openedFrom) {
        MobileShellDestination.Collections, MobileShellDestination.Sources -> openedFrom
        else -> mobileShellBackTarget(MobileShellDestination.Settings, returnDestination)
    }

@Composable
fun AppShell(
    forceLibrary: Boolean,
    playbackViewModel: PlaybackViewModel,
    expandedPlayer: @Composable (expanded: Boolean, panel: String, onPanel: (String) -> Unit, onDismiss: () -> Unit) -> Unit,
    showCollections: Boolean,
    showSources: Boolean,
    libraries: List<LibrarySummary>,
    selectedLibrary: LibrarySummary?,
    onSelectLibrary: (LibrarySummary) -> Unit,
    dashboard: @Composable (onOpenSettings: () -> Unit, onOpenLibrary: () -> Unit, onOpenCollections: () -> Unit, onOpenSources: () -> Unit) -> Unit,
    library: @Composable (onOpenSettings: () -> Unit, onOpenHome: () -> Unit, onOpenCollections: () -> Unit, onOpenSources: () -> Unit) -> Unit,
    collections: @Composable (onBack: () -> Unit, onShowLibrary: () -> Unit) -> Unit,
    sources: @Composable (onBack: () -> Unit) -> Unit,
    settings: @Composable (onBack: () -> Unit) -> Unit,
) {
    var destination by rememberSaveable { mutableStateOf(MobileShellDestination.Home) }
    var returnDestination by rememberSaveable { mutableStateOf(MobileShellDestination.Home) }
    var settingsOpenedFrom by rememberSaveable { mutableStateOf(MobileShellDestination.Home) }
    var playerExpanded by rememberSaveable { mutableStateOf(false) }
    var playerPanel by rememberSaveable { mutableStateOf("player") }
    LaunchedEffect(forceLibrary) {
        if (forceLibrary && destination == MobileShellDestination.Home) {
            destination = MobileShellDestination.Library
            returnDestination = MobileShellDestination.Library
        }
    }
    LaunchedEffect(showCollections, showSources) {
        if ((!showCollections && destination == MobileShellDestination.Collections) ||
            (!showSources && destination == MobileShellDestination.Sources)
        ) destination = mobileShellBackTarget(destination, returnDestination)
    }
    val openAuxiliary: (MobileShellDestination) -> Unit = { target ->
        returnDestination = mobileShellAuxiliaryReturn(destination, returnDestination)
        destination = target
    }
    val navigateBack: () -> Unit = {
        destination = if (destination == MobileShellDestination.Settings) mobileShellSettingsBack(settingsOpenedFrom, returnDestination)
        else mobileShellBackTarget(destination, returnDestination)
    }
    BackHandler(enabled = handlesMobileShellBack(destination), onBack = navigateBack)
    val openHome = { destination = MobileShellDestination.Home }
    val openLibrary: (LibrarySummary) -> Unit = { library ->
        onSelectLibrary(library)
        destination = MobileShellDestination.Library
    }
    val openCollections = { openAuxiliary(MobileShellDestination.Collections) }
    val openSources = { openAuxiliary(MobileShellDestination.Sources) }
    val openSettings = { settingsOpenedFrom = destination; openAuxiliary(MobileShellDestination.Settings) }
    // The chat is its own screen: when it asks to go somewhere (its library switcher), go there once the shell is back.
    val pending = PendingShellOpen.target
    LaunchedEffect(pending, libraries) {
        val target = pending ?: return@LaunchedEffect
        when {
            target == PendingShellOpen.HOME -> openHome()
            target == PendingShellOpen.SETTINGS -> openSettings()
            else -> {
                val wanted = PendingShellOpen.libraryId(target)
                val library = libraries.firstOrNull { it.id == wanted } ?: run {
                    if (libraries.isNotEmpty()) PendingShellOpen.target = null // the library is gone: forget it, do not jump later
                    return@LaunchedEffect // not loaded yet: try again when it is
                }
                openLibrary(library)
            }
        }
        PendingShellOpen.target = null
    }
    val content: @Composable () -> Unit = {
            when (destination) {
                MobileShellDestination.Home -> dashboard(
                    openSettings,
                    { destination = MobileShellDestination.Library },
                    openCollections,
                    openSources,
                )
                MobileShellDestination.Library -> library(
                    openSettings,
                    openHome,
                    openCollections,
                    openSources,
                )
                MobileShellDestination.Collections -> collections(navigateBack) { destination = MobileShellDestination.Library }
                MobileShellDestination.Sources -> sources(navigateBack)
                MobileShellDestination.Settings -> settings(navigateBack)
            }
    }
    ProvideHermesChatEntry {
        Scaffold(bottomBar = { MobileMiniPlayerHost(playbackViewModel) { playerExpanded = true } }) { padding ->
            BoxWithConstraints(Modifier.fillMaxSize().padding(padding)) {
                val expanded = usesExpandedMobileNavigation(maxWidth.value.toInt())
                if (expanded) {
                    CompositionLocalProvider(LocalExpandedMobileNavigation provides true) {
                        PermanentNavigationDrawer(
                            drawerContent = {
                                ExpandedMobileNavigation(destination, libraries, selectedLibrary, showCollections, showSources, openHome, openLibrary, openCollections, openSources, openSettings)
                            },
                        ) { content() }
                    }
                } else {
                    // Screens such as Collections hand their icons up to this row, so they sit beside the switcher like in a library.
                    var headerActions by remember { mutableStateOf<List<HeaderAction>?>(null) }
                    val headerSlot = remember { { content: List<HeaderAction>? -> headerActions = content } }
                    CompositionLocalProvider(LocalHeaderActions provides headerSlot) {
                        Column(Modifier.fillMaxSize()) {
                            if (destination != MobileShellDestination.Home && destination != MobileShellDestination.Library) {
                                Row(Modifier.padding(horizontal = MOBILE_CONTENT_GUTTER), verticalAlignment = Alignment.CenterVertically) {
                                    Box(Modifier.weight(1f)) {
                                        MobileLibraryRail(libraries, selectedLibrary, destination, openHome, showCollections, showSources, openCollections, openSources, openSettings, openLibrary)
                                    }
                                    headerActions?.let { HeaderActionButtons(it) }
                                }
                            }
                            Box(Modifier.weight(1f)) { content() }
                        }
                    }
                }
            }
        }
    }
    expandedPlayer(playerExpanded, playerPanel, { playerPanel = it }) {
        playerExpanded = false
        playerPanel = "player"
    }
}
