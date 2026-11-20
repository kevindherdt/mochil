package net.kdhmedia.mochil.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.border
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.catalogue.ClientDeviceClass
import net.kdhmedia.mochil.catalogue.LibrarySummary

@Composable
fun LibraryVisibilitySettings(
    libraries: List<LibrarySummary>,
    visibleLibraryIds: Set<String>,
    deviceClass: ClientDeviceClass,
    onVisibilityChange: (LibrarySummary, Boolean) -> Unit,
    television: Boolean = false,
    showCollections: Boolean = true,
    showSources: Boolean = true,
    onDestinationVisibility: (OptionalDestination, Boolean) -> Unit = { _, _ -> },
    onMove: ((LibrarySummary, Int) -> Unit)? = null,
) {
    Text(
        when (deviceClass) {
            ClientDeviceClass.Television -> "Reading shelves are hidden by default on TV. Turn on anything you still want to browse here."
            ClientDeviceClass.EInk -> "Video shelves are hidden by default on e-ink. Turn on anything you still want to browse here."
            ClientDeviceClass.Handheld -> "Choose which server libraries appear on this device. This does not delete or disable them on the server."
        },
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        modifier = Modifier.padding(bottom = 8.dp),
    )
    if (!television) {
        DestinationVisibilityRow("Collections", "Every collection, across all libraries", showCollections) { onDestinationVisibility(OptionalDestination.Collections, it) }
        DestinationVisibilityRow("Sources", "OPDS catalogues and WebDAV libraries", showSources) { onDestinationVisibility(OptionalDestination.Sources, it) }
        Text("Server libraries", style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(start = 8.dp, top = 14.dp, bottom = 4.dp))
    }
    libraries.forEach { library ->
        val visible = library.id in visibleLibraryIds
        var focused by remember(library.id) { mutableStateOf(false) }
        val shape = RoundedCornerShape(12.dp)
        if (onMove != null) {
            // On TV the row itself is not a control; Up, Down and Show/Hide are three plain buttons the remote can reach one by one.
            Row(Modifier.fillMaxWidth().padding(vertical = 8.dp, horizontal = 8.dp), verticalAlignment = Alignment.CenterVertically) {
                Column(Modifier.weight(1f).padding(end = 12.dp)) {
                    Text(library.name, style = MaterialTheme.typography.titleMedium)
                    Text("${library.kind.ifBlank { "Library" }} · ${library.count} items", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
                TvButton("Up", { onMove(library, -1) }, enabled = library.id != libraries.first().id)
                androidx.compose.foundation.layout.Spacer(Modifier.padding(4.dp))
                TvButton("Down", { onMove(library, 1) }, enabled = library.id != libraries.last().id)
                androidx.compose.foundation.layout.Spacer(Modifier.padding(4.dp))
                TvButton(if (visible) "Shown" else "Hidden", { onVisibilityChange(library, !visible) }, primary = visible)
            }
            return@forEach
        }
        Row(
            Modifier
                .fillMaxWidth()
                .onFocusChanged { focused = it.hasFocus }
                .border(if (television && focused) 4.dp else 0.dp, if (television && focused) MaterialTheme.colorScheme.primary else Color.Transparent, shape)
                .clickable { onVisibilityChange(library, !visible) }
                .padding(vertical = if (television) 10.dp else 6.dp, horizontal = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(Modifier.weight(1f).padding(end = 12.dp)) {
                Text(library.name, style = if (television) MaterialTheme.typography.titleMedium else MaterialTheme.typography.bodyLarge)
                Text("${library.kind.ifBlank { "Library" }} · ${library.count} items", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            Switch(checked = visible, onCheckedChange = { onVisibilityChange(library, it) })
        }
    }
    if (libraries.isEmpty()) Text("Libraries will appear here after the server catalogue loads.", color = MaterialTheme.colorScheme.onSurfaceVariant)
}

@Composable
private fun DestinationVisibilityRow(title: String, detail: String, visible: Boolean, onChange: (Boolean) -> Unit) {
    Row(
        Modifier.fillMaxWidth().clickable { onChange(!visible) }.padding(vertical = 6.dp, horizontal = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(Modifier.weight(1f).padding(end = 12.dp)) {
            Text(title, style = MaterialTheme.typography.bodyLarge)
            Text(detail, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
        Switch(checked = visible, onCheckedChange = onChange)
    }
}
