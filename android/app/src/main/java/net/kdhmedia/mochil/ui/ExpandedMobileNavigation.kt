package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.hermes.LocalHermesChatEntry
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.FolderOpen
import androidx.compose.material.icons.outlined.Explore
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material.icons.outlined.ChatBubbleOutline
import androidx.compose.material.icons.outlined.Cloud
import androidx.compose.material.icons.automirrored.outlined.MenuBook
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationDrawerItem
import androidx.compose.material3.NavigationDrawerItemDefaults
import androidx.compose.material3.PermanentDrawerSheet
import androidx.compose.runtime.Composable
import androidx.compose.runtime.compositionLocalOf
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.material3.Text
import net.kdhmedia.mochil.catalogue.LibrarySummary

internal const val EXPANDED_MOBILE_NAVIGATION_MIN_WIDTH_DP = 840
internal val LocalExpandedMobileNavigation = compositionLocalOf { false }

internal fun usesExpandedMobileNavigation(widthDp: Int): Boolean =
    widthDp >= EXPANDED_MOBILE_NAVIGATION_MIN_WIDTH_DP

/** Material permanent navigation for tablets, unfolded devices, and wide windows. */
@Composable
internal fun ExpandedMobileNavigation(
    destination: MobileShellDestination,
    libraries: List<LibrarySummary>,
    selectedLibrary: LibrarySummary?,
    showCollections: Boolean,
    showSources: Boolean,
    onHome: () -> Unit,
    onLibrary: (LibrarySummary) -> Unit,
    onCollections: () -> Unit,
    onSources: () -> Unit,
    onSettings: () -> Unit,
) {
    val chatEntry = LocalHermesChatEntry.current
    PermanentDrawerSheet(
        modifier = Modifier.width(220.dp).fillMaxHeight(),
        drawerContainerColor = MaterialTheme.colorScheme.surface,
    ) {
        LazyColumn(contentPadding = PaddingValues(horizontal = 12.dp, vertical = 16.dp)) {
            item("home") {
                ExpandedDestination("Discover", destination == MobileShellDestination.Discover, { Icon(Icons.Outlined.Explore, null) }, onHome)
            }
            items(libraries, key = LibrarySummary::id) { library ->
                ExpandedDestination(
                    library.name,
                    destination == MobileShellDestination.Library && selectedLibrary?.id == library.id,
                    { Icon(Icons.AutoMirrored.Outlined.MenuBook, null) },
                ) { onLibrary(library) }
            }
            if (showCollections) item("collections") {
                ExpandedDestination("Collections", destination == MobileShellDestination.Collections, { Icon(Icons.Outlined.FolderOpen, null) }, onCollections)
            }
            if (showSources) item("sources") {
                ExpandedDestination("Sources", destination == MobileShellDestination.Sources, { Icon(Icons.Outlined.Cloud, null) }, onSources)
            }
            chatEntry?.let { chat ->
                item("chat") { ExpandedDestination(chat.name, false, { Icon(Icons.Outlined.ChatBubbleOutline, null) }, chat.open) }
            }
            item("settings") {
                ExpandedDestination("Settings", destination == MobileShellDestination.Settings, { Icon(Icons.Default.Settings, null) }, onSettings)
            }
        }
    }
}

@Composable
private fun ExpandedDestination(label: String, selected: Boolean, icon: @Composable () -> Unit, onClick: () -> Unit) {
    NavigationDrawerItem(
        label = { Text(label, maxLines = 1, overflow = TextOverflow.Ellipsis) },
        selected = selected,
        onClick = onClick,
        icon = icon,
        modifier = Modifier,
        colors = NavigationDrawerItemDefaults.colors(
            selectedContainerColor = MaterialTheme.colorScheme.primaryContainer,
            selectedTextColor = MaterialTheme.colorScheme.onPrimaryContainer,
            selectedIconColor = MaterialTheme.colorScheme.onPrimaryContainer,
        ),
    )
}
