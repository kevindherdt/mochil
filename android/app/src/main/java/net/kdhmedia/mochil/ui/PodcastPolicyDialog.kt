package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.foundation.selection.toggleable
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.podcast.PodcastPolicy
import net.kdhmedia.mochil.podcast.PodcastDevicePolicy
import net.kdhmedia.mochil.podcast.PodcastShow

@Composable
internal fun PodcastPolicyDialog(show: PodcastShow, devicePolicy: PodcastDevicePolicy, busy: Boolean, dismiss: () -> Unit, save: (PodcastPolicy, PodcastDevicePolicy) -> Unit) {
    var policy by remember(show.id) { mutableStateOf(show.policy) }
    var device by remember(show.id) { mutableStateOf(devicePolicy) }
    AlertDialog(
        onDismissRequest = { if (!busy) dismiss() },
        title = { Text("Download policy") },
        text = {
            Column(Modifier.heightIn(max = 520.dp).verticalScroll(rememberScrollState()), verticalArrangement = Arrangement.spacedBy(14.dp)) {
                Text(show.title, style = MaterialTheme.typography.titleSmall)
                Text("On this device", style = MaterialTheme.typography.titleSmall)
                Text(deviceSummary(device), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                SwitchRow("Download automatically", "Keep the newest episodes offline on this device", device.autoDownload, busy) { device = device.copy(autoDownload = it) }
                PolicyChoice("Keep newest", deviceRetainOptions, device.retainDownloads, { "$it episodes" }, busy) { device = device.copy(retainDownloads = it) }
                SwitchRow("Wi-Fi only", "Avoid metered mobile networks", device.unmeteredOnly, busy) { device = device.copy(unmeteredOnly = it) }
                HorizontalDivider()
                Text("On the server", style = MaterialTheme.typography.titleSmall)
                Text("The server's own podcast cache. It does not change what this device keeps.", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                SwitchRow("Cache new episodes", "Download new episodes to the server", policy.autoDownload, busy) { enabled -> policy = policy.copy(autoDownload = enabled, retainDownloads = if (enabled && policy.retainDownloads == 0) 3 else policy.retainDownloads) }
                PolicyChoice("Keep newest on server", retainOptions, policy.retainDownloads, { if (it == 0) "No retention limit" else "$it downloads" }, busy) { policy = policy.copy(retainDownloads = it, autoDownload = policy.autoDownload && it > 0) }
                PolicyChoice("Remove played", playedOptions, policy.deletePlayedAfterDays, { if (it == 0) "Never automatically" else "After $it ${if (it == 1) "day" else "days"}" }, busy) { policy = policy.copy(deletePlayedAfterDays = it) }
            }
        },
        confirmButton = { TextButton(enabled = !busy, onClick = { save(policy, device) }) { Text(if (busy) "Saving…" else "Save") } },
        dismissButton = { TextButton(enabled = !busy, onClick = dismiss) { Text("Cancel") } },
    )
}

/** One sentence for the stored device fields, so the dialog never says "manual" while automatic downloads are on. */
private fun deviceSummary(device: PodcastDevicePolicy) =
    if (!device.autoDownload) "Nothing downloads to this device by itself. You can still download episodes by hand."
    else "Keeps the newest ${device.retainDownloads} episodes on this device, ${if (device.unmeteredOnly) "Wi-Fi only" else "on any network"}."

/** The whole row toggles, so the label is as tappable as the switch (and is announced as one switch to TalkBack). */
@Composable
private fun SwitchRow(title: String, detail: String, checked: Boolean, busy: Boolean, onChange: (Boolean) -> Unit) {
    Row(Modifier.fillMaxWidth().toggleable(value = checked, enabled = !busy, role = Role.Switch, onValueChange = onChange), verticalAlignment = Alignment.CenterVertically) {
        Column(Modifier.weight(1f)) { Text(title); Text(detail, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant) }
        Switch(checked = checked, enabled = !busy, onCheckedChange = null)
    }
}

@Composable
private fun PolicyChoice(label: String, choices: List<Int>, selected: Int, valueLabel: (Int) -> String, busy: Boolean, select: (Int) -> Unit) {
    var open by remember { mutableStateOf(false) }
    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        Text(label, Modifier.weight(1f))
        Box {
            TextButton(enabled = !busy, onClick = { open = true }) { Text(valueLabel(selected)) }
            DropdownMenu(expanded = open, onDismissRequest = { open = false }) {
                choices.forEach { value -> DropdownMenuItem(text = { Text(valueLabel(value)) }, onClick = { open = false; select(value) }) }
            }
        }
    }
}

private val retainOptions = listOf(0, 1, 3, 5, 10, 20, 50)
private val playedOptions = listOf(0, 1, 3, 7, 14, 30, 90)
private val deviceRetainOptions = listOf(1, 3, 5, 10)
