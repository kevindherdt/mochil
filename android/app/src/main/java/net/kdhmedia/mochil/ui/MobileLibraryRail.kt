package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.hermes.LocalHermesChatEntry
import net.kdhmedia.mochil.ui.theme.LocalEInkPalette
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.gestures.snapping.rememberSnapFlingBehavior
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.ChatBubbleOutline
import androidx.compose.material.icons.outlined.Cloud
import androidx.compose.material.icons.outlined.FolderOpen
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material3.FilterChip
import androidx.compose.material3.FilterChipDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Modifier
import androidx.compose.ui.Alignment
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.catalogue.LibrarySummary
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.filled.KeyboardArrowDown
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.draw.clip
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight

/** One compact switcher for touch devices: the current place as a coloured icon and a chevron; the menu lists Home, every library, then Collections, Sources and Settings. */
@Composable
internal fun MobileLibraryRail(
    libraries: List<LibrarySummary>,
    selectedLibrary: LibrarySummary?,
    destination: MobileShellDestination,
    onOpenHome: () -> Unit,
    showCollections: Boolean,
    showSources: Boolean,
    onOpenCollections: () -> Unit,
    onOpenSources: () -> Unit,
    onOpenSettings: () -> Unit,
    onSelectLibrary: (LibrarySummary) -> Unit,
    chatName: String? = null,
) {
    // chatName is set when the switcher sits inside the chat itself: the chat is then the current place.
    var open by remember { mutableStateOf(false) }
    val muted = MaterialTheme.colorScheme.onSurfaceVariant
    val current = if (chatName != null) Triple(Icons.Outlined.ChatBubbleOutline, muted, chatName) else when (destination) {
        MobileShellDestination.Library -> selectedLibrary?.let {
            mediaKindStyle(
                it.kind,
                monochromeColor = if (LocalEInkPalette.current) MaterialTheme.colorScheme.onSurface else null,
            ).let { style -> Triple(style.icon, style.tint(MaterialTheme.colorScheme.background), selectedLibrary.name) }
        }
        MobileShellDestination.Collections -> Triple(Icons.Outlined.FolderOpen, muted, "Collections")
        MobileShellDestination.Sources -> Triple(Icons.Outlined.Cloud, muted, "Sources")
        MobileShellDestination.Settings -> Triple(Icons.Default.Settings, muted, "Settings")
        else -> null
    } ?: Triple(Icons.Default.Home, muted, "Home")
    Box(Modifier.fillMaxWidth().height(MOBILE_LIBRARY_RAIL_HEIGHT), contentAlignment = Alignment.CenterStart) {
        val shape = androidx.compose.foundation.shape.RoundedCornerShape(16.dp)
        Row(
            Modifier.clip(shape).background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = .45f))
                .border(androidx.compose.foundation.BorderStroke(1.dp, MaterialTheme.colorScheme.outline.copy(alpha = .22f)), shape)
                .clickable(onClickLabel = "Change library", role = androidx.compose.ui.semantics.Role.Button) { open = true }
                .semantics { contentDescription = "${current.third}, change library" }
                .padding(horizontal = 14.dp, vertical = 10.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(current.first, null, Modifier.size(22.dp), tint = current.second)
            // Two libraries of one kind share an icon and colour, so the name has to be visible.
            Text(
                current.third,
                Modifier.padding(start = 8.dp).widthIn(max = 160.dp),
                color = MaterialTheme.colorScheme.onSurface,
                style = MaterialTheme.typography.titleSmall,
                fontWeight = FontWeight.SemiBold,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            Icon(Icons.Default.KeyboardArrowDown, null, Modifier.padding(start = 6.dp).size(20.dp), tint = muted)
        }
        DropdownMenu(open, { open = false }) {
            @Composable
            fun entry(icon: androidx.compose.ui.graphics.vector.ImageVector, tint: Color, name: String, count: Int?, on: Boolean, click: () -> Unit) {
                DropdownMenuItem(
                    text = { Text(name, color = if (on) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurface, fontWeight = if (on) FontWeight.SemiBold else FontWeight.Medium, maxLines = 1, overflow = TextOverflow.Ellipsis) },
                    leadingIcon = { Icon(icon, null, tint = tint) },
                    trailingIcon = count?.let { { Text(java.text.NumberFormat.getIntegerInstance().format(it), color = muted, style = MaterialTheme.typography.labelMedium) } },
                    onClick = { open = false; click() },
                )
            }
            entry(Icons.Default.Home, muted, "Home", null, chatName == null && destination == MobileShellDestination.Home, onOpenHome)
            libraries.forEach { library ->
                val style = mediaKindStyle(
                    library.kind,
                    monochromeColor = if (LocalEInkPalette.current) MaterialTheme.colorScheme.onSurface else null,
                )
                entry(style.icon, style.tint(MaterialTheme.colorScheme.surface), library.name, library.count.takeIf { it > 0 }, isRailLibrarySelected(destination, selectedLibrary?.id, library.id)) { onSelectLibrary(library) }
            }
            if (showCollections) entry(Icons.Outlined.FolderOpen, muted, "Collections", null, destination == MobileShellDestination.Collections, onOpenCollections)
            if (showSources) entry(Icons.Outlined.Cloud, muted, "Sources", null, destination == MobileShellDestination.Sources, onOpenSources)
            LocalHermesChatEntry.current?.let { chat -> entry(Icons.Outlined.ChatBubbleOutline, muted, chat.name, null, false, chat.open) }
            if (chatName != null) entry(Icons.Outlined.ChatBubbleOutline, muted, chatName, null, true) {}
            entry(Icons.Default.Settings, muted, "Settings", null, destination == MobileShellDestination.Settings, onOpenSettings)
        }
    }
}

internal val MOBILE_LIBRARY_RAIL_HEIGHT = 48.dp
internal val MOBILE_CONTENT_GUTTER = 16.dp

internal fun isRailLibrarySelected(destination: MobileShellDestination, selectedLibraryId: String?, libraryId: String): Boolean =
    destination == MobileShellDestination.Library && libraryId == selectedLibraryId
