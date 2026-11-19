package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.FilterChip
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.playback.PlaybackQuality
import net.kdhmedia.mochil.playback.PlaybackQualityPolicy
import net.kdhmedia.mochil.playback.PlaybackQualityPreferences

@Composable
internal fun PlaybackQualitySettings(television: Boolean = false) {
    val context = LocalContext.current
    val preferences = remember(context) { PlaybackQualityPreferences(context) }
    var policy by remember { mutableStateOf(preferences.load()) }
    Column(Modifier.fillMaxWidth().padding(horizontal = if (television) 0.dp else 8.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Text("Streaming quality", style = MaterialTheme.typography.titleMedium)
        QualityChoice(if (television) "Unmetered (usually Wi-Fi or Ethernet)" else "Wi-Fi", policy.wifi, television) { value ->
            policy = policy.copy(wifi = value).also(preferences::save)
        }
        QualityChoice(if (television) "Metered (including cellular)" else "Mobile data", policy.mobile, television) { value ->
            policy = policy.copy(mobile = value).also(preferences::save)
        }
        if (television) {
            Text("Android selects the profile by whether the network is metered, not its connection type. If network information is unavailable, the unmetered profile is used.", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
        Text("Maximum keeps the original stream when supported. Lower profiles allow Mochil to reduce network use.", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
    }
}

@Composable
private fun QualityChoice(label: String, selected: PlaybackQuality, television: Boolean, onSelected: (PlaybackQuality) -> Unit) {
    Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
        Text(label, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
        if (television) TvSettingsChoiceRail(
            choices = PlaybackQuality.entries,
            selected = { selected == it },
            label = PlaybackQuality::label,
            key = PlaybackQuality::name,
            onSelect = onSelected,
            modifier = Modifier.fillMaxWidth(),
        ) else Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            PlaybackQuality.entries.forEach { quality ->
                FilterChip(selected = selected == quality, onClick = { onSelected(quality) }, label = { Text(quality.label) })
            }
        }
    }
}
