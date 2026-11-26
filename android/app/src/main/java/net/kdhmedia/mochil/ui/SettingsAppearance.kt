package net.kdhmedia.mochil.ui

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.selection.toggleable
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.Role
import net.kdhmedia.mochil.ui.kit.PillItem
import net.kdhmedia.mochil.ui.kit.PillRow
import net.kdhmedia.mochil.ui.kit.SwatchItem
import net.kdhmedia.mochil.ui.kit.SwatchRow
import net.kdhmedia.mochil.ui.theme.AppPalette
import net.kdhmedia.mochil.ui.theme.AppThemeMode
import net.kdhmedia.mochil.ui.theme.AppThemeSelection
import net.kdhmedia.mochil.ui.theme.MochilSpace
import net.kdhmedia.mochil.ui.theme.backdrop
import net.kdhmedia.mochil.ui.theme.swatch

/**
 * Settings → Appearance: the four themes as round swatches (Kevin's 21 Nov ruling, sheet 6), the mode as pills, and the
 * motion switch. "Device" stays here though the web has no such mode: on a phone, following the system is expected.
 */
@Composable
internal fun AppearanceSettings(selection: AppThemeSelection, onSelection: (AppThemeSelection) -> Unit) {
    val dark = when (selection.mode) { AppThemeMode.System -> isSystemInDarkTheme(); AppThemeMode.Dark -> true; AppThemeMode.Light -> false }
    SettingsCard("Theme") {
        SwatchRow(
            AppPalette.entries.map { SwatchItem(it.name, it.label, it.backdrop(dark), it.swatch(dark)) },
            selectedId = selection.palette.name,
            onSelect = { name -> AppPalette.entries.firstOrNull { it.name == name }?.let { onSelection(selection.copy(palette = it)) } },
            modifier = Modifier.padding(vertical = MochilSpace.s4),
        )
    }
    SettingsCard("Mode") {
        PillRow(
            AppThemeMode.entries.map { PillItem(it.name, it.label) },
            activeId = selection.mode.name,
            onSelect = { name -> AppThemeMode.entries.firstOrNull { it.name == name }?.let { onSelection(selection.copy(mode = it)) } },
            modifier = Modifier.padding(horizontal = MochilSpace.s8),
        )
        Text(
            "E-ink uses maximum contrast and restrained surfaces.",
            Modifier.padding(horizontal = MochilSpace.s8, vertical = MochilSpace.s4),
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
    SettingsCard("Motion") {
        Row(
            Modifier.fillMaxWidth().toggleable(value = selection.reduceMotion, role = Role.Switch, onValueChange = { onSelection(selection.copy(reduceMotion = it)) })
                .padding(horizontal = MochilSpace.s8),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(Modifier.weight(1f).padding(end = MochilSpace.s12)) {
                Text("Reduce animations", style = MaterialTheme.typography.bodyLarge, color = MaterialTheme.colorScheme.onSurface)
                Text("Lists, panels and search open at once instead of sliding. On by default on e-ink screens.", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            // The row is the control; the switch only shows its state, so TalkBack reads the text and the state as one.
            Switch(checked = selection.reduceMotion, onCheckedChange = null)
        }
    }
}
