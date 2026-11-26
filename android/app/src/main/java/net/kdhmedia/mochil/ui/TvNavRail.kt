package net.kdhmedia.mochil.ui

import android.view.KeyEvent
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.animateDpAsState
import androidx.compose.animation.core.snap
import androidx.compose.animation.core.tween
import net.kdhmedia.mochil.catalogue.LibraryKind
import net.kdhmedia.mochil.ui.theme.MochilChrome
import net.kdhmedia.mochil.ui.theme.MochilRadius
import net.kdhmedia.mochil.ui.theme.MochilSpace
import net.kdhmedia.mochil.ui.theme.LocalReduceMotion
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.focusGroup
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.ui.draw.clip
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Album
import androidx.compose.material.icons.filled.AutoStories
import androidx.compose.material.icons.filled.Headphones
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.filled.LibraryBooks
import androidx.compose.material.icons.filled.Mic
import androidx.compose.material.icons.filled.Movie
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material.icons.filled.Tv
import androidx.compose.material.icons.filled.VideoLibrary
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.focus.FocusDirection
import androidx.compose.ui.geometry.CornerRadius
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusProperties
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.input.key.KeyEventType
import androidx.compose.ui.input.key.onPreviewKeyEvent
import androidx.compose.ui.input.key.type
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.selected
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.map
import net.kdhmedia.mochil.catalogue.LibraryShellProjection
import net.kdhmedia.mochil.catalogue.LibrarySummary
import net.kdhmedia.mochil.catalogue.LibraryViewModel
import net.kdhmedia.mochil.catalogue.shellProjection

/**
 * Width the rail always occupies; it opens over the content, never pushing it, while it has focus. The resting width
 * follows the TV sheets: a slim icon column at ten feet, wide enough for a 20dp glyph with air around it.
 */
internal val TvNavRailWidth = 48.dp
private val TvNavRailOpenWidth = 224.dp

internal const val TV_RAIL_HOME = "home"
internal const val TV_RAIL_SETTINGS = "settings"
internal fun tvRailLibraryKey(id: String) = "library:$id"

/** The rail entry that matches the screen being shown, so Left always lands on "where I am". */
internal fun tvRailSelectedKey(route: String, selectedLibraryId: String?): String? = when (route) {
    "Home" -> TV_RAIL_HOME
    "Settings" -> TV_RAIL_SETTINGS
    "Library" -> selectedLibraryId?.let(::tvRailLibraryKey)
    else -> null
}

/** Delegates to [net.kdhmedia.mochil.catalogue.LibraryKind] so the rail's grouping of manga/comics with
 * books can never drift from MediaKindStyle's or TvConnectedApp's grouping again. */
internal fun tvRailIcon(kind: String): ImageVector = when (LibraryKind.fromWire(kind)) {
    LibraryKind.Movies -> Icons.Default.Movie
    LibraryKind.Tv, LibraryKind.Anime -> Icons.Default.Tv
    LibraryKind.Music -> Icons.Default.Album
    LibraryKind.Audiobooks -> Icons.Default.Headphones
    LibraryKind.Podcasts -> Icons.Default.Mic
    LibraryKind.Books, LibraryKind.LightNovels, LibraryKind.Manga, LibraryKind.Comics, LibraryKind.Mixed, LibraryKind.Calibre -> Icons.Default.AutoStories
    LibraryKind.Vault -> Icons.Default.VideoLibrary
}

/**
 * The one place to move between Home, every library and Settings. A slim icon column that opens with labels when it
 * has focus; Left from any screen's left edge reaches it, and Back from it goes Home.
 */
/** Watches only the library list and selection, so catalogue paging never recomposes the rail. */
@Composable
internal fun TvNavRailHost(
    libraryViewModel: LibraryViewModel,
    route: String,
    onHome: () -> Unit,
    onLibrary: (LibrarySummary) -> Unit,
    onSettings: () -> Unit,
    onBack: () -> Unit,
    entry: FocusRequester,
    modifier: Modifier = Modifier,
) {
    val flow = remember(libraryViewModel) { libraryViewModel.state.map { it.shellProjection() }.distinctUntilChanged() }
    val shell by flow.collectAsStateWithLifecycle(initialValue = LibraryShellProjection())
    TvNavRail(shell.libraries.inTvOrder(), tvRailSelectedKey(route, shell.selected?.id), onHome, onLibrary, onSettings, onBack, entry, modifier)
}

