package net.kdhmedia.mochil.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.automirrored.filled.Sort
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.unit.dp

internal data class BrowseModeOption<T>(val value: T, val label: String)

@Composable
internal fun <T> BrowseModeMenu(selected: T, options: List<BrowseModeOption<T>>, noun: String, onSelected: (T) -> Unit) {
    var expanded by remember { mutableStateOf(false) }
    val selectedLabel = options.firstOrNull { it.value == selected }?.label.orEmpty()
    Box {
        // Icon alone used to carry the current mode only in its content description (screen-reader-only);
        // a sighted user had no on-screen cue which of several browse modes was active. Show it as visible
        // text too, matching the pattern already used elsewhere (e.g. MobileLibraryRail's current-place label).
        Row(
            Modifier
                .clickable(onClickLabel = "$noun browse mode", role = Role.Button, onClick = { expanded = true })
                .heightIn(min = 48.dp)
                .padding(horizontal = 8.dp, vertical = 6.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(Icons.AutoMirrored.Filled.Sort, contentDescription = null, tint = MaterialTheme.colorScheme.onSurfaceVariant)
            Text(selectedLabel, modifier = Modifier.padding(start = 6.dp), style = MaterialTheme.typography.labelLarge, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
        DropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
            options.forEach { option ->
                DropdownMenuItem(
                    text = { Text(option.label) },
                    onClick = { expanded = false; if (option.value != selected) onSelected(option.value) },
                    leadingIcon = { if (option.value == selected) Icon(Icons.Default.Check, contentDescription = null) },
                )
            }
        }
    }
}
