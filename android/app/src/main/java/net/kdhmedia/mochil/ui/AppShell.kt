package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.ui.kit.PageHeader
import net.kdhmedia.mochil.hermes.LocalHermesChatEntry
import net.kdhmedia.mochil.hermes.ProvideHermesChatEntry
import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.ime
import androidx.compose.foundation.layout.navigationBars
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.PermanentNavigationDrawer
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.foundation.layout.Row
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.text.style.TextAlign
import net.kdhmedia.mochil.playback.PlaybackViewModel
import net.kdhmedia.mochil.catalogue.LibrarySummary
import net.kdhmedia.mochil.ui.theme.MochilSpace

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
    library: @Composable (onOpenCollections: () -> Unit) -> Unit,
    collections: @Composable (onBack: () -> Unit, onShowLibrary: () -> Unit) -> Unit,
    sources: @Composable (onBack: () -> Unit) -> Unit,
    settings: @Composable (onBack: () -> Unit) -> Unit,
) {
    var destination by rememberSaveable(stateSaver = MobileShellDestinationSaver) { mutableStateOf(MOBILE_SHELL_START) }
    // The book library the Library tab returns to after a visit to Notes (a vault is a library too, and one is selected at a time).
    var lastLibraryId by rememberSaveable { mutableStateOf<String?>(null) }
    var playerExpanded by rememberSaveable { mutableStateOf(false) }
    var playerPanel by rememberSaveable { mutableStateOf("player") }
    val openDiscover = { destination = MobileShellDestination.Discover }
    val openLibrary: (LibrarySummary) -> Unit = { library ->
        onSelectLibrary(library)
        destination = MobileShellDestination.Library
    }
    // The Library tab, and where Back lands: a book library, never the vault that was open.
    val openLibraryTab = {
        libraryTabTarget(libraries, selectedLibrary, lastLibraryId)?.let(onSelectLibrary)
        destination = MobileShellDestination.Library
    }
    val openNotes = {
        val vault = notesLibrary(libraries, selectedLibrary)
        if (vault != null) {
            onSelectLibrary(vault)
            destination = MobileShellDestination.Library
        } else {
            destination = MobileShellDestination.Notes
        }
    }
    val openCollections = { destination = MobileShellDestination.Collections }
    val openSources = { destination = MobileShellDestination.Sources }
    val openSettings = { destination = MobileShellDestination.Settings }
    LaunchedEffect(forceLibrary) {
        if (forceLibrary && destination == MobileShellDestination.Discover) destination = MobileShellDestination.Library
    }
    LaunchedEffect(showCollections, showSources) {
        if ((!showCollections && destination == MobileShellDestination.Collections) ||
            (!showSources && destination == MobileShellDestination.Sources)
        ) openLibraryTab()
    }
    LaunchedEffect(selectedLibrary?.id, selectedLibrary?.kind) {
        selectedLibrary?.takeUnless(::isVault)?.let { lastLibraryId = it.id }
    }
    // Notes asked for before the library list arrived (or restored after the process died): open the vault once there is one.
    LaunchedEffect(destination, libraries) {
        if (destination == MobileShellDestination.Notes) notesLibrary(libraries, selectedLibrary)?.let { vault ->
            onSelectLibrary(vault)
            destination = MobileShellDestination.Library
        }
    }
    val vaultOpen = vaultBackAvailable(libraries, selectedLibrary, lastLibraryId)
    BackHandler(enabled = handlesMobileShellBack(destination, vaultOpen), onBack = openLibraryTab)
    // The chat is its own screen: when it asks to go somewhere (its library switcher), go there once the shell is back.
    val pending = PendingShellOpen.target
    LaunchedEffect(pending, libraries) {
        val target = pending ?: return@LaunchedEffect
        when {
            target == PendingShellOpen.DISCOVER -> openDiscover()
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
                MobileShellDestination.Discover -> dashboard(
                    openSettings,
                    openLibraryTab,
                    openCollections,
                    openSources,
                )
                // Library and Notes share one call site: Notes is the library screen with a vault open.
                MobileShellDestination.Library -> library(openCollections)
                MobileShellDestination.Notes ->
                    if (showsNotesEmpty(libraries)) NotesEmpty() else library(openCollections)
                MobileShellDestination.Collections -> collections(openLibraryTab) { destination = MobileShellDestination.Library }
                MobileShellDestination.Sources -> sources(openLibraryTab)
                MobileShellDestination.Settings -> settings(openLibraryTab)
            }
    }
    val expandedNavigation = usesExpandedMobileNavigation(LocalConfiguration.current.screenWidthDp)
    // Read through derivedStateOf: the raw inset changes every frame of the keyboard animation, and only "is it up" matters here.
    val ime = WindowInsets.ime
    val density = LocalDensity.current
    val keyboardVisible by remember(ime, density) { derivedStateOf { ime.getBottom(density) > 0 } }
    ProvideHermesChatEntry {
        val chat = LocalHermesChatEntry.current
        Scaffold(
            bottomBar = {
                // The navigation-bar inset is owned here, once: by the dock when it is shown, otherwise by this column. A column
                // that holds only an idle player would measure zero and let Scaffold stop reserving the inset for the content.
                val dockShown = dockVisible(expandedNavigation, keyboardVisible)
                Column(if (dockShown) Modifier else Modifier.windowInsetsPadding(WindowInsets.navigationBars)) {
                    MobileMiniPlayerHost(playbackViewModel, navigationInset = false) { playerExpanded = true }
                    if (dockShown) {
                        ShellDock(destination, selectedLibrary, chat) { target ->
                            when (target) {
                                MobileShellDestination.Library -> openLibraryTab()
                                MobileShellDestination.Discover -> openDiscover()
                                MobileShellDestination.Notes -> openNotes()
                                MobileShellDestination.Settings -> openSettings()
                                else -> destination = target
                            }
                        }
                    }
                }
            },
        ) { padding ->
            Box(Modifier.fillMaxSize().padding(padding)) {
                if (expandedNavigation) {
                    CompositionLocalProvider(LocalExpandedMobileNavigation provides true) {
                        PermanentNavigationDrawer(
                            drawerContent = {
                                ExpandedMobileNavigation(destination, libraries, selectedLibrary, showCollections, showSources, openDiscover, openLibrary, openCollections, openSources, openSettings)
                            },
                        ) { content() }
                    }
                } else {
                    // Screens such as Collections hand their icons up to this row, so they sit beside the page title.
                    var headerActions by remember { mutableStateOf<List<HeaderAction>?>(null) }
                    val headerSlot = remember { { content: List<HeaderAction>? -> headerActions = content } }
                    CompositionLocalProvider(LocalHeaderActions provides headerSlot) {
                        Column(Modifier.fillMaxSize()) {
                            shellPageTitle(destination)?.let { title ->
                                Row(Modifier.padding(horizontal = MOBILE_CONTENT_GUTTER), verticalAlignment = Alignment.CenterVertically) {
                                    PageHeader(title, openLibraryTab, Modifier.weight(1f))
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

/** Notes with no vault on the server: one line that says what it is and one that says what to do. */
@Composable
private fun NotesEmpty() {
    Column(
        Modifier.fillMaxSize().padding(MochilSpace.s24),
        verticalArrangement = Arrangement.spacedBy(MochilSpace.s8, Alignment.CenterVertically),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text("No notes library", style = MaterialTheme.typography.titleMedium)
        Text(
            "Add a notes folder (a vault) on your server, or show it under Settings → Libraries.",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            textAlign = TextAlign.Center,
        )
    }
}