@OptIn(androidx.compose.ui.ExperimentalComposeUiApi::class)
@Composable
internal fun TvNavRail(
    libraries: List<LibrarySummary>,
    selectedKey: String?,
    onHome: () -> Unit,
    onLibrary: (LibrarySummary) -> Unit,
    onSettings: () -> Unit,
    onBack: () -> Unit,
    entry: FocusRequester,
    modifier: Modifier = Modifier,
) {
    var expanded by remember { mutableStateOf(false) }
    val current = remember { FocusRequester() }
    val currentIsListed = selectedKey == TV_RAIL_HOME || selectedKey == TV_RAIL_SETTINGS ||
        libraries.any { tvRailLibraryKey(it.id) == selectedKey }
    val selectedLibraryIndex = libraries.indexOfFirst { tvRailLibraryKey(it.id) == selectedKey }
    val libraryListState = rememberLazyListState(initialFirstVisibleItemIndex = selectedLibraryIndex.coerceAtLeast(0))
    // Prepare the selected destination only while focus is outside the rail. Re-keying the
    // effect cancels obsolete scrolls when the selection or list changes.
    LaunchedEffect(selectedKey, libraries, expanded) {
        if (!expanded && selectedLibraryIndex >= 0) libraryListState.scrollToItem(selectedLibraryIndex)
    }
    val width by animateDpAsState(
        if (expanded) TvNavRailOpenWidth else TvNavRailWidth,
        if (LocalReduceMotion.current) snap() else tween(durationMillis = 170, easing = FastOutSlowInEasing),
        label = "tvRailWidth",
    )
    BoxWithConstraints(modifier.fillMaxHeight().width(width)) {
        Column(
            Modifier.align(Alignment.CenterStart).width(width).heightIn(max = (maxHeight - MochilSpace.s24).coerceAtLeast(0.dp))
                .padding(horizontal = MochilSpace.s4)
                .background(MaterialTheme.colorScheme.surface.copy(alpha = .94f), RoundedCornerShape(MochilRadius.r16))
                .border(MochilChrome.hairline, MaterialTheme.colorScheme.outlineVariant.copy(alpha = .5f), RoundedCornerShape(MochilRadius.r16))
                .padding(horizontal = 2.dp, vertical = MochilSpace.s4)
                .onFocusChanged { expanded = it.hasFocus }
                .focusRequester(entry)
                .focusProperties { enter = { if (currentIsListed) current else FocusRequester.Default } }
                .focusGroup()
                .onPreviewKeyEvent { event ->
                    if (event.nativeKeyEvent.keyCode != KeyEvent.KEYCODE_BACK) return@onPreviewKeyEvent false
                    if (event.type == KeyEventType.KeyUp) onBack()
                    true
                },
            verticalArrangement = Arrangement.spacedBy(MochilChrome.hairline),
        ) {
            val focusManager = LocalFocusManager.current
            fun item(key: String, tap: () -> Unit): () -> Unit = {
                // Choosing where you already are just steps into it, rather than reloading it.
                if (key == selectedKey) focusManager.moveFocus(FocusDirection.Right) else tap()
            }
            TvNavRailItem("Home", Icons.Default.Home, expanded, selectedKey == TV_RAIL_HOME,
                if (selectedKey == TV_RAIL_HOME) Modifier.focusRequester(current) else Modifier, item(TV_RAIL_HOME, onHome))
            if (expanded && libraries.isNotEmpty()) TvNavRailHeading("Libraries")
            LazyColumn(Modifier.weight(1f, fill = false), state = libraryListState, verticalArrangement = Arrangement.spacedBy(2.dp)) {
                items(libraries, key = LibrarySummary::id) { library ->
                    val key = tvRailLibraryKey(library.id)
                    TvNavRailItem(library.name, tvRailIcon(library.kind), expanded, selectedKey == key,
                        if (selectedKey == key) Modifier.focusRequester(current) else Modifier, item(key) { onLibrary(library) })
                }
            }
            TvNavRailItem("Settings", Icons.Default.Settings, expanded, selectedKey == TV_RAIL_SETTINGS,
                if (selectedKey == TV_RAIL_SETTINGS) Modifier.focusRequester(current) else Modifier, item(TV_RAIL_SETTINGS, onSettings))
        }
    }
}

@Composable
private fun TvNavRailHeading(label: String) {
    Row(
        Modifier.fillMaxWidth().padding(start = MochilSpace.s8, top = MochilSpace.s8, bottom = MochilSpace.s4),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(label, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1)
        Box(Modifier.padding(start = MochilSpace.s8).weight(1f).height(MochilChrome.hairline).background(MaterialTheme.colorScheme.outlineVariant.copy(alpha = .6f)))
    }
}

@Composable
private fun TvNavRailItem(label: String, icon: ImageVector, expanded: Boolean, selected: Boolean, modifier: Modifier, onClick: () -> Unit) {
    var focused by remember { mutableStateOf(false) }
    val shape = RoundedCornerShape(MochilRadius.r12)
    val accent = MaterialTheme.colorScheme.primary
    // Two states, two signals, both from the palette: the destination you are on wears a tinted tile and a leading
    // accent bar; the destination the remote is on wears the accent filled, so focus reads across a room. They used to
    // be white on both counts, which looked the same in every theme and hid the difference between "here" and "about
    // to go here".
    val background = when {
        focused -> accent
        selected -> accent.copy(alpha = .16f)
        else -> androidx.compose.ui.graphics.Color.Transparent
    }
    val tint = when {
        focused -> MaterialTheme.colorScheme.onPrimary
        selected -> accent
        else -> MaterialTheme.colorScheme.onSurface
    }
    Row(
        modifier.fillMaxWidth().height(if (expanded) 40.dp else 36.dp)
            .onFocusChanged { focused = it.isFocused }
            .clip(shape)
            .background(background, shape)
            .drawBehind {
                if (selected && !focused) {
                    drawRoundRect(accent, Offset(0f, size.height * .22f), Size(MochilChrome.ring.toPx(), size.height * .56f), CornerRadius(MochilRadius.r4.toPx()))
                }
            }
            // One node per destination: the label is the description whether or not the rail is open, so a remote that
            // reads the screen names the destination rather than reading an icon; the glyph is decorative.
            .semantics {
                contentDescription = label
                this.selected = selected
            }
            .clickable(interactionSource = remember { androidx.compose.foundation.interaction.MutableInteractionSource() }, indication = null, role = Role.Tab, onClick = onClick)
            // The resting rail is 48dp: 4 + 2 of rail padding either side leaves 36, so an 8dp inset fits a 20dp glyph.
            .padding(horizontal = if (expanded) MochilSpace.s12 else MochilSpace.s8),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(icon, contentDescription = null, Modifier.size(if (expanded) 22.dp else 20.dp), tint = tint)
        if (expanded) {
            Spacer(Modifier.width(MochilSpace.s12))
            Text(
                label,
                color = tint,
                style = MaterialTheme.typography.bodyMedium,
                fontWeight = if (selected || focused) FontWeight.SemiBold else FontWeight.Normal,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
    }
}
